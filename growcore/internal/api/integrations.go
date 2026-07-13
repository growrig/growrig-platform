package api

import (
	"net/http"

	"github.com/growrig/growrig/growcore/internal/integrations"
)

func (s *Server) getIntegrationBundles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.integrations.Bundles())
}
func (s *Server) getIntegrationIcon(w http.ResponseWriter, r *http.Request) {
	raw, err := s.integrations.BundleAsset(r.PathValue("id"), "icon.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(raw)
}
func (s *Server) getIntegrationInstances(w http.ResponseWriter, r *http.Request) {
	items, err := s.integrations.Instances()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s *Server) createIntegrationInstance(w http.ResponseWriter, r *http.Request) {
	var in integrations.InstanceInput
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.integrations.Create(in)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (s *Server) updateIntegrationInstance(w http.ResponseWriter, r *http.Request) {
	var in integrations.InstanceInput
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.integrations.Update(r.PathValue("id"), in)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) deleteIntegrationInstance(w http.ResponseWriter, r *http.Request) {
	if err := s.integrations.Delete(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) testIntegrationInstance(w http.ResponseWriter, r *http.Request) {
	item, err := s.integrations.Test(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "instance": item})
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) invokeIntegration(w http.ResponseWriter, r *http.Request) {
	input := map[string]any{}
	if err := decode(r, &input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	out, err := s.integrations.Invoke(r.Context(), r.PathValue("id"), r.PathValue("capability"), input)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (s *Server) getIntegrationBindings(w http.ResponseWriter, r *http.Request) {
	items, err := s.integrations.Bindings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s *Server) saveIntegrationBinding(w http.ResponseWriter, r *http.Request) {
	var in integrations.BindingInput
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.integrations.SaveBinding(in)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) deleteIntegrationBinding(w http.ResponseWriter, r *http.Request) {
	if err := s.integrations.DeleteBinding(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) resolveIntegrationBinding(w http.ResponseWriter, r *http.Request) {
	item, err := s.integrations.ResolveFor(r.URL.Query().Get("feature"), r.URL.Query().Get("growId"), r.URL.Query().Get("environmentId"), r.URL.Query().Get("capability"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if item == nil {
		writeErr(w, http.StatusNotFound, errNoCompatibleIntegration)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

var errNoCompatibleIntegration = integrationError("no compatible configured integration")

type integrationError string

func (e integrationError) Error() string { return string(e) }
