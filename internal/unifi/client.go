// Package unifi is a self-contained, read-only client for the official UniFi
// Network Integration API — the supported local integration surface. It talks to
// the two endpoints this service needs (list sites, list connected clients) over
// the console's local /proxy/network/integration/v1 base, authenticated by a
// static X-API-KEY header. There is no login, cookie or session to manage. See
// docs/specs/01-unifi-client.md.
package unifi

import "fmt"

// Integration API endpoint paths (relative to the console base URL).
const (
	integrationSitesURI   = "proxy/network/integration/v1/sites"
	integrationClientsURI = "proxy/network/integration/v1/sites/%s/clients"
)

// Client is the read-only UniFi Network Integration API surface used by the
// service. Both methods authenticate with the configured X-API-KEY; on 401/403
// they return *AuthError. The key is static, so an AuthError is a bad key — not a
// signal to re-authenticate.
type Client interface {
	// ListSites returns the console's sites. UNIFI_SITE (a human name) is resolved
	// to a site UUID against this list at startup.
	ListSites() ([]Site, error)
	// GetActiveClients returns the currently connected clients for the site
	// identified by its UUID.
	GetActiveClients(siteID string) ([]ClientResponse, error)
}

// makeURL joins the console base URL and a relative path (which may carry a query
// string).
func makeURL(base, uri string) string {
	return fmt.Sprintf("%s/%s", base, uri)
}
