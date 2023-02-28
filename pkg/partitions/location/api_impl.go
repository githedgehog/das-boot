package location

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/google/uuid"
	"go.githedgehog.com/dasboot/pkg/partitions"
)

type api struct {
	dev *partitions.Device
}

var _ LocationPartition = &api{}

// Open an existing identity partition. If the partition was not previously initialized
// this function returns `ErrUninitializedPartition` in which case the caller should
// call `Init()` instead.
func Open(d *partitions.Device) (LocationPartition, error) {
	// initial checks
	if !d.IsHedgehogLocationPartition() {
		return nil, ErrWrongDevice
	}
	if !d.IsMounted() {
		return nil, ErrNotMounted
	}

	// read version file
	f, err := d.FS.Open(versionFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// this means the caller should call `Init` instead
			return nil, ErrUninitializedPartition
		}
		return nil, err
	}
	defer f.Close()
	var version Version
	if err := json.NewDecoder(f).Decode(&version); err != nil {
		return nil, err
	}

	// only version one is supported right now
	if version.Version != version1 {
		return nil, ErrUnsupportedVersion
	}

	// all validations complete, return the API object
	return &api{
		dev: d,
	}, nil
}

// GetLocation implements LocationPartition
func (a *api) GetLocation() (*Info, error) {
	// uuid
	f1, err := a.dev.FS.Open(locationUUIDPath)
	if err != nil {
		return nil, err
	}
	defer f1.Close()
	uuidBytes, err := io.ReadAll(f1)
	if err != nil {
		return nil, err
	}
	luuid, err := uuid.ParseBytes(uuidBytes)
	if err != nil {
		return nil, err
	}

	// uuid.sig
	f2, err := a.dev.FS.Open(locationUUIDSigPath)
	if err != nil {
		return nil, err
	}
	defer f2.Close()
	uuidSigBytes, err := io.ReadAll(f2)
	if err != nil {
		return nil, err
	}

	// metadata
	f3, err := a.dev.FS.Open(locationMetadataPath)
	if err != nil {
		return nil, err
	}
	defer f3.Close()
	metadataBytes, err := io.ReadAll(f3)
	if err != nil {
		return nil, err
	}
	var md Metadata
	if err := json.Unmarshal(metadataBytes, &md); err != nil {
		return nil, err
	}

	// metadata.sig
	f4, err := a.dev.FS.Open(locationMetadataSigPath)
	if err != nil {
		return nil, err
	}
	defer f4.Close()
	metadataSigBytes, err := io.ReadAll(f4)
	if err != nil {
		return nil, err
	}

	// now return it
	// we validated as good as we can at this point that this is good data
	return &Info{
		UUID:        luuid.String(),
		UUIDSig:     uuidSigBytes,
		Metadata:    string(metadataBytes),
		MetadataSig: metadataSigBytes,
	}, nil
}
