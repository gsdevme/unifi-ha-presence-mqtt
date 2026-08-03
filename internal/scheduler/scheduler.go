// Package scheduler runs the presence poll loop: fetch connected clients, apply the
// presence debounce, publish per-device state and report the outcome to health. It
// retries transient errors with backoff. See docs/specs/04-polling-scheduling.md.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/presence"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

// ClientProvider supplies the console's currently connected clients. The
// Integration API key is static, so there is nothing to re-authenticate: a
// persistent auth failure surfaces as a poll failure (flipping /readyz not-ready).
type ClientProvider interface {
	ActiveClients(ctx context.Context) ([]unifi.ClientResponse, error)
}

// Presence applies a snapshot of active clients to the tracked devices, returning
// the current per-device state.
type Presence interface {
	Apply(active []unifi.ClientResponse) []presence.Device
}

// StatePublisher publishes the current per-device state.
type StatePublisher interface {
	PublishStates(ctx context.Context, devices []presence.Device) error
}

// HealthReporter records poll outcomes for readiness.
type HealthReporter interface {
	MarkSuccess()
	MarkFailure()
}

// Config configures the scheduler.
type Config struct {
	PollInterval time.Duration
	MaxRetries   int

	Logger *slog.Logger
	// Now and After are injectable for deterministic tests (testing/synctest).
	Now   func() time.Time
	After func(time.Duration) <-chan time.Time
}

// Scheduler owns the poll loop.
type Scheduler struct {
	provider  ClientProvider
	presence  Presence
	publisher StatePublisher
	health    HealthReporter
	cfg       Config
	logger    *slog.Logger
	after     func(time.Duration) <-chan time.Time

	// apiMu serialises polls so ticks never overlap.
	apiMu sync.Mutex
}

// New builds a Scheduler.
func New(p ClientProvider, pres Presence, pub StatePublisher, h HealthReporter, cfg Config) *Scheduler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.After == nil {
		cfg.After = time.After
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	return &Scheduler{
		provider: p, presence: pres, publisher: pub, health: h, cfg: cfg,
		logger: cfg.Logger, after: cfg.After,
	}
}

// Run polls immediately, then every PollInterval, blocking until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	s.poll(ctx) // immediate first poll
	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

// PollNow runs a single poll cycle. Exposed for the acceptance suite; the timed
// loop uses the same underlying logic.
func (s *Scheduler) PollNow(ctx context.Context) { s.poll(ctx) }

// poll performs one fetch + apply + publish, updating health.
func (s *Scheduler) poll(ctx context.Context) {
	s.apiMu.Lock()
	defer s.apiMu.Unlock()

	active, err := s.fetchWithRetry(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		s.logger.WarnContext(ctx, "poll failed", "err", err)
		s.health.MarkFailure()
		return
	}
	devices := s.presence.Apply(active)
	if err := s.publisher.PublishStates(ctx, devices); err != nil {
		s.logger.WarnContext(ctx, "publish failed", "err", err)
		s.health.MarkFailure()
		return
	}
	s.logger.InfoContext(ctx, "published presence", "devices", len(devices))
	s.health.MarkSuccess()
}

// fetchWithRetry fetches connected clients, retrying transient errors with
// exponential backoff up to MaxRetries.
func (s *Scheduler) fetchWithRetry(ctx context.Context) ([]unifi.ClientResponse, error) {
	var lastErr error
	backoff := time.Second
	for attempt := 0; attempt <= s.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-s.after(backoff):
			}
			backoff *= 2
		}
		active, err := s.provider.ActiveClients(ctx)
		if err == nil {
			return active, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		lastErr = err
		s.logger.DebugContext(ctx, "poll attempt failed", "attempt", attempt, "err", err)
	}
	return nil, lastErr
}
