package hass

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gsdevme/unifi/pkg/unifi"
	"regexp"
	"strings"
)

const Home = "home"
const NotHome = "not_home"

func PublishClientState(client unifi.ClientResponse, broker mqtt.Client) {
	device, _ := json.Marshal(DeviceAttributes{
		IpAddress:  client.IpAddress,
		MacAddress: client.MacAddress,
	})

	broker.Publish(getStateTopic(client.Hostname), 0, false, Home)
	broker.Publish(getAttributesTopic(client.Hostname), 0, false, string(device))
}

func PublishNotHome(name string, broker mqtt.Client) {
	broker.Publish(getStateTopic(name), 0, false, NotHome)
}

func PublishAutoDiscoveryForClient(client unifi.ClientResponse, broker mqtt.Client) {
	config := newDiscoveryConfig(client.Hostname, client.DeviceName)
	payload, _ := json.Marshal(config)

	broker.Publish(fmt.Sprintf("homeassistant/device_tracker/%s/config", config.UniqueId), 0, false, string(payload))

	PublishClientState(client, broker)
}

func PublishAutoDiscoveryPlaceholder(name string, broker mqtt.Client) {
	config := newDiscoveryConfig(name, "Unknown")
	payload, _ := json.Marshal(config)

	broker.Publish(fmt.Sprintf("homeassistant/device_tracker/%s/config", config.UniqueId), 0, false, string(payload))
}

func newDiscoveryConfig(name string, manufacturer string) *discoveryConfig {
	id := fmt.Sprintf("unifi_%s", sanitizeHostnameForBroker(name))

	return &discoveryConfig{
		StateTopic:      getStateTopic(name),
		AttributesTopic: getAttributesTopic(name),
		Name:            name,
		PayloadHome:     Home,
		PayloadNotHome:  NotHome,
		SourceType:      "router",
		UniqueId:        id,
		Device: deviceConfig{
			Name:         name,
			Identifiers:  []string{id},
			Manufacturer: manufacturer,
		},
	}
}

func sanitizeHostnameForBroker(hostname string) string {
	r := regexp.MustCompile("[^A-Z0-9a-z_]")

	return r.ReplaceAllString(hostname, "_")
}

func getTopic(hostname string) string {
	return fmt.Sprintf("unifi/%s", sanitizeHostnameForBroker(strings.ToLower(hostname)))
}

func getStateTopic(hostname string) string {
	return fmt.Sprintf("%s/state", getTopic(hostname))
}

func getAttributesTopic(hostname string) string {
	return fmt.Sprintf("%s/attributes", getTopic(hostname))
}
