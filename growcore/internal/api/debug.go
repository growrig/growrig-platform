package api

import "net/http"

func (s *Server) getDatabaseTables(w http.ResponseWriter, r *http.Request) {
	tables, err := s.store.DatabaseTables()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tables)
}
