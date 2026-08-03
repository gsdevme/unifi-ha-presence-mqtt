package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/config"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/homeassistant"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/mqtt"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/presence"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/publisher"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/scheduler"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/server"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the poll -> MQTT presence service",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runServe(cmd.Context())
	},
}

func runServe(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := newLogger(cfg.LogLevel, cfg.LogFormat)
	logger.Info("starting", "config", cfg.String())
	if cfg.Mode == "mock" {
		logger.Warn("running in MOCK mode; not using a real UniFi controller")
	}

	// Status/health server listens immediately so probes work during init.
	status := server.New(server.Config{
		ReadyFailureThreshold: cfg.ReadyFailureThreshold,
		PollInterval:          cfg.PollInterval,
	})
	healthSrv := &http.Server{Addr: cfg.HealthAddr, Handler: status.Handler()}
	healthErr := make(chan error, 1)
	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			healthErr <- err
			return
		}
		healthErr <- nil
	}()

	// UniFi client: a single Integration API client authenticated by the static
	// X-API-KEY. Listing sites up front validates the key early (a clear onboarding
	// signal) and resolves the UNIFI_SITE name to its UUID.
	client := unifi.NewHTTPClient(cfg.UnifiHost, cfg.UnifiAPIKey)
	siteID, err := resolveSite(client, cfg.UnifiSite)
	if err != nil {
		return fmt.Errorf("unifi: %w (check UNIFI_API_KEY, UNIFI_HOST, and that the "+
			"Network Integration API is reachable at %s)", err, cfg.UnifiHost)
	}
	logger.Info("resolved unifi site", "name", cfg.UnifiSite, "id", siteID)
	provider := &unifiProvider{client: client, siteID: siteID}

	tracker := presence.NewTracker(cfg.TrackDevices, cfg.FailureThreshold)

	// MQTT connection (with LWT on the service availability topic). The connection
	// manager gets its own lifetime (mqttCtx), independent of the signal context, so
	// on SIGTERM we can still publish a retained "offline" before disconnecting — a
	// cancelled ctx would tear the manager down first and the clean Disconnect would
	// then suppress the LWT, leaving the service stuck "online".
	mqttCtx, mqttCancel := context.WithCancel(context.Background())
	defer mqttCancel()
	availabilityTopic := homeassistant.Config{TopicPrefix: cfg.MQTTTopicPrefix}.AvailabilityTopic()
	mc, err := mqtt.Connect(mqttCtx, mqtt.Options{
		BrokerURL:         cfg.MQTTBrokerURL,
		Username:          cfg.MQTTUsername,
		Password:          cfg.MQTTPassword,
		ClientID:          cfg.MQTTClientID,
		AvailabilityTopic: availabilityTopic,
		Logger:            logger,
	})
	if err != nil {
		return fmt.Errorf("mqtt: %w", err)
	}

	pub := publisher.New(mc, cfg.HADiscoveryPrefix, cfg.MQTTTopicPrefix)

	// Republish availability + discovery + current state on every (re)connection.
	mc.SetOnConnectionUp(func(ctx context.Context) {
		devices := tracker.Devices()
		if err := pub.PublishAvailability(ctx, true); err != nil {
			logger.Warn("republish availability failed", "err", err)
		}
		if err := pub.PublishDiscovery(ctx, devices); err != nil {
			logger.Warn("republish discovery failed", "err", err)
		}
		if err := pub.PublishStates(ctx, devices); err != nil {
			logger.Warn("republish state failed", "err", err)
		}
	})

	// Initial publish (in case the first connection-up fired before the callback
	// was registered): discovery + online + seeded absent state so entities exist.
	initial := tracker.Devices()
	if err := pub.PublishDiscovery(ctx, initial); err != nil {
		return fmt.Errorf("publish discovery: %w", err)
	}
	if err := pub.PublishAvailability(ctx, true); err != nil {
		return fmt.Errorf("publish availability: %w", err)
	}
	if err := pub.PublishStates(ctx, initial); err != nil {
		return fmt.Errorf("publish initial state: %w", err)
	}

	// Scheduler. recordingPublisher decorates pub so the status page records the
	// device snapshot after each successful publish.
	sched := scheduler.New(provider, tracker, recordingPublisher{pub: pub, status: status}, status, scheduler.Config{
		PollInterval: cfg.PollInterval,
		MaxRetries:   cfg.PollMaxRetries,
		Logger:       logger,
	})

	schedDone := make(chan struct{})
	go func() { sched.Run(ctx); close(schedDone) }()

	logger.Info("service running")
	select {
	case err := <-healthErr:
		return fmt.Errorf("health server: %w", err)
	case <-ctx.Done():
	}
	logger.Info("shutting down")

	// Graceful shutdown: explicit offline, clean disconnect, stop everything.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pub.PublishAvailability(shutdownCtx, false); err != nil {
		logger.Warn("publish offline failed", "err", err)
	}
	if err := mc.Disconnect(shutdownCtx); err != nil {
		logger.Warn("mqtt disconnect failed", "err", err)
	}
	<-schedDone
	return shutdownServer(healthSrv)
}

// recordingPublisher publishes state, then records the device snapshot for the
// status page on success.
type recordingPublisher struct {
	pub    *publisher.Service
	status *server.Server
}

func (r recordingPublisher) PublishStates(ctx context.Context, devices []presence.Device) error {
	if err := r.pub.PublishStates(ctx, devices); err != nil {
		return err // keep the last good snapshot, do not record
	}
	r.status.SetDevices(deviceViews(devices))
	return nil
}

// deviceViews maps presence devices into the non-sensitive server view (IP/MAC
// omitted).
func deviceViews(devices []presence.Device) []server.DeviceView {
	views := make([]server.DeviceView, 0, len(devices))
	for _, d := range devices {
		views = append(views, server.DeviceView{Hostname: d.Hostname, Present: d.Present})
	}
	return views
}

// unifiProvider holds the Integration API client and the resolved site UUID. The
// API key is static so the client is built once; there is no re-auth. It satisfies
// scheduler.ClientProvider.
type unifiProvider struct {
	client unifi.Client
	siteID string
}

func (p *unifiProvider) ActiveClients(_ context.Context) ([]unifi.ClientResponse, error) {
	return p.client.GetActiveClients(p.siteID)
}

// resolveSite lists the console's sites (validating the API key early) and returns
// the UUID whose name matches wantName (case-insensitive). When only one site
// exists it is used regardless of name.
func resolveSite(client unifi.Client, wantName string) (string, error) {
	sites, err := client.ListSites()
	if err != nil {
		return "", err
	}
	switch len(sites) {
	case 0:
		return "", errors.New("no sites returned")
	case 1:
		return sites[0].ID, nil
	}
	for _, s := range sites {
		if strings.EqualFold(s.Name, wantName) {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("no site named %q among %d sites", wantName, len(sites))
}

func shutdownServer(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
