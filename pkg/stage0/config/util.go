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

	return &ret
}
