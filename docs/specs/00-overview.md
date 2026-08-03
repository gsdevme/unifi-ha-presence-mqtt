# 00 — Overview

## Purpose

A small, stdlib-first Go service that polls a **UniFi Network** controller (UDM /
UniFi OS) and republishes selected devices' **presence** to **MQTT** with **Home
Assistant (HA) MQTT autodiscovery**, so a tracked phone/laptop appears in HA as a
`device_tracker` (home/not_home) and a `connectivity` `binary_sensor` — no manual HA
configuration, and without importing every client on the network.

Deployed to Kubernetes / Docker (manifests are illustrative in the README; none in
this repo).

## Principles

- **Spec-driven.** These `docs/specs/*` are authored first and are the source of truth.
  Code is reconciled against `REQUIREMENTS.md` via `/spec-reconcile`.
- **stdlib-first.** Latest Go (**go1.26.2**). Only unavoidable external deps: MQTT
  client (`paho.golang`), CLI (`cobra`), `.env` loader (`godotenv`) and BDD runner
  (`godog`, test-only). All core logic — HTTP, JSON, scheduling, logging — is stdlib.
- **Self-contained.** The UniFi controller client is baked into `internal/unifi`; there
  is no external `github.com/gsdevme/unifi` dependency.
- **Read-only.** The service only reads controller status (`stat/sta`). It never
  changes controller or client configuration.
- **Debounced presence.** A device is reported `home` immediately on sighting, but only
  `not_home` after `FAILURE_THRESHOLD` consecutive missed polls — this smooths Wi-Fi
  roaming, power-save and momentary disconnects.
- **Kube-friendly.** HTTP health/readiness probes, env-based config, structured logging
  (`log/slog`), graceful shutdown, non-root minimal (distroless) image.

## High-level flow

1. Load + validate config from environment (`godotenv` loads a local `.env`).
2. `internal/unifi` authenticates (`POST api/auth/login` → `TOKEN` cookie) or reuses a
   supplied token; on `401` it transparently re-authenticates.
3. Publish HA **discovery** configs once on startup for every tracked device (retained,
   QoS 1) so entities exist before the first sighting.
4. `internal/scheduler` polls active clients every `POLL_INTERVAL` (default 10s),
   feeds them to `internal/presence` (the debounce), and publishes a retained JSON state
   document per device.
5. `internal/server` exposes an HTML status page on `/` plus `/healthz` (liveness) and
   `/readyz` (readiness).
6. On `SIGTERM`, publish `offline` availability (retained), `Disconnect()` cleanly,
   exit 0.

## Non-goals

- Changing UniFi/controller configuration of any kind (read-only).
- GPS/zone location — presence is on-network vs not, mapped to `home`/`not_home`.
- Kubernetes manifests, Helm charts, or MQTT broker provisioning.

## Reference

The UniFi client is a self-contained port of `github.com/gsdevme/unifi/pkg/unifi`,
covering the two endpoints this service needs (login, active clients).

See the sibling specs:
- `01-unifi-client.md` — controller endpoints, auth, insecure transport.
- `02-domain-model.md` — `Device`, `Tracker` and the presence debounce.
- `03-mqtt-ha-discovery.md` — MQTT topics and HA discovery design.
- `04-polling-scheduling.md` — poll loop and re-auth.
- `05-config.md` — environment variables.
- `06-lifecycle-health.md` — probes, readiness, graceful shutdown, status page.
- `07-deployment.md` — container, CI/CD, release.
- `REQUIREMENTS.md` — traceable requirement IDs (`REQ-*`).
