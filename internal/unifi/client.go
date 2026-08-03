// Package unifi is a self-contained, read-only client for the two UniFi Network
// controller (UniFi OS / UDM) endpoints this service needs: login and active
// clients. It is a port of github.com/gsdevme/unifi/pkg/unifi so the repo carries
// no external UniFi dependency. See docs/specs/01-unifi-client.md.
package unifi

import "fmt"

// Controller endpoint paths (relative to the controller base URL).
const (
	authLoginURI     = "api/auth/login"
	activeClientsURI = "proxy/network/api/s/%s/stat/sta"
)

// authRequestBody is the JSON body posted to the login endpoint.
type authRequestBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember,omitempty"`
}

// Client is the read-only UniFi controller surface used by the service. On an
// expired session both methods return *HttpAuthError so callers can re-auth via
// errors.As.
type Client interface {
	// GetAuthToken returns the controller session token (the TOKEN cookie value),
	// logging in with credentials on first use, or the token supplied via
	// WithAuthToken.
	GetAuthToken() (string, error)
	// GetActiveClients returns the currently active clients for the given site.
	GetActiveClients(siteID string) ([]ClientResponse, error)
}

// makeURL joins the controller base URL and a relative path.
func makeURL(base, uri string) string {
	return fmt.Sprintf("%s/%s", base, uri)
}
