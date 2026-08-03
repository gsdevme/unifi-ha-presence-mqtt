package unifi

// activeClientsJSON is the decode shape of GET .../stat/sta. The controller
// returns ~90 fields per client; only the identity subset the presence domain
// needs is declared here (unknown fields are ignored by encoding/json).
type activeClientsJSON struct {
	Data []struct {
		SiteID     string `json:"site_id"`
		Mac        string `json:"mac"`
		IP         string `json:"ip"`
		Hostname   string `json:"hostname"`
		Name       string `json:"name"`
		DeviceName string `json:"device_name"`
		LastSeen   int64  `json:"last_seen"`
	} `json:"data"`
}
