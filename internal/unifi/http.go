package unifi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	userAgent = "golang-unifi"
	// AuthCookieName is the cookie the controller sets on login; its value is the
	// session token sent back on subsequent requests.
	AuthCookieName = "TOKEN"
)

// ClientFunc configures a client at construction time.
type ClientFunc func(c *httpClient)

// WithCredentials configures the client to log in with the given credentials the
// first time a token is needed.
func WithCredentials(username, password string) ClientFunc {
	return func(c *httpClient) {
		c.username = username
		c.password = password
	}
}

// WithAuthToken configures the client to reuse an existing session token,
// short-circuiting the login flow.
func WithAuthToken(token string) ClientFunc {
	return func(c *httpClient) {
		c.authToken = token
	}
}

type httpClient struct {
	baseURL   string
	authToken string
	username  string
	password  string
	http      *http.Client
}

// NewHTTPClient builds a UniFi client for the controller at baseURL (e.g.
// "https://192.168.1.1"), authenticating via the provided option. The transport
// skips TLS verification because UniFi OS ships a self-signed certificate.
func NewHTTPClient(baseURL string, auth ClientFunc) Client {
	c := &httpClient{
		baseURL: baseURL,
		http: &http.Client{
			Transport: insecureTransport(),
			Timeout:   10 * time.Second,
		},
	}
	auth(c)
	return c
}

func (c *httpClient) GetAuthToken() (string, error) {
	if c.authToken != "" {
		return c.authToken, nil
	}

	body, err := json.Marshal(&authRequestBody{
		Username: c.username,
		Password: c.password,
		Remember: true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal auth request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, makeURL(c.baseURL, authLoginURI), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("authentication failed to %s: %w", c.baseURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusUnauthorized {
			return "", &HttpAuthError{msg: "unauthorized: invalid credentials"}
		}
		return "", fmt.Errorf("login: expected 200, got %d", res.StatusCode)
	}

	for _, cookie := range res.Cookies() {
		if cookie.Name == AuthCookieName {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("login succeeded but no %s cookie was returned", AuthCookieName)
}

func (c *httpClient) GetActiveClients(siteID string) ([]ClientResponse, error) {
	token, err := c.GetAuthToken()
	if err != nil {
		return nil, fmt.Errorf("get active clients: %w", err)
	}

	requestURL := makeURL(c.baseURL, fmt.Sprintf(activeClientsURI, siteID))
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create active-clients request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: token})
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get active clients from %s: %w", requestURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusUnauthorized {
			return nil, &HttpAuthError{msg: "unauthorized: session expired"}
		}
		return nil, fmt.Errorf("active clients: expected 200, got %d", res.StatusCode)
	}

	var decoded activeClientsJSON
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode active-clients response: %w", err)
	}

	clients := make([]ClientResponse, 0, len(decoded.Data))
	for _, d := range decoded.Data {
		clients = append(clients, ClientResponse{
			SiteID:     d.SiteID,
			IPAddress:  d.IP,
			MACAddress: d.Mac,
			DeviceName: d.DeviceName,
			Hostname:   d.Hostname,
			Name:       d.Name,
			LastSeen:   time.Unix(d.LastSeen, 0),
		})
	}
	return clients, nil
}

// insecureTransport clones the default transport and disables TLS verification
// (UniFi OS uses a self-signed certificate). Cloning avoids mutating the shared
// http.DefaultTransport.
func insecureTransport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // UDM self-signed cert
	return tr
}
