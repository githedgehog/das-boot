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

func ReadFromFile(path string) (*Stage2, error) {
	// test the file type
	var typ FileType
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		typ = YAML
	} else if strings.HasSuffix(path, ".json") {
		typ = JSON
	}
	if typ == Unknown {
		return nil, fmt.Errorf("stage2 config at '%s': unknown file type, not a JSON or YAML file", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("stage2 config at '%s': %w", path, err)
	}
	defer f.Close()

	// pass it on to the reader function
	return ReadFrom(f, typ)
}

func ReadFrom(r io.Reader, typ FileType) (*Stage2, error) {
	var cfg Stage2
	switch typ { //nolint:exhaustive
	case JSON:
		if err := json.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("stage2 config: JSON decoder: %w", err)
		}
	case YAML:
		if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("stage2 config: YAML decoder: %w", err)
		}
	default:
		return nil, fmt.Errorf("stage 2 config: unknown file type")
	}
	return &cfg, nil
}

func MergeConfigs(embedded *Stage2, override *Stage2) *Stage2 {
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

	if override.Platform != "" {
		ret.Platform = override.Platform
	}

	if override.NOSInstallerURL != "" {
		ret.NOSInstallerURL = override.NOSInstallerURL
	}

	if override.ONIEUpdaterURL != "" {
		ret.ONIEUpdaterURL = override.ONIEUpdaterURL
	}

	if override.NOSType != "" {
		ret.NOSType = override.NOSType
	}

	if len(override.HedgehogSonicProvisioners) > 0 {
		provs := make([]HedgehogSonicProvisioner, len(ret.HedgehogSonicProvisioners))
		copy(provs, ret.HedgehogSonicProvisioners)

		for i := range override.HedgehogSonicProvisioners {
			if j := hasProvisioner(ret.HedgehogSonicProvisioners, override.HedgehogSonicProvisioners[i].Name); j >= 0 {
				provs[j] = override.HedgehogSonicProvisioners[i]
			} else {
				provs = append(provs, override.HedgehogSonicProvisioners[i]) //nolint: makezero
			}
		}

		ret.HedgehogSonicProvisioners = provs
	}

	return &ret
}

func hasProvisioner(provs []HedgehogSonicProvisioner, name string) int {
	for i, prov := range provs {
		if prov.Name == name {
			return i
		}
	}
	return -1
}
