package api

import (
	"net/http"
	"strings"

	"github.com/growrig/growrig/growcore/internal/store"
)

// Stage history: the date a grow entered each stage of its (fixed, ordered)
// plan. The timeline, stage durations and graph bands all derive from these,
// so they can be corrected retrospectively — a stage switch logged late, or on
// the wrong day. Stages not yet reached have no date (they stay predicted from
// species estimates) unless the grower fills one in by hand.

// getStageEvents returns a grow's recorded stage-entry dates, oldest first.
func (s *Server) getStageEvents(w http.ResponseWriter, r *http.Request) {
	growID := r.PathValue("id")
	if _, ok, err := s.store.Grow(growID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	} else if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	events, err := s.store.StageEvents(growID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

type stageDatesBody struct {
	// Dates maps stage name -> entry date (RFC3339 or YYYY-MM-DD). An empty
	// string clears that stage's date (back to predicted). Stages absent from
	// the map are left untouched.
	Dates map[string]string `json:"dates"`
}

// putStageDates edits a grow's recorded stage-entry dates in bulk (the "update
// dates" editor). It never changes which stage is current — that stays with the
// advance/revert control — but keeps the current stage's StageStarted in sync.
func (s *Server) putStageDates(w http.ResponseWriter, r *http.Request) {
	grow, ok, err := s.store.Grow(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	var b stageDatesBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	for stage, iso := range b.Dates {
		if !contains(grow.Stages, stage) {
			writeJSON(w, http.StatusBadRequest, errBody("stage is not part of this grow's sequence"))
			return
		}
		if strings.TrimSpace(iso) == "" {
			if err := s.store.ClearStageDate(grow.ID, stage); err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
		} else if err := s.store.SetStageDate(grow.ID, stage, parseDate(iso)); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	events, err := s.store.StageEvents(grow.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// Keep StageStarted aligned with the current stage's (possibly edited) date.
	if e, ok := stageEventFor(events, grow.Stage); ok && !grow.StageStarted.Equal(e.EnteredAt) {
		grow.StageStarted = e.EnteredAt
		if err := s.store.SaveGrow(grow); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	s.growActivity(grow.ID, "", "info", "configuration", "Edited stage dates of "+grow.Name)
	writeJSON(w, http.StatusOK, events)
}

// stageEventFor returns the recorded event for a stage, if any.
func stageEventFor(events []store.StageEvent, stage string) (store.StageEvent, bool) {
	for _, e := range events {
		if e.Stage == stage {
			return e, true
		}
	}
	return store.StageEvent{}, false
}
