# 02 — Domain model (presence)

`internal/presence` is the pure domain that turns a snapshot of connected UniFi
clients into per-device presence, applying the asymmetric debounce. It has no I/O and
is fully unit-tested.

## Types

```go
type Device struct {
    Hostname   string
    IPAddress  string
    MACAddress string
    Present    bool
}
```

`Device` is the state published to HA for one tracked device; `Hostname` is its
stable identity — the MQTT topic key, HA object id and display name (REQ-DM-01).

```go
type Tracker struct { ... } // per-device Device + matcher pattern + miss counter
func NewTracker(entries []string, failureThreshold int) *Tracker
func (t *Tracker) Apply(active []unifi.ClientResponse) []Device
func (t *Tracker) Devices() []Device
```

`NewTracker` seeds one entry per tracked device, each starting **absent**
(`Present=false`, empty ip/mac) so HA entities exist before the first sighting
(REQ-DM-02, REQ-DM-05). `failureThreshold` below 1 is treated as 1.

### Tracked entries and matching (REQ-DM-07)

Each entry is `Name` or `Name=Pattern`:

- `Name` is the device's **stable identity** (topic + HA object id + display name),
  chosen by the user — it is not required to match anything the console reports.
- `Pattern` (defaulting to `Name`) is matched against each connected client. A client
  matches when `Pattern` is a **case-insensitive substring** of the client's
  name/hostname, or — when `Pattern` looks like a (partial) MAC (≥4 hex digits,
  separators ignored) — a substring of the client's MAC.

The substring behaviour is deliberate: the Integration API appends a MAC-octet suffix
to names (`iPhone-X` → `iPhone-X fd:81`), so a plain name still matches; a full or
partial MAC (`e8:78:65:00:fd:81`, `fd:81`) matches too. For devices with rotating /
private MACs, match on a stable UniFi name rather than the MAC. Patterns should be
specific enough to be unique — the **first** matching client wins.

## The debounce (REQ-DM-03, REQ-DM-04)

`Apply(active)` is called once per poll with the console's currently connected clients:

1. For each tracked device, find the first connected client its `Pattern` matches (see
   above).
2. Then:
   - **Sighting** (a client matched): set `Present=true`, refresh `IPAddress`/
     `MACAddress` from the matching client, reset the miss counter to 0. This is
     **immediate** — a device becomes `home` on the first poll that sees it.
   - **Miss** (no client matched): increment the miss counter. Only when it reaches
     `failureThreshold` does `Present` flip to `false` (and the counter is capped so it
     doesn't overflow). The last-known ip/mac are **retained** while absent so the
     attributes don't blank out during a brief disconnect.
3. Return the full slice of `Device` for every tracked device, in the tracker's stable
   (seed) order (REQ-DM-06).

This asymmetry — instant `home`, delayed `not_home` — is the flapping-smoothing
behaviour preserved from the original daemon.

## Files

`presence.go` (the tracker + debounce + matching), `presence_test.go` (table tests:
immediate present, retained-across-miss, flip only at threshold, counter reset on
re-sighting, ip/mac refresh, untracked ignored, plus `Name=MAC` matching, partial-MAC
suffix, separator/case-insensitive MAC, name tolerates the console MAC suffix).
