package homeassistant

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Config identifies the topics and device metadata for one tracked device's
// discovery. The publisher builds one Config per tracked device.
type Config struct {
	DiscoveryPrefix string // e.g. "homeassistant"
	TopicPrefix     string // e.g. "unifi"
	Host            string // tracked hostname, e.g. "phone1"
	Name            string // HA device/entity name (defaults to Host)
}

// Message is a single MQTT publish (topic + payload); discovery and state
// messages are always retained at QoS 1.
type Message struct {
	Topic   string
	Payload []byte
}

// State is the retained JSON state document per device that both entities read.
type State struct {
	Present    bool   `json:"present"`
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address"`
}

var nonTopicChar = regexp.MustCompile(`[^A-Za-z0-9_]`)

// sanitizeHost lower-cases the hostname and replaces any character outside
// [A-Za-z0-9_] so it is safe in an MQTT topic and HA object id.
func sanitizeHost(host string) string {
	return nonTopicChar.ReplaceAllString(strings.ToLower(host), "_")
}

// BaseTopic is the per-device base topic used as the `~` abbreviation.
func (c Config) BaseTopic() string {
	return c.TopicPrefix + "/" + sanitizeHost(c.Host)
}

// StateTopic is the single retained JSON state topic for this device.
func (c Config) StateTopic() string { return c.BaseTopic() + "/state" }

// AvailabilityTopic is the service-wide LWT/availability topic. It is absolute
// (shared by every device), not a `~` child of a single device's base topic.
func (c Config) AvailabilityTopic() string { return c.TopicPrefix + "/availability" }

// deviceID is the stable HA device identifier for this host.
func (c Config) deviceID() string { return "unifi_" + sanitizeHost(c.Host) }

// BuildState marshals the retained state document for this device.
func BuildState(st State) ([]byte, error) {
	return json.Marshal(st)
}

// BuildDiscovery returns the retained discovery config messages for both entities.
func BuildDiscovery(c Config) ([]Message, error) {
	device := map[string]any{
		"identifiers":  []string{c.deviceID()},
		"manufacturer": "Ubiquiti",
		"name":         c.nameOrHost(),
	}
	entities := Entities()
	msgs := make([]Message, 0, len(entities))
	for _, e := range entities {
		body, err := json.Marshal(buildEntityPayload(c, e, device))
		if err != nil {
			return nil, fmt.Errorf("marshal discovery for %s: %w", e.Key, err)
		}
		topic := fmt.Sprintf("%s/%s/%s_%s/config", c.DiscoveryPrefix, e.Component, c.deviceID(), e.Key)
		msgs = append(msgs, Message{Topic: topic, Payload: body})
	}
	return msgs, nil
}

func buildEntityPayload(c Config, e Entity, device map[string]any) map[string]any {
	uniq := c.deviceID() + "_" + e.Key
	p := map[string]any{
		"~":                     c.BaseTopic(),
		"name":                  e.Name,
		"unique_id":             uniq,
		"object_id":             uniq,
		"state_topic":           "~/state",
		"availability_topic":    c.AvailabilityTopic(), // absolute: service-wide
		"payload_available":     "online",
		"payload_not_available": "offline",
		"device":                device,
	}
	if e.DeviceClass != "" {
		p["device_class"] = e.DeviceClass
	}
	switch e.Component {
	case DeviceTracker:
		// The controller genuinely knows on-network vs not, so we emit home/not_home
		// directly (unlike a GPS tracker, which defers zone resolution to HA).
		p["value_template"] = "{{ 'home' if value_json.present else 'not_home' }}"
		p["json_attributes_topic"] = "~/state"
		p["source_type"] = "router"
	case BinarySensor:
		p["value_template"] = "{{ 'ON' if value_json.present else 'OFF' }}"
		p["payload_on"] = "ON"
		p["payload_off"] = "OFF"
	}
	return p
}

func (c Config) nameOrHost() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Host
}
