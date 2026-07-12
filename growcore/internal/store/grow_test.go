package store

import (
	"testing"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

func TestGrowRoundTrip(t *testing.T) {
	st := open(t)
	g := domain.Grow{
		ID: "grow-1", Name: "Basil batch", Species: "basil",
		Stage: "growth", Stages: domain.StagePresets["basil"],
		StartedAt: time.Now().Add(-48 * time.Hour), StageStarted: time.Now().Add(-24 * time.Hour),
		Status: domain.GrowActive, Notes: "windowsill",
	}
	if err := st.SaveGrow(g); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.Grow("grow-1")
	if err != nil || !ok {
		t.Fatalf("grow not found: ok=%v err=%v", ok, err)
	}
	if got.Name != "Basil batch" || got.Species != "basil" || got.Stage != "growth" {
		t.Fatalf("grow fields wrong: %+v", got)
	}
	if len(got.Stages) != 3 || got.Stages[0] != "seedling" {
		t.Fatalf("stages not persisted: %+v", got.Stages)
	}
	if got.Status != domain.GrowActive {
		t.Fatalf("status not persisted: %q", got.Status)
	}
}

func TestBulkCreateAndMovePlant(t *testing.T) {
	st := open(t)
	_ = st.SaveEnvironment(domain.Environment{ID: "tent-a", Name: "Tent A", Kind: domain.KindTent})
	_ = st.SaveEnvironment(domain.Environment{ID: "room-b", Name: "Room B", Kind: domain.KindRoom})
	_ = st.SaveGrow(domain.Grow{ID: "grow-1", Name: "G", Stages: domain.DefaultStages, Status: domain.GrowActive})

	units, err := st.BulkCreatePlants("grow-1", 3, domain.TrackIndividual, 1, "Plant", "Genovese", "tent-a", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 3 {
		t.Fatalf("expected 3 units, got %d", len(units))
	}
	// Cultivar is a per-unit attribute and must round-trip.
	if got, ok, _ := st.PlantUnit(units[0].ID); !ok || got.Cultivar != "Genovese" {
		t.Fatalf("cultivar not persisted on unit: %+v", got)
	}
	// All three should currently be in tent-a.
	inTent, err := st.PlantsInEnvironment("tent-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(inTent) != 3 {
		t.Fatalf("expected 3 plants in tent, got %d", len(inTent))
	}

	// Move one plant to room-b. The old placement must close and a new open one
	// must appear, so the unit is in exactly one place.
	moved := units[0].ID
	before := time.Now()
	if err := st.MovePlant(moved, "room-b", time.Now()); err != nil {
		t.Fatal(err)
	}
	inTent, _ = st.PlantsInEnvironment("tent-a")
	if len(inTent) != 2 {
		t.Fatalf("expected 2 plants left in tent, got %d", len(inTent))
	}
	inRoom, _ := st.PlantsInEnvironment("room-b")
	if len(inRoom) != 1 || inRoom[0].ID != moved {
		t.Fatalf("moved plant not in room: %+v", inRoom)
	}
	// The unit must have exactly one open placement, and its history shows the
	// prior one closed.
	placements, err := st.PlacementsForUnit(moved)
	if err != nil {
		t.Fatal(err)
	}
	if len(placements) != 2 {
		t.Fatalf("expected 2 placements in history, got %d", len(placements))
	}
	open := 0
	for _, p := range placements {
		if p.EndedAt == nil {
			open++
			if p.EnvironmentID != "room-b" {
				t.Fatalf("open placement should be room-b, got %s", p.EnvironmentID)
			}
		} else if p.EndedAt.Before(before.Add(-time.Minute)) {
			t.Fatalf("closed placement ended too early: %v", p.EndedAt)
		}
	}
	if open != 1 {
		t.Fatalf("expected exactly 1 open placement, got %d", open)
	}
}

func TestDeleteGrowCascades(t *testing.T) {
	st := open(t)
	_ = st.SaveEnvironment(domain.Environment{ID: "tent-a", Name: "Tent A", Kind: domain.KindTent, ControlGrowID: "grow-1"})
	_ = st.SaveGrow(domain.Grow{ID: "grow-1", Name: "G", Stages: domain.DefaultStages, Status: domain.GrowActive})
	units, _ := st.BulkCreatePlants("grow-1", 2, domain.TrackGroup, 5, "Tray", "", "tent-a", time.Now())

	if err := st.DeleteGrow("grow-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := st.Grow("grow-1"); ok {
		t.Fatalf("grow should be deleted")
	}
	if u, _ := st.PlantUnits("grow-1"); len(u) != 0 {
		t.Fatalf("plant units should be deleted, got %d", len(u))
	}
	if p, _ := st.PlacementsForUnit(units[0].ID); len(p) != 0 {
		t.Fatalf("placements should be deleted, got %d", len(p))
	}
	// The environment's control grow reference must be cleared.
	envs, _ := st.Environments()
	for _, e := range envs {
		if e.ID == "tent-a" && e.ControlGrowID != "" {
			t.Fatalf("control grow ref should be cleared, got %q", e.ControlGrowID)
		}
	}
}

func TestMigrateCyclesToGrows(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/mig.db"
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a legacy database: an environment with a cannabis cycle row.
	_ = st.SaveEnvironment(domain.Environment{ID: "tent-a", Name: "Tent A", Kind: domain.KindTent})
	_, err = st.db.Exec(
		`INSERT INTO cycles (environment_id, strain, started_at, phase, phase_started, notes)
		 VALUES (?, ?, ?, ?, ?, '')`,
		"tent-a", "OG Kush", time.Now().Add(-21*24*time.Hour).UnixMilli(), "flowering",
		time.Now().Add(-3*24*time.Hour).UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	// Clear grows so the migration runs on reopen, then re-run migrate().
	if err := st.migrateCyclesToGrows(); err != nil {
		t.Fatal(err)
	}
	grows, _ := st.Grows()
	if len(grows) != 1 {
		t.Fatalf("expected 1 migrated grow, got %d", len(grows))
	}
	g := grows[0]
	if g.Species != "cannabis" || g.Stage != "flowering" {
		t.Fatalf("migrated grow fields wrong: %+v", g)
	}
	// The legacy strain becomes the migrated plant unit's cultivar.
	units, _ := st.PlantUnits(g.ID)
	if len(units) != 1 || units[0].Cultivar != "OG Kush" {
		t.Fatalf("migrated plant cultivar wrong: %+v", units)
	}
	// The environment should now nominate the grow as its control grow.
	envs, _ := st.Environments()
	found := false
	for _, e := range envs {
		if e.ID == "tent-a" {
			found = e.ControlGrowID == g.ID
		}
	}
	if !found {
		t.Fatalf("environment control grow not set from migration")
	}
	// A group plant unit placed in the tent should exist.
	if in, _ := st.PlantsInEnvironment("tent-a"); len(in) != 1 {
		t.Fatalf("expected 1 migrated plant in tent, got %d", len(in))
	}
	// Re-running is a no-op (idempotent).
	if err := st.migrateCyclesToGrows(); err != nil {
		t.Fatal(err)
	}
	if grows, _ := st.Grows(); len(grows) != 1 {
		t.Fatalf("migration should be idempotent, got %d grows", len(grows))
	}
	st.Close()
}
