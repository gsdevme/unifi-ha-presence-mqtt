package features

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cucumber/godog"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/mock"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/presence"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/publisher"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/scheduler"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/server"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

// world holds per-scenario state. It wires the real serve components (unifi
// client + provider, tracker, publisher, scheduler, server) against the in-process
// mock controller and the recording publisher — no broker, no real controller.
type world struct {
	tracked   []string
	threshold int
	clients   []mock.Client

	mock  *mock.Server
	srv   *httptest.Server
	rec   *publisher.RecordingPublisher
	pub   *publisher.Service
	stat  *server.Server
	sched *scheduler.Scheduler
	ctx   context.Context
}

func (w *world) reset() {
	w.tracked = nil
	w.threshold = 5
	w.clients = nil
	w.mock = nil
	w.srv = nil
	w.ctx = context.Background()
}

func (w *world) cleanup() {
	if w.srv != nil {
		w.srv.Close()
	}
}

// testProvider mirrors cmd.unifiProvider: a single Integration API client plus the
// resolved site UUID, no re-auth (scheduler.ClientProvider).
type testProvider struct {
	client unifi.Client
	siteID string
}

func (p *testProvider) ActiveClients(context.Context) ([]unifi.ClientResponse, error) {
	return p.client.GetActiveClients(p.siteID)
}

// --- Given ---

func (w *world) trackedDevicesAre(list string) error {
	for _, d := range strings.Split(list, ",") {
		if d = strings.TrimSpace(d); d != "" {
			w.tracked = append(w.tracked, d)
		}
	}
	return nil
}

func (w *world) failureThresholdIs(n int) error {
	w.threshold = n
	return nil
}

func (w *world) deviceActive(host string) error {
	w.clients = append(w.clients, mock.Client{Hostname: host, IPAddress: "10.0.0.1", MACAddress: "aa:bb:cc:dd:ee:ff"})
	return nil
}

// --- When ---

func (w *world) serviceStartsUp() error {
	w.mock = mock.New(mock.Options{Clients: w.clients})
	w.srv = httptest.NewServer(w.mock.Handler())

	// Build the client and resolve the site UUID, mirroring cmd/serve.go startup.
	client := unifi.NewHTTPClient(w.srv.URL, "test-key")
	sites, err := client.ListSites()
	if err != nil {
		return err
	}
	provider := &testProvider{client: client, siteID: sites[0].ID}
	tracker := presence.NewTracker(w.tracked, w.threshold)
	w.rec = publisher.NewRecordingPublisher()
	w.pub = publisher.New(w.rec, "homeassistant", "unifi")
	w.stat = server.New(server.Config{ReadyFailureThreshold: 3})

	initial := tracker.Devices()
	if err := w.pub.PublishDiscovery(w.ctx, initial); err != nil {
		return err
	}
	if err := w.pub.PublishAvailability(w.ctx, true); err != nil {
		return err
	}
	if err := w.pub.PublishStates(w.ctx, initial); err != nil {
		return err
	}

	w.sched = scheduler.New(provider, tracker, w.pub, w.stat, scheduler.Config{MaxRetries: 0})
	return nil
}

func (w *world) pollRuns() error {
	w.sched.PollNow(w.ctx)
	return nil
}

func (w *world) deviceLeaves(host string) error {
	kept := w.clients[:0:0]
	for _, c := range w.clients {
		if c.Hostname != host {
			kept = append(kept, c)
		}
	}
	w.clients = kept
	w.mock.SetClients(w.clients)
	return nil
}

func (w *world) serviceShutsDown() error {
	return w.pub.PublishAvailability(w.ctx, false)
}

// --- Then ---

func (w *world) discoveryPublishedRetained(kind, host string) error {
	component := map[string]string{"presence": "device_tracker", "connectivity": "binary_sensor"}[kind]
	if component == "" {
		return fmt.Errorf("unknown discovery kind %q", kind)
	}
	topic := fmt.Sprintf("homeassistant/%s/unifi_%s_%s/config", component, strings.ToLower(host), kind)
	rec, ok := w.rec.Get(topic)
	if !ok {
		return fmt.Errorf("discovery topic %s not published", topic)
	}
	if !rec.Retain {
		return fmt.Errorf("discovery topic %s not retained", topic)
	}
	return nil
}

func (w *world) availabilityPublished(payload string) error {
	rec, ok := w.rec.Get("unifi/availability")
	if !ok {
		return fmt.Errorf("availability not published")
	}
	if string(rec.Payload) != payload {
		return fmt.Errorf("availability = %q, want %q", rec.Payload, payload)
	}
	if !rec.Retain {
		return fmt.Errorf("availability not retained")
	}
	return nil
}

func (w *world) stateReportsPresent(host string, want string) error {
	rec, ok := w.rec.Get("unifi/" + strings.ToLower(host) + "/state")
	if !ok {
		return fmt.Errorf("state for %s not published", host)
	}
	var st map[string]any
	if err := json.Unmarshal(rec.Payload, &st); err != nil {
		return err
	}
	if got := fmt.Sprintf("%v", st["present"]); got != want {
		return fmt.Errorf("%s present = %v, want %s", host, st["present"], want)
	}
	return nil
}

func (w *world) readinessReady() error {
	if !w.stat.Ready() {
		return fmt.Errorf("readiness = not ready, want ready")
	}
	return nil
}

func TestFeatures(t *testing.T) {
	w := &world{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
				w.reset()
				return c, nil
			})
			ctx.After(func(c context.Context, _ *godog.Scenario, err error) (context.Context, error) {
				w.cleanup()
				return c, err
			})

			ctx.Step(`^the tracked devices are "([^"]*)"$`, w.trackedDevicesAre)
			ctx.Step(`^the failure threshold is (\d+)$`, w.failureThresholdIs)
			ctx.Step(`^"([^"]*)" is active on the controller$`, w.deviceActive)

			ctx.Step(`^the service starts up$`, w.serviceStartsUp)
			ctx.Step(`^a poll runs$`, w.pollRuns)
			ctx.Step(`^"([^"]*)" leaves the controller$`, w.deviceLeaves)
			ctx.Step(`^the service shuts down$`, w.serviceShutsDown)

			ctx.Step(`^the (presence|connectivity) discovery for "([^"]*)" is published retained$`, w.discoveryPublishedRetained)
			ctx.Step(`^availability "([^"]*)" is published retained$`, w.availabilityPublished)
			ctx.Step(`^the state for "([^"]*)" reports present (true|false)$`, w.stateReportsPresent)
			ctx.Step(`^the readiness endpoint reports ready$`, w.readinessReady)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"."},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("godog acceptance suite failed")
	}
}
