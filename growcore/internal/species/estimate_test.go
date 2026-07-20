package species

import "testing"

func TestEstimatedDays(t *testing.T) {
	// The full cannabis sequence sums every stage's typicalDays.
	full := []string{"seedling", "vegetative", "flowering", "flush", "drying", "cure"}
	if got, want := EstimatedDays("cannabis", full), 14+28+63+7+10+21; got != want {
		t.Errorf("EstimatedDays(cannabis, full) = %d, want %d", got, want)
	}

	// A partial sequence sums only the stages given.
	if got, want := EstimatedDays("cannabis", []string{"seedling", "vegetative"}), 14+28; got != want {
		t.Errorf("EstimatedDays(cannabis, partial) = %d, want %d", got, want)
	}

	// Case-insensitive species lookup, matching Get.
	if got, want := EstimatedDays("Cannabis", full), 143; got != want {
		t.Errorf("EstimatedDays(Cannabis) = %d, want %d", got, want)
	}

	// Unknown species carries no estimate.
	if got := EstimatedDays("nope", full); got != 0 {
		t.Errorf("EstimatedDays(nope) = %d, want 0", got)
	}

	// Unknown stage names contribute nothing rather than erroring.
	if got, want := EstimatedDays("cannabis", []string{"seedling", "mystery"}), 14; got != want {
		t.Errorf("EstimatedDays with unknown stage = %d, want %d", got, want)
	}
}
