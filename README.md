# Unifi Presence to MQTT for Home Assistant

The integration supports MQTT auto discovery

# Why?

The current existing integration adds every single device from your unifi OS into home assistant and this clutters home assistant.

## TODO

- [ ] Handle the client better generally in the code (memory and failure wise)
- [ ] Move HTTP and Broker to Channels? 
- [ ] Better error handling
- [ ] Improve ENV

## How to Use

.. write this

# Configuration

```bash
export MQTT_DNS="mqtt://127.0.0.1:1883"
export UNIFI_PASSWORD="password"
```

## Packages

- https://github.com/gsdevme/unifi