// Package mock implements an in-process stand-in for a UniFi Network controller
// (UniFi OS). It serves the two endpoints the client uses — POST api/auth/login
// (sets a TOKEN cookie) and GET proxy/network/api/s/<site>/stat/sta (a configurable
// active-client list) — so the full pipeline can run without a real controller. It
// is shared by the godog acceptance suite and the standalone `mock` subcommand.
// See docs/specs/07-deployment.md (testing) and internal/unifi.
package mock

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

// Client is a canned active client returned by the stat/sta endpoint.
type Client struct {
	Hostname   string
	IPAddress  string
	MACAddress string
}

// Options configure the canned controller state.
type Options struct {
	Clients []Client
	Logger  *slog.Logger
}

// Defaults returns Options listing two active clients (phone1, phone2) so a
// MODE=mock serve with TRACK_DEVICES=phone1,phone2 shows them home.
func Defaults() Options {
	return Options{
		Clients: []Client{
			{Hostname: "phone1", IPAddress: "192.168.1.20", MACAddress: "aa:bb:cc:dd:ee:01"},
			{Hostname: "phone2", IPAddress: "192.168.1.21", MACAddress: "aa:bb:cc:dd:ee:02"},
		},
		Logger: slog.Default(),
	}
}

// Server is the mock controller. Create with New and mount Handler().
type Server struct {
	mu    sync.Mutex
	opts  Options
	token string // the token issued by the most recent login

	// Observability for tests.
	LoginHits int
	StaHits   int
}

// New builds a mock controller.
func New(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Server{opts: opts, token: randToken()}
}

// SetClients replaces the active-client list (used between scenarios / polls).
func (s *Server) SetClients(clients []Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.Clients = clients
}

// ExpireSession rotates the valid token so the client's cached token no longer
// matches: the next stat/sta returns 401, forcing a re-authentication.
func (s *Server) ExpireSession() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = randToken()
}

// Counts returns a snapshot of the request counters.
func (s *Server) Counts() (login, sta int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LoginHits, s.StaHits
}

// Handler returns the HTTP handler for the mock controller. See routes.go.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, s)
	return mux
}

func (s *Server) handleLogin(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.LoginHits++
	token := s.token
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: unifi.AuthCookieName, Value: token})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleActiveClients(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.StaHits++
	valid := s.token
	clients := append([]Client(nil), s.opts.Clients...)
	s.mu.Unlock()

	// Reject a stale/expired session so the client re-authenticates.
	cookie, err := r.Cookie(unifi.AuthCookieName)
	if err != nil || cookie.Value != valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	data := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		data = append(data, map[string]any{
			"site_id":   "default",
			"mac":       c.MACAddress,
			"ip":        c.IPAddress,
			"hostname":  c.Hostname,
			"last_seen": time.Now().Unix(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"meta": map[string]any{"rc": "ok"},
		"data": data,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func randToken() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}
