package ipam

import "github.com/google/uuid"

// Request represents an IPAM request as being performed by the Stage 0 installer
type Request struct {
	Arch                  string   `json:"arch"`
	DevID                 string   `json:"devid"`
	LocationUUID          string   `json:"location_uuid"`
	LocationUUIDSignature []byte   `json:"location_uuid_signature"`
	Interfaces            []string `json:"interfaces,omitempty"`
}

func (r *Request) Validate() error {
	// arch
	switch r.Arch {
	case "x86_64":
		fallthrough
	case "arm64":
		fallthrough
	case "arm":
		// no error
	default:
		return unsupportedArchError(r.Arch)
	}

	// devid
	if _, err := uuid.Parse(r.DevID); err != nil {
		return invalidUUIDError("devid", err)
	}

	// location uuid
	if r.LocationUUID != "" {
		if _, err := uuid.Parse(r.LocationUUID); err != nil {
			return invalidUUIDError("location_uuid", err)
		}

		// location uuid signature
		if len(r.LocationUUIDSignature) == 0 {
			return emptyValueError("location_uuid_signature")
		}
	}

	// interfaces
	if len(r.Interfaces) == 0 {
		return emptyValueError("interfaces")
	}

	return nil
}
