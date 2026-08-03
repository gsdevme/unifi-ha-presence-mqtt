// Package server exposes the daemon's HTTP surface: a human-friendly status page
// on / plus liveness (/healthz) and readiness (/readyz) probes. Liveness is always
// OK while the process runs; readiness flips ready after the first successful
// publish and not-ready after a configurable number of consecutive poll failures.
// See docs/specs/06-lifecycle-health.md.
package server

import (
	"net/http"
	"sync"
	"time"
)

// Config configures the status server. It is deliberately decoupled from
// internal/config so serve.go maps domain values in and there is no import
// coupling here.
type Config struct {
	ReadyFailureThreshold int
	PollInterval          time.Duration
}

// DeviceView is a tracked device's non-sensitive status for the / page. IP and
// MAC are intentionally omitted as mildly sensitive.
type DeviceView struct {
	Hostname string
	Present  bool
}

// Server tracks readiness based on poll outcomes and holds a concurrency-safe
// snapshot of tracked-device presence for the / page.
type Server struct {
	mu                  sync.Mutex
	ready               bool
	consecutiveFailures int
	threshold           int

	// devices is the latest tracked-device snapshot (see SetDevices). Until set,
	// the / page renders "initialising".
	devicesSet bool
	devices    []DeviceView

	cfg       Config
	startedAt time.Time
}

// New returns a Server that flips not-ready after cfg.ReadyFailureThreshold
// consecutive failures. A threshold below 1 is treated as 1.
func New(cfg Config) *Server {
	if cfg.ReadyFailureThreshold < 1 {
		cfg.ReadyFailureThreshold = 1
	}
	return &Server{threshold: cfg.ReadyFailureThreshold, cfg: cfg, startedAt: time.Now()}
}

// MarkSuccess records a successful publish: ready becomes true and the failure
// counter resets.
func (s *Server) MarkSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = true
	s.consecutiveFailures = 0
}

// MarkFailure records a poll failure; readiness flips off once the threshold of
// consecutive failures is reached.
func (s *Server) MarkFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consecutiveFailures++
	if s.consecutiveFailures >= s.threshold {
		s.ready = false
	}
}

// Ready reports the current readiness state.
func (s *Server) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ready
}

// SetDevices records the latest tracked-device snapshot for the status page. It is
// called after each successful publish; until then the / page renders
// "initialising".
func (s *Server) SetDevices(devices []DeviceView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devicesSet = true
	s.devices = append([]DeviceView(nil), devices...)
}

// Handler returns the mux serving /, /healthz and /readyz. See routes.go for the
// route table.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, s)
	return mux
}

func (s *Server) handleLivez(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if s.Ready() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("not ready"))
}
