package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
	"github.com/growrig/growrig-platform/growcore/internal/feeding"
	"github.com/growrig/growrig-platform/growcore/internal/species"
)

// Feeding presets are user-owned nutrient schedules stored as YAML on disk (see
// store/feeding.go). Built-in presets (the BioBizz charts) are a separate,
// read-only catalog exposed only as *templates* to prefill the create form —
// they never appear in the user's own list.

// getFeedingPresets lists the user's presets (the ones shown on the Grows page).
func (s *Server) getFeedingPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := s.store.FeedingPresets()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if presets == nil {
		presets = []domain.FeedingPreset{}
	}
	writeJSON(w, http.StatusOK, presets)
}

// getFeedingTemplates lists the built-in presets used to seed a new preset,
// optionally filtered to one species.
func (s *Server) getFeedingTemplates(w http.ResponseWriter, r *http.Request) {
	sp := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("species")))
	var templates []domain.FeedingPreset
	if sp != "" {
		templates = feeding.BySpecies(sp)
	} else {
		templates = feeding.All()
	}
	if templates == nil {
		templates = []domain.FeedingPreset{}
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) getFeedingPreset(w http.ResponseWriter, r *http.Request) {
	p, ok, err := s.store.FeedingPreset(r.PathValue("id"))
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
	p, ok, err := s.store.FeedingPreset(r.PathValue("id"))
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
	if err := s.store.DeleteFeedingPreset(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
