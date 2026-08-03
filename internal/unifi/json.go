package unifi

// The Integration API wraps list responses in a pagination envelope
// {offset,limit,count,totalCount,data:[...]}. Only the fields the client needs are
// declared; unknown fields are ignored by encoding/json.

// sitesPage is one page of GET /v1/sites.
type sitesPage struct {
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	Count      int        `json:"count"`
	TotalCount int        `json:"totalCount"`
	Data       []siteJSON `json:"data"`
}

type siteJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// clientsPage is one page of GET /v1/sites/{siteId}/clients (List Connected
// Clients).
type clientsPage struct {
	Offset     int          `json:"offset"`
	Limit      int          `json:"limit"`
	Count      int          `json:"count"`
	TotalCount int          `json:"totalCount"`
	Data       []clientJSON `json:"data"`
}

// clientJSON permissively decodes a connected client's identity fields. The base
// schema carries id/name/ipAddress/type/connectedAt; the wired and wireless
// sub-schemas add hostname/macAddress. Decoding is intentionally permissive so
// that confirming the exact name/hostname key against a real console is a one-line
// tweak, not a rewrite.
type clientJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Hostname    string `json:"hostname"`
	IPAddress   string `json:"ipAddress"`
	MACAddress  string `json:"macAddress"`
	Type        string `json:"type"`
	ConnectedAt string `json:"connectedAt"`
}
