package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
	"github.com/growrig/growrig-platform/growcore/internal/feeding"
	"github.com/growrig/growrig-platform/growcore/internal/species"
)

// Feeding presets combine two sources behind one API: read-only built-ins from
// the species/<id>/feedings.yaml tree (Source "builtin", id "<species>/<slug>")
// and editable user presets from the DB (Source "user"). Reads merge both;
// writes only ever touch user presets — a builtin id is rejected.

func isBuiltinFeeding(id string) bool {
	_, ok := feeding.Get(id)
	return ok
}

func (s *Server) getFeedingPresets(w http.ResponseWriter, r *http.Request) {
	sp := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("species")))
	var builtins []domain.FeedingPreset
	if sp != "" {
		builtins = feeding.BySpecies(sp)
	} else {
		builtins = feeding.All()
	}
	user, err := s.store.FeedingPresets(sp)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// Built-ins first, then user presets (newest first from the store).
	out := make([]domain.FeedingPreset, 0, len(builtins)+len(user))
	out = append(out, builtins...)
	out = append(out, user...)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getFeedingPreset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if p, ok := feeding.Get(id); ok {
		writeJSON(w, http.StatusOK, p)
		return
	}
	p, ok, err := s.store.FeedingPreset(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("feeding preset not found"))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type feedingPresetBody struct {
	Species     string                  `json:"species"`
	Name        string                  `json:"name"`
	Brand       string                  `json:"brand"`
	Description string                  `json:"description"`
	Unit        string                  `json:"unit"`
	Products    []domain.FeedingProduct `json:"products"`
	Phases      []domain.FeedingPhase   `json:"phases"`
}

// sanitizeFeeding trims text and drops empty product/phase/week entries so the
// stored preset stays clean regardless of editor churn.
func sanitizeFeeding(b feedingPresetBody) (products []domain.FeedingProduct, phases []domain.FeedingPhase) {
	valid := map[string]bool{}
	for _, p := range b.Products {
		key := strings.TrimSpace(p.Key)
		label := strings.TrimSpace(p.Label)
		if key == "" || label == "" {
			continue
		}
		valid[key] = true
		products = append(products, domain.FeedingProduct{
			Key:   key,
			Label: label,
			Unit:  strings.TrimSpace(p.Unit),
		})
	}
	for _, ph := range b.Phases {
		name := strings.TrimSpace(ph.Name)
		if name == "" {
			continue
		}
		phase := domain.FeedingPhase{Name: name, Stage: strings.TrimSpace(ph.Stage)}
		for _, wk := range ph.Weeks {
			doses := map[string]float64{}
			for k, v := range wk.Doses {
				if valid[k] && v > 0 {
					doses[k] = v
				}
			}
			phase.Weeks = append(phase.Weeks, domain.FeedingWeek{Doses: doses})
		}
		phases = append(phases, phase)
	}
	return products, phases
}

func (s *Server) createFeedingPreset(w http.ResponseWriter, r *http.Request) {
	var b feedingPresetBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(b.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errBody("name is required"))
		return
	}
	sp, ok := species.Get(b.Species)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errBody("species must be one of the predefined crop families"))
		return
	}
	products, phases := sanitizeFeeding(b)
	p := domain.FeedingPreset{
		ID:          id(b.Name, "feeding"),
		Species:     sp.ID,
		Name:        strings.TrimSpace(b.Name),
		Brand:       strings.TrimSpace(b.Brand),
		Description: strings.TrimSpace(b.Description),
		Source:      "user",
		Unit:        strings.TrimSpace(b.Unit),
		Products:    products,
		Phases:      phases,
		CreatedAt:   time.Now(),
	}
	if err := s.store.SaveFeedingPreset(p); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "configuration", "Created feeding preset "+p.Name)
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) updateFeedingPreset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if isBuiltinFeeding(id) {
		writeJSON(w, http.StatusForbidden, errBody("built-in presets are read-only; duplicate it to customize"))
		return
	}
	p, ok, err := s.store.FeedingPreset(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("feeding preset not found"))
		return
	}
	var b feedingPresetBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(b.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errBody("name is required"))
		return
	}
	sp, ok := species.Get(b.Species)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errBody("species must be one of the predefined crop families"))
		return
	}
	products, phases := sanitizeFeeding(b)
	p.Species = sp.ID
	p.Name = strings.TrimSpace(b.Name)
	p.Brand = strings.TrimSpace(b.Brand)
	p.Description = strings.TrimSpace(b.Description)
	p.Unit = strings.TrimSpace(b.Unit)
	p.Products = products
	p.Phases = phases
	if err := s.store.SaveFeedingPreset(p); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) deleteFeedingPreset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if isBuiltinFeeding(id) {
		writeJSON(w, http.StatusForbidden, errBody("built-in presets cannot be deleted"))
		return
	}
	if err := s.store.DeleteFeedingPreset(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
