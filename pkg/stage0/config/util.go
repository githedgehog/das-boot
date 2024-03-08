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

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"gopkg.in/yaml.v3"
)

type FileType int

const (
	Unknown FileType = iota
	JSON
	YAML
)

func ReadFromFile(path string) (*Stage0, error) {
	// test the file type
	var typ FileType
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		typ = YAML
	} else if strings.HasSuffix(path, ".json") {
		typ = JSON
	}
	if typ == Unknown {
		return nil, fmt.Errorf("stage0 config at '%s': unknown file type, not a JSON or YAML file", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("stage0 config at '%s': %w", path, err)
	}
	defer f.Close()

	// pass it on to the reader function
	return ReadFrom(f, typ)
}

func ReadFrom(r io.Reader, typ FileType) (*Stage0, error) {
	var cfg Stage0
	switch typ { //nolint:exhaustive
	case JSON:
		if err := json.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("stage0 config: JSON decoder: %w", err)
		}
	case YAML:
		if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("stage0 config: YAML decoder: %w", err)
		}
	default:
		return nil, fmt.Errorf("stage 0 config: unknown file type")
	}
	return &cfg, nil
}

func MergeConfigs(embedded *Stage0, override *Stage0) *Stage0 {
	// clone the values from the embedded config
	// so that we don't override the arguments for the caller
	// also short-circuit things to avoid pointer shenanigans
	if embedded == nil {
		return nil
	}
	ret := *embedded
	if override == nil {
		return &ret
	}

	// CA can be overridden
	if len(override.CA) > 0 {
		ret.CA = make([]byte, len(override.CA))
		copy(ret.CA, override.CA)
	}

	// SignatureCA can be overridden
	if len(override.SignatureCA) > 0 {
		ret.SignatureCA = make([]byte, len(override.SignatureCA))
		copy(ret.SignatureCA, override.SignatureCA)
	}

	// IPAMURL can be overridden
	if override.IPAMURL != "" {
		ret.IPAMURL = override.IPAMURL
	}

	// Stage1URL can be overridden
	if override.Stage1URL != "" {
		ret.Stage1URL = override.Stage1URL
	}

	// Services can be overridden
	if override.Services.ControlVIP != "" {
		ret.Services.ControlVIP = override.Services.ControlVIP
	}
	if len(override.Services.NTPServers) > 0 {
		ret.Services.NTPServers = make([]string, len(override.Services.NTPServers))
		copy(ret.Services.NTPServers, override.Services.NTPServers)
	}
	if len(override.Services.SyslogServers) > 0 {
		ret.Services.SyslogServers = make([]string, len(override.Services.SyslogServers))
		copy(ret.Services.SyslogServers, override.Services.SyslogServers)
	}

	// location information can be overridden
	if override.Location != nil {
		ret.Location = &location.Info{
			UUID:        override.Location.UUID,
			UUIDSig:     override.Location.UUIDSig,
			Metadata:    override.Location.Metadata,
			MetadataSig: override.Location.MetadataSig,
		}
	}

	return &ret
}
