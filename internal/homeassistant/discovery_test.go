package homeassistant

import (
	"encoding/json"
	"testing"
)

func testConfig() Config {
	return Config{
		DiscoveryPrefix: "homeassistant",
		TopicPrefix:     "unifi",
		Host:            "Phone-1",
		Name:            "Phone 1",
	}
}

func decodeByTopic(t *testing.T, msgs []Message) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, m := range msgs {
		var p map[string]any
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("bad payload for %s: %v", m.Topic, err)
		}
		out[m.Topic] = p
	}
	return out
}

func mustBuild(t *testing.T) map[string]map[string]any {
	t.Helper()
	msgs, err := BuildDiscovery(testConfig())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(msgs) != len(Entities()) {
		t.Fatalf("got %d messages, want %d", len(msgs), len(Entities()))
	}
	return decodeByTopic(t, msgs)
}

func TestSanitizedTopicsAndDevice(t *testing.T) {
	byTopic := mustBuild(t)
	tr, ok := byTopic["homeassistant/device_tracker/unifi_phone_1_presence/config"]
	if !ok {
		t.Fatalf("missing tracker discovery topic; got %v", keys(byTopic))
	}
	if tr["~"] != "unifi/phone_1" {
		t.Errorf("~ = %v, want unifi/phone_1 (sanitized, lower-cased)", tr["~"])
	}
	if tr["state_topic"] != "~/state" {
		t.Errorf("state_topic = %v", tr["state_topic"])
	}
	if tr["unique_id"] != "unifi_phone_1_presence" || tr["object_id"] != "unifi_phone_1_presence" {
		t.Errorf("unique/object id = %v / %v", tr["unique_id"], tr["object_id"])
	}
	dev, _ := tr["device"].(map[string]any)
	if dev == nil || dev["manufacturer"] != "Ubiquiti" || dev["name"] != "Phone 1" {
		t.Errorf("device block = %v", tr["device"])
	}
	ids, _ := dev["identifiers"].([]any)
	if len(ids) != 1 || ids[0] != "unifi_phone_1" {
		t.Errorf("identifiers = %v", dev["identifiers"])
	}
}

func TestDeviceTrackerEmitsHomeNotHome(t *testing.T) {
	tr := mustBuild(t)["homeassistant/device_tracker/unifi_phone_1_presence/config"]
	if tr["value_template"] != "{{ 'home' if value_json.present else 'not_home' }}" {
		t.Errorf("tracker value_template = %v", tr["value_template"])
	}
	if tr["source_type"] != "router" {
		t.Errorf("source_type = %v, want router", tr["source_type"])
	}
	if tr["json_attributes_topic"] != "~/state" {
		t.Errorf("json_attributes_topic = %v", tr["json_attributes_topic"])
	}
}

func TestConnectivityBinarySensor(t *testing.T) {
	bs := mustBuild(t)["homeassistant/binary_sensor/unifi_phone_1_connectivity/config"]
	if bs["device_class"] != "connectivity" {
		t.Errorf("device_class = %v, want connectivity", bs["device_class"])
	}
	if bs["value_template"] != "{{ 'ON' if value_json.present else 'OFF' }}" {
		t.Errorf("connectivity value_template = %v", bs["value_template"])
	}
	if bs["state_topic"] != "~/state" {
		t.Errorf("state_topic = %v", bs["state_topic"])
	}
}

func TestServiceWideAvailabilityIsAbsolute(t *testing.T) {
	for topic, p := range mustBuild(t) {
		// Availability is service-wide, so it must be the absolute topic, never ~/...
		if p["availability_topic"] != "unifi/availability" {
			t.Errorf("%s availability_topic = %v, want unifi/availability", topic, p["availability_topic"])
		}
		if p["payload_available"] != "online" || p["payload_not_available"] != "offline" {
			t.Errorf("%s availability payloads = %v / %v", topic, p["payload_available"], p["payload_not_available"])
		}
	}
	if got := testConfig().AvailabilityTopic(); got != "unifi/availability" {
		t.Errorf("AvailabilityTopic() = %q, want unifi/availability", got)
	}
}

func TestBuildState(t *testing.T) {
	body, err := BuildState(State{Present: true, IPAddress: "10.0.0.5", MACAddress: "aa:bb"})
	if err != nil {
		t.Fatalf("build state: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["present"] != true || got["ip_address"] != "10.0.0.5" || got["mac_address"] != "aa:bb" {
		t.Errorf("state doc = %v", got)
	}
}

func keys(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
