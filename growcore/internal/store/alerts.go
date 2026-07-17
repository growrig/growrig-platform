package store

import (
	"database/sql"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

const alertCols = `id, key, environment_id, grow_id, device_id, severity, kind, title, message, status, first_seen_at, last_seen_at, acknowledged_at, resolved_at`

func scanAlert(scan func(dst ...any) error) (domain.Alert, error) {
	var a domain.Alert
	var first, last int64
	var ack, resolved sql.NullInt64
	if err := scan(&a.ID, &a.Key, &a.EnvironmentID, &a.GrowID, &a.DeviceID,
		&a.Severity, &a.Kind, &a.Title, &a.Message, &a.Status, &first, &last, &ack, &resolved); err != nil {
		return domain.Alert{}, err
	}
	a.FirstSeenAt = time.UnixMilli(first)
	a.LastSeenAt = time.UnixMilli(last)
	if ack.Valid {
		t := time.UnixMilli(ack.Int64)
		a.AcknowledgedAt = &t
	}
	if resolved.Valid {
		t := time.UnixMilli(resolved.Int64)
		a.ResolvedAt = &t
	}
	return a, nil
}

// OpenAlert opens (or refreshes) the alert identified by a.Key. If a
// non-resolved alert with that key already exists, its message/severity and
// LastSeenAt are bumped; otherwise a new open alert is inserted. This lets a
// long-running condition remain a single row that survives restarts.
func (s *Store) OpenAlert(a domain.Alert) error {
	now := time.Now()
	if a.Severity == "" {
		a.Severity = domain.AlertWarning
	}
	var existingID string
	err := s.db.QueryRow(`SELECT id FROM alerts WHERE key = ? AND status != 'resolved'`, a.Key).Scan(&existingID)
	switch {
	case err == sql.ErrNoRows:
		if a.ID == "" {
			a.ID = newID("alert")
		}
		_, err := s.db.Exec(
			`INSERT INTO alerts (`+alertCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?, ?, NULL, NULL)`,
			a.ID, a.Key, a.EnvironmentID, a.GrowID, a.DeviceID, a.Severity, a.Kind, a.Title, a.Message,
			now.UnixMilli(), now.UnixMilli())
		return err
	case err != nil:
		return err
	default:
		_, err := s.db.Exec(
			`UPDATE alerts SET severity = ?, message = ?, title = ?, last_seen_at = ? WHERE id = ?`,
			a.Severity, a.Message, a.Title, now.UnixMilli(), existingID)
		return err
	}
}

// ResolveAlert marks any non-resolved alert with the given key resolved.
func (s *Store) ResolveAlert(key string) error {
	_, err := s.db.Exec(
		`UPDATE alerts SET status = 'resolved', resolved_at = ? WHERE key = ? AND status != 'resolved'`,
		time.Now().UnixMilli(), key)
	return err
}

// AckAlert acknowledges an open alert by id (it stays visible but flagged seen).
func (s *Store) AckAlert(id string) error {
	_, err := s.db.Exec(
		`UPDATE alerts SET status = 'acknowledged', acknowledged_at = ? WHERE id = ? AND status = 'open'`,
		time.Now().UnixMilli(), id)
	return err
}

// ResolveAlertID resolves a single alert by id (manual dismissal from the UI).
func (s *Store) ResolveAlertID(id string) error {
	_, err := s.db.Exec(
		`UPDATE alerts SET status = 'resolved', resolved_at = ? WHERE id = ? AND status != 'resolved'`,
		time.Now().UnixMilli(), id)
	return err
}

// OpenAlerts returns every non-resolved alert, most-recently-seen first.
func (s *Store) OpenAlerts() ([]domain.Alert, error) {
	rows, err := s.db.Query(`SELECT ` + alertCols + ` FROM alerts WHERE status != 'resolved' ORDER BY last_seen_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Alert{}
	for rows.Next() {
		a, err := scanAlert(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
