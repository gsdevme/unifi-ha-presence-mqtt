package main

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/martian/log"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/hass"
	"github.com/gsdevme/unifi/pkg/unifi"
	"os"
	"time"
)

func setupBroker() (mqtt.Client, error) {


	clientOptions := mqtt.NewClientOptions().
		SetClientID("gsdevme/unifi-ha-presence-mqtt").
		AddBroker(os.Getenv("MQTT_DNS")).
		SetCleanSession(true).
		SetOrderMatters(false).
		SetPingTimeout(5).
		SetKeepAlive(5 * time.Second).
		SetAutoReconnect(false)
	mqttClient := mqtt.NewClient(clientOptions)

	token := mqttClient.Connect()

	// TODO sort this out better
	for !token.WaitTimeout(3 * time.Second) {

	}

	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("cannot connect to broker: %s", err)
	}

	return mqttClient, nil
}

func main() {
	log.SetLevel(log.Debug) // @TODO

	broker, err := setupBroker()

	if err != nil {
		fmt.Println(err.Error())

		os.Exit(1)
	}

	// TODO move to configuration (yaml... maybe csv style ENV)
	devices := []string{"Ceilidhs-Phone", "iPhone-X"}


	// TODO this likely makes sense to do but have another thing about it
	for _, device := range devices {
		hass.PublishAutoDiscoveryPlaceholder(device, broker)
	}

	defer broker.Disconnect(5)

	// TODO move to configuration/envs
	c := unifi.NewHTTPClient("https://172.16.16.1", unifi.WithCredentials("admin", os.Getenv("UNIFI_PASSWORD")))
	clients, _ := c.GetActiveClients("default")


	var activeClients []unifi.ClientResponse
	var presentClients []unifi.ClientResponse

	for _, client := range *clients {
		if client.Hostname != "" {
			activeClients = append(activeClients, client)
		}
	}

	for _, client := range activeClients {
		for deviceIndex, deviceName := range devices {
			if deviceName == client.Hostname {

				presentClients = append(presentClients, client)

				devices = append(devices[:deviceIndex], devices[deviceIndex+1:]...)
			}
		}
	}

	for _, client := range presentClients {
		log.Debugf("%s", client)
		hass.PublishAutoDiscoveryForClient(client, broker)
	}

	for _, device := range devices {
		log.Debugf("device %s not found to be connected to the wifi and is not present", device)
		hass.PublishNotHome(device, broker)
	}

	// @TODO move MQTT into channel?
	// @TODO move HTTP into channel?
	// @TODO handle signal to close channels
	// @TODO handle the client better in general, memory wise also
}
