package ipam

// Request represents an IPAM request as being performed by the Stage 0 installer
type Request struct {
	Arch                  string   `json:"arch"`
	DevID                 string   `json:"devid"`
	LocationUUID          string   `json:"location_uuid"`
	LocationUUIDSignature []byte   `json:"location_uuid_signature"`
	Interfaces            []string `json:"interfaces,omitempty"`
}
