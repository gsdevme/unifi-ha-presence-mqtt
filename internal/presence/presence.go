// Package presence turns a snapshot of connected UniFi clients into per-device
// presence, applying the asymmetric debounce: a device is present immediately on
// sighting, but only absent after FailureThreshold consecutive missed polls. It is
// pure (no I/O). See docs/specs/02-domain-model.md.
package presence

import (
	"strings"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

// Device is the presence state published to Home Assistant for one tracked device.
// Hostname is the stable identity (MQTT topic + HA object id + display name).
type Device struct {
	Hostname   string
	IPAddress  string
	MACAddress string
	Present    bool
}

// Tracker holds, per tracked device, its Device, the matcher pattern and a
// consecutive-miss counter.
type Tracker struct {
	failureThreshold int
	order            []string
	devices          map[string]*Device
	patterns         map[string]string
	misses           map[string]int
}

// NewTracker seeds one absent Device per tracked entry so HA entities exist before
// the first sighting. Each entry is `Name` or `Name=Pattern`: `Name` is the stable
// identity, `Pattern` is matched (case-insensitively, as a substring) against a
// connected client's name or MAC; when omitted, `Name` itself is the pattern. A
// failureThreshold below 1 is treated as 1.
func NewTracker(entries []string, failureThreshold int) *Tracker {
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	t := &Tracker{
		failureThreshold: failureThreshold,
		order:            make([]string, 0, len(entries)),
		devices:          make(map[string]*Device, len(entries)),
		patterns:         make(map[string]string, len(entries)),
		misses:           make(map[string]int, len(entries)),
	}
	for _, e := range entries {
		name, pattern := parseEntry(e)
		if name == "" {
			continue
		}
		if _, seen := t.devices[name]; seen {
			continue
		}
		t.order = append(t.order, name)
		t.devices[name] = &Device{Hostname: name}
		t.patterns[name] = pattern
	}
	return t
}

// Apply updates presence from the console's currently connected clients and returns
// the full per-device snapshot in seed order. On a sighting a device becomes present
// immediately (ip/mac refreshed, miss counter reset); on a miss the counter
// increments and presence flips to false only once it reaches the failure threshold,
// with the last-known ip/mac retained while absent.
func (t *Tracker) Apply(active []unifi.ClientResponse) []Device {
	for _, id := range t.order {
		dev := t.devices[id]
		if c, ok := matchActive(t.patterns[id], active); ok {
			dev.Present = true
			dev.IPAddress = c.IPAddress
			dev.MACAddress = c.MACAddress
			t.misses[id] = 0
			continue
		}
		if t.misses[id] < t.failureThreshold {
			t.misses[id]++
		}
		if t.misses[id] >= t.failureThreshold {
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

// parseEntry splits a `Name` or `Name=Pattern` tracked entry. Whitespace is trimmed;
// an empty pattern (trailing `=`) falls back to the name.
func parseEntry(e string) (name, pattern string) {
	e = strings.TrimSpace(e)
	if i := strings.Index(e, "="); i >= 0 {
		name = strings.TrimSpace(e[:i])
		pattern = strings.TrimSpace(e[i+1:])
		if pattern == "" {
			pattern = name
		}
		return name, pattern
	}
	return e, e
}

// matchActive returns the first connected client the pattern matches.
func matchActive(pattern string, active []unifi.ClientResponse) (unifi.ClientResponse, bool) {
	for _, c := range active {
		if clientMatches(pattern, c) {
			return c, true
		}
	}
	return unifi.ClientResponse{}, false
}

// clientMatches reports whether pattern matches a connected client. A pattern matches
// if it is a case-insensitive substring of the client's name/hostname, or — when it
// looks like a (partial) MAC — a substring of the client's MAC with separators
// ignored. The substring behaviour tolerates the MAC-octet suffix UniFi appends to
// names (e.g. "iPhone-X" matches "iPhone-X fd:81") and partial MACs (e.g. "fd:81").
func clientMatches(pattern string, c unifi.ClientResponse) bool {
	p := strings.ToLower(strings.TrimSpace(pattern))
	if p == "" {
		return false
	}
	if strings.Contains(strings.ToLower(c.Hostname), p) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Name), p) {
		return true
	}
	if mp := macPattern(pattern); mp != "" && strings.Contains(macPattern(c.MACAddress), mp) {
		return true
	}
	return false
}

// macPattern normalises a MAC (or partial MAC) to its lower-case hex digits, or ""
// if the input is not MAC-like (fewer than 4 hex digits after stripping separators,
// or containing a non-hex character). Requiring all-hex avoids a name such as
// "iPhone-X" being treated as a MAC fragment.
func macPattern(s string) string {
	s = strings.ToLower(strings.NewReplacer(":", "", "-", "", ".", "", " ", "").Replace(s))
	if len(s) < 4 {
		return ""
	}
	for _, r := range s {
		isDigit := r >= '0' && r <= '9'
		isHexLetter := r >= 'a' && r <= 'f'
		if !isDigit && !isHexLetter {
			return ""
		}
	}
	return s
}
