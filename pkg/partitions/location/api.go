package location

import (
	"encoding/json"
	"errors"
)

//go:generate mockgen -destination ../../../test/mock/mockpartitions/mocklocation/api_mock.go -package mocklocation . LocationPartition
type LocationPartition interface {
	// GetLocation reads the location information from the partition, and returns an error otherwise.
	GetLocation() (*Info, error)
}

// Version is the contents of the version file.
type Version struct {
	// Version is the version number of the partition format. This field
	// must always be present.
	Version int
}

type Info struct {
	UUID        string
	UUIDSig     []byte
	Metadata    string
	MetadataSig []byte
}

func (i *Info) MetadataDecoded() Metadata {
	ret := Metadata{}
	if err := json.Unmarshal([]byte(i.Metadata), &ret); err != nil {
		return nil
	}

	return ret
}

type Metadata map[string]string

const (
	version1 int = 1

	versionFilePath         = "/version"
	locationDirPath         = "/location"
	locationUUIDPath        = locationDirPath + "/uuid"
	locationUUIDSigPath     = locationDirPath + "/uuid.sig"
	locationMetadataPath    = locationDirPath + "/metadata"
	locationMetadataSigPath = locationDirPath + "/metadata.sig"
)

var (
	ErrWrongDevice            = errors.New("identity: not the identity partition")
	ErrNotMounted             = errors.New("identity: partition not mounted")
	ErrUnsupportedVersion     = errors.New("identity: unsupported identity partition version")
	ErrUninitializedPartition = errors.New("identity: partition uninitialized")
)
