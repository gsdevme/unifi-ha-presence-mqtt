// Package publisher turns presence.Device values into retained MQTT messages: the
// Home Assistant discovery configs (once, on startup and reconnect), the per-device
// JSON state document (per poll), and service-wide availability. See
// docs/specs/03-mqtt-ha-discovery.md.
package publisher

import (
	"context"
	"fmt"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/homeassistant"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/presence"
)

// Publisher is the MQTT transport. Discovery/state/availability are always
// published retained; the transport uses QoS 1. Implementations: the autopaho
// client (prod) and a recording fake (tests).
type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte, retain bool) error
}

// Service publishes HA discovery, per-device presence state and service-wide
// availability for the tracked devices.
type Service struct {
	pub             Publisher
	discoveryPrefix string
	topicPrefix     string
}

// New builds a Service from a Publisher and the discovery/topic prefixes.
func New(pub Publisher, discoveryPrefix, topicPrefix string) *Service {
	return &Service{pub: pub, discoveryPrefix: discoveryPrefix, topicPrefix: topicPrefix}
}

// configFor builds the per-device discovery config for a hostname.
func (s *Service) configFor(host string) homeassistant.Config {
	return homeassistant.Config{
		DiscoveryPrefix: s.discoveryPrefix,
		TopicPrefix:     s.topicPrefix,
		Host:            host,
	}
}

// AvailabilityTopic exposes the service-wide topic used for LWT and explicit
// availability.
func (s *Service) AvailabilityTopic() string {
	return s.configFor("").AvailabilityTopic()
}

// PublishDiscovery publishes every tracked device's retained entity discovery
// configs.
func (s *Service) PublishDiscovery(ctx context.Context, devices []presence.Device) error {
	for _, d := range devices {
		msgs, err := homeassistant.BuildDiscovery(s.configFor(d.Hostname))
		if err != nil {
			return fmt.Errorf("build discovery for %s: %w", d.Hostname, err)
		}
		for _, m := range msgs {
			if err := s.pub.Publish(ctx, m.Topic, m.Payload, true); err != nil {
				return fmt.Errorf("publish discovery %s: %w", m.Topic, err)
			}
		}
	}
	return nil
}

// PublishState publishes the retained JSON state document for one device.
func (s *Service) PublishState(ctx context.Context, d presence.Device) error {
	cfg := s.configFor(d.Hostname)
	body, err := homeassistant.BuildState(homeassistant.State{
		Present:    d.Present,
		IPAddress:  d.IPAddress,
		MACAddress: d.MACAddress,
	})
	if err != nil {
		return fmt.Errorf("build state for %s: %w", d.Hostname, err)
	}
	if err := s.pub.Publish(ctx, cfg.StateTopic(), body, true); err != nil {
		return fmt.Errorf("publish state %s: %w", cfg.StateTopic(), err)
	}
	return nil
}

// PublishStates publishes the state document for every device in the snapshot.
func (s *Service) PublishStates(ctx context.Context, devices []presence.Device) error {
	for _, d := range devices {
		if err := s.PublishState(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// PublishAvailability publishes online/offline (retained) to the service-wide
// availability topic.
func (s *Service) PublishAvailability(ctx context.Context, online bool) error {
	payload := "offline"
	if online {
		payload = "online"
	}
	return s.pub.Publish(ctx, s.AvailabilityTopic(), []byte(payload), true)
}
