// Package mock implements an in-process stand-in for the official UniFi Network
// Integration API. It serves the two endpoints the client uses —
// GET proxy/network/integration/v1/sites and
// GET proxy/network/integration/v1/sites/<site>/clients — authenticated by a
// non-empty X-API-KEY header, so the full pipeline can run without a real console.
// It is shared by the godog acceptance suite and the standalone `mock` subcommand.
// See docs/specs/07-deployment.md (testing) and internal/unifi.
package mock

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SiteID is the fixed UUID the mock reports for its single "default" site.
const SiteID = "00000000-0000-4000-8000-000000000000"

// Client is a canned connected client returned by the clients endpoint.
type Client struct {
	Hostname   string
	IPAddress  string
	MACAddress string
}

// Options configure the canned console state.
type Options struct {
	Clients []Client
	Logger  *slog.Logger
}

// Defaults returns Options listing two connected clients (phone1, phone2) so a
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

// Server is the mock console. Create with New and mount Handler().
type Server struct {
	mu   sync.Mutex
	opts Options

	// Observability for tests.
	SitesHits   int
	ClientsHits int
}

// New builds a mock console.
func New(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Server{opts: opts}
}

// SetClients replaces the connected-client list (used between scenarios / polls).
func (s *Server) SetClients(clients []Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.Clients = clients
}

// Counts returns a snapshot of the request counters.
func (s *Server) Counts() (sites, clients int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.SitesHits, s.ClientsHits
}

// Handler returns the HTTP handler for the mock console. See routes.go.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, s)
	return mux
}

func (s *Server) handleSites(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	s.mu.Lock()
	s.SitesHits++
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, page([]map[string]any{
		{"id": SiteID, "name": "default"},
	}))
}

func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	s.mu.Lock()
	s.ClientsHits++
	clients := append([]Client(nil), s.opts.Clients...)
	s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	data := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		data = append(data, map[string]any{
			"id":          c.MACAddress,
			"name":        c.Hostname,
			"hostname":    c.Hostname,
			"ipAddress":   c.IPAddress,
			"macAddress":  c.MACAddress,
			"type":        "WIRELESS",
			"connectedAt": now,
		})
	}
	writeJSON(w, http.StatusOK, page(data))
}

// authorized enforces a non-empty X-API-KEY header, returning 401 otherwise. The
// mock does not check the key's value — any non-empty key is accepted.
func authorized(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-API-KEY") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

// page wraps data in the Integration API pagination envelope as a single full page.
func page(data []map[string]any) map[string]any {
	return map[string]any{
		"offset":     0,
		"limit":      len(data),
		"count":      len(data),
		"totalCount": len(data),
		"data":       data,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
