package unifi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

// newController spins up a fake UniFi OS controller. handleLogin serves
// POST api/auth/login; the stat/sta route returns stataBody with stataStatus.
func newController(t *testing.T, handleLogin http.HandlerFunc, stataStatus int, stataBody string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /proxy/network/api/s/{site}/stat/sta", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(stataStatus)
		_, _ = w.Write([]byte(stataBody))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func loginOK(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: unifi.AuthCookieName, Value: token})
		w.WriteHeader(http.StatusOK)
	}
}

func TestGetAuthToken(t *testing.T) {
	t.Run("extracts the TOKEN cookie from login", func(t *testing.T) {
		srv := newController(t, loginOK("session-1337"), http.StatusOK, "")
		c := unifi.NewHTTPClient(srv.URL, unifi.WithCredentials("admin", "pass"))
		got, err := c.GetAuthToken()
		if err != nil {
			t.Fatalf("GetAuthToken: %v", err)
		}
		if got != "session-1337" {
			t.Errorf("token = %q, want session-1337", got)
		}
	})

	t.Run("401 returns *HttpAuthError", func(t *testing.T) {
		srv := newController(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}, http.StatusOK, "")
		c := unifi.NewHTTPClient(srv.URL, unifi.WithCredentials("admin", "wrong"))
		_, err := c.GetAuthToken()
		var authErr *unifi.HttpAuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("err = %v, want *HttpAuthError", err)
		}
	})

	t.Run("missing cookie errors", func(t *testing.T) {
		srv := newController(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}, http.StatusOK, "")
		c := unifi.NewHTTPClient(srv.URL, unifi.WithCredentials("admin", "pass"))
		if _, err := c.GetAuthToken(); err == nil {
			t.Fatal("expected error when no TOKEN cookie is returned")
		}
	})

	t.Run("WithAuthToken short-circuits login", func(t *testing.T) {
		c := unifi.NewHTTPClient("https://example.invalid", unifi.WithAuthToken("preset-token"))
		got, err := c.GetAuthToken()
		if err != nil || got != "preset-token" {
			t.Fatalf("token = %q err = %v, want preset-token", got, err)
		}
	})
}

const staBody = `{
  "meta": {"rc": "ok"},
  "data": [
    {"site_id": "s1", "mac": "aa:bb:cc:dd:ee:ff", "ip": "192.168.1.20",
     "hostname": "phone1", "name": "Phone One", "device_name": "iPhone", "last_seen": 1700000000},
    {"site_id": "s1", "mac": "11:22:33:44:55:66", "ip": "192.168.1.21", "last_seen": 1700000001}
  ]
}`

func TestGetActiveClients(t *testing.T) {
	t.Run("decodes the identity subset", func(t *testing.T) {
		srv := newController(t, loginOK("tok"), http.StatusOK, staBody)
		c := unifi.NewHTTPClient(srv.URL, unifi.WithAuthToken("tok"))
		clients, err := c.GetActiveClients("default")
		if err != nil {
			t.Fatalf("GetActiveClients: %v", err)
		}
		if len(clients) != 2 {
			t.Fatalf("got %d clients, want 2", len(clients))
		}
		c0 := clients[0]
		if c0.Hostname != "phone1" || c0.IPAddress != "192.168.1.20" || c0.MACAddress != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("client[0] = %+v", c0)
		}
		if c0.SiteID != "s1" || c0.DeviceName != "iPhone" || c0.Name != "Phone One" {
			t.Errorf("client[0] identity = %+v", c0)
		}
		if c0.LastSeen.Unix() != 1700000000 {
			t.Errorf("client[0] LastSeen = %v", c0.LastSeen)
		}
		if clients[1].Hostname != "" {
			t.Errorf("client[1] hostname = %q, want empty (unnamed client)", clients[1].Hostname)
		}
	})

	t.Run("401 returns *HttpAuthError to drive re-auth", func(t *testing.T) {
		srv := newController(t, loginOK("tok"), http.StatusUnauthorized, "")
		c := unifi.NewHTTPClient(srv.URL, unifi.WithAuthToken("stale"))
		_, err := c.GetActiveClients("default")
		var authErr *unifi.HttpAuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("err = %v, want *HttpAuthError", err)
		}
	})
}
