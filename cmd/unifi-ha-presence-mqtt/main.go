package main

import (
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/martian/log"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/hass"
	"github.com/gsdevme/unifi/pkg/unifi"
	"os"
	"strings"
	"time"
)

const FailureThreshold = 5

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

func pollUnifi(client unifi.Client, broker mqtt.Client, devices []string, failureThresholdCounter map[string]int) error {
	var err error

	// TODO this likely makes sense to do but have another thing about it
	for _, device := range devices {
		if err = hass.PublishAutoDiscoveryPlaceholder(device, broker); err != nil {
			return err
		}
	}

	defer broker.Disconnect(5)

	// TODO move to configuration/envs
	clients, err := client.GetActiveClients("default")

	if err != nil {
		return err
	}

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
		if err = hass.PublishAutoDiscoveryForClient(client, broker); err != nil {
			return err
		}

		failureThresholdCounter[client.Hostname] = 0
	}

	for _, device := range devices {
		failureThresholdCounter[device] = failureThresholdCounter[device]+1

		if failureThresholdCounter[device] >= FailureThreshold {
			log.Debugf("device %s not found to be connected to the wifi and is not present", device)

			if err = hass.PublishNotHome(device, broker); err != nil {
				return err
			}

			failureThresholdCounter[device] = 0
		}
	}

	return nil
}

func fetchAuthTokenWithCredentials() (string, error) {
	httpClient := unifi.NewHTTPClient(os.Getenv("UNIFI_HOST"), unifi.WithCredentials(os.Getenv("UNIFI_USERNAME"), os.Getenv("UNIFI_PASSWORD")))
	authToken, err := httpClient.GetAuthToken()

	if err != nil {
		return "", err
	}

	return authToken, nil
}

func main() {
	log.SetLevel(log.Debug) // @TODO

	broker, err := setupBroker()

	if err != nil {
		fmt.Println(err.Error())

		os.Exit(1)
	}

	authToken, err := fetchAuthTokenWithCredentials()

	if err != nil {
		fmt.Println(err.Error())

		os.Exit(1)
	}

	httpClient := unifi.NewHTTPClient(os.Getenv("UNIFI_HOST"), unifi.WithAuthToken(authToken))
	devices := strings.Split(os.Getenv("TRACK_DEVICES"), ",")

	failureThresholdCounter := make(map[string]int)

	for _, device := range devices {
		failureThresholdCounter[device] = 0
	}

	for {
		err = pollUnifi(httpClient, broker, devices, failureThresholdCounter)

		if err != nil {
			var authError *unifi.HttpAuthError
			if errors.As(err, &authError) {
				fmt.Println("Token expired, Refreshing token")

				authToken, err := fetchAuthTokenWithCredentials()

				if err != nil {
					fmt.Println(err.Error())

					os.Exit(1)
				}

				httpClient = unifi.NewHTTPClient(os.Getenv("UNIFI_HOST"), unifi.WithAuthToken(authToken))

				continue
			}

			fmt.Println(err.Error())

			os.Exit(1)
		}

		time.Sleep(10 * time.Second)
	}

	// @TODO move MQTT into channel?
	// @TODO move HTTP into channel?
	// @TODO handle signal to close channels
	// @TODO handle the client better in general, memory wise also
}
