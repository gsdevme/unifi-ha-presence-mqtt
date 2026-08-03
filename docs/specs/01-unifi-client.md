# 01 — UniFi client

`internal/unifi` is a self-contained port of the two endpoints this service needs from
a UniFi Network controller running on UniFi OS (UDM / UDM-Pro / Cloud Key Gen2+). It is
read-only.

## Construction

```go
c := unifi.NewHTTPClient(host, unifi.WithCredentials(user, pass)) // login flow
c := unifi.NewHTTPClient(host, unifi.WithAuthToken(token))        // reuse a token
```

- `host` is the controller base URL, e.g. `https://192.168.1.1` (no trailing slash).
- Exactly one auth option is applied. `WithAuthToken` short-circuits login:
  `GetAuthToken()` returns the supplied token without a network call.
- The HTTP transport sets `InsecureSkipVerify: true` because UniFi OS ships a
  self-signed certificate, and a 10s request timeout. (REQ-UF-04)

## Endpoints

### `GetAuthToken() (string, error)` — `POST api/auth/login` (REQ-UF-02)

- Body: `{"username","password","remember":true}`, `Content-Type: application/json`.
- On `200`, the controller sets a `TOKEN` cookie; its value is the bearer token
  returned. If the cookie is absent the call errors.
- On `401`, returns `*HttpAuthError` (REQ-UF-05). Any other non-200 is a generic error.
- When the client was built `WithAuthToken`, returns that token immediately.

### `GetActiveClients(site) ([]ClientResponse, error)` — `GET proxy/network/api/s/<site>/stat/sta` (REQ-UF-03)

- Authenticated by sending the `TOKEN` cookie (value from `GetAuthToken`, so a
  credentials client logs in lazily on first use).
- On `200`, decodes the controller's `stat/sta` payload (`{meta, data:[...]}`) into
  `[]ClientResponse`, keeping only the fields the presence domain needs (REQ-UF-06).
- On `401`, returns `*HttpAuthError` so the scheduler can re-authenticate (REQ-UF-05).

## Types

`ClientResponse` (REQ-UF-06): `Hostname, IPAddress, MACAddress, DeviceName, Name,
SiteID string` and `LastSeen time.Time` (Unix-seconds `last_seen` decoded to a time).
The controller's full `stat/sta` element has ~90 fields; only the identity subset is
mapped (`json.go`).

`*HttpAuthError` (REQ-UF-05) is a distinct error type returned on HTTP 401 so callers
do `errors.As(err, &authErr)` to drive transparent re-authentication.

## Files

`client.go` (interface, URL helper, request bodies), `http.go` (transport, the two
calls, options), `model.go` (`ClientResponse`), `json.go` (the `stat/sta` decode
struct), `errors.go` (`HttpAuthError`), `client_test.go` (httptest coverage of the
login-cookie extraction, active-clients decode, and the 401 → `HttpAuthError` path).
