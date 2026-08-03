# Requirements

Stable, traceable requirement IDs. `/spec-reconcile` maps each `REQ-*` to its
implementation and flags drift. Prefix meanings: `UF` UniFi client, `DM` domain
model (presence), `HA` MQTT/Home Assistant, `SC` scheduling, `CF` config, `LC`
lifecycle/health, `TS` testing, `DP` deployment, `AS` Claude assets.

## UniFi client (`internal/unifi`)

- **REQ-UF-01** `Client` interface `ListSites`/`GetActiveClients(siteID)`, built by
  `NewHTTPClient(url, apiKey)`. → `client.go`, `http.go`
- **REQ-UF-02** Every request authenticates with a static `X-API-KEY` header (the local
  Integration API key); no login, cookie or session. → `http.go`
- **REQ-UF-03** Sites `GET proxy/network/integration/v1/sites` and connected clients
  `GET proxy/network/integration/v1/sites/<siteId>/clients`, `X-API-KEY`-authed. → `http.go`
- **REQ-UF-04** Transport uses `InsecureSkipVerify` (UDM self-signed certs) with a 10s
  timeout. → `http.go`
- **REQ-UF-05** A `401`/`403` on either endpoint returns `*AuthError` (bad key); it is
  not retried as re-auth. → `http.go`, `errors.go`
- **REQ-UF-06** `ClientResponse` exposes `Hostname, IPAddress, MACAddress, Name,
  LastSeen`, decoded from the Integration API connected-client JSON (`Hostname` falls
  back from `hostname` to `name`; `LastSeen` from ISO-8601 `connectedAt`). → `model.go`, `json.go`
- **REQ-UF-07** Client is fully self-contained in this repo (no external
  `github.com/gsdevme/unifi` module). → `go.mod`
- **REQ-UF-08** List responses use the pagination envelope
  `{offset,limit,count,totalCount,data}`; the client walks all pages. → `http.go`, `json.go`
- **REQ-UF-09** `UNIFI_SITE` (a human name) is resolved to a site UUID at startup via
  `ListSites`, falling back to the sole site when only one exists. → `cmd/serve.go`

## Domain model — presence (`internal/presence`)

- **REQ-DM-01** `Device{Hostname, IPAddress, MACAddress, Present}` is the per-device
  presence state published to HA; `Hostname` is the stable identity (topic/object-id/
  display name). → `presence.go`
- **REQ-DM-02** `Tracker` holds, per tracked device, a `Device`, a matcher pattern and a
  consecutive-miss counter, seeded from the tracked-entry list. → `presence.go`
- **REQ-DM-03** `Apply(active)` matches connected clients by the device's pattern; on a
  sighting it sets `Present=true`, refreshes ip/mac and resets the miss counter
  immediately. → `presence.go`
- **REQ-DM-04** Asymmetric debounce: a miss increments the counter and only flips
  `Present=false` once it reaches `FailureThreshold`; last-known ip/mac are retained
  while absent. → `presence.go`
- **REQ-DM-05** Devices start absent (`Present=false`) so entities exist before the
  first sighting. → `presence.go`
- **REQ-DM-06** `Apply` returns the full current per-device state (every tracked
  device) in a stable order. → `presence.go`
- **REQ-DM-07** Tracked entries are `Name` or `Name=Pattern`; `Pattern` (default
  `Name`) matches a client as a case-insensitive substring of its name/hostname, or —
  when MAC-like — of its MAC (separators ignored). First match wins. → `presence.go`

## MQTT & Home Assistant (`internal/mqtt`, `internal/homeassistant`, `internal/publisher`)

- **REQ-HA-01** Per-entity discovery topics `homeassistant/<component>/unifi_<host>_<key>/config`,
  retained, QoS 1. → `homeassistant/discovery.go`
- **REQ-HA-02** One retained JSON state doc per device at `~/state`
  (`{present, ip_address, mac_address}`); both entities read it via `value_template`. → `homeassistant/discovery.go`, `publisher/publisher.go`
- **REQ-HA-03** Shared `device` block (`identifiers:[unifi_<host>]`, `manufacturer:"Ubiquiti"`,
  `name:<hostname>`) + `~` base-topic (`<TopicPrefix>/<sanitizedHost>`) abbreviation. → `homeassistant/discovery.go`
- **REQ-HA-04** Two entities per device: a `device_tracker` (key `presence`,
  `source_type router`, `value_template` → `home`/`not_home`, `json_attributes_topic ~/state`)
  and a `binary_sensor` (key `connectivity`, `device_class connectivity`,
  `value_template` → `ON`/`OFF`). → `homeassistant/entities.go`
- **REQ-HA-05** Unlike a GPS tracker, the UniFi tracker emits `home`/`not_home`
  directly (the controller genuinely knows on-network vs not). → `homeassistant/entities.go`, `homeassistant/discovery.go`
- **REQ-HA-06** Service-wide availability via LWT: an absolute (non-`~`) topic
  `<TopicPrefix>/availability`; retained `offline` will, `online` on connect; every
  entity references it (`payload_available:"online"`, `payload_not_available:"offline"`). → `mqtt/client.go`, `homeassistant/discovery.go`
- **REQ-HA-07** All discovery/state/availability publishes are retained + QoS 1. → `mqtt/client.go`, `publisher/publisher.go`
- **REQ-HA-08** `Publisher` interface with autopaho MQTT5 prod backend + recording fake
  for tests. → `publisher/publisher.go`, `publisher/recording.go`, `mqtt/client.go`
- **REQ-HA-09** On every (re)connection the client republishes availability + discovery
  + last state (broker-restart safe). → `mqtt/client.go`, `cmd/serve.go`

## Scheduling (`internal/scheduler`)

