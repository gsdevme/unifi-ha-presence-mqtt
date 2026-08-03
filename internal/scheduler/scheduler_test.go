package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/presence"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

type fakeProvider struct {
	mu      sync.Mutex
	calls   int
	failN   int // fail the first N ActiveClients calls with a transient error
	clients []unifi.ClientResponse
}

func (f *fakeProvider) ActiveClients(context.Context) ([]unifi.ClientResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.failN {
		return nil, errors.New("transient")
	}
	return f.clients, nil
}

type fakePublisher struct {
	mu    sync.Mutex
	count int
	last  []presence.Device
}

func (p *fakePublisher) PublishStates(_ context.Context, devices []presence.Device) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.count++
	p.last = devices
	return nil
}

func (p *fakePublisher) publishCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count
}

type fakeHealth struct {
	mu               sync.Mutex
	success, failure int
}

func (h *fakeHealth) MarkSuccess() { h.mu.Lock(); h.success++; h.mu.Unlock() }
func (h *fakeHealth) MarkFailure() { h.mu.Lock(); h.failure++; h.mu.Unlock() }

func (h *fakeHealth) counts() (int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.success, h.failure
}

func baseConfig() Config {
	return Config{
		PollInterval: time.Hour, // effectively one tick in these tests
		MaxRetries:   3,
		Now:          time.Now,
		After:        time.After,
	}
}

func TestImmediatePollPublishes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeProvider{clients: []unifi.ClientResponse{{Hostname: "phone1"}}}
		p := &fakePublisher{}
		h := &fakeHealth{}
		tr := presence.NewTracker([]string{"phone1"}, 5)
		s := New(f, tr, p, h, baseConfig())

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { s.Run(ctx); close(done) }()

		synctest.Wait()
		if p.publishCount() != 1 {
			t.Fatalf("publish count = %d, want 1 (immediate poll)", p.publishCount())
		}
		if success, _ := h.counts(); success != 1 {
			t.Fatalf("health success = %d, want 1", success)
		}
		if len(p.last) != 1 || !p.last[0].Present {
			t.Fatalf("published devices = %+v, want phone1 present", p.last)
		}
		cancel()
		<-done
	})
}

func TestTickerCadence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeProvider{clients: []unifi.ClientResponse{{Hostname: "phone1"}}}
		p := &fakePublisher{}
		h := &fakeHealth{}
		tr := presence.NewTracker([]string{"phone1"}, 5)
		s := New(f, tr, p, h, baseConfig())

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { s.Run(ctx); close(done) }()

		synctest.Wait() // immediate poll
		time.Sleep(time.Hour + time.Second)
		synctest.Wait() // one ticker tick
		if p.publishCount() != 2 {
			t.Fatalf("publish count = %d, want 2 (immediate + 1 tick)", p.publishCount())
		}
		cancel()
		<-done
	})
}

func TestTransientRetrySucceeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeProvider{clients: []unifi.ClientResponse{{Hostname: "phone1"}}, failN: 2}
		p := &fakePublisher{}
		h := &fakeHealth{}
		tr := presence.NewTracker([]string{"phone1"}, 5)
		s := New(f, tr, p, h, baseConfig())

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { s.Run(ctx); close(done) }()

		// The immediate poll fails twice (1s then 2s backoff) before succeeding.
		time.Sleep(10 * time.Second)
		synctest.Wait()
		if p.publishCount() != 1 {
			t.Fatalf("publish count = %d, want 1 after retries", p.publishCount())
		}
		if _, failure := h.counts(); failure != 0 {
			t.Fatalf("health failure = %d, want 0 (retries absorbed the transient errors)", failure)
		}
		cancel()
		<-done
	})
}
