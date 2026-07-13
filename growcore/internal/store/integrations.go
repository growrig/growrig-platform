package store

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

type IntegrationRecord struct {
	Instance domain.IntegrationInstance
	Secrets  string
}

func (s *Store) IntegrationInstances() ([]IntegrationRecord, error) {
	rows, err := s.db.Query(`SELECT id,bundle_id,name,config,secrets,enabled,status,status_message,last_checked,created,updated FROM integration_instances ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []IntegrationRecord{}
	for rows.Next() {
		var rec IntegrationRecord
		var raw string
		var enabled int
		var checked, created, updated int64
		if err := rows.Scan(&rec.Instance.ID, &rec.Instance.BundleID, &rec.Instance.Name, &raw, &rec.Secrets, &enabled, &rec.Instance.Status, &rec.Instance.StatusMessage, &checked, &created, &updated); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &rec.Instance.Config); err != nil {
			return nil, err
		}
		rec.Instance.Enabled = enabled != 0
		rec.Instance.CreatedAt, rec.Instance.UpdatedAt = time.UnixMilli(created), time.UnixMilli(updated)
		if checked > 0 {
			t := time.UnixMilli(checked)
			rec.Instance.LastCheckedAt = &t
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) IntegrationInstance(id string) (IntegrationRecord, error) {
	recs, err := s.IntegrationInstances()
	if err != nil {
		return IntegrationRecord{}, err
	}
	for _, rec := range recs {
		if rec.Instance.ID == id {
			return rec, nil
		}
	}
	return IntegrationRecord{}, errors.New("integration instance not found")
}

func (s *Store) SaveIntegrationInstance(rec IntegrationRecord) error {
	raw, err := json.Marshal(rec.Instance.Config)
	if err != nil {
		return err
	}
	checked := int64(0)
	if rec.Instance.LastCheckedAt != nil {
		checked = rec.Instance.LastCheckedAt.UnixMilli()
	}
	_, err = s.db.Exec(`INSERT INTO integration_instances(id,bundle_id,name,config,secrets,enabled,status,status_message,last_checked,created,updated)
		VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET bundle_id=excluded.bundle_id,name=excluded.name,config=excluded.config,secrets=excluded.secrets,enabled=excluded.enabled,status=excluded.status,status_message=excluded.status_message,last_checked=excluded.last_checked,updated=excluded.updated`,
		rec.Instance.ID, rec.Instance.BundleID, rec.Instance.Name, string(raw), rec.Secrets, rec.Instance.Enabled, rec.Instance.Status, rec.Instance.StatusMessage, checked, rec.Instance.CreatedAt.UnixMilli(), rec.Instance.UpdatedAt.UnixMilli())
	return err
}

func (s *Store) DeleteIntegrationInstance(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM integration_bindings WHERE instance_id=?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM integration_instances WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("integration instance not found")
	}
	return tx.Commit()
}

func (s *Store) IntegrationBindings() ([]domain.IntegrationBinding, error) {
	rows, err := s.db.Query(`SELECT id,feature,grow_id,environment_id,capability,instance_id,created,updated FROM integration_bindings ORDER BY feature,grow_id,environment_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.IntegrationBinding{}
	for rows.Next() {
		var b domain.IntegrationBinding
		var c, u int64
		if err := rows.Scan(&b.ID, &b.Feature, &b.GrowID, &b.EnvironmentID, &b.Capability, &b.InstanceID, &c, &u); err != nil {
			return nil, err
		}
		b.CreatedAt = time.UnixMilli(c)
		b.UpdatedAt = time.UnixMilli(u)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) SaveIntegrationBinding(b domain.IntegrationBinding) error {
	_, err := s.db.Exec(`INSERT INTO integration_bindings(id,feature,grow_id,environment_id,capability,instance_id,created,updated) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(feature,grow_id,environment_id,capability) DO UPDATE SET id=excluded.id,instance_id=excluded.instance_id,created=excluded.created,updated=excluded.updated`, b.ID, b.Feature, b.GrowID, b.EnvironmentID, b.Capability, b.InstanceID, b.CreatedAt.UnixMilli(), b.UpdatedAt.UnixMilli())
	return err
}

func (s *Store) DeleteIntegrationBinding(id string) error {
	_, err := s.db.Exec(`DELETE FROM integration_bindings WHERE id=?`, id)
	return err
}
