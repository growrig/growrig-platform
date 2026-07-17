package store

import (
	"time"
)

// StageEvent is one entry in a grow's stage history: the moment it entered a
// given stage. Durations, timeline milestones and graph stage-bands derive from
// the ordered sequence.
type StageEvent struct {
	ID        string    `json:"id"`
	GrowID    string    `json:"growId"`
	Stage     string    `json:"stage"`
	EnteredAt time.Time `json:"enteredAt"`
}

// AddStageEvent appends a stage transition to a grow's history.
func (s *Store) AddStageEvent(growID, stage string, enteredAt time.Time) error {
	if enteredAt.IsZero() {
		enteredAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO stage_events (id, grow_id, stage, entered_at) VALUES (?, ?, ?, ?)`,
		newID("stage"), growID, stage, enteredAt.UnixMilli())
	return err
}

// StageEvents returns a grow's stage history, oldest first.
func (s *Store) StageEvents(growID string) ([]StageEvent, error) {
	rows, err := s.db.Query(
		`SELECT id, grow_id, stage, entered_at FROM stage_events WHERE grow_id = ? ORDER BY entered_at ASC, rowid ASC`,
		growID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StageEvent{}
	for rows.Next() {
		var e StageEvent
		var entered int64
		if err := rows.Scan(&e.ID, &e.GrowID, &e.Stage, &entered); err != nil {
			return nil, err
		}
		e.EnteredAt = time.UnixMilli(entered)
		out = append(out, e)
	}
	return out, rows.Err()
}
