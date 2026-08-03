package presence

import (
	"testing"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

func client(hostname, ip, mac string) unifi.ClientResponse {
	return unifi.ClientResponse{Hostname: hostname, IPAddress: ip, MACAddress: mac}
}

// byHost indexes a device snapshot by hostname.
func byHost(devices []Device) map[string]Device {
	m := make(map[string]Device, len(devices))
	for _, d := range devices {
		m[d.Hostname] = d
	}
	return m
}

func TestSeededDevicesStartAbsent(t *testing.T) {
	tr := NewTracker([]string{"phone1", "phone2"}, 5)
	devices := tr.Devices()
	if len(devices) != 2 {
		t.Fatalf("got %d devices, want 2", len(devices))
	}
	for _, d := range devices {
		if d.Present {
			t.Errorf("%s should start absent", d.Hostname)
		}
		if d.IPAddress != "" || d.MACAddress != "" {
			t.Errorf("%s should start with empty ip/mac, got %+v", d.Hostname, d)
		}
	}
}

func TestImmediatePresentOnSighting(t *testing.T) {
	tr := NewTracker([]string{"phone1", "phone2"}, 5)
	got := byHost(tr.Apply([]unifi.ClientResponse{client("phone1", "192.168.1.20", "aa:bb")}))

	if !got["phone1"].Present {
		t.Error("phone1 should be present immediately on first sighting")
	}
	if got["phone1"].IPAddress != "192.168.1.20" || got["phone1"].MACAddress != "aa:bb" {
		t.Errorf("phone1 ip/mac not set: %+v", got["phone1"])
	}
	if got["phone2"].Present {
		t.Error("phone2 was never seen; should be absent")
	}
}

func TestMissBelowThresholdRetainsPresence(t *testing.T) {
	tr := NewTracker([]string{"phone1"}, 3)
	tr.Apply([]unifi.ClientResponse{client("phone1", "10.0.0.5", "mac1")}) // present
	// Two consecutive misses (below the threshold of 3): stays present, ip/mac retained.
	for i := 0; i < 2; i++ {
		got := byHost(tr.Apply(nil))
		if !got["phone1"].Present {
			t.Fatalf("miss %d: phone1 flipped absent below threshold", i+1)
		}
		if got["phone1"].IPAddress != "10.0.0.5" || got["phone1"].MACAddress != "mac1" {
			t.Errorf("miss %d: ip/mac not retained: %+v", i+1, got["phone1"])
		}
	}
}

func TestFlipsAbsentExactlyAtThreshold(t *testing.T) {
	tr := NewTracker([]string{"phone1"}, 3)
	tr.Apply([]unifi.ClientResponse{client("phone1", "10.0.0.5", "mac1")})

	if got := byHost(tr.Apply(nil)); !got["phone1"].Present { // miss 1
		t.Fatal("still present after 1 miss")
	}
	if got := byHost(tr.Apply(nil)); !got["phone1"].Present { // miss 2
		t.Fatal("still present after 2 misses")
	}
	got := byHost(tr.Apply(nil)) // miss 3 == threshold
	if got["phone1"].Present {
		t.Error("phone1 should be absent at the 3rd consecutive miss")
	}
	// ip/mac are retained even while absent.
	if got["phone1"].IPAddress != "10.0.0.5" {
		t.Errorf("ip should be retained while absent, got %+v", got["phone1"])
	}
}

func TestResightingResetsCounter(t *testing.T) {
	tr := NewTracker([]string{"phone1"}, 3)
	tr.Apply([]unifi.ClientResponse{client("phone1", "10.0.0.5", "mac1")})
	tr.Apply(nil) // miss 1
	tr.Apply(nil) // miss 2 (one short of flipping)
	// Re-sighting resets the counter, so subsequent misses never accumulate to a flip
	// from the earlier run — this is the flapping-smoothing guarantee.
	tr.Apply([]unifi.ClientResponse{client("phone1", "10.0.0.9", "mac1")})
	if got := byHost(tr.Apply(nil)); !got["phone1"].Present { // miss 1 again
		t.Fatal("counter did not reset on re-sighting")
	}
	if got := byHost(tr.Apply(nil)); !got["phone1"].Present { // miss 2
		t.Fatal("flipped too early after reset")
	}
	if got := byHost(tr.Apply(nil)); got["phone1"].Present { // miss 3
		t.Error("should flip absent 3 misses after the re-sighting")
	}
}

func TestReSightingRefreshesIPMAC(t *testing.T) {
	tr := NewTracker([]string{"phone1"}, 5)
	tr.Apply([]unifi.ClientResponse{client("phone1", "10.0.0.5", "mac-old")})
	got := byHost(tr.Apply([]unifi.ClientResponse{client("phone1", "10.0.0.99", "mac-new")}))
	if got["phone1"].IPAddress != "10.0.0.99" || got["phone1"].MACAddress != "mac-new" {
		t.Errorf("ip/mac not refreshed on re-sighting: %+v", got["phone1"])
	}
}

func TestUnknownAndBlankHostnamesIgnored(t *testing.T) {
	tr := NewTracker([]string{"phone1"}, 5)
	got := byHost(tr.Apply([]unifi.ClientResponse{
		client("", "10.0.0.1", "blank"),       // unnamed controller client
		client("laptop", "10.0.0.2", "other"), // not tracked
	}))
	if got["phone1"].Present {
		t.Error("phone1 should be absent; only blank/untracked clients were active")
	}
	if len(got) != 1 {
		t.Errorf("snapshot should only contain tracked devices, got %v", got)
	}
}

func TestThresholdBelowOneTreatedAsOne(t *testing.T) {
	tr := NewTracker([]string{"phone1"}, 0)
	tr.Apply([]unifi.ClientResponse{client("phone1", "10.0.0.5", "mac1")})
	if got := byHost(tr.Apply(nil)); got["phone1"].Present {
		t.Error("with threshold clamped to 1, a single miss should flip absent")
	}
}

func TestApplyReturnsStableOrder(t *testing.T) {
	tr := NewTracker([]string{"c", "a", "b"}, 5)
	devices := tr.Apply(nil)
	order := []string{devices[0].Hostname, devices[1].Hostname, devices[2].Hostname}
	want := []string{"c", "a", "b"} // seed order preserved, not sorted
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v (seed order)", order, want)
		}
	}
}
