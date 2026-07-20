package api

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
	"github.com/growrig/growrig/growcore/internal/species"
)

// --- Species catalog ---

func (s *Server) getSpecies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, species.All())
}

// getSpeciesIcon serves a species' catalog icon.svg (the grow form's species
// picker artwork). 404 when the species ships no icon.
func (s *Server) getSpeciesIcon(w http.ResponseWriter, r *http.Request) {
	raw, err := species.Asset(r.PathValue("id"), "icon.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(raw)
}

// --- Cultivars ---

func (s *Server) getCultivars(w http.ResponseWriter, r *http.Request) {
	sp := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("species")))
	cultivars, err := s.store.Cultivars(sp)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if cultivars == nil {
		cultivars = []domain.Cultivar{}
	}
	writeJSON(w, http.StatusOK, cultivars)
}

func (s *Server) getCultivar(w http.ResponseWriter, r *http.Request) {
	c, ok, err := s.store.Cultivar(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("cultivar not found"))
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type cultivarBody struct {
	Species     string            `json:"species"`
	Name        string            `json:"name"`
	Creator     string            `json:"creator"`
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"`
	// Image is an optional data URL ("data:image/png;base64,…"). Empty leaves the
	// existing image unchanged on update; RemoveImage explicitly clears it.
	Image       string `json:"image"`
	RemoveImage bool   `json:"removeImage"`
}

// sanitizeAttributes keeps only the attribute keys declared by the species and,
// for enum attributes, only values present in the declared options.
func sanitizeAttributes(sp species.Species, in map[string]string) map[string]string {
	out := map[string]string{}
	for _, attr := range sp.CultivarAttributes {
		v := strings.TrimSpace(in[attr.Key])
		if v == "" {
			continue
		}
		if attr.Type == species.AttrEnum && len(attr.Options) > 0 && !containsStr(attr.Options, v) {
			continue
		}
		out[attr.Key] = v
	}
	return out
}

func (s *Server) createCultivar(w http.ResponseWriter, r *http.Request) {
	var b cultivarBody
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
	c := domain.Cultivar{
		ID:          id(b.Name, "cultivar"),
		Species:     sp.ID,
		Name:        strings.TrimSpace(b.Name),
		Creator:     strings.TrimSpace(b.Creator),
		Description: strings.TrimSpace(b.Description),
		Attributes:  sanitizeAttributes(sp, b.Attributes),
		CreatedAt:   time.Now(),
	}
	if err := s.store.SaveCultivar(c); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if data, mime, ok := decodeDataURL(b.Image); ok {
		if err := s.store.SetCultivarImage(c.ID, data, mime); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		c.ImageType = mime
	}
	s.activity("", "", "info", "configuration", "Created cultivar "+c.Name)
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) updateCultivar(w http.ResponseWriter, r *http.Request) {
	c, ok, err := s.store.Cultivar(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("cultivar not found"))
		return
	}
	var b cultivarBody
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
	c.Species = sp.ID
	c.Name = strings.TrimSpace(b.Name)
	c.Creator = strings.TrimSpace(b.Creator)
	c.Description = strings.TrimSpace(b.Description)
	c.Attributes = sanitizeAttributes(sp, b.Attributes)
	if err := s.store.SaveCultivar(c); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	switch {
	case b.RemoveImage:
		if err := s.store.ClearCultivarImage(c.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		c.ImageType = ""
	default:
		if data, mime, ok := decodeDataURL(b.Image); ok {
			if err := s.store.SetCultivarImage(c.ID, data, mime); err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			c.ImageType = mime
		}
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) deleteCultivar(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteCultivar(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getCultivarImage(w http.ResponseWriter, r *http.Request) {
	data, mime, ok, err := s.store.CultivarImage(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	serveImage(w, r, data, mime, ok, "no-cache")
}

// decodeDataURL parses a "data:<mime>;base64,<payload>" image URL. It returns
// ok=false for empty input or anything that isn't a base64 image data URL.
func decodeDataURL(s string) (data []byte, mime string, ok bool) {
	if !strings.HasPrefix(s, "data:") {
		return nil, "", false
	}
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return nil, "", false
	}
	header := s[len("data:"):comma]
	if !strings.HasSuffix(header, ";base64") {
		return nil, "", false
	}
	mime = strings.TrimSuffix(header, ";base64")
	if !strings.HasPrefix(mime, "image/") {
		return nil, "", false
	}
	raw, err := base64.StdEncoding.DecodeString(s[comma+1:])
	if err != nil || len(raw) == 0 {
		return nil, "", false
	}
	return raw, mime, true
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
