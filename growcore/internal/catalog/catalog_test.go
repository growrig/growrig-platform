package catalog

import "testing"

// TestProducts loads the committed repo-root devices/ tree and checks the
// directory-derived invariants hold.
func TestProducts(t *testing.T) {
	products := Products()
	if len(products) == 0 {
		t.Fatal("no products loaded; expected the repo-root devices/ tree")
	}

	seen := map[string]bool{}
	for _, p := range products {
		if p.ID == "" {
			t.Errorf("product with empty id: %+v", p)
		}
		if seen[p.ID] {
			t.Errorf("duplicate product id %q", p.ID)
		}
		seen[p.ID] = true

		if !validCategory(p.Category) {
			t.Errorf("product %q has invalid category %q", p.ID, p.Category)
		}
		if p.Brand == "" || p.Model == "" {
			t.Errorf("product %q missing brand/model", p.ID)
		}
	}

	// Categories must be emitted in categoryOrder.
	last := -1
	for _, p := range products {
		if r := categoryRank(p.Category); r < last {
			t.Errorf("products not ordered by category: %q (rank %d) after rank %d", p.ID, r, last)
		} else {
			last = r
		}
	}

	// Spot-check a multi-binding device survived the round-trip.
	xiaomi := find(products, "xiaomi-lywsd03mmc")
	if xiaomi == nil {
		t.Fatal("expected xiaomi-lywsd03mmc in catalog")
	}
	if xiaomi.Category != CatSensor {
		t.Errorf("xiaomi category = %q, want sensor", xiaomi.Category)
	}
	if len(xiaomi.Provides) != 2 {
		t.Fatalf("xiaomi provides %d bindings, want 2", len(xiaomi.Provides))
	}
	if xiaomi.Provides[0].Measurement != "temperature" || xiaomi.Provides[1].Measurement != "humidity" {
		t.Errorf("xiaomi provides = %+v, want temperature+humidity", xiaomi.Provides)
	}
}

func find(products []Product, id string) *Product {
	for i := range products {
		if products[i].ID == id {
			return &products[i]
		}
	}
	return nil
}
