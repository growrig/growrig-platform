package store

import (
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

const photoCols = `id, grow_id, plant_unit_id, caption, taken_at, file, image_type, created_at`

func scanGrowPhoto(scan func(dst ...any) error) (domain.GrowPhoto, error) {
	var p domain.GrowPhoto
	var taken, created int64
	if err := scan(&p.ID, &p.GrowID, &p.PlantUnitID, &p.Caption, &taken, &p.File, &p.ImageType, &created); err != nil {
		return domain.GrowPhoto{}, err
	}
	p.TakenAt = time.UnixMilli(taken)
	p.CreatedAt = time.UnixMilli(created)
	return p, nil
}

// AddGrowPhoto records a photo's metadata. The image bytes must already be
// written to disk under the data directory by the caller.
func (s *Store) AddGrowPhoto(p domain.GrowPhoto) (domain.GrowPhoto, error) {
	if p.ID == "" {
		p.ID = newID("photo")
	}
	if p.TakenAt.IsZero() {
		p.TakenAt = time.Now()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO grow_photos (`+photoCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.GrowID, p.PlantUnitID, p.Caption, p.TakenAt.UnixMilli(), p.File, p.ImageType, p.CreatedAt.UnixMilli())
	if err != nil {
		return domain.GrowPhoto{}, err
	}
	return p, nil
}

// GrowPhotos lists a grow's photos, newest taken first.
func (s *Store) GrowPhotos(growID string) ([]domain.GrowPhoto, error) {
	rows, err := s.db.Query(`SELECT `+photoCols+` FROM grow_photos WHERE grow_id = ? ORDER BY taken_at DESC, created_at DESC`, growID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GrowPhoto{}
	for rows.Next() {
		p, err := scanGrowPhoto(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GrowPhoto returns one photo's metadata by id.
func (s *Store) GrowPhoto(id string) (domain.GrowPhoto, bool, error) {
	p, err := scanGrowPhoto(s.db.QueryRow(`SELECT `+photoCols+` FROM grow_photos WHERE id = ?`, id).Scan)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return domain.GrowPhoto{}, false, nil
		}
		return domain.GrowPhoto{}, false, err
	}
	return p, true, nil
}

// DeleteGrowPhoto removes a photo's metadata row (the caller unlinks the file
// when no rows reference it anymore — see PhotoFileRefCount).
func (s *Store) DeleteGrowPhoto(id string) error {
	_, err := s.db.Exec(`DELETE FROM grow_photos WHERE id = ?`, id)
	return err
}

// PhotoFileRefCount counts how many rows still reference a given on-disk file
// for a grow, so a content-addressed file is unlinked only when the last row is
// gone.
func (s *Store) PhotoFileRefCount(growID, file string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM grow_photos WHERE grow_id = ? AND file = ?`, growID, file).Scan(&n)
	return n, err
}
