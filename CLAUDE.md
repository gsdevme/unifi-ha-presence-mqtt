# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What this is

A stdlib-first Go service (go1.26) that polls a **UniFi Network** controller (UDM /
UniFi OS) and republishes selected devices' **presence** to **MQTT** with **Home
Assistant autodiscovery**. Read-only. A tracked device is reported `home` immediately
on sighting but only `not_home` after `FAILURE_THRESHOLD` consecutive missed polls
(the debounce). Structurally a twin of the sibling `hyundai-bluelink-mqtt`.

## Spec-driven workflow (read this first)

`docs/specs/*.md` are authored first and are the **source of truth**; code is
reconciled against them. `docs/specs/REQUIREMENTS.md` holds stable `REQ-*` IDs
(prefixes: `UF` unifi client, `DM` presence domain, `HA` mqtt/HA, `SC` scheduling,
`CF` config, `LC` lifecycle/health, `TS` testing, `DP` deployment, `AS` Claude
assets), each pointing at its implementation file. When you change behaviour, update
the relevant spec and `REQ-*` entry in the same change. `/spec-reconcile` maps
`REQ-*` to code and flags drift — run it after non-trivial changes.

## Commands

```sh
make build      # -> ./bin/unifi-ha-presence-mqtt
make run        # build + serve (poll -> MQTT; honours MODE from .env)
make run-mock   # build + run the standalone mock UniFi controller (:8090)
make test       # unit + integration (excludes the godog features suite)
make test-e2e   # godog acceptance suite (./features/...)
make lint       # golangci-lint (installs the pinned binary into ./bin)

gofmt -l $(git ls-files '*.go')   # must be empty
go vet ./... && go build ./...
```

## Architecture

- **`internal/unifi`** — self-contained, read-only client for the official UniFi
  Network **Integration API** (static `X-API-KEY` header; sites + connected clients
  under `/proxy/network/integration/v1`, paginated). `*AuthError` on 401/403 (bad
  key — no re-auth). No external UniFi module.
- **`internal/presence`** — pure domain: `Tracker` + the asymmetric debounce.
- **`internal/homeassistant`** — pure HA discovery/state payload builder (one
  device_tracker + one connectivity binary_sensor per device, shared retained state
  doc, service-wide availability).
- **`internal/mqtt` / `internal/publisher`** — autopaho MQTT5 (LWT + reconnect
  republish) behind a `Publisher` interface with a recording fake.
- **`internal/scheduler`** — poll loop (immediate first run + ticker), transient
  retry with backoff (no re-auth; the API key is static); `Now`/`After` injectable
  for `testing/synctest`.
- **`internal/server`** — `/healthz`, `/readyz`, HTML status page (devices
  present/absent; IP/MAC omitted).
- **`internal/mock`** — in-process controller mock shared by godog + the `mock`
  subcommand.
- **`internal/cmd`** — cobra root + `serve` (composition root) + `mock`.

## Conventions

- **stdlib-first.** Only external deps: cobra, autopaho (`paho.golang`), godotenv,
  and godog (test-only). Reach for stdlib first.
- Structured logging via `log/slog`; config `String()` redacts secrets — never log
  credentials.
- Accept interfaces, return structs; small interfaces defined at the consumer.
