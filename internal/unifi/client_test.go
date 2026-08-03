package unifi_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/unifi"
)

const testAPIKey = "secret-api-key"

// requireAPIKey writes 401 unless the request carries the expected X-API-KEY.
func requireAPIKey(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-API-KEY") != testAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func TestListSites(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /proxy/network/integration/v1/sites", func(w http.ResponseWriter, r *http.Request) {
		if !requireAPIKey(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"offset":0,"limit":200,"count":1,"totalCount":1,` +
			`"data":[{"id":"11111111-2222-3333-4444-555555555555","name":"Default"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := unifi.NewHTTPClient(srv.URL, testAPIKey)
	sites, err := c.ListSites()
	if err != nil {
		t.Fatalf("ListSites: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("got %d sites, want 1", len(sites))
	}
	if sites[0].ID != "11111111-2222-3333-4444-555555555555" || sites[0].Name != "Default" {
		t.Errorf("site = %+v", sites[0])
	}
}

func TestGetActiveClients(t *testing.T) {
	t.Run("sends X-API-KEY and decodes clients across pages", func(t *testing.T) {
		// One client per page so the client must follow the pagination envelope.
		clients := []string{
			`{"id":"c1","name":"Phone One","hostname":"phone1","ipAddress":"192.168.1.20",` +
				`"macAddress":"aa:bb:cc:dd:ee:ff","type":"WIRELESS","connectedAt":"2024-01-02T03:04:05Z"}`,
			`{"id":"c2","name":"laptop","ipAddress":"192.168.1.21",` +
				`"macAddress":"11:22:33:44:55:66","type":"WIRED","connectedAt":"2024-01-02T03:04:06Z"}`,
		}
		var gotKey string
		mux := http.NewServeMux()
		mux.HandleFunc("GET /proxy/network/integration/v1/sites/{site}/clients", func(w http.ResponseWriter, r *http.Request) {
			gotKey = r.Header.Get("X-API-KEY")
			if !requireAPIKey(w, r) {
				return
			}
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			w.Header().Set("Content-Type", "application/json")
			if offset >= len(clients) {
				_, _ = fmt.Fprintf(w, `{"offset":%d,"limit":1,"count":0,"totalCount":%d,"data":[]}`, offset, len(clients))
				return
			}
			_, _ = fmt.Fprintf(w, `{"offset":%d,"limit":1,"count":1,"totalCount":%d,"data":[%s]}`, offset, len(clients), clients[offset])
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		c := unifi.NewHTTPClient(srv.URL, testAPIKey)
		got, err := c.GetActiveClients("site-uuid")
		if err != nil {
			t.Fatalf("GetActiveClients: %v", err)
		}
		if gotKey != testAPIKey {
			t.Errorf("X-API-KEY header = %q, want %q", gotKey, testAPIKey)
		}
		if len(got) != 2 {
			t.Fatalf("got %d clients, want 2 (paginated)", len(got))
		}
		c0 := got[0]
		if c0.Hostname != "phone1" || c0.Name != "Phone One" || c0.IPAddress != "192.168.1.20" || c0.MACAddress != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("client[0] = %+v", c0)
		}
		if c0.LastSeen.IsZero() {
			t.Errorf("client[0].LastSeen not parsed from connectedAt")
		}
		// The wired sub-schema omits hostname, so it falls back to name.
		if got[1].Hostname != "laptop" {
			t.Errorf("client[1].Hostname = %q, want laptop (fallback to name)", got[1].Hostname)
		}
	})

	t.Run("401 returns *AuthError", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /proxy/network/integration/v1/sites/{site}/clients", func(w http.ResponseWriter, r *http.Request) {
			if !requireAPIKey(w, r) {
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)

		c := unifi.NewHTTPClient(srv.URL, "wrong-key")
		_, err := c.GetActiveClients("site-uuid")
		var authErr *unifi.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("err = %v, want *AuthError", err)
		}
	})
}
