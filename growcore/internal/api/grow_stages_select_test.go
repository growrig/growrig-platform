package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/growrig/growrig/growcore/internal/domain"
	"github.com/growrig/growrig/growcore/internal/store"
)

func createGrowReq(t *testing.T, s *Server, body string) domain.Grow {
	t.Helper()
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.createGrow(rec, req)
	if rec.Code != 200 {
		t.Fatalf("createGrow: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var g domain.Grow
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func updateGrowReq(t *testing.T, s *Server, id, body string) int {
	t.Helper()
	req := httptest.NewRequest("PUT", "/", strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	s.updateGrow(rec, req)
	return rec.Code
}

// A grow's stage selection keeps only the requested optional stages (required
// stages are always in), and updates may not drop stages already entered.
func TestCreateGrowStageSelection(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := &Server{store: st}

	// Omit every optional stage: only the required stages remain.
	g := createGrowReq(t, s, `{"name":"Lean","species":"cannabis","stages":["vegetative","flowering"]}`)
	if got, want := strings.Join(g.Stages, ","), "vegetative,flowering"; got != want {
		t.Fatalf("stages = %q, want %q", got, want)
	}
	if g.Stage != "vegetative" {
		t.Fatalf("starting stage = %q, want vegetative", g.Stage)
	}

	// Adding a future optional stage (cure) is allowed.
	if code := updateGrowReq(t, s, g.ID, `{"name":"Lean","species":"cannabis","stages":["vegetative","flowering","cure"]}`); code != 200 {
		t.Fatalf("adding cure: code=%d", code)
	}
	g2, _, _ := st.Grow(g.ID)
	if got, want := strings.Join(g2.Stages, ","), "vegetative,flowering,cure"; got != want {
		t.Fatalf("stages after add = %q, want %q", got, want)
	}

	// A default grow starts at seedling (an optional stage) and records it. That
	// stage has been entered, so a later edit may not drop it — but a future
	// optional stage (cure) that hasn't been reached can still be removed.
	full := createGrowReq(t, s, `{"name":"Full","species":"cannabis"}`)
	if full.Stage != "seedling" {
		t.Fatalf("default starting stage = %q, want seedling", full.Stage)
	}
	if code := updateGrowReq(t, s, full.ID, `{"name":"Full","species":"cannabis","stages":["seedling","vegetative","flowering","flush","drying"]}`); code != 200 {
		t.Fatalf("removing future optional cure: code=%d, want 200", code)
	}
	if code := updateGrowReq(t, s, full.ID, `{"name":"Full","species":"cannabis","stages":["vegetative","flowering"]}`); code != 400 {
		t.Fatalf("dropping entered seedling stage: code=%d, want 400", code)
	}
}
