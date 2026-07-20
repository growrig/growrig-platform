// Package tailscale gives GrowRig optional private remote access by embedding a
// Tailscale node (tsnet) into the process. When enabled, it exposes the same web
// UI/API over a tailnet-only HTTPS listener, alongside the normal LAN server —
// no port forwarding, no separate tailscaled daemon, no root.
//
// Design choices:
//   - The node joins the *user's own* tailnet via interactive login; GrowRig
//     never owns a shared tailnet and never uses an embedded auth key.
//   - Only the web handler is served. SQLite, MQTT, Home Assistant and device
//     ports are never exposed.
//   - Node state is persisted (Dir) so the user authenticates once, not every
//     restart. GrowRig's own login/permissions still apply on top.
//   - Funnel (public internet exposure) is never used.
package tailscale

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"tailscale.com/tsnet"
)

// State is the coarse lifecycle of the embedded node, surfaced to the UI.
type State string

const (
	StateStopped    State = "stopped"     // not running
	StateStarting   State = "starting"    // bringing the node up
	StateNeedsLogin State = "needs-login" // waiting for the user to authorize (AuthURL set)
	StateRunning    State = "running"     // reachable on the tailnet
	StateError      State = "error"       // failed to start / run (Error set)
)

// Settings are the persisted, user-controlled options (dataDir/tailscale.yaml).
type Settings struct {
	Enabled bool `yaml:"enabled"`
	// Hostname the node presents to the tailnet; becomes the remote hostname.
	Hostname string `yaml:"hostname"`
	// ControlURL optionally points at an alternative coordination server. Empty
	// uses Tailscale's default. A path toward self-hosted control planes.
	ControlURL string `yaml:"controlURL,omitempty"`
}

// Status is a snapshot of the node for the UI.
type Status struct {
	Enabled    bool       `json:"enabled"`
	State      State      `json:"state"`
	Hostname   string     `json:"hostname"`
	AuthURL    string     `json:"authUrl,omitempty"` // shown as link/QR while needs-login
	URL        string     `json:"url,omitempty"`     // https remote URL once running
	DNSName    string     `json:"dnsName,omitempty"` // MagicDNS FQDN
	ControlURL string     `json:"controlUrl,omitempty"`
	KeyExpiry  *time.Time `json:"keyExpiry,omitempty"` // node-key expiry; nil when disabled
	KeyExpired bool       `json:"keyExpired"`
	Error      string     `json:"error,omitempty"`
}

const defaultHostname = "growrig"

// Manager owns the embedded node's lifecycle. It is safe for concurrent use.
type Manager struct {
	stateDir     string
	settingsPath string
	handler      http.Handler

	mu       sync.Mutex
	settings Settings

	// baseCtx bounds the node's lifetime to the process; request contexts must
	// never be used to start it, or the node would stop when the request ends.
	baseCtx context.Context

	// Live runtime, replaced wholesale on start/stop.
	srv    *tsnet.Server
	ln     net.Listener
	cancel context.CancelFunc

	// Watcher-maintained status (guarded by mu).
	state     State
	authURL   string
	dnsName   string
	keyExpiry *time.Time
	lastErr   string
}

// New loads persisted settings and returns a manager that will serve handler
// over the tailnet when enabled. stateDir holds the tsnet node state.
func New(stateDir, settingsPath string, handler http.Handler) (*Manager, error) {
	m := &Manager{
		stateDir:     stateDir,
		settingsPath: settingsPath,
		handler:      handler,
		state:        StateStopped,
		settings:     Settings{Hostname: defaultHostname},
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) load() error {
	raw, err := os.ReadFile(m.settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var s Settings
	if err := yaml.Unmarshal(raw, &s); err != nil {
		return err
	}
	if strings.TrimSpace(s.Hostname) == "" {
		s.Hostname = defaultHostname
	}
	m.settings = s
	return nil
}

func (m *Manager) persist() error {
	raw, err := yaml.Marshal(m.settings)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.settingsPath), 0o750); err != nil {
		return err
	}
	return os.WriteFile(m.settingsPath, raw, 0o600)
}

// Start binds the manager to the process lifetime and brings the node up if it
// was left enabled. Called once at boot. A start error is recorded in Status
// rather than returned, so a transient tailnet problem never blocks boot.
func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	m.baseCtx = ctx
	enabled := m.settings.Enabled
	m.mu.Unlock()
	if enabled {
		if err := m.start(); err != nil {
			m.setError(err)
		}
	}
}

// Enable turns remote access on with the given hostname/controlURL (blank
// hostname keeps the current one) and starts the node. The choice is persisted.
func (m *Manager) Enable(hostname, controlURL string) error {
	m.mu.Lock()
	if h := sanitizeHostname(hostname); h != "" {
		m.settings.Hostname = h
	}
	m.settings.ControlURL = strings.TrimSpace(controlURL)
	m.settings.Enabled = true
	if err := m.persist(); err != nil {
		m.mu.Unlock()
		return err
	}
	already := m.srv != nil
	m.mu.Unlock()
	if already {
		return nil
	}
	return m.start()
}

// Disable stops the node and remembers the choice. The persisted node identity
// is kept, so re-enabling does not require re-authenticating.
func (m *Manager) Disable() error {
	m.mu.Lock()
	m.settings.Enabled = false
	err := m.persist()
	m.mu.Unlock()
	m.stop()
	return err
}

