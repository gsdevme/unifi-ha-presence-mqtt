# 04 — Polling & scheduling

`internal/scheduler` owns the poll loop that drives presence. It is deliberately simple:
there is no daily force-refresh (unlike the Bluelink sibling) — presence is cheap and
polled frequently.

## Poll loop (REQ-SC-01, REQ-SC-02)

- Immediate first poll, then a `time.Ticker` at `POLL_INTERVAL` (default 10s).
- Polls are **serialised**: a tick never overlaps the previous one (a mutex guards the
  API call), and the loop stops on context cancellation.
- Each tick:
  1. `GetActiveClients(siteID)` against the UniFi Integration API client.
  2. `presence.Apply(active)` → the current per-device state (with the debounce).
  3. `PublishState(device)` for each tracked device (retained JSON).
  4. Report `MarkSuccess`/`MarkFailure` to the health reporter.

## No re-authentication (REQ-SC-03)

The Integration API key is **static** (a header, not a session), so there is nothing
to re-authenticate: the scheduler no longer branches on an auth error. A poll error —
including a `*unifi.AuthError` from a bad key — is treated like any other transient
failure: retried up to `POLL_MAX_RETRIES` with exponential backoff (1s, 2s, …), and a
poll that still fails counts as a failure for readiness. A persistent bad key therefore
simply keeps `/readyz` not-ready (and is logged), rather than triggering a re-login.

## Testability (REQ-SC-04)

`Now func() time.Time` and `After func(time.Duration) <-chan time.Time` are injectable
so `testing/synctest` drives the loop deterministically. `PollNow(ctx)` runs a single
cycle for the acceptance suite; the timed loop calls the same underlying `poll`.

## Interfaces

The scheduler depends on small interfaces, not concrete types:

```go
type ClientProvider interface { // supplies the console's connected clients
    ActiveClients(ctx context.Context) ([]unifi.ClientResponse, error)
}
type StatePublisher interface { PublishState(ctx, []presence.Device) error }
type HealthReporter interface { MarkSuccess(); MarkFailure() }
```

The composition root (`cmd/serve.go`) supplies a `ClientProvider` holding the single
Integration API client and the resolved site UUID.

## Files

`scheduler.go`, `scheduler_test.go` (synctest: immediate publish, ticker cadence,
transient retry absorbed).
