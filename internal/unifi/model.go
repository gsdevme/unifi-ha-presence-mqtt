package unifi

import "time"

// Site is a UniFi site from GET /v1/sites. The Integration API identifies sites by
// a UUID (ID), not the legacy "default" string; UNIFI_SITE (a human name) is
// resolved to this ID at startup.
type Site struct {
	ID   string
	Name string
}

// ClientResponse is the identity subset of a connected client the presence domain
// needs, mapped from the Integration API's "List Connected Clients" schema. The
// full client element has many more fields; only these are decoded (see json.go).
type ClientResponse struct {
	IPAddress  string
	MACAddress string
	Hostname   string
	Name       string
	LastSeen   time.Time
}
