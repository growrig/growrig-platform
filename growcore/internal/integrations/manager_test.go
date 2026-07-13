package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/growrig/growrig-platform/growcore/internal/store"
)

func TestDeclarativeIntegrationSecretsRuntimeAndBindings(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"received": body})
	}))
	defer server.Close()

	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundles", "notification", "test")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `
id: test-webhook
name: Test webhook
version: "1"
category: notification
capabilities: [notification.send]
config:
  - {key: endpoint, label: Endpoint, type: url, required: true}
  - {key: token, label: Token, type: password, secret: true, required: true}
runtime:
  type: http
  test:
    urlField: endpoint
    method: POST
    headers: {Authorization: "Bearer {{config.token}}"}
    body: {event: test}
  operations:
    notification.send:
      urlField: endpoint
      method: POST
      headers: {Authorization: "Bearer {{config.token}}"}
      body: {message: "{{input.message}}"}
`
	if err := os.WriteFile(filepath.Join(bundleDir, "integration.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "growcore.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m, err := NewManager(st, filepath.Join(dir, "bundles"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	inst, err := m.Create(InstanceInput{BundleID: "test-webhook", Name: "Alerts", Config: map[string]string{"endpoint": server.URL, "token": "secret-token"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := inst.Config["token"]; found {
		t.Fatal("secret exposed in public config")
	}
	rec, err := st.IntegrationInstance(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Secrets == "" || rec.Secrets == "secret-token" {
		t.Fatal("secret was not encrypted")
	}
	if _, err = m.Test(context.Background(), inst.ID); err != nil {
		t.Fatal(err)
	}
	result, err := m.Invoke(context.Background(), inst.ID, "notification.send", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.(map[string]any)["received"].(map[string]any)["message"] != "hello" {
		t.Fatalf("unexpected result: %#v", result)
	}
	binding, err := m.SaveBinding(BindingInput{Feature: "critical-alerts", Capability: "notification.send", InstanceID: inst.ID})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := m.Resolve("critical-alerts", "grow-1", "notification.send")
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil || resolved.ID != inst.ID {
		t.Fatal("global binding did not resolve as grow fallback")
	}
	if err := m.DeleteBinding(binding.ID); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestCreateRejectsMissingAndInvalidConfiguration(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundles", "ai", "x")
	_ = os.MkdirAll(bundleDir, 0o755)
	manifest := `id: x
name: X
version: "1"
category: ai
capabilities: [ai.chat]
config:
  - {key: endpoint, label: Endpoint, type: url, required: true}
runtime: {type: builtin, handler: ollama}
`
	_ = os.WriteFile(filepath.Join(bundleDir, "integration.yaml"), []byte(manifest), 0o644)
	st, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m, err := NewManager(st, filepath.Join(dir, "bundles"), filepath.Join(dir, "key"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Create(InstanceInput{BundleID: "x", Name: "bad", Config: map[string]string{"endpoint": "ftp://nope"}}); err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestFirstAIChatInstanceGetsDefaultBinding(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundles", "ai", "x")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `id: x
name: X
version: "1"
category: ai
capabilities: [ai.chat]
config:
  - {key: endpoint, label: Endpoint, type: url, required: true}
runtime: {type: builtin, handler: ollama}
`
	if err := os.WriteFile(filepath.Join(bundleDir, "integration.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m, err := NewManager(st, filepath.Join(dir, "bundles"), filepath.Join(dir, "key"))
	if err != nil {
		t.Fatal(err)
	}
	instance, err := m.Create(InstanceInput{BundleID: "x", Name: "Local AI", Config: map[string]string{"endpoint": "http://localhost:11434"}})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := m.Resolve("grow-assistant", "grow-1", "ai.chat")
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil || resolved.ID != instance.ID {
		t.Fatal("first AI chat instance was not selected as the global Grow assistant")
	}
}

func TestOpenMeteoIsCreatedAndBoundByDefault(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundles", "data", "open-meteo")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `id: open-meteo
name: Open-Meteo
version: "1"
category: data
capabilities: [weather.forecast]
config:
  - {key: baseUrl, label: API URL, type: url, required: true, default: "https://api.open-meteo.com"}
runtime: {type: builtin, handler: open-meteo}
`
	if err := os.WriteFile(filepath.Join(bundleDir, "integration.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m, err := NewManager(st, filepath.Join(dir, "bundles"), filepath.Join(dir, "key"))
	if err != nil {
		t.Fatal(err)
	}
	instances, err := m.Instances()
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 || instances[0].BundleID != "open-meteo" || !instances[0].Enabled {
		t.Fatalf("unexpected defaults: %#v", instances)
	}
	resolved, err := m.Resolve("weather-context", "grow-1", "weather.forecast")
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil || resolved.ID != instances[0].ID {
		t.Fatal("default weather binding did not resolve")
	}
	// Reopening the manager must remain idempotent.
	if _, err := NewManager(st, filepath.Join(dir, "bundles"), filepath.Join(dir, "key")); err != nil {
		t.Fatal(err)
	}
	instances, _ = m.Instances()
	if len(instances) != 1 {
		t.Fatalf("default duplicated: %d instances", len(instances))
	}
}
