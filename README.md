# unifi-ha-presence-mqtt

A small, stdlib-first Go service that polls a **UniFi Network** controller (UDM /
UniFi OS) and republishes selected devices' **presence** to **MQTT** with **Home
Assistant autodiscovery** — so a tracked phone/laptop shows up in HA as a
`device_tracker` (home/not_home) and a `connectivity` `binary_sensor`, without
importing every client on your network.

## Why

The existing UniFi integration adds every single client on your network to Home
Assistant, cluttering it. This service tracks only the hostnames you list, debounces
Wi-Fi blips (a device is reported `home` instantly but only `not_home` after several
consecutive missed polls), and publishes clean retained discovery so entities exist
before the first sighting.

## Home Assistant entities

For each tracked device you get one HA **device** ("Ubiquiti / <hostname>") with two
entities, both reading a single retained JSON state doc (`unifi/<host>/state`):

| Entity | Component | Behaviour |
| --- | --- | --- |
| Presence | `device_tracker` | `home` / `not_home`; `source_type: router`; `ip_address`/`mac_address` as attributes |
| Connectivity | `binary_sensor` | `device_class: connectivity` (ON = on-network) |

Availability is service-wide (retained LWT on `unifi/availability`): all entities go
`unavailable` together if the service dies.

## Configuration

All configuration is via environment variables (a local `.env` is loaded automatically).

| Env var | Default | Required | Description |
| --- | --- | --- | --- |
| `MODE` | `live` | no | `live` targets a real controller; `mock` targets the local `mock` server. |
| `MOCK_URL` | `http://localhost:8090` | no | Mock base URL (only when `MODE=mock`). |
| `UNIFI_HOST` | — | if `live` | Controller base URL, e.g. `https://192.168.1.1`. |
| `UNIFI_USERNAME` | — | if `live` | Controller local-admin username. |
| `UNIFI_PASSWORD` | — | if `live` | Controller local-admin password. |
| `UNIFI_SITE` | `default` | no | UniFi site id. |
| `TRACK_DEVICES` | — | yes | Comma-separated hostnames to track, e.g. `phone1,phone2`. |
| `POLL_INTERVAL` | `10s` | no | Poll cadence (floor `2s`). |
| `FAILURE_THRESHOLD` | `5` | no | Consecutive missed polls before a device flips to `not_home` (≥1). |
| `POLL_MAX_RETRIES` | `3` | no | Transient-error retries per poll. |
| `READY_FAILURE_THRESHOLD` | `3` | no | Consecutive poll failures before `/readyz` flips not-ready. |
| `MQTT_BROKER_URL` | — | yes | e.g. `mqtt://mosquitto:1883` or `tls://host:8883`. |
| `MQTT_USERNAME` / `MQTT_PASSWORD` | — | no | MQTT auth. |
| `MQTT_CLIENT_ID` | `unifi-ha-presence-mqtt` | no | MQTT client id. |
| `MQTT_TOPIC_PREFIX` | `unifi` | no | Base topic prefix. |
| `HA_DISCOVERY_PREFIX` | `homeassistant` | no | HA discovery prefix. |
| `HEALTH_ADDR` | `:8080` | no | Listen address for `/`, `/healthz`, `/readyz`. |
| `LOG_LEVEL` / `LOG_FORMAT` | `info` / `json` | no | `log/slog` config. |

See `.env.dist` for a ready-to-copy template (it ships `MODE=mock`).

## Running

### Docker (GHCR)

```bash
docker run --name unifi-ha-presence \
  -e UNIFI_HOST="https://192.168.1.1" \
  -e UNIFI_USERNAME="admin" \
  -e UNIFI_PASSWORD="password" \
  -e TRACK_DEVICES="phone1,phone2" \
  -e MQTT_BROKER_URL="mqtt://mqtt:1883" \
  -p 8080:8080 \
  --restart on-failure \
  ghcr.io/gsdevme/unifi-ha-presence-mqtt:latest
```

### Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: unifi-ha-presence
data:
  UNIFI_HOST: "https://192.168.1.1"
  TRACK_DEVICES: "phone1,phone2"
  MQTT_BROKER_URL: "mqtt://mqtt:1883"
---
apiVersion: v1
kind: Secret
metadata:
  name: unifi-ha-presence
data:
  UNIFI_USERNAME: YWRtaW4=       # admin
  UNIFI_PASSWORD: cGFzc3dvcmQ=   # password
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: unifi-ha-presence
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
          image: ghcr.io/gsdevme/unifi-ha-presence-mqtt:latest
          envFrom:
            - configMapRef:
                name: unifi-ha-presence
            - secretRef:
                name: unifi-ha-presence
          ports:
            - containerPort: 8080
          readinessProbe:
            httpGet: { path: /readyz, port: 8080 }
          livenessProbe:
            httpGet: { path: /healthz, port: 8080 }
          resources:
            limits:
              memory: "32Mi"
              cpu: "50m"
```

### Local (no real controller or broker credentials)

```sh
cp .env.dist .env          # ships MODE=mock, TRACK_DEVICES=phone1,phone2
docker compose up -d       # or any local mosquitto on :1883
make run-mock              # terminal 1: fake UniFi controller on :8090
make run                   # terminal 2: the service (MODE=mock)
```

Then `mosquitto_sub -t 'homeassistant/#' -t 'unifi/#'` to watch discovery and state.

## Development

```sh
make build      # -> ./bin/unifi-ha-presence-mqtt
make test       # unit + integration (excludes the godog features suite)
make test-e2e   # godog acceptance suite (./features/...)
make lint       # golangci-lint (installs the pinned binary into ./bin)
```

This project is **spec-driven**: `docs/specs/*.md` are authored first and are the
source of truth; `docs/specs/REQUIREMENTS.md` holds stable `REQ-*` IDs mapped to code.
Run `/spec-reconcile` after non-trivial changes to catch drift. The UniFi controller
client is vendored in-tree at `internal/unifi` (no external module).
