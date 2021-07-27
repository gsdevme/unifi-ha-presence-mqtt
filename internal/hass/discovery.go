package hass

type DeviceAttributes struct {
	IpAddress string `json:"ip_address"`
	MacAddress string `json:"mac_address"`
}

type deviceConfig struct {
	Name         string   `json:"name"`
	Identifiers  []string `json:"identifiers"`
	Manufacturer string   `json:"manufacturer"`
}

type discoveryConfig struct {
	StateTopic      string       `json:"state_topic"`
	AttributesTopic string       `json:"json_attributes_topic"`
	Name            string       `json:"name"`
	PayloadHome     string       `json:"payload_home"`
	PayloadNotHome  string       `json:"payload_not_home"`
	SourceType      string       `json:"source_type"`
	UniqueId        string       `json:"unique_id"`
	Device          deviceConfig `json:"device"`
}
