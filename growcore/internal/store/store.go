// Package store persists Grow Core configuration and history in SQLite.
//
// It uses the pure-Go modernc.org/sqlite driver so Grow Core builds and runs
// without CGO, which keeps cross-compilation for the Grow Hub simple.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path and applies the
// schema. Use ":memory:" for ephemeral runs.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // modernc/sqlite is safest single-writer
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS environments (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'tent',
    air_source      TEXT NOT NULL DEFAULT '',
    model           TEXT NOT NULL DEFAULT '',
    size            TEXT NOT NULL DEFAULT '', -- legacy freeform; superseded by width/depth/height
    width_cm        REAL NOT NULL DEFAULT 0,
    depth_cm        REAL NOT NULL DEFAULT 0,
    height_cm       REAL NOT NULL DEFAULT 0,
    target_temp     REAL NOT NULL DEFAULT 24,
    target_humidity REAL NOT NULL DEFAULT 55,
    target_co2      REAL NOT NULL DEFAULT 0,
    emergency_temp  REAL NOT NULL DEFAULT 35
);
CREATE TABLE IF NOT EXISTS cycles (
    environment_id TEXT PRIMARY KEY,
    strain         TEXT NOT NULL DEFAULT '',
    started_at     INTEGER NOT NULL DEFAULT 0,
    phase          TEXT NOT NULL DEFAULT '',
    phase_started  INTEGER NOT NULL DEFAULT 0,
    notes          TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS bindings (
    id             TEXT PRIMARY KEY,
    environment_id TEXT NOT NULL,
    kind           TEXT NOT NULL,
    name           TEXT NOT NULL,
    entity         TEXT NOT NULL,
    measurement    TEXT NOT NULL DEFAULT '',
    role           TEXT NOT NULL DEFAULT '',
    rpm_entity     TEXT NOT NULL DEFAULT '',
    wattage        REAL NOT NULL DEFAULT 0,
    is_primary     INTEGER NOT NULL DEFAULT 0,
    created        INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_bindings_env ON bindings (environment_id);
CREATE TABLE IF NOT EXISTS readings (
    environment_id TEXT NOT NULL,
    ts             INTEGER NOT NULL,
    temp           REAL NOT NULL,
    humidity       REAL NOT NULL,
    co2            REAL NOT NULL DEFAULT 0,
    vpd            REAL NOT NULL DEFAULT 0,
    exhaust_speed  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_readings_env_ts ON readings (environment_id, ts);
-- Superseded by the bindings model.
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS devices;
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	// Additive migrations for databases created before these columns existed.
	for _, m := range []struct{ table, column, def string }{
		{"environments", "model", "TEXT NOT NULL DEFAULT ''"},
		{"environments", "size", "TEXT NOT NULL DEFAULT ''"},
		{"environments", "width_cm", "REAL NOT NULL DEFAULT 0"},
		{"environments", "depth_cm", "REAL NOT NULL DEFAULT 0"},
		{"environments", "height_cm", "REAL NOT NULL DEFAULT 0"},
		{"bindings", "wattage", "REAL NOT NULL DEFAULT 0"},
		{"bindings", "is_primary", "INTEGER NOT NULL DEFAULT 0"},
	} {
		if err := s.ensureColumn(m.table, m.column, m.def); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureColumn(table, column, def string) error {
	rows, err := s.db.Query("SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + def)
	return err
}

// --- Environments ---

func (s *Store) SaveEnvironment(e domain.Environment) error {
	if e.Kind == "" {
		e.Kind = domain.KindTent
	}
	_, err := s.db.Exec(
		`INSERT INTO environments
		   (id, name, kind, air_source, model, width_cm, depth_cm, height_cm, target_temp, target_humidity, target_co2, emergency_temp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name, kind=excluded.kind, air_source=excluded.air_source,
		   model=excluded.model,
		   width_cm=excluded.width_cm, depth_cm=excluded.depth_cm, height_cm=excluded.height_cm,
		   target_temp=excluded.target_temp, target_humidity=excluded.target_humidity,
		   target_co2=excluded.target_co2, emergency_temp=excluded.emergency_temp`,
		e.ID, e.Name, string(e.Kind), e.AirSourceID, e.Model,
		e.WidthCm, e.DepthCm, e.HeightCm,
		e.TargetTempC, e.TargetHumidity, e.TargetCO2, e.EmergencyTempC,
	)
	return err
}

func (s *Store) UpdateTargets(id string, targetTemp, targetHumidity float64) error {
	res, err := s.db.Exec(
		`UPDATE environments SET target_temp=?, target_humidity=? WHERE id=?`,
		targetTemp, targetHumidity, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("environment %q not found", id)
	}
	return nil
}

func (s *Store) Environments() ([]domain.Environment, error) {
	rows, err := s.db.Query(
		`SELECT id, name, kind, air_source, model, width_cm, depth_cm, height_cm, target_temp, target_humidity, target_co2, emergency_temp
		 FROM environments ORDER BY kind DESC, name`) // tents before rooms
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Environment
	for rows.Next() {
		var e domain.Environment
		var kind string
		if err := rows.Scan(&e.ID, &e.Name, &kind, &e.AirSourceID, &e.Model, &e.WidthCm, &e.DepthCm, &e.HeightCm,
			&e.TargetTempC, &e.TargetHumidity, &e.TargetCO2, &e.EmergencyTempC); err != nil {
			return nil, err
		}
		e.Kind = domain.EnvironmentKind(kind)
		out = append(out, e)
	}
	return out, rows.Err()
}

// --- Cycles ---

func (s *Store) SaveCycle(c domain.Cycle) error {
	_, err := s.db.Exec(
		`INSERT INTO cycles (environment_id, strain, started_at, phase, phase_started, notes)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(environment_id) DO UPDATE SET
		   strain=excluded.strain, started_at=excluded.started_at,
		   phase=excluded.phase, phase_started=excluded.phase_started, notes=excluded.notes`,
		c.EnvironmentID, c.Strain, c.StartedAt.UnixMilli(), string(c.Phase), c.PhaseStarted.UnixMilli(), c.Notes,
	)
	return err
}

func (s *Store) DeleteCycle(envID string) error {
	_, err := s.db.Exec(`DELETE FROM cycles WHERE environment_id=?`, envID)
	return err
}

func (s *Store) Cycles() ([]domain.Cycle, error) {
	rows, err := s.db.Query(
		`SELECT environment_id, strain, started_at, phase, phase_started, notes FROM cycles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Cycle
	for rows.Next() {
		var c domain.Cycle
		var started, phaseStarted int64
		var phase string
		if err := rows.Scan(&c.EnvironmentID, &c.Strain, &started, &phase, &phaseStarted, &c.Notes); err != nil {
			return nil, err
		}
		c.StartedAt = time.UnixMilli(started)
		c.PhaseStarted = time.UnixMilli(phaseStarted)
		c.Phase = domain.Phase(phase)
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteEnvironment removes an environment. It fails if bindings still
// reference it, or if another environment uses it as an air source.
func (s *Store) DeleteEnvironment(id string) error {
	var bindings, refs int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM bindings WHERE environment_id=?`, id).Scan(&bindings); err != nil {
		return err
	}
	if bindings > 0 {
		return fmt.Errorf("environment %q still has %d binding(s)", id, bindings)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM environments WHERE air_source=?`, id).Scan(&refs); err != nil {
		return err
	}
	if refs > 0 {
		return fmt.Errorf("environment %q is used as an air source by %d tent(s)", id, refs)
	}
	res, err := s.db.Exec(`DELETE FROM environments WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("environment %q not found", id)
	}
	return nil
}

// --- Bindings ---

func (s *Store) SaveBinding(b domain.Binding) error {
	_, err := s.db.Exec(
		`INSERT INTO bindings (id, environment_id, kind, name, entity, measurement, role, rpm_entity, wattage, is_primary, created)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   environment_id=excluded.environment_id, kind=excluded.kind, name=excluded.name,
		   entity=excluded.entity, measurement=excluded.measurement, role=excluded.role,
		   rpm_entity=excluded.rpm_entity, wattage=excluded.wattage, is_primary=excluded.is_primary`,
		b.ID, b.EnvironmentID, string(b.Kind), b.Name, b.Entity,
		string(b.Measurement), string(b.Role), b.RPMEntity, b.Wattage, boolToInt(b.Primary), time.Now().UnixNano(),
	)
	return err
}

func (s *Store) DeleteBinding(id string) error {
	res, err := s.db.Exec(`DELETE FROM bindings WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("binding %q not found", id)
	}
	return nil
}

// SetPrimaryLight makes bindingID the sole primary light in its environment.
func (s *Store) SetPrimaryLight(envID, bindingID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(
		`UPDATE bindings SET is_primary=0 WHERE environment_id=? AND kind='light'`, envID); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE bindings SET is_primary=1 WHERE id=? AND kind='light'`, bindingID); err != nil {
		return err
	}
	return tx.Commit()
}

// EnsurePrimaryLight guarantees that, if an environment has any lights, exactly
// one is primary — promoting the oldest light when none is currently marked.
func (s *Store) EnsurePrimaryLight(envID string) error {
	var count int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM bindings WHERE environment_id=? AND kind='light' AND is_primary=1`,
		envID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // a primary already exists
	}
	var oldest string
	err := s.db.QueryRow(
		`SELECT id FROM bindings WHERE environment_id=? AND kind='light' ORDER BY created LIMIT 1`,
		envID).Scan(&oldest)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // no lights at all
	}
	if err != nil {
		return err
	}
	return s.SetPrimaryLight(envID, oldest)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) Bindings() ([]domain.Binding, error) {
	rows, err := s.db.Query(
		`SELECT id, environment_id, kind, name, entity, measurement, role, rpm_entity, wattage, is_primary
		 FROM bindings ORDER BY created`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Binding
	for rows.Next() {
		var b domain.Binding
		var kind, measurement, role string
		var isPrimary int
		if err := rows.Scan(&b.ID, &b.EnvironmentID, &kind, &b.Name, &b.Entity, &measurement, &role, &b.RPMEntity, &b.Wattage, &isPrimary); err != nil {
			return nil, err
		}
		b.Kind = domain.BindingKind(kind)
		b.Measurement = domain.Measurement(measurement)
		b.Role = domain.Role(role)
		b.Primary = isPrimary != 0
		out = append(out, b)
	}
	return out, rows.Err()
}

// --- Readings history ---

func (s *Store) InsertReading(r domain.Reading) error {
	_, err := s.db.Exec(
		`INSERT INTO readings (environment_id, ts, temp, humidity, co2, vpd, exhaust_speed)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.EnvironmentID, r.Time.UnixMilli(), r.TempC, r.Humidity, r.CO2, r.VPD, r.ExhaustSpeed,
	)
	return err
}

// RecentReadings returns up to limit most-recent readings, oldest first.
func (s *Store) RecentReadings(envID string, limit int) ([]domain.Reading, error) {
	rows, err := s.db.Query(
		`SELECT ts, temp, humidity, co2, vpd, exhaust_speed FROM readings
		 WHERE environment_id=? ORDER BY ts DESC LIMIT ?`, envID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Reading
	for rows.Next() {
		r := domain.Reading{EnvironmentID: envID}
		var ts int64
		if err := rows.Scan(&ts, &r.TempC, &r.Humidity, &r.CO2, &r.VPD, &r.ExhaustSpeed); err != nil {
			return nil, err
		}
		r.Time = time.UnixMilli(ts)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
