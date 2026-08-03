package unifi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	userAgent = "golang-unifi"
	// apiKeyHeader authenticates every Integration API request; its value is the
	// local API key generated on the console.
	apiKeyHeader = "X-API-KEY"
	// pageLimit is the page size requested when walking the pagination envelope.
	pageLimit = 200
)

type httpClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewHTTPClient builds a UniFi Integration API client for the console at baseURL
// (e.g. "https://192.168.1.1"), authenticating every request with the static
// apiKey via the X-API-KEY header. The transport skips TLS verification because
// UniFi OS ships a self-signed certificate, with a 10s request timeout.
func NewHTTPClient(baseURL, apiKey string) Client {
	return &httpClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http: &http.Client{
			Transport: insecureTransport(),
			Timeout:   10 * time.Second,
		},
	}
}

// ListSites returns every site, walking the pagination envelope.
func (c *httpClient) ListSites() ([]Site, error) {
	var sites []Site
	for offset := 0; ; {
		var page sitesPage
		path := fmt.Sprintf("%s?offset=%d&limit=%d", integrationSitesURI, offset, pageLimit)
		if err := c.getJSON(path, &page); err != nil {
			return nil, fmt.Errorf("list sites: %w", err)
		}
		for _, s := range page.Data {
			sites = append(sites, Site(s))
		}
		offset += page.Count
		if page.Count == 0 || offset >= page.TotalCount {
			return sites, nil
		}
	}
}

// GetActiveClients returns the currently connected clients for siteID, walking the
// pagination envelope. "Connected" already means present, so the presence domain
// treats every returned client as a sighting.
func (c *httpClient) GetActiveClients(siteID string) ([]ClientResponse, error) {
	base := fmt.Sprintf(integrationClientsURI, siteID)
	var clients []ClientResponse
	for offset := 0; ; {
		var page clientsPage
		path := fmt.Sprintf("%s?offset=%d&limit=%d", base, offset, pageLimit)
		if err := c.getJSON(path, &page); err != nil {
			return nil, fmt.Errorf("list connected clients: %w", err)
		}
		for _, d := range page.Data {
			clients = append(clients, ClientResponse{
				IPAddress:  d.IPAddress,
				MACAddress: d.MACAddress,
				Hostname:   firstNonEmpty(d.Hostname, d.Name),
				Name:       d.Name,
				LastSeen:   parseTimestamp(d.ConnectedAt),
			})
		}
		offset += page.Count
		if page.Count == 0 || offset >= page.TotalCount {
			return clients, nil
		}
	}
}

// getJSON issues an authenticated GET for path (relative to the base URL) and
// decodes a 200 JSON body into out. A 401/403 becomes *AuthError; any other
// non-200 is a generic error.
func (c *httpClient) getJSON(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, makeURL(c.baseURL, path), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set(apiKeyHeader, c.apiKey)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s: %w", c.baseURL, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
			return &AuthError{msg: fmt.Sprintf("unauthorized (%d): check UNIFI_API_KEY", res.StatusCode)}
		}
		return fmt.Errorf("expected 200, got %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// firstNonEmpty returns the first non-empty string, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseTimestamp parses an Integration API ISO-8601 timestamp (e.g. connectedAt),
// returning the zero time when it is empty or unparseable.
func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

// insecureTransport clones the default transport and disables TLS verification
// (UniFi OS uses a self-signed certificate). Cloning avoids mutating the shared
// http.DefaultTransport.
func insecureTransport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // UDM self-signed cert
	return tr
}
