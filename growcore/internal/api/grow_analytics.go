package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

// Grow analytics interpret the grow's data in context — stage durations, care
// cadence, where it lived, and how well its environment tracked target. Heavy
// raw series stay on the history endpoints; this returns aggregates only.

type stageDuration struct {
	Stage string     `json:"stage"`
	From  time.Time  `json:"from"`
	To    *time.Time `json:"to,omitempty"` // nil = still in this stage
	Days  float64    `json:"days"`
}

type careWeek struct {
	WeekStart time.Time `json:"weekStart"`
	Count     int       `json:"count"`
	FeedCount int       `json:"feedCount"`
	TotalML   float64   `json:"totalMl"`
}

type placementSpan struct {
	EnvironmentID   string     `json:"environmentId"`
	EnvironmentName string     `json:"environmentName"`
	From            time.Time  `json:"from"`
	To              *time.Time `json:"to,omitempty"`
}

type growAnalytics struct {
	StageDurations []stageDuration `json:"stageDurations"`
	CareByWeek     []careWeek      `json:"careByWeek"`
	CareFrequency  map[string]int  `json:"careFrequency"`
	Placements     []placementSpan `json:"placements"`
	PctInTarget    *float64        `json:"pctInTarget,omitempty"` // % of samples within temp+humidity target
	SampleCount    int             `json:"sampleCount"`
}

func (s *Server) getGrowAnalytics(w http.ResponseWriter, r *http.Request) {
	growID := r.PathValue("id")
	grow, ok, err := s.store.Grow(growID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	now := time.Now()
	out := growAnalytics{
		StageDurations: []stageDuration{},
		CareByWeek:     []careWeek{},
		CareFrequency:  map[string]int{},
		Placements:     []placementSpan{},
	}

	// --- stage durations (backfill from current stage if no history yet) ---
	events, err := s.store.StageEvents(growID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	type se struct {
		stage string
		at    time.Time
	}
	spans := make([]se, 0, len(events))
	for _, e := range events {
		spans = append(spans, se{e.Stage, e.EnteredAt})
	}
	if len(spans) == 0 && grow.Stage != "" {
		spans = append(spans, se{grow.Stage, grow.StageStarted})
	}
	for i, sp := range spans {
		var to *time.Time
		end := now
		if i+1 < len(spans) {
			t := spans[i+1].at
			to = &t
			end = t
		}
		out.StageDurations = append(out.StageDurations, stageDuration{
			Stage: sp.stage, From: sp.at, To: to,
			Days: end.Sub(sp.at).Hours() / 24,
		})
	}

	// --- care cadence ---
	events2, err := s.store.CareEvents(growID, 500, 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	weekBuckets := map[int64]*careWeek{}
	for _, ev := range events2 {
		out.CareFrequency[ev.Type]++
		wk := weekStart(ev.OccurredAt)
		b := weekBuckets[wk.UnixMilli()]
		if b == nil {
			b = &careWeek{WeekStart: wk}
			weekBuckets[wk.UnixMilli()] = b
		}
		b.Count++
		if ev.Type == "feed" {
			b.FeedCount++
		}
		b.TotalML += ev.TotalML()
	}
	for _, b := range weekBuckets {
		out.CareByWeek = append(out.CareByWeek, *b)
	}
	sort.Slice(out.CareByWeek, func(i, j int) bool {
		return out.CareByWeek[i].WeekStart.Before(out.CareByWeek[j].WeekStart)
	})

	// --- placement history + environments the grow occupied ---
	units, err := s.store.PlantUnits(growID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	unitIDs := make([]string, len(units))
	for i, u := range units {
		unitIDs[i] = u.ID
	}
	placementsByUnit, err := s.store.PlacementsForUnits(unitIDs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	envName := map[string]string{}
	if envs, err := s.store.Environments(); err == nil {
		for _, e := range envs {
			envName[e.ID] = e.Name
		}
	}
	envSet := map[string]bool{}
	for _, ps := range placementsByUnit {
		for _, p := range ps {
			envSet[p.EnvironmentID] = true
			out.Placements = append(out.Placements, placementSpan{
				EnvironmentID: p.EnvironmentID, EnvironmentName: envName[p.EnvironmentID],
				From: p.StartedAt, To: p.EndedAt,
			})
		}
	}
	sort.Slice(out.Placements, func(i, j int) bool {
		return out.Placements[i].From.Before(out.Placements[j].From)
	})

	// --- % of climate samples within target, across occupied environments ---
	envTargets := map[string]domain.Environment{}
	if envs, err := s.store.Environments(); err == nil {
		for _, e := range envs {
			envTargets[e.ID] = e
		}
	}
	var inTarget, total int
	for envID := range envSet {
		env, ok := envTargets[envID]
		if !ok {
			continue
		}
		readings, err := s.store.ReadingsSince(envID, grow.StartedAt, 400)
		if err != nil {
			continue
		}
		for _, rd := range readings {
			if rd.TempC == 0 && rd.Humidity == 0 {
				continue
			}
			total++
			if withinTarget(rd.TempC, env.TargetTempC, env.TargetTempMinC, env.TargetTempMaxC, 1.5) &&
				withinTarget(rd.Humidity, env.TargetHumidity, env.TargetHumidityMin, env.TargetHumidityMax, 5) {
				inTarget++
			}
		}
	}
	out.SampleCount = total
	if total > 0 {
		pct := float64(inTarget) / float64(total) * 100
		out.PctInTarget = &pct
	}

	writeJSON(w, http.StatusOK, out)
}

// withinTarget reports whether v sits inside [min,max] when a range is set, else
// within tol of the single setpoint.
func withinTarget(v, setpoint, min, max, tol float64) bool {
	if min > 0 && max > 0 && max > min {
		return v >= min && v <= max
	}
	if setpoint > 0 {
		return v >= setpoint-tol && v <= setpoint+tol
	}
	return true
}

// weekStart returns local Monday 00:00 for t's week.
func weekStart(t time.Time) time.Time {
	y, m, d := t.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	// Go's Weekday: Sunday=0..Saturday=6; shift so Monday is the week's first day.
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}