// Close stops the node without changing the enabled setting (process shutdown).
func (m *Manager) Close() { m.stop() }

// start builds and brings up the tsnet node, serves the handler over a
// tailnet-only HTTPS listener, and launches the status watcher.
func (m *Manager) start() error {
	m.mu.Lock()
	if m.srv != nil {
		m.mu.Unlock()
		return nil
	}
	parent := m.baseCtx
	if parent == nil {
		parent = context.Background()
	}
	if err := os.MkdirAll(m.stateDir, 0o700); err != nil {
		m.mu.Unlock()
		return err
	}
	srv := &tsnet.Server{
		Hostname:   m.settings.Hostname,
		Dir:        m.stateDir,
		ControlURL: m.settings.ControlURL,
		// tsnet is chatty; keep its logs out of GrowRig's. Status comes from the
		// LocalClient watcher below, not from parsing these.
		Logf:     func(string, ...any) {},
		UserLogf: func(string, ...any) {},
	}
	m.srv = srv
	m.state = StateStarting
	m.authURL = ""
	m.dnsName = ""
	m.keyExpiry = nil
	m.lastErr = ""
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.mu.Unlock()

	if err := srv.Start(); err != nil {
		cancel()
		m.clear()
		return err
	}
	// Tailnet-only HTTPS. Traffic is already WireGuard-encrypted; TLS gives the
	// browser a valid cert and an https:// URL. Requires MagicDNS + HTTPS on the
	// tailnet — a cert failure surfaces via Status.Error.
	ln, err := srv.ListenTLS("tcp", ":443")
	if err != nil {
		cancel()
		m.clear()
		return err
	}
	m.mu.Lock()
	m.ln = ln
	m.mu.Unlock()

	go func() {
		// Serve returns when the listener is closed on stop.
		if err := http.Serve(ln, m.handler); err != nil && !errors.Is(err, net.ErrClosed) {
			m.setError(err)
		}
	}()
	go m.watch(ctx, srv)
	return nil
}

// watch polls the local node status and mirrors it into the manager, so the UI
// can show the auth URL, the running state, the remote URL and key expiry.
func (m *Manager) watch(ctx context.Context, srv *tsnet.Server) {
	lc, err := srv.LocalClient()
	if err != nil {
		m.setError(err)
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	loginRequested := false
	for {
		st, err := lc.Status(ctx)
		if err == nil && st != nil {
			m.mu.Lock()
			switch st.BackendState {
			case "Running":
				m.state = StateRunning
				m.authURL = ""
			case "NeedsLogin", "NeedsMachineAuth":
				m.state = StateNeedsLogin
				m.authURL = st.AuthURL
			case "Starting", "NoState":
				if m.state != StateNeedsLogin {
					m.state = StateStarting
				}
			case "Stopped":
				m.state = StateStopped
			}
			if st.Self != nil {
				m.dnsName = strings.TrimSuffix(st.Self.DNSName, ".")
				m.keyExpiry = st.Self.KeyExpiry
			}
			needLogin := st.BackendState == "NeedsLogin" && st.AuthURL == ""
			m.mu.Unlock()
			// Kick off interactive login once if the backend is waiting but hasn't
			// produced a URL yet.
			if needLogin && !loginRequested {
				loginRequested = true
				_ = lc.StartLoginInteractive(ctx)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Manager) stop() {
	m.mu.Lock()
	cancel, ln, srv := m.cancel, m.ln, m.srv
	m.cancel, m.ln, m.srv = nil, nil, nil
	m.state = StateStopped
	m.authURL, m.dnsName, m.keyExpiry, m.lastErr = "", "", nil, ""
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if ln != nil {
		_ = ln.Close()
	}
	if srv != nil {
		_ = srv.Close()
	}
}

// clear tears down a half-started node after a start failure, without touching
// the enabled setting or the recorded error.
func (m *Manager) clear() {
	m.mu.Lock()
	ln, srv := m.ln, m.srv
	m.ln, m.srv, m.cancel = nil, nil, nil
	m.mu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}
	if srv != nil {
		_ = srv.Close()
	}
}

func (m *Manager) setError(err error) {
	m.mu.Lock()
	m.state = StateError
	m.lastErr = err.Error()
	m.mu.Unlock()
}

// Status returns a snapshot for the API/UI.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := Status{
		Enabled:    m.settings.Enabled,
		State:      m.state,
		Hostname:   m.settings.Hostname,
		AuthURL:    m.authURL,
		DNSName:    m.dnsName,
		ControlURL: m.settings.ControlURL,
		KeyExpiry:  m.keyExpiry,
		Error:      m.lastErr,
	}
	if !m.settings.Enabled && m.srv == nil {
		s.State = StateStopped
	}
	if m.dnsName != "" {
		s.URL = "https://" + m.dnsName
	}
	if m.keyExpiry != nil && time.Now().After(*m.keyExpiry) {
		s.KeyExpired = true
	}
	return s
}

// sanitizeHostname keeps a DNS-label-safe hostname (letters, digits, hyphen).
func sanitizeHostname(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	var b strings.Builder
	for _, r := range h {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ' || r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
