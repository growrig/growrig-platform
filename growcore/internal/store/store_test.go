package store

import (
	"path/filepath"
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
