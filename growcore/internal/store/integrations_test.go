package store

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func TestIntegrationBindingMigrationAddsEnvironmentScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE integration_bindings (
		id TEXT PRIMARY KEY, feature TEXT NOT NULL, grow_id TEXT NOT NULL DEFAULT '',
		capability TEXT NOT NULL, instance_id TEXT NOT NULL, created INTEGER NOT NULL,
		updated INTEGER NOT NULL, UNIQUE(feature, grow_id, capability));
		INSERT INTO integration_bindings VALUES ('global', 'grow-assistant', '', 'ai.chat', 'one', 1, 1);`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now()
	if err := st.SaveIntegrationBinding(domain.IntegrationBinding{
		ID: "environment", Feature: "grow-assistant", EnvironmentID: "env-1",
		Capability: "ai.chat", InstanceID: "two", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	bindings, err := st.IntegrationBindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 2 || bindings[1].EnvironmentID != "env-1" {
		t.Fatalf("unexpected migrated bindings: %#v", bindings)
	}
}
