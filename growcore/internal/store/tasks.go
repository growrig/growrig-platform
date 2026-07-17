package store

import (
	"database/sql"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

const taskCols = `id, grow_id, environment_id, plant_unit_id, action_type, title, due_at, status, source, completed_care_event_id, created_at, completed_at`

func scanTask(scan func(dst ...any) error) (domain.Task, error) {
	var t domain.Task
	var created int64
	var due, completed sql.NullInt64
	if err := scan(&t.ID, &t.GrowID, &t.EnvironmentID, &t.PlantUnitID, &t.ActionType, &t.Title,
		&due, &t.Status, &t.Source, &t.CompletedCareEventID, &created, &completed); err != nil {
		return domain.Task{}, err
	}
	t.CreatedAt = time.UnixMilli(created)
	if due.Valid {
		d := time.UnixMilli(due.Int64)
		t.DueAt = &d
	}
	if completed.Valid {
		c := time.UnixMilli(completed.Int64)
		t.CompletedAt = &c
	}
	return t, nil
}

func nullMillis(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UnixMilli()
}

// CreateTask persists a new task, generating an id and defaults where absent.
func (s *Store) CreateTask(t domain.Task) (domain.Task, error) {
	if t.ID == "" {
		t.ID = newID("task")
	}
	if t.Status == "" {
		t.Status = domain.TaskOpen
	}
	if t.Source == "" {
		t.Source = domain.TaskManual
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO tasks (`+taskCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.GrowID, t.EnvironmentID, t.PlantUnitID, t.ActionType, t.Title,
		nullMillis(t.DueAt), t.Status, t.Source, t.CompletedCareEventID,
		t.CreatedAt.UnixMilli(), nullMillis(t.CompletedAt))
	if err != nil {
		return domain.Task{}, err
	}
	return t, nil
}

// Task returns a single task by id.
func (s *Store) Task(id string) (domain.Task, error) {
	return scanTask(s.db.QueryRow(`SELECT `+taskCols+` FROM tasks WHERE id = ?`, id).Scan)
}

// ListTasks returns tasks, newest-created first, optionally filtered by status
// (empty status = all).
func (s *Store) ListTasks(status domain.TaskStatus) ([]domain.Task, error) {
	query := `SELECT ` + taskCols + ` FROM tasks`
	var args []any
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DueTasks returns open tasks due at or before the given cutoff (overdue and
// due-now), soonest first. Tasks without a due date are excluded.
func (s *Store) DueTasks(before time.Time) ([]domain.Task, error) {
	rows, err := s.db.Query(
		`SELECT `+taskCols+` FROM tasks WHERE status = 'open' AND due_at IS NOT NULL AND due_at <= ? ORDER BY due_at ASC`,
		before.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CompleteTask marks a task completed, linking the care event it produced.
func (s *Store) CompleteTask(id, careEventID string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET status = 'completed', completed_at = ?, completed_care_event_id = ? WHERE id = ? AND status = 'open'`,
		time.Now().UnixMilli(), careEventID, id)
	return err
}

// SkipTask marks a task skipped without recording any care.
func (s *Store) SkipTask(id string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET status = 'skipped', completed_at = ? WHERE id = ? AND status = 'open'`,
		time.Now().UnixMilli(), id)
	return err
}
