package api

import (
	"net/http"
	"os"
	"syscall"
	"time"
)

// restart triggers a graceful process restart. It records the intent, replies
// immediately, then signals itself (SIGTERM) so main's normal shutdown path
// runs — which logs "Grow Core stopped" and exits. Under a process supervisor
// (the Home Assistant add-on's container, systemd, `make dev`) the process is
// then brought back up, emitting a fresh "Grow Core started". In a bare `go run`
// with no supervisor it simply stops.
func (s *Server) restart(w http.ResponseWriter, r *http.Request) {
	s.activity("", "", "info", "system", "Grow Core restart requested")
	w.WriteHeader(http.StatusAccepted)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	// Small delay so this response reaches the client before we tear down.
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}()
}

// clearActivity empties the activity log, then drops a single marker so the log
// isn't jarringly blank and there's a record of who cleared it.
func (s *Server) clearActivity(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ClearActivities(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "system", "Activity log cleared")
	w.WriteHeader(http.StatusNoContent)
}
