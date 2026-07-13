package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

func open(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestBindingRoundTrip(t *testing.T) {
	st := open(t)
	if err := st.SaveEnvironment(domain.Environment{ID: "tent-a", Name: "Tent A", Kind: domain.KindTent}); err != nil {
		t.Fatal(err)
	}
	b := domain.Binding{
		ID: "b1", EnvironmentID: "tent-a", Kind: domain.KindFan, Name: "Exhaust",
		Entity: "fan.exhaust", Role: domain.RoleExhaust, RPMEntity: "sensor.rpm",
	}
	if err := st.SaveBinding(b); err != nil {
		t.Fatal(err)
	}
	got, err := st.Bindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Entity != "fan.exhaust" || got[0].Role != domain.RoleExhaust {
		t.Fatalf("binding not persisted: %+v", got)
	}
}

func TestDeleteEnvironmentBlockedByBinding(t *testing.T) {
	st := open(t)
	_ = st.SaveEnvironment(domain.Environment{ID: "tent-a", Name: "A", Kind: domain.KindTent})
	_ = st.SaveBinding(domain.Binding{ID: "b1", EnvironmentID: "tent-a", Kind: domain.KindSensor, Name: "T", Entity: "sensor.t", Measurement: domain.MeasureTemperature})

	if err := st.DeleteEnvironment("tent-a"); err == nil {
		t.Fatal("expected deletion blocked while a binding references the environment")
	}
	if err := st.DeleteBinding("b1"); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteEnvironment("tent-a"); err != nil {
		t.Fatalf("delete after removing binding: %v", err)
	}
}

func TestDeleteEnvironmentBlockedByAirSourceRef(t *testing.T) {
	st := open(t)
	_ = st.SaveEnvironment(domain.Environment{ID: "room", Name: "Lung", Kind: domain.KindRoom})
	_ = st.SaveEnvironment(domain.Environment{ID: "tent", Name: "Tent", Kind: domain.KindTent, AirSourceID: "room"})

	if err := st.DeleteEnvironment("room"); err == nil {
		t.Fatal("expected deletion blocked while a tent uses the room as air source")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")
	st1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st1.Close()
	st2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	st2.Close()
}

func TestEnvironmentYAMLExportAndImport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "growcore.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	env := domain.Environment{ID: "box-a", Name: "Box A", Kind: domain.KindTent, Model: "Test Tent", TargetTempC: 24, TargetHumidity: 55, EmergencyTempC: 35}
	if err := st.SaveEnvironment(env); err != nil {
		t.Fatal(err)
	}
	b := domain.Binding{ID: "light-a", DeviceID: "fixture-a", DeviceName: "Test Light", EnvironmentID: env.ID, Kind: domain.KindLight, Name: "Test Light", Wattage: 100, Primary: true}
	if err := st.SaveBinding(b); err != nil {
		t.Fatal(err)
	}
	st.Close()

	yamlPath := filepath.Join(dir, "environments", env.ID, "environment.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "devices:") || !strings.Contains(string(raw), "wattage: 100") {
		t.Fatalf("unexpected YAML:\n%s", raw)
	}
	updated := strings.Replace(string(raw), "name: Box A", "name: Box A edited", 1)
	if err := os.WriteFile(yamlPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	envs, err := st.Environments()
	if err != nil {
		t.Fatal(err)
	}
	if len(envs) != 1 || envs[0].Name != "Box A edited" {
		t.Fatalf("YAML edit not imported: %+v", envs)
	}
}

func TestActivityLogFiltersByEnvironment(t *testing.T) {
	st := open(t)
	if err := st.AddActivity(domain.Activity{EnvironmentID: "a", Level: "info", Type: "control", Message: "fan changed"}); err != nil {
		t.Fatal(err)
	}
	if err := st.AddActivity(domain.Activity{EnvironmentID: "b", Level: "warning", Type: "warning", Message: "sensor offline"}); err != nil {
		t.Fatal(err)
	}
	events, err := st.Activities("a", "", nil, nil, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Message != "fan changed" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestActivityLogFiltersByGrow(t *testing.T) {
	st := open(t)
	if err := st.AddActivity(domain.Activity{GrowID: "g1", Level: "info", Type: "configuration", Message: "created grow"}); err != nil {
		t.Fatal(err)
	}
	if err := st.AddActivity(domain.Activity{GrowID: "g2", Level: "info", Type: "configuration", Message: "other grow"}); err != nil {
		t.Fatal(err)
	}
	events, err := st.Activities("", "g1", nil, nil, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Message != "created grow" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestActivityLogFiltersByLevel(t *testing.T) {
	st := open(t)
	_ = st.AddActivity(domain.Activity{EnvironmentID: "a", Level: "info", Type: "control", Message: "fan changed"})
	_ = st.AddActivity(domain.Activity{EnvironmentID: "a", Level: "warning", Type: "warning", Message: "sensor offline"})
	_ = st.AddActivity(domain.Activity{EnvironmentID: "a", Level: "error", Type: "warning", Message: "device unreachable"})
	events, err := st.Activities("a", "", []string{"warning", "error"}, nil, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected only warning+error events, got: %+v", events)
	}
	for _, e := range events {
		if e.Level == "info" {
			t.Fatalf("info event leaked through level filter: %+v", e)
		}
	}
}

func TestActivityLogPaginates(t *testing.T) {
	st := open(t)
	for i := 0; i < 5; i++ {
		if err := st.AddActivity(domain.Activity{ID: fmt.Sprintf("e%d", i), EnvironmentID: "a", Level: "info", Type: "control", Message: fmt.Sprintf("msg %d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	total, err := st.CountActivities("a", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Fatalf("expected count 5, got %d", total)
	}
	page1, err := st.Activities("a", "", nil, nil, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	page2, err := st.Activities("a", "", nil, nil, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || len(page2) != 2 {
		t.Fatalf("expected 2 per page, got %d and %d", len(page1), len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Fatalf("pages overlap: %s == %s", page1[0].ID, page2[0].ID)
	}
}
