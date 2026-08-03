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
| `MODE` | `live` | no | `live` targets a real console (Integration API); `mock` targets the local `mock` server. |
| `MOCK_URL` | `http://localhost:8090` | no | Mock base URL (only when `MODE=mock`). |
| `UNIFI_HOST` | — | if `live` | Console base URL, e.g. `https://192.168.1.1`. |
| `UNIFI_API_KEY` | — | if `live` | Local Integration API key (sent as `X-API-KEY`). See [Onboarding](#onboarding-local-api-key). |
| `UNIFI_SITE` | `default` | no | UniFi site **name**, resolved to its UUID at startup. |
| `TRACK_DEVICES` | — | yes | Comma-separated `Name` or `Name=Pattern` entries. See [Matching devices](#matching-devices). |
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

## Onboarding (local API key)

This service uses the official **UniFi Network Integration API** over a **local
connection** — a static API key, no username/password login (which breaks with MFA).

1. **Generate a local API key.** In the console's UniFi Network app:
   **Settings → Control Plane → Integrations → API Keys → Create API Key**, then copy
   it (it is shown once). *(The exact menu location can vary by firmware; if the local
   menu differs, `unifi.ui.com` can also mint a key for the console.)*
2. **Verify locally** (the console uses a self-signed cert, hence `-k`):
   ```sh
   curl -k -H "X-API-KEY: <key>" https://<gateway-ip>/proxy/network/integration/v1/sites
   curl -k -H "X-API-KEY: <key>" https://<gateway-ip>/proxy/network/integration/v1/sites/<siteId>/clients
   ```
   The second call shows the `name` and `macAddress` of each connected device — that is
   what `TRACK_DEVICES` matches against (see [Matching devices](#matching-devices)).
3. **Configure env:** `UNIFI_HOST=https://<gateway-ip>`, `UNIFI_API_KEY=<key>`,
   `UNIFI_SITE=default` (or your site name), `TRACK_DEVICES=iPhone-X=e8:78:65:00:fd:81`,
   `MQTT_BROKER_URL=…`.

At startup the service lists sites (validating the key) and logs the resolved site
UUID; a bad key keeps `/readyz` not-ready with a clear log.

## Matching devices

`TRACK_DEVICES` is a comma-separated list of `Name` or `Name=Pattern` entries:

- **`Name`** is the Home Assistant identity — the entity/device name and MQTT topic.
  You choose it; it need not match anything the console reports.
- **`Pattern`** (defaults to `Name`) is matched against each connected client. A client
  matches when the pattern is a **case-insensitive substring** of the client's name, or
  — when the pattern looks like a full or partial **MAC** — of the client's MAC
  (separators/case ignored).

The UniFi Integration API reports device names with a MAC-octet suffix appended (e.g.
`iPhone-X fd:81`) and offers no clean DHCP hostname, so matching is substring-based:

| Entry | Matches |
| --- | --- |
| `iPhone-X=e8:78:65:00:fd:81` | HA entity **iPhone-X**, matched by full MAC (rename-proof) |
| `iPhone-X` | HA entity **iPhone-X**, matched by name substring (`iPhone-X fd:81`) |
| `Phone=fd:81` | matched by a partial MAC suffix |

For phones that use a **rotating / private Wi-Fi MAC**, match on a stable UniFi
**name** rather than the MAC (set a fixed alias on the client in UniFi if needed).
Patterns should be specific enough to be unique — the first matching client wins.

## Running

### Docker (GHCR)

```bash
docker run --name unifi-ha-presence \
  -e UNIFI_HOST="https://192.168.1.1" \
  -e UNIFI_API_KEY="your-local-api-key" \
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
  UNIFI_API_KEY: eW91ci1sb2NhbC1hcGkta2V5   # your-local-api-key
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
