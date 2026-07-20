package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
	"github.com/growrig/growrig/growcore/internal/store"
)

// stageNames returns the ordered stage names of a grow's recorded history.
func stageNames(t *testing.T, st *store.Store, growID string) []string {
	t.Helper()
	events, err := st.StageEvents(growID)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Stage
	}
	return out
}

func changeStageReq(t *testing.T, s *Server, growID, stage string) domain.Grow {
	t.Helper()
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"stage":"`+stage+`"}`))
	req.SetPathValue("id", growID)
	rec := httptest.NewRecorder()
	s.changeStage(rec, req)
	if rec.Code != 200 {
		t.Fatalf("changeStage(%s): code=%d body=%s", stage, rec.Code, rec.Body.String())
	}
	var g domain.Grow
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	return g
}

// Advancing forward records each stage; reverting to an earlier stage discards
// the stages entered past it and keeps the earlier stage's original date.
func TestChangeStageAdvanceThenRevert(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	stages := []string{"seedling", "vegetative", "flowering", "flush"}
	if err := st.SaveGrow(domain.Grow{ID: "g1", Name: "G", Species: "cannabis", Stage: "seedling", Stages: stages, Status: domain.GrowActive, StageStarted: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetStageDate("g1", "seedling", time.Now().Add(-72*time.Hour)); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st}

	// Advance seedling -> vegetative -> flowering; each gets a recorded date.
	changeStageReq(t, s, "g1", "vegetative")
	vegGrow := changeStageReq(t, s, "g1", "flowering")
	if vegGrow.Stage != "flowering" {
		t.Fatalf("expected current stage flowering, got %s", vegGrow.Stage)
	}
	if got := stageNames(t, st, "g1"); len(got) != 3 {
		t.Fatalf("expected 3 recorded stages after advancing, got %v", got)
	}
	vegDate := stageDateOf(t, st, "g1", "vegetative")

	// Revert flowering -> vegetative: flowering must be discarded, vegetative kept.
	g := changeStageReq(t, s, "g1", "vegetative")
	if g.Stage != "vegetative" {
		t.Fatalf("expected current stage vegetative after revert, got %s", g.Stage)
	}
	got := stageNames(t, st, "g1")
	if len(got) != 2 || got[1] != "vegetative" {
		t.Fatalf("expected [seedling vegetative] after revert, got %v", got)
	}
	if !g.StageStarted.Equal(vegDate) {
		t.Fatalf("expected StageStarted to keep vegetative's original date %v, got %v", vegDate, g.StageStarted)
	}
}

func stageDateOf(t *testing.T, st *store.Store, growID, stage string) time.Time {
	t.Helper()
	events, err := st.StageEvents(growID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.Stage == stage {
			return e.EnteredAt
		}
	}
	t.Fatalf("no recorded date for stage %s", stage)
	return time.Time{}
}

// putStageDates sets and clears recorded dates and keeps StageStarted aligned
// with the current stage.
func TestPutStageDates(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	stages := []string{"seedling", "vegetative", "flowering"}
	if err := st.SaveGrow(domain.Grow{ID: "g1", Name: "G", Species: "cannabis", Stage: "vegetative", Stages: stages, Status: domain.GrowActive, StageStarted: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetStageDate("g1", "seedling", time.Now().Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := st.SetStageDate("g1", "vegetative", time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st}

	// Correct vegetative's date and clear seedling's.
	body := `{"dates":{"vegetative":"2026-01-10","seedling":""}}`
	req := httptest.NewRequest("PUT", "/", strings.NewReader(body))
	req.SetPathValue("id", "g1")
	rec := httptest.NewRecorder()
	s.putStageDates(rec, req)
	if rec.Code != 200 {
		t.Fatalf("putStageDates: code=%d body=%s", rec.Code, rec.Body.String())
	}

	got := stageNames(t, st, "g1")
	if len(got) != 1 || got[0] != "vegetative" {
		t.Fatalf("expected only [vegetative] recorded, got %v", got)
	}
	// StageStarted follows the current stage's edited date.
	g, _, _ := st.Grow("g1")
	want := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	if !g.StageStarted.Equal(want) {
		t.Fatalf("expected StageStarted %v, got %v", want, g.StageStarted)
	}

	// A stage outside the sequence is rejected.
	bad := httptest.NewRequest("PUT", "/", strings.NewReader(`{"dates":{"harvest":"2026-01-10"}}`))
	bad.SetPathValue("id", "g1")
	brec := httptest.NewRecorder()
	s.putStageDates(brec, bad)
	if brec.Code != 400 {
		t.Fatalf("expected 400 for unknown stage, got %d", brec.Code)
	}
}
