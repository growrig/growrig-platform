package api

import (
	"net/http"

	"github.com/growrig/growrig/growcore/internal/tailscale"
)

// tailscaleResponse is the node status plus whether the feature is compiled/
// wired in at all (nil manager => remote access unavailable in this build).
type tailscaleResponse struct {
	Available bool `json:"available"`
	tailscale.Status
}

func (s *Server) tailscaleResp() tailscaleResponse {
	if s.tailscale == nil {
		return tailscaleResponse{Available: false, Status: tailscale.Status{State: tailscale.StateStopped}}
	}
	return tailscaleResponse{Available: true, Status: s.tailscale.Status()}
}

// getTailscale returns the current remote-access status.
func (s *Server) getTailscale(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.tailscaleResp())
}

// enableTailscale turns on private remote access and starts the node. The
// response includes the auth URL (once available) for the UI to show.
func (s *Server) enableTailscale(w http.ResponseWriter, r *http.Request) {
	if s.tailscale == nil {
		writeJSON(w, http.StatusServiceUnavailable, errBody("remote access is not available in this build"))
		return
	}
	var body struct {
		Hostname   string `json:"hostname"`
		ControlURL string `json:"controlUrl"`
	}
	// Body is optional — enabling with no options keeps the stored hostname.
	_ = decode(r, &body)
	if err := s.tailscale.Enable(body.Hostname, body.ControlURL); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "configuration", "Enabled Tailscale remote access")
	writeJSON(w, http.StatusOK, s.tailscaleResp())
}

// disableTailscale stops the node. The persisted node identity is kept, so
// re-enabling does not require re-authenticating.
func (s *Server) disableTailscale(w http.ResponseWriter, r *http.Request) {
	if s.tailscale == nil {
		writeJSON(w, http.StatusServiceUnavailable, errBody("remote access is not available in this build"))
		return
	}
	if err := s.tailscale.Disable(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.activity("", "", "info", "configuration", "Disabled Tailscale remote access")
	writeJSON(w, http.StatusOK, s.tailscaleResp())
}
