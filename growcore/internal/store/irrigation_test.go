package store

import (
	"testing"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func TestIrrigationBindingRoundTrip(t *testing.T) {
	st := open(t)
	if err := st.SaveEnvironment(domain.Environment{ID: "tent-a", Name: "Tent A", Kind: domain.KindTent}); err != nil {
		t.Fatal(err)
	}
	b := domain.Binding{
		ID: "irr1", DeviceID: "dev-autopot", DeviceName: "AutoPot 4Pot",
		EnvironmentID: "tent-a", Kind: domain.KindIrrigation, Name: "Irrigation",
		Entity: "", IrrigationType: domain.IrrigationAutoPot, IrrigationMode: domain.IrrigationPassive,
		ReservoirL: 47, ValveCount: 4,
	}
	if err := st.SaveBinding(b); err != nil {
		t.Fatal(err)
	}
	got, err := st.Bindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d bindings, want 1", len(got))
	}
	g := got[0]
	if g.Kind != domain.KindIrrigation || g.IrrigationType != domain.IrrigationAutoPot ||
		g.IrrigationMode != domain.IrrigationPassive || g.ReservoirL != 47 || g.ValveCount != 4 || g.Entity != "" {
		t.Fatalf("round-trip mismatch: %+v", g)
	}
	t.Logf("irrigation binding OK: %+v", g)
}
