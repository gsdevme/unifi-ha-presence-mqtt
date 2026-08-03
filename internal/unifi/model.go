package unifi

import "time"

// ClientResponse is the identity subset of a UniFi active client the presence
// domain needs. The controller's stat/sta element has ~90 fields; only these are
// mapped (see json.go).
type ClientResponse struct {
	SiteID     string
	IPAddress  string
	MACAddress string
	DeviceName string
	Hostname   string
	Name       string
	LastSeen   time.Time
}
