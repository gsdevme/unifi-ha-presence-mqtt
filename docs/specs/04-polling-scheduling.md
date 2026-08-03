# 04 — Polling & scheduling

`internal/scheduler` owns the poll loop that drives presence. It is deliberately simple:
there is no daily force-refresh (unlike the Bluelink sibling) — presence is cheap and
polled frequently.

## Poll loop (REQ-SC-01, REQ-SC-02)

- Immediate first poll, then a `time.Ticker` at `POLL_INTERVAL` (default 10s).
- Polls are **serialised**: a tick never overlaps the previous one (a mutex guards the
  API call), and the loop stops on context cancellation.
- Each tick:
  1. `GetActiveClients(site)` against the UniFi client.
  2. `presence.Apply(active)` → the current per-device state (with the debounce).
  3. `PublishState(device)` for each tracked device (retained JSON).
  4. Report `MarkSuccess`/`MarkFailure` to the health reporter.

## Re-authentication (REQ-SC-03)

The UniFi `TOKEN` expires periodically. When a poll returns `*HttpAuthError`
(`errors.As`), the scheduler re-authenticates with the stored credentials and rebuilds
the client (`WithAuthToken`), then continues — this preserves the token-reuse +
transparent-re-auth behaviour of the original daemon. Other transient errors are retried
up to `POLL_MAX_RETRIES` with exponential backoff (1s, 2s, …); a poll that still fails
counts as a failure for readiness.

## Testability (REQ-SC-04)

`Now func() time.Time` and `After func(time.Duration) <-chan time.Time` are injectable
so `testing/synctest` drives the loop deterministically. `PollNow(ctx)` runs a single
cycle for the acceptance suite; the timed loop calls the same underlying `poll`.

## Interfaces

The scheduler depends on small interfaces, not concrete types:

```go
type ClientProvider interface { // supplies (and re-auths) the UniFi client
    ActiveClients(ctx context.Context) ([]unifi.ClientResponse, error)
}
type StatePublisher interface { PublishState(ctx, []presence.Device) error }
type HealthReporter interface { MarkSuccess(); MarkFailure() }
```

The composition root (`cmd/serve.go`) supplies a `ClientProvider` that owns the
credentials and performs the re-auth-on-401 rebuild.

## Files

`scheduler.go`, `scheduler_test.go` (synctest: immediate publish, ticker cadence,
transient retry absorbed, re-auth on 401 rebuilds and succeeds).
