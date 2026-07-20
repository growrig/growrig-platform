package species

import (
	"reflect"
	"testing"
)

func TestResolveStages(t *testing.T) {
	sp, ok := Get("cannabis")
	if !ok {
		t.Fatal("cannabis species not found")
	}

	// Default selection: required stages plus optional stages that default in
	// (germination defaults off).
	if got, want := sp.DefaultStageNames(), []string{"seedling", "vegetative", "flowering", "flush", "drying", "cure"}; !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultStageNames() = %v, want %v", got, want)
	}

	// A nil request yields the default sequence.
	if got, want := sp.ResolveStages(nil), sp.DefaultStageNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("ResolveStages(nil) = %v, want %v", got, want)
	}

	// Required stages are forced in even when omitted; optional ones appear only
	// when named; canonical order is preserved regardless of request order.
	got := sp.ResolveStages([]string{"cure", "germination"})
	want := []string{"germination", "vegetative", "flowering", "cure"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ResolveStages(cure,germination) = %v, want %v", got, want)
	}

	// Unknown stage names are ignored, leaving just the required stages.
	if got, want := sp.ResolveStages([]string{"mystery"}), []string{"vegetative", "flowering"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ResolveStages(mystery) = %v, want %v", got, want)
	}
}
