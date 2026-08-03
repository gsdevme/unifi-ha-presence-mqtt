# 02 — Domain model (presence)

`internal/presence` is the pure domain that turns a snapshot of active UniFi clients
into per-device presence, applying the asymmetric debounce. It has no I/O and is fully
unit-tested.

## Types

```go
type Device struct {
    Hostname   string
    IPAddress  string
    MACAddress string
    Present    bool
}
```

`Device` is the state published to HA for one tracked hostname (REQ-DM-01).

```go
type Tracker struct { ... } // per-hostname Device + consecutive-miss counter
func NewTracker(hostnames []string, failureThreshold int) *Tracker
func (t *Tracker) Apply(active []unifi.ClientResponse) []Device
func (t *Tracker) Devices() []Device
```

`NewTracker` seeds one entry per tracked hostname, each starting **absent**
(`Present=false`, empty ip/mac) so HA entities exist before the first sighting
(REQ-DM-02, REQ-DM-05). `failureThreshold` below 1 is treated as 1.

## The debounce (REQ-DM-03, REQ-DM-04)

`Apply(active)` is called once per poll with the controller's current active clients:

1. Build the set of present hostnames from `active` (matched by `Hostname`; blank
   hostnames are ignored — the controller reports unnamed clients).
2. For each tracked device:
   - **Sighting** (hostname present): set `Present=true`, refresh `IPAddress`/
     `MACAddress` from the matching client, reset the miss counter to 0. This is
     **immediate** — a device becomes `home` on the first poll that sees it.
   - **Miss** (hostname absent): increment the miss counter. Only when it reaches
     `failureThreshold` does `Present` flip to `false` (and the counter is capped so it
     doesn't overflow). The last-known ip/mac are **retained** while absent so the
     attributes don't blank out during a brief disconnect.
3. Return the full slice of `Device` for every tracked hostname, in the tracker's
   stable (seed) order (REQ-DM-06).

This asymmetry — instant `home`, delayed `not_home` — is the flapping-smoothing
behaviour preserved from the original daemon.

## Files

`presence.go` (the tracker + debounce), `presence_test.go` (table tests: immediate
present, retained-across-miss, flip only at threshold, counter reset on re-sighting,
ip/mac refresh, unknown/blank hostnames ignored).
