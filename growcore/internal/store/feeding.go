package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

// feedingBody is the JSON blob stored in feeding_presets.body: the flexible
// part of a preset (default unit, products, phases). The columns hold the
// searchable/displayable scalars; the body keeps the rest schema-light.
type feedingBody struct {
	Unit     string                  `json:"unit"`
	Products []domain.FeedingProduct `json:"products"`
	Phases   []domain.FeedingPhase   `json:"phases"`
}

const feedingCols = `id, species, name, brand, description, body, created_at`

func scanFeedingPreset(scan func(dst ...any) error) (domain.FeedingPreset, error) {
	var p domain.FeedingPreset
	var bodyJSON string
	var created int64
	if err := scan(&p.ID, &p.Species, &p.Name, &p.Brand, &p.Description, &bodyJSON, &created); err != nil {
		return domain.FeedingPreset{}, err
	}
	var b feedingBody
	if bodyJSON != "" {
		_ = json.Unmarshal([]byte(bodyJSON), &b)
	}
	p.Unit = b.Unit
	p.Products = b.Products
	p.Phases = b.Phases
	p.Source = "user"
	p.CreatedAt = time.UnixMilli(created)
	return p, nil
}

// SaveFeedingPreset inserts or updates a user feeding preset.
func (s *Store) SaveFeedingPreset(p domain.FeedingPreset) error {
	body, err := json.Marshal(feedingBody{Unit: p.Unit, Products: p.Products, Phases: p.Phases})
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO feeding_presets (id, species, name, brand, description, body, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   species=excluded.species, name=excluded.name, brand=excluded.brand,
		   description=excluded.description, body=excluded.body`,
		p.ID, p.Species, p.Name, p.Brand, p.Description, string(body), p.CreatedAt.UnixMilli(),
	)
	return err
}

// FeedingPresets returns all user presets (optionally filtered by species),
// newest first.
func (s *Store) FeedingPresets(species string) ([]domain.FeedingPreset, error) {
	q := `SELECT ` + feedingCols + ` FROM feeding_presets`
	args := []any{}
	if species != "" {
		q += ` WHERE species=?`
		args = append(args, species)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FeedingPreset
	for rows.Next() {
		p, err := scanFeedingPreset(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// FeedingPreset returns one user preset by id.
func (s *Store) FeedingPreset(id string) (domain.FeedingPreset, bool, error) {
	p, err := scanFeedingPreset(s.db.QueryRow(`SELECT `+feedingCols+` FROM feeding_presets WHERE id=?`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.FeedingPreset{}, false, nil
	}
	if err != nil {
		return domain.FeedingPreset{}, false, err
	}
	return p, true, nil
}

// DeleteFeedingPreset removes a user preset.
func (s *Store) DeleteFeedingPreset(id string) error {
	_, err := s.db.Exec(`DELETE FROM feeding_presets WHERE id=?`, id)
	return err
}
