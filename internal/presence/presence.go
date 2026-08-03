// Package presence turns a snapshot of active UniFi clients into per-device
// presence, applying the asymmetric debounce: a device is present immediately on
// sighting, but only absent after FailureThreshold consecutive missed polls. It is
// pure (no I/O). See docs/specs/02-domain-model.md.
package presence

import "github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"

// Device is the presence state published to Home Assistant for one tracked
// hostname.
type Device struct {
	Hostname   string
	IPAddress  string
	MACAddress string
	Present    bool
}

// Tracker holds, per tracked hostname, its Device plus a consecutive-miss counter.
type Tracker struct {
	failureThreshold int
	order            []string
	devices          map[string]*Device
	misses           map[string]int
}

// NewTracker seeds one absent Device per tracked hostname so HA entities exist
// before the first sighting. A failureThreshold below 1 is treated as 1.
func NewTracker(hostnames []string, failureThreshold int) *Tracker {
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	t := &Tracker{
		failureThreshold: failureThreshold,
		order:            make([]string, 0, len(hostnames)),
		devices:          make(map[string]*Device, len(hostnames)),
		misses:           make(map[string]int, len(hostnames)),
	}
	for _, h := range hostnames {
		if _, seen := t.devices[h]; seen {
			continue
		}
		t.order = append(t.order, h)
		t.devices[h] = &Device{Hostname: h}
	}
	return t
}

// Apply updates presence from the controller's current active clients and returns
// the full per-device snapshot in seed order. On a sighting a device becomes
// present immediately (ip/mac refreshed, miss counter reset); on a miss the counter
// increments and presence flips to false only once it reaches the failure
// threshold, with the last-known ip/mac retained while absent.
func (t *Tracker) Apply(active []unifi.ClientResponse) []Device {
	present := make(map[string]unifi.ClientResponse, len(active))
	for _, c := range active {
		if c.Hostname == "" {
			continue // the controller reports unnamed clients; ignore them
		}
		if _, tracked := t.devices[c.Hostname]; tracked {
			present[c.Hostname] = c
		}
	}

	for host, dev := range t.devices {
		if c, ok := present[host]; ok {
			dev.Present = true
			dev.IPAddress = c.IPAddress
			dev.MACAddress = c.MACAddress
			t.misses[host] = 0
			continue
		}
		if t.misses[host] < t.failureThreshold {
			t.misses[host]++
		}
		if t.misses[host] >= t.failureThreshold {
			dev.Present = false // ip/mac retained
		}
	}
	return t.Devices()
}

// Devices returns the current per-device snapshot in seed order.
func (t *Tracker) Devices() []Device {
	out := make([]Device, 0, len(t.order))
	for _, h := range t.order {
		out = append(out, *t.devices[h])
	}
	return out
}
