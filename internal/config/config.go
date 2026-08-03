// Package config binds and validates the service configuration from environment
// variables. See docs/specs/05-config.md.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// MinPollInterval is the floor for POLL_INTERVAL, protecting the controller from
// excessive polling.
const MinPollInterval = 2 * time.Second

// defaultMockURL is the mock target used when MODE=mock and MOCK_URL is unset.
// Must match the mock command's default --addr :8090 (see internal/cmd/mock.go).
const defaultMockURL = "http://localhost:8090"

// Config is the fully-parsed, validated service configuration.
type Config struct {
	// UniFi
	Mode         string // "live" | "mock"
	UnifiHost    string // console base URL (resolved to MOCK_URL in mock mode)
	UnifiAPIKey  string // Integration API key sent as the X-API-KEY header
	UnifiSite    string // human site name, resolved to a site UUID at startup
	TrackDevices []string

	// Polling & scheduling
	PollInterval          time.Duration
	FailureThreshold      int
	PollMaxRetries        int
	ReadyFailureThreshold int

	// MQTT / HA
	MQTTBrokerURL     string
	MQTTUsername      string
	MQTTPassword      string
	MQTTClientID      string
	MQTTTopicPrefix   string
	HADiscoveryPrefix string

	// Health & logging
	HealthAddr string
	LogLevel   string
	LogFormat  string
}

// Load reads configuration from the environment, applies defaults and validates.
func Load() (*Config, error) {
	c := &Config{
		UnifiHost:         os.Getenv("UNIFI_HOST"),
		UnifiAPIKey:       os.Getenv("UNIFI_API_KEY"),
		UnifiSite:         getEnv("UNIFI_SITE", "default"),
		MQTTBrokerURL:     os.Getenv("MQTT_BROKER_URL"),
		MQTTUsername:      os.Getenv("MQTT_USERNAME"),
		MQTTPassword:      os.Getenv("MQTT_PASSWORD"),
		MQTTClientID:      getEnv("MQTT_CLIENT_ID", "unifi-ha-presence-mqtt"),
		MQTTTopicPrefix:   getEnv("MQTT_TOPIC_PREFIX", "unifi"),
		HADiscoveryPrefix: getEnv("HA_DISCOVERY_PREFIX", "homeassistant"),
		HealthAddr:        getEnv("HEALTH_ADDR", ":8080"),
		LogLevel:          strings.ToLower(getEnv("LOG_LEVEL", "info")),
		LogFormat:         strings.ToLower(getEnv("LOG_FORMAT", "json")),
	}

	var errs []error

	// Resolve the target: live hits the real console over the Integration API, mock
	// points UNIFI_HOST at the local mock server and supplies a dummy API key.
	c.Mode = strings.ToLower(getEnv("MODE", "live"))
	switch c.Mode {
	case "live":
		if c.UnifiHost == "" {
			errs = append(errs, errors.New("UNIFI_HOST is required"))
		}
		if c.UnifiAPIKey == "" {
			errs = append(errs, errors.New("UNIFI_API_KEY is required"))
		}
	case "mock":
		c.UnifiHost = getEnv("MOCK_URL", defaultMockURL)
		if c.UnifiAPIKey == "" {
			c.UnifiAPIKey = "mock"
		}
	default:
		errs = append(errs, fmt.Errorf("MODE must be live or mock, got %q", c.Mode))
	}

	c.TrackDevices = parseList(os.Getenv("TRACK_DEVICES"))
	if len(c.TrackDevices) == 0 {
		errs = append(errs, errors.New("TRACK_DEVICES is required (comma-separated hostnames)"))
	}

	if c.MQTTBrokerURL == "" {
		errs = append(errs, errors.New("MQTT_BROKER_URL is required"))
	} else if _, err := url.Parse(c.MQTTBrokerURL); err != nil {
		errs = append(errs, fmt.Errorf("MQTT_BROKER_URL is invalid: %w", err))
	}

	interval, err := parseDuration("POLL_INTERVAL", 10*time.Second)
	if err != nil {
		errs = append(errs, err)
	} else if interval < MinPollInterval {
		errs = append(errs, fmt.Errorf("POLL_INTERVAL %s is below the %s floor", interval, MinPollInterval))
	}
	c.PollInterval = interval

	c.FailureThreshold = getInt("FAILURE_THRESHOLD", 5)
	if c.FailureThreshold < 1 {
		errs = append(errs, errors.New("FAILURE_THRESHOLD must be >= 1"))
	}
	c.PollMaxRetries = getInt("POLL_MAX_RETRIES", 3)
	if c.PollMaxRetries < 0 {
		errs = append(errs, errors.New("POLL_MAX_RETRIES must be >= 0"))
	}
	c.ReadyFailureThreshold = getInt("READY_FAILURE_THRESHOLD", 3)
	if c.ReadyFailureThreshold < 1 {
		errs = append(errs, errors.New("READY_FAILURE_THRESHOLD must be >= 1"))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return c, nil
}

// String renders the config with secrets redacted (safe to log). The UniFi API
// key and MQTT password are redacted by omission — they are never printed.
func (c *Config) String() string {
	host := c.UnifiHost
	if c.Mode == "mock" {
		host = fmt.Sprintf("mock(%s)", c.UnifiHost)
	}
	return fmt.Sprintf("Config{mode=%s host=%s site=%s track=%v poll=%s failThreshold=%d "+
		"maxRetries=%d readyThreshold=%d broker=%s clientID=%s topicPrefix=%s haPrefix=%s health=%s log=%s/%s}",
		c.Mode, host, c.UnifiSite, c.TrackDevices, c.PollInterval, c.FailureThreshold,
		c.PollMaxRetries, c.ReadyFailureThreshold, c.MQTTBrokerURL, c.MQTTClientID, c.MQTTTopicPrefix,
		c.HADiscoveryPrefix, c.HealthAddr, c.LogLevel, c.LogFormat)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func parseDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid duration: %w", key, err)
	}
	return d, nil
}

// parseList splits a comma-separated list, trimming spaces and dropping empties.
func parseList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
