package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

// cultivarCols excludes the image blob; image bytes are fetched on demand via
// CultivarImage so listing stays cheap.
const cultivarCols = `id, species, name, creator, description, attributes, image_type, created_at`

func scanCultivar(scan func(dst ...any) error) (domain.Cultivar, error) {
	var c domain.Cultivar
	var attrsJSON string
	var created int64
	if err := scan(&c.ID, &c.Species, &c.Name, &c.Creator, &c.Description, &attrsJSON, &c.ImageType, &created); err != nil {
		return domain.Cultivar{}, err
	}
	c.Attributes = map[string]string{}
	if attrsJSON != "" {
		_ = json.Unmarshal([]byte(attrsJSON), &c.Attributes)
	}
	c.CreatedAt = time.UnixMilli(created)
	return c, nil
}

// SaveCultivar inserts or updates a cultivar's metadata. It never touches the
// image blob; use SetCultivarImage / ClearCultivarImage for that. On update the
// stored image_type is preserved.
func (s *Store) SaveCultivar(c domain.Cultivar) error {
	if c.Attributes == nil {
		c.Attributes = map[string]string{}
	}
	attrs, err := json.Marshal(c.Attributes)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO cultivars (id, species, name, creator, description, attributes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   species=excluded.species, name=excluded.name, creator=excluded.creator,
		   description=excluded.description, attributes=excluded.attributes`,
		c.ID, c.Species, c.Name, c.Creator, c.Description, string(attrs), c.CreatedAt.UnixMilli(),
	)
	return err
}

// Cultivars returns all cultivars (optionally filtered by species), newest first.
func (s *Store) Cultivars(species string) ([]domain.Cultivar, error) {
	q := `SELECT ` + cultivarCols + ` FROM cultivars`
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
	var out []domain.Cultivar
	for rows.Next() {
		c, err := scanCultivar(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Cultivar(id string) (domain.Cultivar, bool, error) {
	c, err := scanCultivar(s.db.QueryRow(`SELECT `+cultivarCols+` FROM cultivars WHERE id=?`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Cultivar{}, false, nil
	}
	if err != nil {
		return domain.Cultivar{}, false, err
	}
	return c, true, nil
}

func (s *Store) DeleteCultivar(id string) error {
	_, err := s.db.Exec(`DELETE FROM cultivars WHERE id=?`, id)
	return err
}

// SetCultivarImage stores the image bytes and MIME type for a cultivar.
func (s *Store) SetCultivarImage(id string, data []byte, mime string) error {
	_, err := s.db.Exec(`UPDATE cultivars SET image_data=?, image_type=? WHERE id=?`, data, mime, id)
	return err
}

// ClearCultivarImage removes any stored image.
func (s *Store) ClearCultivarImage(id string) error {
	_, err := s.db.Exec(`UPDATE cultivars SET image_data=NULL, image_type='' WHERE id=?`, id)
	return err
}

// CultivarImage returns the stored image bytes and MIME type. ok is false when
// the cultivar has no image.
func (s *Store) CultivarImage(id string) (data []byte, mime string, ok bool, err error) {
	var blob []byte
	var t string
	err = s.db.QueryRow(`SELECT image_data, image_type FROM cultivars WHERE id=?`, id).Scan(&blob, &t)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	if len(blob) == 0 || t == "" {
		return nil, "", false, nil
	}
	return blob, t, true, nil
}
