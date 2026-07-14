package api

import (
	"net/http"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

// The calendar aggregates the care journal across every grow into one dated
// feed, so the web calendar can plot what happened (and, with the grows from the
// live snapshot, what is coming up) without fetching each grow individually.

// calendarEvent is one care action on the calendar: a lightweight, cross-grow
// projection of a CareEvent carrying just what the month/day views render.
type calendarEvent struct {
	ID         string            `json:"id"`
	GrowID     string            `json:"growId"`
	GrowName   string            `json:"growName"`
	Type       string            `json:"type"` // care action key (water, feed, …)
	OccurredAt time.Time         `json:"occurredAt"`
	Source     domain.CareSource `json:"source"`
	PlantCount int               `json:"plantCount"`
	TotalML    float64           `json:"totalMl,omitempty"`
	RecipeName string            `json:"recipeName,omitempty"`
	Notes      string            `json:"notes,omitempty"`
}

type calendarResponse struct {
	From   *time.Time      `json:"from,omitempty"`
	To     *time.Time      `json:"to,omitempty"`
	Events []calendarEvent `json:"events"`
}

// getCalendar returns care events across all grows within an optional [from, to]
// window (YYYY-MM-DD or RFC3339). Absent bounds default to a wide window around
// now so a first paint has context without the client computing dates.
func (s *Server) getCalendar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var from, to time.Time
	if v := q.Get("from"); v != "" {
		from = parseDate(v)
	} else {
		from = time.Now().AddDate(0, -3, 0)
	}
	if v := q.Get("to"); v != "" {
		// Include the whole "to" day by pushing to its end.
		to = parseDate(v).Add(24*time.Hour - time.Millisecond)
	} else {
		to = time.Now().AddDate(0, 1, 0)
	}

	events, err := s.store.AllCareEvents(from, to, 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	grows, err := s.store.Grows()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	growName := make(map[string]string, len(grows))
	for _, g := range grows {
		growName[g.ID] = g.Name
	}

	out := calendarResponse{From: &from, To: &to, Events: make([]calendarEvent, 0, len(events))}
	for _, e := range events {
		out.Events = append(out.Events, calendarEvent{
			ID:         e.ID,
			GrowID:     e.GrowID,
			GrowName:   growName[e.GrowID],
			Type:       e.Type,
			OccurredAt: e.OccurredAt,
			Source:     e.Source,
			PlantCount: len(e.Applications),
			TotalML:    e.TotalML(),
			RecipeName: s.recipeName(e.RecipeID),
			Notes:      e.Notes,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
