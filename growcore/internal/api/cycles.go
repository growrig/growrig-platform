package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
	"github.com/growrig/growrig-platform/growcore/internal/sim"
)

type cycleBody struct {
	Strain    string       `json:"strain"`
	StartedAt string       `json:"startedAt"` // RFC3339 or YYYY-MM-DD; empty = now
	Phase     domain.Phase `json:"phase"`
	Notes     string       `json:"notes"`
}

func (s *Server) putCycle(w http.ResponseWriter, r *http.Request) {
	var b cycleBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if !validPhase(b.Phase) {
		writeJSON(w, http.StatusBadRequest, errBody("unknown phase"))
		return
	}
	envs, err := s.store.Environments()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	id := r.PathValue("id")
	if !containsEnv(envs, id) {
		writeJSON(w, http.StatusNotFound, errBody("environment not found"))
		return
	}
	cycle := domain.Cycle{
		EnvironmentID: id,
		Strain:        b.Strain,
		StartedAt:     parseDate(b.StartedAt),
		Phase:         b.Phase,
		PhaseStarted:  time.Now(),
		Notes:         b.Notes,
	}
	if err := s.store.SaveCycle(cycle); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity(id, "", "info", "configuration", "Started or updated grow cycle for "+cycle.Strain)
	writeJSON(w, http.StatusOK, cycle)
}

func (s *Server) deleteCycle(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("id")
	if err := s.store.DeleteCycle(envID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity(envID, "", "info", "configuration", "Ended grow cycle")
	w.WriteHeader(http.StatusNoContent)
}

// postDemo seeds the simulator's demo tent + lung room into an empty database,
// so users can explore the app instantly. Only valid in simulator mode.
func (s *Server) postDemo(w http.ResponseWriter, r *http.Request) {
	if s.adapterType != "simulator" {
		writeJSON(w, http.StatusConflict, errBody("demo data is only available with the simulator adapter"))
		return
	}
	envs, err := s.store.Environments()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if len(envs) > 0 {
		writeJSON(w, http.StatusConflict, errBody("environments already exist"))
		return
	}
	demoEnvs, bindings := sim.SeedTopology()
	for _, env := range demoEnvs {
		if err := s.store.SaveEnvironment(env); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	for _, b := range bindings {
		if err := s.store.SaveBinding(b); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	// A sample cycle so the demo shows the full picture.
	_ = s.store.SaveCycle(domain.Cycle{
		EnvironmentID: sim.TentID, Strain: "Demo Kush",
		StartedAt: time.Now().AddDate(0, 0, -21), Phase: domain.PhaseVegetative, PhaseStarted: time.Now().AddDate(0, 0, -7),
	})
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) getSchedule(w http.ResponseWriter, r *http.Request) {
	sched, _, err := s.store.LightSchedule(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, sched)
}

type scheduleBody struct {
	Mode         domain.LightScheduleMode `json:"mode"`
	LightsOnAt   string                   `json:"lightsOnAt"`
	OnHours      float64                  `json:"onHours"`
	PhaseOnHours map[domain.Phase]float64 `json:"phaseOnHours"`
}

func (s *Server) putSchedule(w http.ResponseWriter, r *http.Request) {
	var b scheduleBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if !validScheduleMode(b.Mode) {
		writeJSON(w, http.StatusBadRequest, errBody("unknown schedule mode"))
		return
	}
	if b.LightsOnAt != "" && !validHHMM(b.LightsOnAt) {
		writeJSON(w, http.StatusBadRequest, errBody("lightsOnAt must be HH:MM"))
		return
	}
	envs, err := s.store.Environments()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	id := r.PathValue("id")
	if !containsEnv(envs, id) {
		writeJSON(w, http.StatusNotFound, errBody("environment not found"))
		return
	}
	phaseOn := map[domain.Phase]float64{}
	for p, h := range b.PhaseOnHours {
		if validPhase(p) {
			phaseOn[p] = h
		}
	}
	onAt := strings.TrimSpace(b.LightsOnAt)
	if onAt == "" {
		onAt = "06:00"
	}
	sched := domain.LightSchedule{
		EnvironmentID: id,
		Mode:          b.Mode,
		LightsOnAt:    onAt,
		OnHours:       b.OnHours,
		PhaseOnHours:  phaseOn,
	}
	if err := s.store.SaveLightSchedule(sched); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity(id, "", "info", "configuration", "Updated light schedule")
	writeJSON(w, http.StatusOK, sched)
}

// getLightingDefaults returns the recommended photoperiod (hours of light) for
// each grow phase, used to seed the schedule editor.
func (s *Server) getLightingDefaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.PhotoperiodDefaults)
}

func validScheduleMode(m domain.LightScheduleMode) bool {
	for _, x := range domain.AllLightScheduleModes {
		if x == m {
			return true
		}
	}
	return false
}

func validHHMM(s string) bool {
	var h, m int
	if n, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil || n != 2 {
		return false
	}
	return h >= 0 && h <= 23 && m >= 0 && m <= 59
}

func validPhase(p domain.Phase) bool {
	for _, x := range domain.AllPhases {
		if x == p {
			return true
		}
	}
	return false
}

// parseDate accepts RFC3339 or YYYY-MM-DD, defaulting to now.
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return time.Now()
}
