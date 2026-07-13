package api

import (
	"errors"
	"net/http"

	"github.com/growrig/growrig/growcore/internal/catalogsource"
)

// Custom catalog sources: supported public Git repositories with a catalog.yaml
// manifest that extend the built-in device/integration catalogs. Grow Core
// derives and downloads a provider source archive; it never clones the repo.

func (s *Server) getCatalogSources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"sources":     s.catalogSources.List(),
		"mergedKinds": catalogsource.MergedKinds,
	})
}

func (s *Server) createCatalogSource(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Repository string `json:"repository"`
		Repo       string `json:"repo"` // accepted for compatibility with older clients
		Ref        string `json:"ref"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	repository := in.Repository
	if repository == "" {
		repository = in.Repo
	}
	src, err := s.catalogSources.Add(repository, in.Ref)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	s.activity("", "", "info", "catalog", "Catalog source added: "+src.Name+" ("+src.Repository+")")
	writeJSON(w, http.StatusCreated, src)
}

func (s *Server) refreshCatalogSource(w http.ResponseWriter, r *http.Request) {
	src, err := s.catalogSources.Refresh(r.PathValue("id"))
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, catalogsource.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeErr(w, status, err)
		return
	}
	s.activity("", "", "info", "catalog", "Catalog source refreshed: "+src.Name+" ("+src.Repository+")")
	writeJSON(w, http.StatusOK, src)
}

func (s *Server) deleteCatalogSource(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.catalogSources.Remove(id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, catalogsource.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeErr(w, status, err)
		return
	}
	s.activity("", "", "info", "catalog", "Catalog source removed: "+id)
	w.WriteHeader(http.StatusNoContent)
}
