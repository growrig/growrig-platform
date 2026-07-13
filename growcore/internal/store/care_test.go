package store

import (
	"testing"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

// seedCareGrow makes a grow with n individual plants and returns their unit ids.
func seedCareGrow(t *testing.T, st *Store) []string {
	t.Helper()
	_ = st.SaveGrow(domain.Grow{ID: "grow-1", Name: "G", Species: "cannabis", Stages: domain.DefaultStages, Status: domain.GrowActive})
	units, err := st.BulkCreatePlants("grow-1", 3, domain.TrackIndividual, 1, "Plant", "Oreoz", "tent-a", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(units))
	for i, u := range units {
		ids[i] = u.ID
	}
	return ids
}

func TestCareEventRoundTrip(t *testing.T) {
	st := open(t)
	ids := seedCareGrow(t, st)

	e := domain.CareEvent{
		ID: "care-1", GrowID: "grow-1", Type: "feed", OccurredAt: time.Now(),
		RecipeID: "veg-base", PH: 6.2, EC: 1.4,
		Applications: []domain.CareApplication{
			{ID: "a1", PlantUnitID: ids[0], AmountML: 900},
			{ID: "a2", PlantUnitID: ids[1], AmountML: 900},
		},
	}
	if err := st.SaveCareEvent(e); err != nil {
		t.Fatal(err)
	}

	got, ok, err := st.CareEvent("care-1")
	if err != nil || !ok {
		t.Fatalf("CareEvent: ok=%v err=%v", ok, err)
	}
	if got.Type != "feed" || got.EC != 1.4 || got.Source != domain.CareManual {
		t.Fatalf("event fields not persisted: %+v", got)
	}
	if len(got.Applications) != 2 {
		t.Fatalf("expected 2 applications, got %d", len(got.Applications))
	}
	if got.TotalML() != 1800 {
		t.Fatalf("expected 1800 ml total, got %v", got.TotalML())
	}
}

func TestCareEventsNewestFirst(t *testing.T) {
	st := open(t)
	ids := seedCareGrow(t, st)
	base := time.Now()
	_ = st.SaveCareEvent(domain.CareEvent{ID: "c1", GrowID: "grow-1", Type: "water", OccurredAt: base.Add(-2 * time.Hour),
		Applications: []domain.CareApplication{{ID: "x1", PlantUnitID: ids[0], AmountML: 500}}})
	_ = st.SaveCareEvent(domain.CareEvent{ID: "c2", GrowID: "grow-1", Type: "water", OccurredAt: base,
		Applications: []domain.CareApplication{{ID: "x2", PlantUnitID: ids[0], AmountML: 600}}})

	events, err := st.CareEvents("grow-1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].ID != "c2" {
		t.Fatalf("expected newest first (c2), got %+v", events)
	}
}

func TestLastCareByType(t *testing.T) {
	st := open(t)
	ids := seedCareGrow(t, st)
	base := time.Now()
	_ = st.SaveCareEvent(domain.CareEvent{ID: "w1", GrowID: "grow-1", Type: "water", OccurredAt: base.Add(-3 * time.Hour),
		Applications: []domain.CareApplication{{ID: "a", PlantUnitID: ids[0], AmountML: 1}}})
	_ = st.SaveCareEvent(domain.CareEvent{ID: "w2", GrowID: "grow-1", Type: "water", OccurredAt: base.Add(-1 * time.Hour),
		Applications: []domain.CareApplication{{ID: "b", PlantUnitID: ids[0], AmountML: 2}}})
	_ = st.SaveCareEvent(domain.CareEvent{ID: "f1", GrowID: "grow-1", Type: "feed", OccurredAt: base.Add(-2 * time.Hour),
		Applications: []domain.CareApplication{{ID: "c", PlantUnitID: ids[0], AmountML: 3}}})

	last, err := st.LastCareByType("grow-1")
	if err != nil {
		t.Fatal(err)
	}
	if last["water"].ID != "w2" {
		t.Fatalf("expected latest water w2, got %q", last["water"].ID)
	}
	if last["feed"].ID != "f1" {
		t.Fatalf("expected feed f1, got %q", last["feed"].ID)
	}
	if len(last["water"].Applications) != 1 {
		t.Fatalf("applications not attached to lastByType")
	}
}

func TestLastCarePerPlant(t *testing.T) {
	st := open(t)
	ids := seedCareGrow(t, st)
	base := time.Now()
	// Plant 0 fed twice, plant 1 once, plant 2 never.
	_ = st.SaveCareEvent(domain.CareEvent{ID: "e1", GrowID: "grow-1", Type: "water", OccurredAt: base.Add(-2 * time.Hour),
		Applications: []domain.CareApplication{{ID: "a", PlantUnitID: ids[0], AmountML: 1}, {ID: "b", PlantUnitID: ids[1], AmountML: 1}}})
	_ = st.SaveCareEvent(domain.CareEvent{ID: "e2", GrowID: "grow-1", Type: "water", OccurredAt: base,
		Applications: []domain.CareApplication{{ID: "c", PlantUnitID: ids[0], AmountML: 1}}})

	perPlant, err := st.LastCarePerPlant("grow-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := perPlant[ids[2]]; ok {
		t.Fatalf("plant 2 never cared for should be absent")
	}
	if !perPlant[ids[0]].After(perPlant[ids[1]]) {
		t.Fatalf("plant 0's last care should be after plant 1's")
	}
}

func TestDeleteCareEvent(t *testing.T) {
	st := open(t)
	ids := seedCareGrow(t, st)
	_ = st.SaveCareEvent(domain.CareEvent{ID: "c1", GrowID: "grow-1", Type: "water", OccurredAt: time.Now(),
		Applications: []domain.CareApplication{{ID: "a1", PlantUnitID: ids[0], AmountML: 500}}})
	if err := st.DeleteCareEvent("c1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := st.CareEvent("c1"); ok {
		t.Fatalf("event should be gone")
	}
	apps, err := st.applicationsByEvent([]string{"c1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("applications should cascade on delete, got %+v", apps)
	}
}

func TestGrowCareConfigRoundTrip(t *testing.T) {
	st := open(t)
	seedCareGrow(t, st)

	// No config yet -> ok=false (fall back to species defaults).
	if _, ok, err := st.GrowCareConfig("grow-1"); err != nil || ok {
		t.Fatalf("expected no config, ok=%v err=%v", ok, err)
	}

	cfg := domain.GrowCareConfig{Actions: []domain.GrowCareActionConfig{
		{Key: "water", Enabled: true, Quick: true},
		{Key: "feed", Enabled: false, Quick: false},
		{Key: "foliar", Label: "Foliar spray", Enabled: true, Custom: true, Fields: []string{"product", "note"}},
	}}
	if err := st.SaveGrowCareConfig("grow-1", cfg); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.GrowCareConfig("grow-1")
	if err != nil || !ok {
		t.Fatalf("GrowCareConfig: ok=%v err=%v", ok, err)
	}
	if len(got.Actions) != 3 || got.Actions[1].Key != "feed" || got.Actions[1].Enabled {
		t.Fatalf("config not persisted: %+v", got.Actions)
	}
	if got.Actions[2].Label != "Foliar spray" || len(got.Actions[2].Fields) != 2 {
		t.Fatalf("custom action not persisted: %+v", got.Actions[2])
	}

	// Empty list clears the config back to species defaults.
	if err := st.SaveGrowCareConfig("grow-1", domain.GrowCareConfig{}); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := st.GrowCareConfig("grow-1"); ok {
		t.Fatalf("empty config should clear back to defaults")
	}
}

func TestDeleteGrowCascadesCare(t *testing.T) {
	st := open(t)
	ids := seedCareGrow(t, st)
	_ = st.SaveCareEvent(domain.CareEvent{ID: "c1", GrowID: "grow-1", Type: "water", OccurredAt: time.Now(),
		Applications: []domain.CareApplication{{ID: "a1", PlantUnitID: ids[0], AmountML: 500}}})
	if err := st.DeleteGrow("grow-1"); err != nil {
		t.Fatal(err)
	}
	events, err := st.CareEvents("grow-1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("care events should be removed with the grow, got %+v", events)
	}
}
