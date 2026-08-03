# 01 — UniFi client

`internal/unifi` is a self-contained, read-only client for the official **UniFi
Network Integration API** (the supported local integration surface, v10.4.57) as
exposed by a UniFi OS console (UDM / UDM-Pro / Cloud Key Gen2+). It replaces the
legacy `api/auth/login` → `TOKEN` cookie flow, which is undocumented and breaks with
MFA.

## Construction

```go
c := unifi.NewHTTPClient(host, apiKey) // static X-API-KEY on every request
```

- `host` is the console base URL, e.g. `https://192.168.1.1` (no trailing slash).
- `apiKey` is a **local API key** minted on the console (Settings → Control Plane →
  Integrations → API Keys). It is sent as the `X-API-KEY` header on every request —
  there is no login, cookie, or session to refresh. (REQ-UF-02)
- The HTTP transport sets `InsecureSkipVerify: true` because UniFi OS ships a
  self-signed certificate, and a 10s request timeout. (REQ-UF-04)

## Endpoints

All endpoints live under `proxy/network/integration/v1` and return a **pagination
envelope** `{offset, limit, count, totalCount, data:[…]}`. The client walks pages
(fixed `limit`, advancing `offset` by each page's `count`) until every element is
read. (REQ-UF-08)

### `ListSites() ([]Site, error)` — `GET /v1/sites` (REQ-UF-03)

- Returns the console's sites, each a `Site{ID, Name}` where `ID` is a UUID (not the
  legacy `"default"` string). Used at startup to resolve the `UNIFI_SITE` name.

### `GetActiveClients(siteID) ([]ClientResponse, error)` — `GET /v1/sites/<siteId>/clients` (REQ-UF-03)

- "List Connected Clients": only currently connected clients are returned, so every
  element is a present sighting for the presence domain.
- Decodes the identity subset the domain needs into `[]ClientResponse` (REQ-UF-06);
  unknown fields are ignored.

## Types

`Site` (REQ-UF-03): `ID, Name string` — a site UUID and its human name.

`ClientResponse` (REQ-UF-06): `IPAddress, MACAddress, Hostname, Name string` and
`LastSeen time.Time`. Mapping from the client JSON: `Hostname =
firstNonEmpty(hostname, name)`, `Name = name`, `IPAddress = ipAddress`,
`MACAddress = macAddress`, `LastSeen` parsed from the ISO-8601 `connectedAt`. The
`hostname`/`macAddress` keys live on the wired/wireless sub-schemas; decoding is
permissive so confirming the exact name/hostname key against a real console is a
one-line change.

`*AuthError` (REQ-UF-05) is returned on HTTP **401/403** — a missing or invalid
`X-API-KEY`. Because the key is static there is no re-auth: a persistent `AuthError`
means the key is wrong. It is exported so tests inspect it via `errors.As`.

## Files

`client.go` (interface, endpoint paths, URL helper), `http.go` (transport, the two
paginated calls, the `getJSON` helper), `model.go` (`Site`, `ClientResponse`),
`json.go` (pagination-envelope + client decode structs), `errors.go` (`AuthError`),
`client_test.go` (httptest coverage of the `X-API-KEY` header, sites/clients decode
including a paginated case, and the 401 → `*AuthError` path).
