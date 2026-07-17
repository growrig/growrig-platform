package api

import "net/http"

func (s *Server) getAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.store.OpenAlerts()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (s *Server) ackAlert(w http.ResponseWriter, r *http.Request) {
	if err := s.store.AckAlert(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resolveAlert(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ResolveAlertID(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
