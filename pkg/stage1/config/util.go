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

func ReadFromFile(path string) (*Stage1, error) {
	// test the file type
	var typ FileType
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		typ = YAML
	} else if strings.HasSuffix(path, ".json") {
		typ = JSON
	}
	if typ == Unknown {
		return nil, fmt.Errorf("stage1 config at '%s': unknown file type, not a JSON or YAML file", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("stage1 config at '%s': %w", path, err)
	}
	defer f.Close()

	// pass it on to the reader function
	return ReadFrom(f, typ)
}

func ReadFrom(r io.Reader, typ FileType) (*Stage1, error) {
	var cfg Stage1
	switch typ { //nolint:exhaustive
	case JSON:
		if err := json.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("stage1 config: JSON decoder: %w", err)
		}
	case YAML:
		if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("stage1 config: YAML decoder: %w", err)
		}
	default:
		return nil, fmt.Errorf("stage 1 config: unknown file type")
	}
	return &cfg, nil
}

func MergeConfigs(embedded *Stage1, override *Stage1) *Stage1 {
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

	// Keylime settings can be overridden
	if override.Keylime != nil {
		if ret.Keylime == nil {
			// if keylime isn't even set in the embedded we simply copy all over
			keylimeOverride := *override.Keylime
			ret.Keylime = &keylimeOverride
		} else {
			// otherwise we treat it as single field setting overrides
			if override.Keylime.CVCAURL != "" {
				ret.Keylime.CVCAURL = override.Keylime.CVCAURL
			}
			if override.Keylime.RegistrarIP != "" {
				ret.Keylime.RegistrarIP = override.Keylime.RegistrarIP
			}
			if override.Keylime.RegistrarPort > 0 {
				ret.Keylime.RegistrarPort = override.Keylime.RegistrarPort
			}
			if override.Keylime.RevocationNotificationIP != "" {
				ret.Keylime.RevocationNotificationIP = override.Keylime.RevocationNotificationIP
			}
			if override.Keylime.RevocationNotificationPort > 0 {
				ret.Keylime.RevocationNotificationPort = override.Keylime.RevocationNotificationPort
			}
			if override.Keylime.TenantTriggerURL != "" {
				ret.Keylime.TenantTriggerURL = override.Keylime.TenantTriggerURL
			}
		}
	}

	// RegisterURL can be overridden
	if override.RegisterURL != "" {
		ret.RegisterURL = override.RegisterURL
	}

	// Stage2URL can be overridden
	if override.Stage2URL != "" {
		ret.Stage2URL = override.Stage2URL
	}

	return &ret
}