- **REQ-SC-01** Poll loop at `POLL_INTERVAL` (default 10s), immediate first run,
  serialised (ticks never overlap). → `scheduler.go`
- **REQ-SC-02** Each tick: `GetActiveClients(siteID)` → `presence.Apply` → publish state
  per device → report success/failure to health. → `scheduler.go`
- **REQ-SC-03** No re-auth (the API key is static): every poll error — including a bad
  key — is retried up to `POLL_MAX_RETRIES` with exponential backoff and otherwise counts
  as a failure for readiness. → `scheduler.go`
- **REQ-SC-04** `Now`/`After` are injectable for deterministic `testing/synctest`
  tests. → `scheduler.go`

## Config (`internal/config`)

- **REQ-CF-01** All env vars in `05-config.md` bound with defaults. → `config.go`
- **REQ-CF-02** Fail-fast validation (`errors.Join`) — required present; duration/URL
  parse; `FAILURE_THRESHOLD >= 1`; `POLL_INTERVAL` floor. → `config.go`
- **REQ-CF-03** Secrets redacted in logs/`String()`. → `config.go`
- **REQ-CF-04** `MODE` (`live`|`mock`) resolves the UniFi target: `live` requires
  `UNIFI_HOST`/`UNIFI_API_KEY`; `mock` points the host at `MOCK_URL` and supplies a dummy
  API key. Code/image default `live`; `.env.dist` ships `mock`. → `config.go`, `.env.dist`, `Dockerfile`
- **REQ-CF-05** `TRACK_DEVICES` (comma list, required) parsed into the tracked-entry set;
  each entry is `Name` or `Name=Pattern` (see REQ-DM-07). → `config.go`, `presence.go`
- **REQ-CF-06** `godotenv` loads local `.env`; `.env.dist` template committed. → `cmd/root.go`, `.env.dist`

## Lifecycle & health (`internal/server`, `cmd`, `main.go`)

- **REQ-LC-01** `/healthz` liveness always-ok while running. → `internal/server`
- **REQ-LC-02** `/readyz` ready after first successful publish. → `internal/server`
- **REQ-LC-03** `/readyz` not-ready after `READY_FAILURE_THRESHOLD` consecutive
  failures; recovers on success. → `internal/server`, `scheduler.go`
- **REQ-LC-04** Structured `log/slog` logging; level/format configurable; no secrets. → `cmd/root.go`
- **REQ-LC-05** Graceful shutdown: explicit retained `offline` publish, clean
  `Disconnect()`, then exit. → `cmd/serve.go`
- **REQ-LC-06** Status/health server listens during init so probes work at startup. → `cmd/serve.go`
- **REQ-LC-07** Root `/` HTML status page: service, readiness, uptime, poll interval, Go
  version and the tracked devices (present/absent — IP/MAC omitted as sensitive); always
  `200`, no secrets. → `internal/server`
- **REQ-LC-08** `cmd/main.go` reports errors to stderr and exits non-zero on failure. → `cmd/main.go`

## Testing (`internal/mock`, `features`)

- **REQ-TS-01** Mock UniFi console implements the Integration API `GET
  proxy/network/integration/v1/sites` and `.../sites/<site>/clients` (configurable client
  list), requiring a non-empty `X-API-KEY`. → `mock/server.go`, `mock/routes.go`
- **REQ-TS-02** Mock is shared by the godog suite and the standalone `mock` subcommand. → `mock/server.go`, `cmd/mock.go`
- **REQ-TS-03** godog scenarios: startup site-resolution + discovery retained,
  present→`home` immediately, absent→`not_home` only after the threshold, availability
  `online`/graceful `offline`. → `features/presence.feature`
- **REQ-TS-04** Unit tests for `unifi`, `config`, `presence`, discovery, publisher,
  scheduler, server. → `*_test.go`
- **REQ-TS-05** Recording fake `Publisher` — no broker in tests. → `publisher/recording.go`
- **REQ-TS-06** `golangci-lint` (default linters + gofmt) run via `make lint`. → `.golangci.yml`, `Makefile`

## Deployment (`Dockerfile`, `go.mod`, `.github/workflows/`)

- **REQ-DP-01** Multi-stage Dockerfile: static `CGO_ENABLED=0`, distroless nonroot,
  `EXPOSE 8080`. → `Dockerfile`
- **REQ-DP-02** Image default env `MODE=live`; default `CMD serve`. → `Dockerfile`
- **REQ-DP-03** Module path `github.com/gsdevme/unifi-ha-presence-mqtt`, `go 1.26`. → `go.mod`
- **REQ-DP-04** GitHub Actions PR gate runs `make lint`/`test`/`test-e2e` and a
  build-only Docker validation on every pull request. → `.github/workflows/ci.yml`, `.github/workflows/checks.yml`
- **REQ-DP-05** On merge to `master`, release-please cuts a semver tag + GitHub Release
  from Conventional Commits; a release then builds and pushes a multi-arch
  (`amd64`+`arm64`) image to `ghcr.io/gsdevme/unifi-ha-presence-mqtt` (`:vX.Y.Z`,
  `:latest`). → `.github/workflows/release.yml`, `release-please-config.json`

## Claude assets

- **REQ-AS-01** Skill `home-assistant-mqtt-discovery`. → `.claude/skills/...`
- **REQ-AS-02** Skill `effective-go`. → `.claude/skills/...`
- **REQ-AS-03** Skill `go-1.26`. → `.claude/skills/...`
- **REQ-AS-04** Command `/spec-reconcile`. → `.claude/commands/spec-reconcile.md`
- **REQ-AS-05** Planner agent `go-http-pattern-planner`. → `.claude/agents/go-http-pattern-planner.md`
