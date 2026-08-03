package mock_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/mock"
	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

func TestMockSitesAndClients(t *testing.T) {
	m := mock.New(mock.Defaults())
	srv := httptest.NewServer(m.Handler())
	defer srv.Close()

	c := unifi.NewHTTPClient(srv.URL, "any-key")

	sites, err := c.ListSites()
	if err != nil {
		t.Fatalf("ListSites: %v", err)
	}
	if len(sites) != 1 || sites[0].ID != mock.SiteID || sites[0].Name != "default" {
		t.Fatalf("sites = %+v", sites)
	}

	clients, err := c.GetActiveClients(sites[0].ID)
	if err != nil {
		t.Fatalf("GetActiveClients: %v", err)
	}
	if len(clients) != 2 || clients[0].Hostname != "phone1" || clients[0].IPAddress != "192.168.1.20" {
		t.Fatalf("clients = %+v", clients)
	}
	if sitesHits, clientsHits := m.Counts(); sitesHits != 1 || clientsHits != 1 {
		t.Fatalf("counts sites=%d clients=%d, want 1/1", sitesHits, clientsHits)
	}
}

func TestMockSetClients(t *testing.T) {
	m := mock.New(mock.Defaults())
	srv := httptest.NewServer(m.Handler())
	defer srv.Close()
	m.SetClients([]mock.Client{{Hostname: "laptop", IPAddress: "10.0.0.9", MACAddress: "ff:ee"}})

	c := unifi.NewHTTPClient(srv.URL, "any-key")
	clients, err := c.GetActiveClients(mock.SiteID)
	if err != nil {
		t.Fatalf("GetActiveClients: %v", err)
	}
	if len(clients) != 1 || clients[0].Hostname != "laptop" {
		t.Fatalf("clients = %+v, want single laptop", clients)
	}
}

func TestMockRequiresAPIKey(t *testing.T) {
	m := mock.New(mock.Defaults())
	srv := httptest.NewServer(m.Handler())
	defer srv.Close()

	// An empty X-API-KEY is rejected with 401 -> *AuthError.
	c := unifi.NewHTTPClient(srv.URL, "")
	_, err := c.ListSites()
	var authErr *unifi.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %v, want *AuthError for a missing API key", err)
	}
}
