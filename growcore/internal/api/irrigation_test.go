package api

import (
	"testing"

	"github.com/growrig/growrig/growcore/internal/domain"
	"github.com/growrig/growrig/growcore/internal/store"
)

func TestBuildIrrigationBinding(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SaveEnvironment(domain.Environment{ID: "tent-a", Name: "A", Kind: domain.KindTent}); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st}

	// Passive AutoPot: no entity required, defaults mode to passive.
	b, err := s.buildBinding("irr-1", bindingBody{
		DeviceID: "d1", DeviceName: "AutoPot", EnvironmentID: "tent-a",
		Kind: domain.KindIrrigation, Name: "Irrigation", Entity: "",
		IrrigationType: "autopot", ReservoirL: 47, ValveCount: 4,
	})
	if err != nil {
		t.Fatalf("passive irrigation rejected: %v", err)
	}
	if b.IrrigationMode != domain.IrrigationPassive || b.Entity != "" || b.ReservoirL != 47 || b.ValveCount != 4 {
		t.Fatalf("passive binding wrong: %+v", b)
	}

	// Missing type is rejected.
	if _, err := s.buildBinding("irr-2", bindingBody{
		DeviceID: "d2", DeviceName: "X", EnvironmentID: "tent-a",
		Kind: domain.KindIrrigation, Name: "Irrigation",
	}); err == nil {
		t.Fatal("expected error for missing irrigation type")
	}

	// Controlled mode requires an entity.
	if _, err := s.buildBinding("irr-3", bindingBody{
		DeviceID: "d3", DeviceName: "X", EnvironmentID: "tent-a",
		Kind: domain.KindIrrigation, Name: "Pump", IrrigationType: "drip",
		IrrigationMode: "controlled", Entity: "",
	}); err == nil {
		t.Fatal("expected error for controlled irrigation without entity")
	}
	t.Log("irrigation binding validation OK")
}
