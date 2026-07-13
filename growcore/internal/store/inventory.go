package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

// inventoryCols excludes the image blob; image bytes are fetched on demand via
// InventoryItemImage so listing stays cheap (mirrors cultivarCols).
const inventoryCols = `id, category, name, variants, location, status, notes, attributes, product_id, image_type, created_at, updated_at`

func scanInventoryItem(scan func(dst ...any) error) (domain.InventoryItem, error) {
	var it domain.InventoryItem
	var status, variantsJSON, attrsJSON string
	var created, updated int64
	if err := scan(&it.ID, &it.Category, &it.Name, &variantsJSON, &it.Location,
		&status, &it.Notes, &attrsJSON, &it.ProductID, &it.ImageType, &created, &updated); err != nil {
		return domain.InventoryItem{}, err
	}
	it.Status = domain.InventoryStatus(status)
	it.Variants = []domain.StockLine{}
	if variantsJSON != "" {
		_ = json.Unmarshal([]byte(variantsJSON), &it.Variants)
	}
	it.Attributes = map[string]string{}
	if attrsJSON != "" {
		_ = json.Unmarshal([]byte(attrsJSON), &it.Attributes)
	}
	it.CreatedAt = time.UnixMilli(created)
	it.UpdatedAt = time.UnixMilli(updated)
	return it, nil
}

// SaveInventoryItem inserts or updates a stock record. It never touches the
// image blob; use SetInventoryItemImage / ClearInventoryItemImage for that. On
// update the stored image_type is preserved.
func (s *Store) SaveInventoryItem(it domain.InventoryItem) error {
	if it.Status == "" {
		it.Status = domain.InventoryActive
	}
	if it.Attributes == nil {
		it.Attributes = map[string]string{}
	}
	if it.Variants == nil {
		it.Variants = []domain.StockLine{}
	}
	attrs, err := json.Marshal(it.Attributes)
	if err != nil {
		return err
	}
	variants, err := json.Marshal(it.Variants)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO inventory_items
		   (id, category, name, variants, location, status, notes, attributes, product_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   category=excluded.category, name=excluded.name, variants=excluded.variants,
		   location=excluded.location, status=excluded.status, notes=excluded.notes,
		   attributes=excluded.attributes, product_id=excluded.product_id, updated_at=excluded.updated_at`,
		it.ID, it.Category, it.Name, string(variants), it.Location, string(it.Status),
		it.Notes, string(attrs), it.ProductID, it.CreatedAt.UnixMilli(), it.UpdatedAt.UnixMilli(),
	)
	return err
}

// SetInventoryItemImage stores the image bytes and MIME type for an item.
func (s *Store) SetInventoryItemImage(id string, data []byte, mime string) error {
	_, err := s.db.Exec(`UPDATE inventory_items SET image_data=?, image_type=? WHERE id=?`, data, mime, id)
	return err
}

// ClearInventoryItemImage removes any user-uploaded image.
func (s *Store) ClearInventoryItemImage(id string) error {
	_, err := s.db.Exec(`UPDATE inventory_items SET image_data=NULL, image_type='' WHERE id=?`, id)
	return err
}

// InventoryItemImage returns the stored image bytes and MIME type. ok is false
// when the item has no user-uploaded image.
func (s *Store) InventoryItemImage(id string) (data []byte, mime string, ok bool, err error) {
	var blob []byte
	var t string
	err = s.db.QueryRow(`SELECT image_data, image_type FROM inventory_items WHERE id=?`, id).Scan(&blob, &t)
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

// InventoryItems returns all stock records (optionally filtered by category),
// newest first.
func (s *Store) InventoryItems(category string) ([]domain.InventoryItem, error) {
	q := `SELECT ` + inventoryCols + ` FROM inventory_items`
	args := []any{}
	if category != "" {
		q += ` WHERE category=?`
		args = append(args, category)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.InventoryItem
	for rows.Next() {
		it, err := scanInventoryItem(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) InventoryItem(id string) (domain.InventoryItem, bool, error) {
	it, err := scanInventoryItem(s.db.QueryRow(`SELECT `+inventoryCols+` FROM inventory_items WHERE id=?`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.InventoryItem{}, false, nil
	}
	if err != nil {
		return domain.InventoryItem{}, false, err
	}
	return it, true, nil
}

func (s *Store) DeleteInventoryItem(id string) error {
	_, err := s.db.Exec(`DELETE FROM inventory_items WHERE id=?`, id)
	return err
}
