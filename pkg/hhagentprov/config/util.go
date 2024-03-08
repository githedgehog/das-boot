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

	"gopkg.in/yaml.v3"
)

type FileType int

const (
	Unknown FileType = iota
	JSON
	YAML
)

func ReadFromFile(path string) (*HedgehogAgentProvisioner, error) {
	// test the file type
	var typ FileType
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		typ = YAML
	} else if strings.HasSuffix(path, ".json") {
		typ = JSON
	}
	if typ == Unknown {
		return nil, fmt.Errorf("hedgehog agent provisioner config at '%s': unknown file type, not a JSON or YAML file", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("hedgehog agent provisioner config at '%s': %w", path, err)
	}
	defer f.Close()

	// pass it on to the reader function
	return ReadFrom(f, typ)
}

func ReadFrom(r io.Reader, typ FileType) (*HedgehogAgentProvisioner, error) {
	var cfg HedgehogAgentProvisioner
	switch typ { //nolint:exhaustive
	case JSON:
		if err := json.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("hedgehog agent provisioner config: JSON decoder: %w", err)
		}
	case YAML:
		if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("hedgehog agent provisioner config: YAML decoder: %w", err)
		}
	default:
		return nil, fmt.Errorf("hedgehog agent provisioner config: unknown file type")
	}
	return &cfg, nil
}

func MergeConfigs(embedded *HedgehogAgentProvisioner, override *HedgehogAgentProvisioner) *HedgehogAgentProvisioner {
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

	if override.AgentURL != "" {
		ret.AgentURL = override.AgentURL
	}

	if override.AgentConfigURL != "" {
		ret.AgentConfigURL = override.AgentConfigURL
	}

	if override.AgentKubeconfigURL != "" {
		ret.AgentKubeconfigURL = override.AgentKubeconfigURL
	}

	return &ret
}
