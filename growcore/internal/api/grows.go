package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

// --- Grows ---

func (s *Server) getGrows(w http.ResponseWriter, r *http.Request) {
	grows, err := s.store.Grows()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if grows == nil {
		grows = []domain.Grow{}
	}
	writeJSON(w, http.StatusOK, grows)
}

func (s *Server) getStagePresets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.StagePresets)
}

type growBody struct {
	Name      string `json:"name"`
	Species   string `json:"species"`
	StartedAt string `json:"startedAt"` // RFC3339 or YYYY-MM-DD; empty = now
	Notes     string `json:"notes"`
}

// speciesStages validates a species against the predefined presets and returns
// its (auto-derived) stage sequence. Species is the single source of truth for
// stages; a grow cannot use an unknown crop family.
func speciesStages(species string) (key string, stages []string, ok bool) {
	key = strings.ToLower(strings.TrimSpace(species))
	stages, ok = domain.StagePresets[key]
	return key, stages, ok
}

func (s *Server) createGrow(w http.ResponseWriter, r *http.Request) {
	var b growBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(b.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errBody("name is required"))
		return
	}
	species, stages, ok := speciesStages(b.Species)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errBody("species must be one of the predefined crop families"))
		return
	}
	now := time.Now()
	grow := domain.Grow{
		ID:           id(b.Name, "grow"),
		Name:         strings.TrimSpace(b.Name),
		Species:      species,
		Stage:        stages[0],
		Stages:       stages,
		StartedAt:    parseDate(b.StartedAt),
		StageStarted: now,
		Status:       domain.GrowActive,
		Notes:        b.Notes,
	}
	if err := s.store.SaveGrow(grow); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "configuration", "Created grow "+grow.Name)
	writeJSON(w, http.StatusOK, grow)
}

func (s *Server) updateGrow(w http.ResponseWriter, r *http.Request) {
	grow, ok, err := s.store.Grow(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	var b growBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(b.Name) != "" {
		grow.Name = strings.TrimSpace(b.Name)
	}
	species, stages, ok := speciesStages(b.Species)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errBody("species must be one of the predefined crop families"))
		return
	}
	grow.Species = species
	grow.Stages = stages
	if b.StartedAt != "" {
		grow.StartedAt = parseDate(b.StartedAt)
	}
	grow.Notes = b.Notes
	// Species (hence the stage sequence) may have changed; keep the current stage
	// valid against the derived sequence.
	if !contains(grow.Stages, grow.Stage) {
		grow.Stage = grow.Stages[0]
	}
	if err := s.store.SaveGrow(grow); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, grow)
}

