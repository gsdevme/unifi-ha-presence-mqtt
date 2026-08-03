# 06 — Lifecycle & health

`internal/server` exposes the daemon's HTTP surface; `internal/cmd` is the composition
root and process lifecycle.

## HTTP surface (`internal/server`)

- **`GET /healthz`** — liveness, always `200 ok` while the process runs (REQ-LC-01).
- **`GET /readyz`** — readiness (REQ-LC-02, REQ-LC-03):
  - `503` until the first successful publish.
  - `200 ready` after the first success.
  - flips back to `503` after `READY_FAILURE_THRESHOLD` consecutive poll failures;
    recovers on the next success.
- **`GET /{$}`** — an HTML status page (REQ-LC-07): service name, readiness, uptime,
  poll interval, Go version, and the tracked devices each shown as **present**/**absent**.
  IP and MAC are intentionally omitted as mildly sensitive. Always `200`, never leaks
  secrets. Unknown paths `404` (exact-root pattern).

`server.Config` and `server.Metrics`/device snapshot are decoupled from
`internal/config`/`internal/presence` so `serve.go` maps domain values in and there is no
import coupling in the server package (mirrors the Bluelink reference). The server tracks
readiness via `MarkSuccess`/`MarkFailure` (it is the `HealthReporter`).

## Process lifecycle (`internal/cmd`)

Startup (`serve.go`, REQ-LC-05, REQ-LC-06):

1. Load + validate config; build the logger; warn if `MODE=mock`.
2. Start the status/health server **immediately** (listens during init so probes work
   at startup).
3. Build the UniFi client (authenticate, or reuse a token) — a `ClientProvider` that
   owns credentials and re-auths on `401`.
4. Build one `homeassistant.Config` per tracked device; `mqtt.Connect` with the LWT set
   to the service availability topic.
5. Register the on-connection-up callback (republish availability + discovery + last
   state); publish discovery + `online` once initially.
6. Start the scheduler and block on `ctx` (SIGTERM/SIGINT).

Graceful shutdown (REQ-LC-05): publish retained `offline`, `Disconnect()` the broker
cleanly, stop the scheduler, shut the health server down.

`cmd/main.go` (REQ-LC-08) maps a returned error to stderr + a non-zero exit code.

Logging (REQ-LC-04) is structured `log/slog`; `LOG_LEVEL`/`LOG_FORMAT` configurable;
the config `String()` redacts secrets so credentials never reach the logs.
