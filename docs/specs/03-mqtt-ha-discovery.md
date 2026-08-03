# 03 — MQTT & Home Assistant discovery

`internal/homeassistant` is a **pure payload builder**; `internal/mqtt` is the autopaho
MQTT5 transport; `internal/publisher` glues them together. The
`home-assistant-mqtt-discovery` skill is the source of truth for the discovery rules.

## Config (one per tracked device)

```go
type Config struct {
    DiscoveryPrefix string // e.g. "homeassistant"
    TopicPrefix     string // e.g. "unifi"
    Host            string // tracked hostname, e.g. "phone1"
    Name            string // HA device/entity name (defaults to Host)
}
```

The publisher builds one `Config` per tracked device and calls the builder per device.

## Topics

- `BaseTopic = <TopicPrefix>/<sanitizedHost>` — the `~` abbreviation. `sanitizedHost`
  lower-cases the hostname and replaces any character outside `[A-Za-z0-9_]` with `_`.
- `StateTopic = ~/state` — a single **retained** JSON document per device:
  `{"present": bool, "ip_address": string, "mac_address": string}` (REQ-HA-02).
- `AvailabilityTopic = <TopicPrefix>/availability` — **service-wide**, absolute (not a
  `~` child), shared by every device (REQ-HA-06).
- Discovery: `homeassistant/<component>/unifi_<sanitizedHost>_<key>/config`, retained
  (REQ-HA-01).

## Entities (REQ-HA-04)

Both entities read the shared `~/state` document via `value_template`.

| Component | key | Distinctive fields |
| --- | --- | --- |
| `device_tracker` | `presence` | `state_topic ~/state`, `value_template {{ 'home' if value_json.present else 'not_home' }}`, `source_type router`, `json_attributes_topic ~/state` |
| `binary_sensor` | `connectivity` | `device_class connectivity`, `state_topic ~/state`, `value_template {{ 'ON' if value_json.present else 'OFF' }}` |

Unlike a GPS `device_tracker`, the UniFi tracker emits `home`/`not_home` **directly**:
the controller genuinely knows whether the device is on the network, so there is no GPS
zone resolution to defer to HA (REQ-HA-05). `json_attributes_topic` surfaces
`ip_address`/`mac_address` as entity attributes.

## Shared device block (REQ-HA-03)

```json
{ "identifiers": ["unifi_<sanitizedHost>"], "manufacturer": "Ubiquiti", "name": "<hostname>" }
```

Every entity payload also carries `~`, `unique_id`/`object_id` = `unifi_<host>_<key>`,
and the availability triple:

```json
{ "availability_topic": "<TopicPrefix>/availability",
  "payload_available": "online", "payload_not_available": "offline" }
```

The availability topic is absolute (service-wide), so it is **not** written as `~/...`.

## Transport & availability (REQ-HA-06, REQ-HA-07, REQ-HA-08, REQ-HA-09)

- autopaho MQTT5. LWT: retained `offline` on `<TopicPrefix>/availability`, QoS 1.
- `online` published (retained) on connect; `offline` published explicitly on graceful
  shutdown.
- All discovery/state/availability publishes are retained at QoS 1.
- On every (re)connection the client republishes availability + discovery + the latest
  per-device state, so a broker restart re-primes HA.

## Files

`homeassistant/discovery.go` (topics, `BuildDiscovery`, `BuildState`),
`homeassistant/entities.go` (the two-entity catalogue), `homeassistant/discovery_test.go`;
`mqtt/client.go`; `publisher/publisher.go`, `publisher/recording.go`,
`publisher/publisher_test.go`.
