# Unifi Presence to MQTT for Home Assistant

The integration supports MQTT auto discovery

# Why?

The current existing integration adds every single device from your unifi OS into home assistant and this clutters home assistant.

# Running

## Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: unifi-ha-presence
data:
  TRACK_DEVICES: "phone1,phone2"
  UNIFI_HOST: "https://192.168.1.1"
  MQTT_DNS: "mqtt://mqtt:1883"
---
apiVersion: v1
kind: Secret
metadata:
  name: unifi-ha-presence
data:
  UNIFI_PASSWORD: cGFzc3dvcmQ=
  UNIFI_USERNAME: YWRtaW4=
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: unifi-ha-presence
  labels:
    app: unifi-ha-presence
spec:
  selector:
    matchLabels:
      app: unifi-ha-presence
  template:
    metadata:
      labels:
        app: unifi-ha-presence
    spec:
      containers:
        - name: unifi-ha-presence
          image: gsdevme/unifi-ha-presence-mqtt:latest
          envFrom:
            - configMapRef:
                name: unifi-ha-presence
            - secretRef:
                name: unifi-ha-presence
          resources:
            limits:
              memory: "32Mi"
              cpu: "50m"
```              

## Docker 

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
