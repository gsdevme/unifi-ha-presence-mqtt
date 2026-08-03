# 05 — Configuration

All configuration is via **environment variables**, bound in `internal/config` with
stdlib `os.Getenv` + validation. For local dev, `godotenv` loads a `.env` file (a no-op
in prod where the file is absent). `.env.dist` is the committed template; `.env` is
git-ignored and holds real secrets.

## Variables

| Env var | Default | Required | Description |
| --- | --- | --- | --- |
| `MODE` | `live` | no | `live` targets the real UniFi console (Integration API); `mock` targets the local `mock` server. |
| `MOCK_URL` | `http://localhost:8090` | no | Mock server base URL; used only when `MODE=mock`. Must match the `mock` command's `--addr`. |
| `UNIFI_HOST` | — | if `live` | Console base URL, e.g. `https://192.168.1.1`. Ignored in `mock` (resolved to `MOCK_URL`). |
| `UNIFI_API_KEY` | — | if `live` | Local Integration API key, sent as the `X-API-KEY` header. Ignored in `mock` (a dummy is used). |
| `UNIFI_SITE` | `default` | no | UniFi site **name**, resolved to a site UUID at startup (falls back to the sole site). |
| `TRACK_DEVICES` | — | yes | Comma-separated `Name` or `Name=Pattern` entries, e.g. `iPhone-X=e8:78:65:00:fd:81`. `Name` is the HA identity; `Pattern` (default `Name`) is a case-insensitive substring matched against the client's name or MAC (see REQ-DM-07). |
| `POLL_INTERVAL` | `10s` | no | Poll cadence (`time.ParseDuration`). Floor `2s`. |
| `FAILURE_THRESHOLD` | `5` | no | Consecutive missed polls before a device flips to `not_home`. Must be `>= 1`. |
| `POLL_MAX_RETRIES` | `3` | no | Transient-error retries per poll before it counts as failed. |
| `READY_FAILURE_THRESHOLD` | `3` | no | Consecutive poll failures before `/readyz` flips not-ready. |
| `MQTT_BROKER_URL` | — | yes | e.g. `mqtt://mosquitto:1883` or `tls://host:8883`. |
| `MQTT_USERNAME` | — | no | MQTT auth username. |
| `MQTT_PASSWORD` | — | no | MQTT auth password. |
| `MQTT_CLIENT_ID` | `unifi-ha-presence-mqtt` | no | MQTT client id. |
| `MQTT_TOPIC_PREFIX` | `unifi` | no | Base topic prefix (sanitized host appended). |
| `HA_DISCOVERY_PREFIX` | `homeassistant` | no | HA discovery prefix. |
| `HEALTH_ADDR` | `:8080` | no | Listen address for `/`, `/healthz` and `/readyz`. |
| `LOG_LEVEL` | `info` | no | `debug`/`info`/`warn`/`error` (`log/slog`). |
| `LOG_FORMAT` | `json` | no | `json` or `text`. |

## Validation rules (REQ-CF-02)

- Fail fast via `errors.Join` (exit non-zero with a clear, aggregated message) when a
  required var is missing or a duration/URL fails to parse.
- `MODE` must be `live` or `mock`. In `mock`, `UNIFI_HOST` resolves to `MOCK_URL` and
  the API-key requirement is dropped (a dummy is supplied); in `live`, `UNIFI_HOST` and
  `UNIFI_API_KEY` are required.
- `TRACK_DEVICES` must contain at least one non-empty hostname (REQ-CF-05).
- `POLL_INTERVAL` below the `2s` floor is rejected.
- `FAILURE_THRESHOLD` must be `>= 1`; `POLL_MAX_RETRIES` and `READY_FAILURE_THRESHOLD`
  must be `>= 0` / `>= 1` respectively.
- Secrets (`UNIFI_API_KEY`, `MQTT_PASSWORD`) are never logged; the config's `String()`
  redacts them by omission (REQ-CF-03).

## Defaults differ by surface (REQ-CF-04)

The container image sets `MODE=live` in its default env; local runs default to `live`
too, but the committed `.env.dist` ships `MODE=mock` so a fresh `cp .env.dist .env`
runs against the local mock out of the box. `MQTT_BROKER_URL` and `TRACK_DEVICES` are
required in both modes.
