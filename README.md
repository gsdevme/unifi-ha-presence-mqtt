# Unifi Presence to MQTT for Home Assistant

The integration supports MQTT auto discovery

# Why?

The current existing integration adds every single device from your unifi OS into home assistant and this clutters home assistant.

# Running

```bash
docker run --name=unifi-ha \
-e MQTT_DNS="mqtt://127.0.0.1:1883" \
-e TRACK_DEVICES="phone1,phone2" \
-e UNIFI_HOST="https://192.168.1.1" \
-e UNIFI_USERNAME="admin" \
-e UNIFI_PASSWORD="password" \
--restart=on-failure \
gsdevme/unifi-ha-presence-mqtt:latest
```

## TODO

- [ ] Handle the client better generally in the code (memory and failure wise)
- [ ] Move HTTP and Broker to Channels? 
- [ ] Better error handling

## Packages

- https://github.com/gsdevme/unifi
- https://hub.docker.com/r/gsdevme/unifi-ha-presence-mqtt