package publisher

import (
	"encoding/json"
	"testing"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/presence"
)

func newService() (*Service, *RecordingPublisher) {
	rp := NewRecordingPublisher()
	return New(rp, "homeassistant", "unifi"), rp
}

func TestPublishDiscoveryRetained(t *testing.T) {
	s, rp := newService()
	devices := []presence.Device{{Hostname: "phone1"}, {Hostname: "phone2"}}
	if err := s.PublishDiscovery(t.Context(), devices); err != nil {
		t.Fatalf("discovery: %v", err)
	}
	for _, topic := range []string{
		"homeassistant/device_tracker/unifi_phone1_presence/config",
		"homeassistant/binary_sensor/unifi_phone1_connectivity/config",
		"homeassistant/device_tracker/unifi_phone2_presence/config",
		"homeassistant/binary_sensor/unifi_phone2_connectivity/config",
	} {
		rec, ok := rp.Get(topic)
		if !ok {
			t.Errorf("discovery topic %s not published", topic)
			continue
		}
		if !rec.Retain {
			t.Errorf("discovery topic %s not retained", topic)
		}
	}
}

func TestPublishStateJSON(t *testing.T) {
	s, rp := newService()
	d := presence.Device{Hostname: "phone1", IPAddress: "10.0.0.5", MACAddress: "aa:bb", Present: true}
	if err := s.PublishState(t.Context(), d); err != nil {
		t.Fatalf("state: %v", err)
	}
	rec, ok := rp.Get("unifi/phone1/state")
	if !ok || !rec.Retain {
		t.Fatal("state not published retained")
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Payload, &got); err != nil {
		t.Fatalf("state payload: %v", err)
	}
	if got["present"] != true || got["ip_address"] != "10.0.0.5" || got["mac_address"] != "aa:bb" {
		t.Errorf("state doc = %v", got)
	}
}

func TestPublishStateAbsent(t *testing.T) {
	s, rp := newService()
	if err := s.PublishState(t.Context(), presence.Device{Hostname: "phone1"}); err != nil {
		t.Fatalf("state: %v", err)
	}
	rec, _ := rp.Get("unifi/phone1/state")
	var got map[string]any
	_ = json.Unmarshal(rec.Payload, &got)
	if got["present"] != false {
		t.Errorf("absent device present = %v, want false", got["present"])
	}
}

func TestPublishStates(t *testing.T) {
	s, rp := newService()
	devices := []presence.Device{
		{Hostname: "phone1", Present: true},
		{Hostname: "phone2", Present: false},
	}
	if err := s.PublishStates(t.Context(), devices); err != nil {
		t.Fatalf("states: %v", err)
	}
	if _, ok := rp.Get("unifi/phone1/state"); !ok {
		t.Error("phone1 state not published")
	}
	if _, ok := rp.Get("unifi/phone2/state"); !ok {
		t.Error("phone2 state not published")
	}
}

func TestPublishAvailability(t *testing.T) {
	s, rp := newService()
	_ = s.PublishAvailability(t.Context(), true)
	rec, ok := rp.Get("unifi/availability")
	if !ok || string(rec.Payload) != "online" || !rec.Retain {
		t.Errorf("availability = %q retain=%v ok=%v", rec.Payload, rec.Retain, ok)
	}
	_ = s.PublishAvailability(t.Context(), false)
	rec, _ = rp.Get("unifi/availability")
	if string(rec.Payload) != "offline" {
		t.Errorf("availability = %q, want offline", rec.Payload)
	}
}

func TestAvailabilityTopic(t *testing.T) {
	s, _ := newService()
	if got := s.AvailabilityTopic(); got != "unifi/availability" {
		t.Errorf("AvailabilityTopic() = %q, want unifi/availability", got)
	}
}
