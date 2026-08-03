package mock_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/mock"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

func TestMockLoginAndActiveClients(t *testing.T) {
	m := mock.New(mock.Defaults())
	srv := httptest.NewServer(m.Handler())
	defer srv.Close()

	c := unifi.NewHTTPClient(srv.URL, unifi.WithCredentials("admin", "pass"))
	clients, err := c.GetActiveClients("default")
	if err != nil {
		t.Fatalf("GetActiveClients: %v", err)
	}
	if len(clients) != 2 || clients[0].Hostname != "phone1" || clients[0].IPAddress != "192.168.1.20" {
		t.Fatalf("clients = %+v", clients)
	}
	if login, sta := m.Counts(); login != 1 || sta != 1 {
		t.Fatalf("counts login=%d sta=%d, want 1/1", login, sta)
	}
}

func TestMockSetClients(t *testing.T) {
	m := mock.New(mock.Defaults())
	srv := httptest.NewServer(m.Handler())
	defer srv.Close()
	m.SetClients([]mock.Client{{Hostname: "laptop", IPAddress: "10.0.0.9", MACAddress: "ff:ee"}})

	c := unifi.NewHTTPClient(srv.URL, unifi.WithCredentials("admin", "pass"))
	clients, err := c.GetActiveClients("default")
	if err != nil {
		t.Fatalf("GetActiveClients: %v", err)
	}
	if len(clients) != 1 || clients[0].Hostname != "laptop" {
		t.Fatalf("clients = %+v, want single laptop", clients)
	}
}

func TestMockExpireSessionForcesReauth(t *testing.T) {
	m := mock.New(mock.Defaults())
	srv := httptest.NewServer(m.Handler())
	defer srv.Close()

	// Log in once and reuse the token, like the provider does.
	loginClient := unifi.NewHTTPClient(srv.URL, unifi.WithCredentials("admin", "pass"))
	tok, err := loginClient.GetAuthToken()
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	tokenClient := unifi.NewHTTPClient(srv.URL, unifi.WithAuthToken(tok))
	if _, err := tokenClient.GetActiveClients("default"); err != nil {
		t.Fatalf("first poll should succeed: %v", err)
	}

	// Expiring the session makes the cached token stale -> 401.
	m.ExpireSession()
	_, err = tokenClient.GetActiveClients("default")
	var authErr *unifi.HttpAuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("stale token err = %v, want *HttpAuthError", err)
	}

	// Re-authenticating yields the new token, which works again.
	tok2, err := unifi.NewHTTPClient(srv.URL, unifi.WithCredentials("admin", "pass")).GetAuthToken()
	if err != nil {
		t.Fatalf("reauth login: %v", err)
	}
	if _, err := unifi.NewHTTPClient(srv.URL, unifi.WithAuthToken(tok2)).GetActiveClients("default"); err != nil {
		t.Fatalf("poll after reauth should succeed: %v", err)
	}
}
