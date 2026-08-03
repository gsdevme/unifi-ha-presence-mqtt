// Package homeassistant builds Home Assistant MQTT autodiscovery payloads for a
// tracked UniFi device. See docs/specs/03-mqtt-ha-discovery.md and the
// home-assistant-mqtt-discovery skill, which are the source of truth for these
// rules.
package homeassistant

// Component types.
const (
	BinarySensor  = "binary_sensor"
	DeviceTracker = "device_tracker"
)

// Entity describes a single HA entity derived from the shared JSON state document.
// Key is both the discovery object-id suffix and the semantic entity name; both
// entities read the same `present` field of the state document.
type Entity struct {
	Component   string
	Key         string
	Name        string
	DeviceClass string
}

// Entities returns the two entities published for every tracked device: a
// device_tracker for presence (home/not_home) and a connectivity binary_sensor.
func Entities() []Entity {
	return []Entity{
		{Component: DeviceTracker, Key: "presence", Name: "Presence"},
		{Component: BinarySensor, Key: "connectivity", Name: "Connectivity", DeviceClass: "connectivity"},
	}
}
