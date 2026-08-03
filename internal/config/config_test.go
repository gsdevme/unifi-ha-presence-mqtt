package config

import (
	"slices"
	"strings"
	"testing"
)

// setEnv sets the minimum required vars plus any overrides for a test.
func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	base := map[string]string{
		"UNIFI_HOST":      "https://192.168.1.1",
		"UNIFI_USERNAME":  "admin",
		"UNIFI_PASSWORD":  "secret",
		"TRACK_DEVICES":   "phone1,phone2",
		"MQTT_BROKER_URL": "mqtt://localhost:1883",
	}
	for k, v := range base {
		if _, ok := kv[k]; !ok {
			t.Setenv(k, v)
		}
	}
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)
	c, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.PollInterval.Seconds() != 10 {
		t.Errorf("poll interval = %s, want 10s", c.PollInterval)
	}
	if c.FailureThreshold != 5 {
		t.Errorf("failure threshold = %d, want 5", c.FailureThreshold)
	}
	if c.UnifiSite != "default" {
		t.Errorf("site = %q, want default", c.UnifiSite)
	}
	if c.MQTTTopicPrefix != "unifi" {
		t.Errorf("topic prefix = %q, want unifi", c.MQTTTopicPrefix)
	}
	if c.Mode != "live" {
		t.Errorf("mode = %q, want live", c.Mode)
	}
	if !slices.Equal(c.TrackDevices, []string{"phone1", "phone2"}) {
		t.Errorf("track devices = %v", c.TrackDevices)
	}
}

func TestTrackDevicesTrimmed(t *testing.T) {
	setEnv(t, map[string]string{"TRACK_DEVICES": " phone1 , ,phone2,"})
	c, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !slices.Equal(c.TrackDevices, []string{"phone1", "phone2"}) {
		t.Errorf("track devices = %v, want [phone1 phone2]", c.TrackDevices)
	}
}

func TestMissingTrackDevices(t *testing.T) {
	setEnv(t, map[string]string{"TRACK_DEVICES": ""})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TRACK_DEVICES") {
		t.Fatalf("expected TRACK_DEVICES error, got %v", err)
	}
}

func TestMissingRequiredLive(t *testing.T) {
	t.Setenv("UNIFI_HOST", "")
	t.Setenv("UNIFI_USERNAME", "")
	t.Setenv("UNIFI_PASSWORD", "")
	t.Setenv("TRACK_DEVICES", "")
	t.Setenv("MQTT_BROKER_URL", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing required vars")
	}
	for _, want := range []string{"UNIFI_HOST", "UNIFI_USERNAME", "UNIFI_PASSWORD", "TRACK_DEVICES", "MQTT_BROKER_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %s: %v", want, err)
		}
	}
}

func TestPollFloor(t *testing.T) {
	setEnv(t, map[string]string{"POLL_INTERVAL": "500ms"})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "floor") {
		t.Fatalf("expected poll floor error, got %v", err)
	}
}

func TestBadFailureThreshold(t *testing.T) {
	setEnv(t, map[string]string{"FAILURE_THRESHOLD": "0"})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "FAILURE_THRESHOLD") {
		t.Fatalf("expected FAILURE_THRESHOLD error, got %v", err)
	}
}

func TestModeMockResolvesDefaultURL(t *testing.T) {
	// Empty creds must be accepted in mock mode.
	setEnv(t, map[string]string{"MODE": "mock", "UNIFI_HOST": "", "UNIFI_USERNAME": "", "UNIFI_PASSWORD": ""})
	c, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.UnifiHost != defaultMockURL {
		t.Errorf("mock host = %q, want %q", c.UnifiHost, defaultMockURL)
	}
	if c.UnifiUsername == "" || c.UnifiPassword == "" {
		t.Errorf("mock should supply dummy creds, got user=%q pass=%q", c.UnifiUsername, c.UnifiPassword)
	}
}

func TestModeMockCustomURL(t *testing.T) {
	const custom = "http://localhost:9000"
	setEnv(t, map[string]string{"MODE": "mock", "MOCK_URL": custom})
	c, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.UnifiHost != custom {
		t.Errorf("mock host = %q, want %q", c.UnifiHost, custom)
	}
}

func TestModeBogus(t *testing.T) {
	setEnv(t, map[string]string{"MODE": "bogus"})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "MODE") {
		t.Fatalf("expected MODE validation error, got %v", err)
	}
}

func TestRedaction(t *testing.T) {
	setEnv(t, map[string]string{
		"UNIFI_PASSWORD": "unifi-secret-pw",
		"MQTT_PASSWORD":  "hunter2",
	})
	c, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	s := c.String()
	for _, leak := range []string{"unifi-secret-pw", "hunter2"} {
		if strings.Contains(s, leak) {
			t.Errorf("String() leaked %q: %s", leak, s)
		}
	}
}