func (s *Server) deleteGrow(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteGrow(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type stageBody struct {
	Stage string `json:"stage"`
}

func (s *Server) changeStage(w http.ResponseWriter, r *http.Request) {
	grow, ok, err := s.store.Grow(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	var b stageBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	stage := strings.TrimSpace(b.Stage)
	if !contains(grow.Stages, stage) {
		writeJSON(w, http.StatusBadRequest, errBody("stage is not part of this grow's sequence"))
		return
	}
	grow.Stage = stage
	grow.StageStarted = time.Now()
	if err := s.store.SaveGrow(grow); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "configuration", grow.Name+" advanced to "+stage)
	writeJSON(w, http.StatusOK, grow)
}

func (s *Server) completeGrow(w http.ResponseWriter, r *http.Request) {
	grow, ok, err := s.store.Grow(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	grow.Status = domain.GrowCompleted
	if err := s.store.SaveGrow(grow); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "configuration", "Completed grow "+grow.Name)
	writeJSON(w, http.StatusOK, grow)
}

// --- Grow detail (grow + plants + placements) ---

type placementView struct {
	domain.PlantPlacement
	EnvironmentName string `json:"environmentName"`
}

type plantDetail struct {
	domain.PlantUnit
	CurrentEnvironmentID   string           `json:"currentEnvironmentId"`
	CurrentEnvironmentName string           `json:"currentEnvironmentName"`
	Placements             []placementView  `json:"placements"`
}

type growDetail struct {
	domain.Grow
	StageDays  int           `json:"stageDays"`
	TotalDays  int           `json:"totalDays"`
	PlantCount int           `json:"plantCount"`
	Plants     []plantDetail `json:"plants"`
}

func (s *Server) getGrow(w http.ResponseWriter, r *http.Request) {
	grow, ok, err := s.store.Grow(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	envName, err := s.environmentNames()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	units, err := s.store.PlantUnits(grow.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now()
	detail := growDetail{
		Grow:      grow,
		StageDays: domain.DaysSince(grow.StageStarted, now),
		TotalDays: domain.DaysSince(grow.StartedAt, now),
		Plants:    []plantDetail{},
	}
	for _, u := range units {
		if u.Status == domain.PlantActive {
			detail.PlantCount += u.Quantity
		}
		placements, err := s.store.PlacementsForUnit(u.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		pd := plantDetail{PlantUnit: u, Placements: []placementView{}}
		for _, p := range placements {
			pd.Placements = append(pd.Placements, placementView{PlantPlacement: p, EnvironmentName: envName[p.EnvironmentID]})
			if p.EndedAt == nil {
				pd.CurrentEnvironmentID = p.EnvironmentID
				pd.CurrentEnvironmentName = envName[p.EnvironmentID]
			}
		}
		detail.Plants = append(detail.Plants, pd)
	}
	writeJSON(w, http.StatusOK, detail)
}

// --- Plants ---

type plantView struct {
	domain.PlantUnit
	GrowName               string          `json:"growName"`
	CurrentEnvironmentID   string          `json:"currentEnvironmentId"`
	CurrentEnvironmentName string          `json:"currentEnvironmentName"`
	Placements             []placementView `json:"placements"`
}

func (s *Server) getPlant(w http.ResponseWriter, r *http.Request) {
	unit, ok, err := s.store.PlantUnit(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("plant not found"))
		return
	}
	envName, err := s.environmentNames()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	placements, err := s.store.PlacementsForUnit(unit.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	pv := plantView{PlantUnit: unit, Placements: []placementView{}}
	if grow, ok, _ := s.store.Grow(unit.GrowID); ok {
		pv.GrowName = grow.Name
	}
	for _, p := range placements {
		pv.Placements = append(pv.Placements, placementView{PlantPlacement: p, EnvironmentName: envName[p.EnvironmentID]})
		if p.EndedAt == nil {
			pv.CurrentEnvironmentID = p.EnvironmentID
			pv.CurrentEnvironmentName = envName[p.EnvironmentID]
		}
	}
	writeJSON(w, http.StatusOK, pv)
}

type bulkPlantsBody struct {
	Count         int                 `json:"count"`
	Tracking      domain.TrackingMode `json:"tracking"`
	QuantityPer   int                 `json:"quantityPer"`
	Label         string              `json:"label"`
	Cultivar      string              `json:"cultivar"`
	EnvironmentID string              `json:"environmentId"`
}

func (s *Server) createPlants(w http.ResponseWriter, r *http.Request) {
	grow, ok, err := s.store.Grow(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	var b bulkPlantsBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if b.Count <= 0 {
		writeJSON(w, http.StatusBadRequest, errBody("count must be positive"))
		return
	}
	tracking := b.Tracking
	if tracking != domain.TrackIndividual && tracking != domain.TrackGroup {
		tracking = domain.TrackGroup
	}
	if b.EnvironmentID != "" {
		envs, err := s.store.Environments()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !containsEnv(envs, b.EnvironmentID) {
			writeJSON(w, http.StatusBadRequest, errBody("environment not found"))
			return
		}
	}
	label := strings.TrimSpace(b.Label)
	if label == "" {
		if tracking == domain.TrackGroup {
			label = "Group"
		} else {
			label = "Plant"
		}
	}
	units, err := s.store.BulkCreatePlants(grow.ID, b.Count, tracking, b.QuantityPer, label, strings.TrimSpace(b.Cultivar), b.EnvironmentID, time.Now())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity(b.EnvironmentID, "", "info", "configuration", "Added plants to "+grow.Name)
	writeJSON(w, http.StatusOK, units)
}

type updatePlantBody struct {
	Label    string `json:"label"`
	Cultivar string `json:"cultivar"`
	Quantity *int   `json:"quantity"` // pointer: omitted leaves quantity unchanged
}

// updatePlant edits a plant unit's per-unit attributes (label, cultivar, and —
// for groups — quantity). Placement, grow and status are changed via their own
// endpoints.
func (s *Server) updatePlant(w http.ResponseWriter, r *http.Request) {
	unit, ok, err := s.store.PlantUnit(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("plant not found"))
		return
	}
	var b updatePlantBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	unit.Label = strings.TrimSpace(b.Label)
	unit.Cultivar = strings.TrimSpace(b.Cultivar)
	if b.Quantity != nil && unit.Tracking == domain.TrackGroup && *b.Quantity > 0 {
		unit.Quantity = *b.Quantity
	}
	if err := s.store.SavePlantUnit(unit); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "configuration", "Updated "+plantLabel(unit))
	writeJSON(w, http.StatusOK, unit)
}

type moveBody struct {
	EnvironmentID string `json:"environmentId"`
}

func (s *Server) movePlant(w http.ResponseWriter, r *http.Request) {
	unit, ok, err := s.store.PlantUnit(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("plant not found"))
		return
	}
	var b moveBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	envs, err := s.store.Environments()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !containsEnv(envs, b.EnvironmentID) {
		writeJSON(w, http.StatusBadRequest, errBody("environment not found"))
		return
	}
	if err := s.store.MovePlant(unit.ID, b.EnvironmentID, time.Now()); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity(b.EnvironmentID, "", "info", "configuration", "Moved "+plantLabel(unit)+" here")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) setPlantStatus(status domain.PlantStatus, verb string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		unit, ok, err := s.store.PlantUnit(r.PathValue("id"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, errBody("plant not found"))
			return
		}
		unit.Status = status
		if err := s.store.SavePlantUnit(unit); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		s.activity("", "", "info", "configuration", verb+" "+plantLabel(unit))
		writeJSON(w, http.StatusOK, unit)
	}
}

func plantLabel(u domain.PlantUnit) string {
	if u.Label != "" {
		return u.Label
	}
	return "plant"
}

// --- Environment occupancy & control grow ---

type envPlantsGroup struct {
	Grow  domain.Grow        `json:"grow"`
	Units []domain.PlantUnit `json:"units"`
}

func (s *Server) getEnvironmentPlants(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("id")
	units, err := s.store.PlantsInEnvironment(envID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	byGrow := map[string][]domain.PlantUnit{}
	order := []string{}
	for _, u := range units {
		if _, seen := byGrow[u.GrowID]; !seen {
			order = append(order, u.GrowID)
		}
		byGrow[u.GrowID] = append(byGrow[u.GrowID], u)
	}
	groups := []envPlantsGroup{}
	for _, growID := range order {
		grow, ok, err := s.store.Grow(growID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			continue
		}
		groups = append(groups, envPlantsGroup{Grow: grow, Units: byGrow[growID]})
	}
	writeJSON(w, http.StatusOK, groups)
}

type controlGrowBody struct {
	GrowID string `json:"growId"`
}

func (s *Server) putControlGrow(w http.ResponseWriter, r *http.Request) {
	envID := r.PathValue("id")
	envs, err := s.store.Environments()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	env, ok := findEnvByID(envs, envID)
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("environment not found"))
		return
	}
	var b controlGrowBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if b.GrowID != "" {
		if _, exists, err := s.store.Grow(b.GrowID); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		} else if !exists {
			writeJSON(w, http.StatusBadRequest, errBody("grow not found"))
			return
		}
	}
	env.ControlGrowID = b.GrowID
	if err := s.store.SaveEnvironment(env); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if b.GrowID == "" {
		s.activity(envID, "", "info", "configuration", "Cleared control grow")
	} else {
		s.activity(envID, "", "info", "configuration", "Set control grow")
	}
	writeJSON(w, http.StatusOK, env)
}

// environmentNames returns a map of environment id -> display name.
func (s *Server) environmentNames() (map[string]string, error) {
	envs, err := s.store.Environments()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(envs))
	for _, e := range envs {
		m[e.ID] = e.Name
	}
	return m, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
