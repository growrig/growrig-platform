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

// SetStageDate records (or corrects) the date a grow entered a given stage.
// Stages are strictly ordered and entered once, so this keeps a single event
// per (grow, stage): any existing one is replaced.
func (s *Store) SetStageDate(growID, stage string, enteredAt time.Time) error {
	if enteredAt.IsZero() {
		enteredAt = time.Now()
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM stage_events WHERE grow_id = ? AND stage = ?`, growID, stage); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO stage_events (id, grow_id, stage, entered_at) VALUES (?, ?, ?, ?)`,
		newID("stage"), growID, stage, enteredAt.UnixMilli()); err != nil {
		return err
	}
	return tx.Commit()
}

// ClearStageDate removes a grow's recorded entry date for a stage (the stage
// reverts to being predicted). Used both to blank a date and to discard stages
// that were entered by mistake when reverting.
func (s *Store) ClearStageDate(growID, stage string) error {
	_, err := s.db.Exec(`DELETE FROM stage_events WHERE grow_id = ? AND stage = ?`, growID, stage)
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
