package hass

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gsdevme/unifi/pkg/unifi"
	"regexp"
	"strings"
	"time"
)

const Home = "home"
const NotHome = "not_home"

func ensureBrokerOpen(broker mqtt.Client) error {
	if !broker.IsConnectionOpen() {
		t := broker.Connect()

		// TODO sort this out better
		for !t.WaitTimeout(3 * time.Second) {

		}

		if err := t.Error(); err != nil {
			return fmt.Errorf("cannot connect to broker: %s", err)
		}
	}

	return nil
}

func PublishClientState(client unifi.ClientResponse, broker mqtt.Client) error {
	if err := ensureBrokerOpen(broker); err != nil {
		return err
	}

	device, _ := json.Marshal(DeviceAttributes{
		IpAddress:  client.IpAddress,
		MacAddress: client.MacAddress,
	})

	var t mqtt.Token

	t = broker.Publish(getStateTopic(client.Hostname), 0, false, Home)

	if !t.WaitTimeout(time.Second * 1000) {
		return fmt.Errorf("failed to publish home %w", t.Error())
	}

	t = broker.Publish(getAttributesTopic(client.Hostname), 0, false, string(device))

	if !t.WaitTimeout(time.Second * 1000) {
		return fmt.Errorf("failed to publish attributes %w", t.Error())
	}

	return nil
}

func PublishNotHome(name string, broker mqtt.Client) error {
	if err := ensureBrokerOpen(broker); err != nil {
		return err
	}

	t := broker.Publish(getStateTopic(name), 0, false, NotHome)

	if !t.WaitTimeout(time.Second * 1000) {
		return fmt.Errorf("failed to publish not_home %w", t.Error())
	}

	return nil
}

func PublishAutoDiscoveryForClient(client unifi.ClientResponse, broker mqtt.Client) error {
	if err := ensureBrokerOpen(broker); err != nil {
		return err
	}

	config := newDiscoveryConfig(client.Hostname, client.DeviceName)
	payload, _ := json.Marshal(config)

	t := broker.Publish(fmt.Sprintf("homeassistant/device_tracker/%s/config", config.UniqueId), 0, false, string(payload))

	if !t.WaitTimeout(time.Second * 1000) {
		return fmt.Errorf("failed to publish auto discovery %w", t.Error())
	}

	return PublishClientState(client, broker)
}

func PublishAutoDiscoveryPlaceholder(name string, broker mqtt.Client) error {
	config := newDiscoveryConfig(name, "Unknown")
	payload, _ := json.Marshal(config)

	t := broker.Publish(fmt.Sprintf("homeassistant/device_tracker/%s/config", config.UniqueId), 0, false, string(payload))

	if !t.WaitTimeout(time.Second * 1000) {
		return fmt.Errorf("failed to publish auto discovery %w", t.Error())
	}

	return nil
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
