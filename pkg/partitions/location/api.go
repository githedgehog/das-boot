// Copyright 2023 Hedgehog
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	Version int `json:"version"`
}

type Info struct {
	UUID        string `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	UUIDSig     []byte `json:"uuid_sig,omitempty" yaml:"uuid_sig,omitempty"`
	Metadata    string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	MetadataSig []byte `json:"metadata_sig,omitempty" yaml:"metadata_sig,omitempty"`
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
