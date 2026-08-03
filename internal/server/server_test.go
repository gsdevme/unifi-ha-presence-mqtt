package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadinessLifecycle(t *testing.T) {
	s := New(Config{ReadyFailureThreshold: 3})
	if s.Ready() {
		t.Fatal("should start not ready")
	}
	s.MarkSuccess()
	if !s.Ready() {
		t.Fatal("should be ready after first success")
	}
	s.MarkFailure()
	s.MarkFailure()
	if !s.Ready() {
		t.Fatal("should stay ready below threshold")
	}
	s.MarkFailure() // 3rd consecutive
	if s.Ready() {
		t.Fatal("should be not ready at threshold")
	}
	s.MarkSuccess()
	if !s.Ready() {
		t.Fatal("should recover on success")
	}
}

func TestProbes(t *testing.T) {
	s := New(Config{ReadyFailureThreshold: 1})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %v / %v", resp, err)
	}
	resp, _ = http.Get(srv.URL + "/readyz")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("readyz before success = %d, want 503", resp.StatusCode)
	}
	s.MarkSuccess()
	resp, _ = http.Get(srv.URL + "/readyz")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("readyz after success = %d, want 200", resp.StatusCode)
	}
}

func TestRootStatusPage(t *testing.T) {
	s := New(Config{ReadyFailureThreshold: 1})
	s.SetDevices([]DeviceView{
		{Hostname: "phone1", Present: true},
		{Hostname: "phone2", Present: false},
	})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET / error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	page := string(body)
	for _, want := range []string{"not ready", "phone1: present", "phone2: absent"} {
		if !strings.Contains(page, want) {
			t.Errorf("page missing %q:\n%s", want, page)
		}
	}
}

// sensitiveMarkers must never appear on the / page — it shares HEALTH_ADDR with the
// probes and may be reachable by others.
var sensitiveMarkers = []string{"ip_address", "mac_address", "192.168", "aa:bb"}

func TestRootOmitsSensitive(t *testing.T) {
	s := New(Config{ReadyFailureThreshold: 1})
	s.SetDevices([]DeviceView{{Hostname: "phone1", Present: true}})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/")
	body, _ := io.ReadAll(resp.Body)
	page := string(body)
	for _, marker := range sensitiveMarkers {
		if strings.Contains(page, marker) {
			t.Errorf("page leaked sensitive marker %q:\n%s", marker, page)
		}
	}
}

func TestRootBeforeDevices(t *testing.T) {
	s := New(Config{ReadyFailureThreshold: 1})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/")
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "initialising") {
		t.Errorf("page should show initialising before devices are set:\n%s", body)
	}
}

func TestUnknownPath404(t *testing.T) {
	s := New(Config{ReadyFailureThreshold: 1})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatalf("GET /nope error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /nope = %d, want 404", resp.StatusCode)
	}
}
