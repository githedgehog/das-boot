package registration

import "go.githedgehog.com/dasboot/pkg/partitions/location"

// Request represents a registration request as performed by the stage 1 installer
type Request struct {
	DeviceID     string         `json:"devid,omitempty"`
	CSR          []byte         `json:"csr,omitempty"`
	LocationInfo *location.Info `json:"location_info,omitempty"`
}
