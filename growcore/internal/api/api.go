// Package api exposes Grow Core over HTTP: a REST surface for configuration and
// discovery, plus a WebSocket that streams the live system snapshot.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/coder/websocket"

	"github.com/growrig/growrig-platform/growcore/internal/catalog"
	"github.com/growrig/growrig-platform/growcore/internal/control"
	"github.com/growrig/growrig-platform/growcore/internal/domain"
	"github.com/growrig/growrig-platform/growcore/internal/store"
)

type Server struct {
	store       *store.Store
	engine      *control.Engine
	adapter     control.Adapter
	hub         *Hub
	adapterType string
	static      http.Handler
}

func NewServer(st *store.Store, eng *control.Engine, adapter control.Adapter, hub *Hub, adapterType string, static http.Handler) *Server {
	return &Server{store: st, engine: eng, adapter: adapter, hub: hub, adapterType: adapterType, static: static}
}

// Handler builds the HTTP router.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/info", s.getInfo)
	mux.HandleFunc("GET /api/state", s.getState)
	mux.HandleFunc("GET /api/roles", s.getRoles)
	mux.HandleFunc("GET /api/phases", s.getPhases)
	mux.HandleFunc("GET /api/catalog", s.getCatalog)
	mux.HandleFunc("GET /api/discovery", s.getDiscovery)
	mux.HandleFunc("POST /api/demo", s.postDemo)

	mux.HandleFunc("GET /api/environments", s.getEnvironments)
	mux.HandleFunc("POST /api/environments", s.createEnvironment)
	mux.HandleFunc("PUT /api/environments/{id}", s.updateEnvironment)
	mux.HandleFunc("DELETE /api/environments/{id}", s.deleteEnvironment)
	mux.HandleFunc("PUT /api/environments/{id}/targets", s.putTargets)
	mux.HandleFunc("GET /api/environments/{id}/history", s.getHistory)
	mux.HandleFunc("PUT /api/environments/{id}/cycle", s.putCycle)
	mux.HandleFunc("DELETE /api/environments/{id}/cycle", s.deleteCycle)

	mux.HandleFunc("GET /api/bindings", s.getBindings)
	mux.HandleFunc("POST /api/bindings", s.createBinding)
	mux.HandleFunc("PUT /api/bindings/{id}", s.updateBinding)
	mux.HandleFunc("DELETE /api/bindings/{id}", s.deleteBinding)
	mux.HandleFunc("PUT /api/bindings/{id}/switch", s.putSwitch)

	mux.HandleFunc("GET /api/ws", s.ws)

	if s.static != nil {
		mux.Handle("/", s.static)
	}
	return withCORS(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"adapter": s.adapterType})
}

func (s *Server) getState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.engine.Latest())
}

func (s *Server) getRoles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.AllFanRoles)
}

func (s *Server) getPhases(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.AllPhases)
}

func (s *Server) getCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, catalog.Products())
}

func (s *Server) getDiscovery(w http.ResponseWriter, r *http.Request) {
	found := s.adapter.Discover()
	if found == nil {
		found = []control.DiscoveredEntity{}
	}
	writeJSON(w, http.StatusOK, found)
}

func (s *Server) getEnvironments(w http.ResponseWriter, r *http.Request) {
	envs, err := s.store.Environments()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if envs == nil {
		envs = []domain.Environment{}
	}
	writeJSON(w, http.StatusOK, envs)
}

func (s *Server) putTargets(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetTempC    float64 `json:"targetTempC"`
		TargetHumidity float64 `json:"targetHumidity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if body.TargetTempC < 5 || body.TargetTempC > 45 {
		writeJSON(w, http.StatusBadRequest, errBody("targetTempC must be between 5 and 45"))
		return
	}
	if body.TargetHumidity < 10 || body.TargetHumidity > 95 {
		writeJSON(w, http.StatusBadRequest, errBody("targetHumidity must be between 10 and 95"))
		return
	}
	if err := s.store.UpdateTargets(r.PathValue("id"), body.TargetTempC, body.TargetHumidity); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getHistory(w http.ResponseWriter, r *http.Request) {
	limit := 120
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			limit = n
		}
	}
	readings, err := s.store.RecentReadings(r.PathValue("id"), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if readings == nil {
		readings = []domain.Reading{}
	}
	writeJSON(w, http.StatusOK, readings)
}

func (s *Server) getBindings(w http.ResponseWriter, r *http.Request) {
	bindings, err := s.store.Bindings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if bindings == nil {
		bindings = []domain.Binding{}
	}
	writeJSON(w, http.StatusOK, bindings)
}

func (s *Server) putSwitch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		On bool `json:"on"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	bindings, err := s.store.Bindings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	id := r.PathValue("id")
	for _, b := range bindings {
		if b.ID == id {
			if err := s.adapter.SetSwitch(b.Entity, body.On); err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, errBody("binding not found"))
}

func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer c.CloseNow()
	s.hub.serveWS(c, s.engine.Latest())
}

func validRole(role domain.Role) bool {
	for _, r := range domain.AllFanRoles {
		if r == role {
			return true
		}
	}
	return false
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: encode: %v", err)
	}
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errBody(err.Error()))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
