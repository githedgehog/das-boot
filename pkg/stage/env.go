package stage

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.githedgehog.com/dasboot/pkg/devid"
	"go.githedgehog.com/dasboot/pkg/stage0/config"
)

// OnieEnv represents a set of environment variables that *should* always
// be set in any running ONIE installer
type OnieEnv struct {
	BootReason string
	ExecURL    string
	Platform   string
	VendorID   string
	SerialNum  string
	EthAddr    string
}

// GetOnieEnv returns the set of ONIE environment variables that *should* always
// bet in any running ONIE installer
func GetOnieEnv() *OnieEnv {
	return &OnieEnv{
		BootReason: os.Getenv("onie_boot_reason"),
		ExecURL:    os.Getenv("onie_exec_url"),
		Platform:   os.Getenv("onie_platform"),
		VendorID:   os.Getenv("onie_vendor_id"),
		SerialNum:  os.Getenv("onie_serial_num"),
		EthAddr:    os.Getenv("onie_eth_addr"),
	}
}

type StagingInfo struct {
	StagingDir        string
	ServerCA          []byte
	ConfigSignatureCA []byte
	LogSettings       LogSettings
	OnieHeaders       *config.OnieHeaders
	DeviceID          string
}

const (
	envNameStagingDir        = "dasboot_staging_dir"
	envNameServerCA          = "dasboot_server_ca"
	envNameConfigSignatureCA = "dasboot_config_signature_ca"
	envNameLogSettings       = "dasboot_log_settings"
	envNameOnieHeaders       = "dasboot_onie_headers"
	envNameDeviceID          = "dasboot_hhdevid"
	pathServerCA             = "server-ca.der"
	pathConfigSignatureCA    = "config-signature-ca.der"
	pathLogSettings          = "log-settings.json"
	pathOnieHeaders          = "onie-headers.json"
)

func (si *StagingInfo) Export() error {
	logSettingsBytes, err := json.Marshal(&si.LogSettings)
	if err != nil {
		return fmt.Errorf("failed to JSON encode log settings: %w", err)
	}

	var onieHeadersBytes []byte
	if si.OnieHeaders != nil {
		var err error
		onieHeadersBytes, err = json.Marshal(si.OnieHeaders)
		if err != nil {
			return fmt.Errorf("failed to JSON encode ONIE headers: %w", err)
		}
	}

	// we only persist to disk if staging dir is set already, otherwise we only
	// export environment variables
	if si.StagingDir != "" {
		// we will write the certificates to disk
		// mainly for debugging purposes and in cases an installer fails and we need that
		// information when manually starting a subsequent installer
		pwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		if pwd != si.StagingDir {
			if err := os.Chdir(si.StagingDir); err != nil {
				return fmt.Errorf("failed to change directory to staging directory '%s': %w", si.StagingDir, err)
			}
		}

		if len(si.ServerCA) > 0 {
			serverCAPath := filepath.Join(si.StagingDir, pathServerCA)
			if err := writeFile(serverCAPath, si.ServerCA); err != nil {
				return fmt.Errorf("failed to write server CA to disk at '%s': %w", serverCAPath, err)
			}
		}

		if len(si.ConfigSignatureCA) > 0 {
			configSignatureCAPath := filepath.Join(si.StagingDir, pathConfigSignatureCA)
			if err := writeFile(configSignatureCAPath, si.ConfigSignatureCA); err != nil {
				return fmt.Errorf("failed to write config signature CA to disk at '%s': %w", configSignatureCAPath, err)
			}
		}

		logSettingsPath := filepath.Join(si.StagingDir, pathLogSettings)
		if err := writeFile(logSettingsPath, logSettingsBytes); err != nil {
			return fmt.Errorf("failed to write log settings to disk at '%s': %w", logSettingsPath, err)
		}

		if len(onieHeadersBytes) > 0 {
			onieHeadersPath := filepath.Join(si.StagingDir, pathOnieHeaders)
			if err := writeFile(onieHeadersPath, onieHeadersBytes); err != nil {
				return fmt.Errorf("failed to write ONIE headers to disk at '%s': %w", onieHeadersPath, err)
			}
		}
	}

	// now export environment variables
	if si.StagingDir != "" {
		if err := os.Setenv(envNameStagingDir, si.StagingDir); err != nil {
			return fmt.Errorf("failed to set '%s' environment variable: %w", envNameStagingDir, err)
		}
	}
	if len(si.ServerCA) > 0 {
		if err := os.Setenv(envNameServerCA, base64.StdEncoding.EncodeToString(si.ServerCA)); err != nil {
			return fmt.Errorf("failed to set '%s' environment variable: %w", envNameServerCA, err)
		}
	}
	if len(si.ConfigSignatureCA) > 0 {
		if err := os.Setenv(envNameConfigSignatureCA, base64.StdEncoding.EncodeToString(si.ConfigSignatureCA)); err != nil {
			return fmt.Errorf("failed to set '%s' environment variable: %w", envNameConfigSignatureCA, err)
		}
	}
	if string(logSettingsBytes) != "{}" {
		if err := os.Setenv(envNameLogSettings, string(logSettingsBytes)); err != nil {
			return fmt.Errorf("failed to set '%s' environment variable: %w", envNameLogSettings, err)
		}
	}
	if len(onieHeadersBytes) > 0 {
		if err := os.Setenv(envNameOnieHeaders, string(onieHeadersBytes)); err != nil {
			return fmt.Errorf("failed to set '%s' environment variable: %w", envNameOnieHeaders, err)
		}
	}
	if si.DeviceID != "" {
		if err := os.Setenv(envNameDeviceID, si.DeviceID); err != nil {
			return fmt.Errorf("failed to set '%s' environment variable: %w", envNameDeviceID, err)
		}
	}

	return nil
}

func writeFile(path string, contents []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.Write(contents)
	if err != nil {
		return err
	}
	if n != len(contents) {
		return fmt.Errorf("not all contents written to file")
	}

	return nil
}

func ReadStagingInfo() (*StagingInfo, error) {
	ret := &StagingInfo{}
	var ok bool

	ret.StagingDir, ok = os.LookupEnv(envNameStagingDir)
	if !ok {
		// we are assuming that the staging directory is then the current working directory if this is not set
		var err error
		ret.StagingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("environment variable '%s' not set, and could not get current working directory: %w", envNameStagingDir, err)
		}
	}

	serverCABase64String, ok := os.LookupEnv(envNameServerCA)
	if !ok {
		// environment variable not set, so we'll try to read it from disk
		var err error
		serverCAPath := filepath.Join(ret.StagingDir, pathServerCA)
		ret.ServerCA, err = readFile(serverCAPath)
		if err != nil {
			return nil, fmt.Errorf("environment variable '%s' not set, and failed to read Server CA from staging file '%s': %w", envNameServerCA, serverCAPath, err)
		}
	} else {
		// environment variable is set, try to base64 decode the value from it
		var err error
		ret.ServerCA, err = base64.StdEncoding.DecodeString(serverCABase64String)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode Server CA bytes from environment variable '%s': %w", envNameServerCA, err)
		}
	}

	configSignatureCABase64String, ok := os.LookupEnv(envNameConfigSignatureCA)
	if !ok {
		// environment variable not set, so we'll try to read it from disk
		var err error
		configSignatureCAPath := filepath.Join(ret.StagingDir, pathConfigSignatureCA)
		ret.ConfigSignatureCA, err = readFile(configSignatureCAPath)
		if err != nil {
			return nil, fmt.Errorf("environment variable '%s' not set, and failed to read Config Signature CA from staging file '%s': %w", envNameConfigSignatureCA, configSignatureCAPath, err)
		}
	} else {
		// environment variable is set, try to base64 decode the value from it
		var err error
		ret.ConfigSignatureCA, err = base64.StdEncoding.DecodeString(configSignatureCABase64String)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode Config Signature CA bytes from environment variable '%s': %w", envNameConfigSignatureCA, err)
		}
	}

	logSettingsJSONString, ok := os.LookupEnv(envNameLogSettings)
	if !ok {
		// environment variable not set, so we'll try to read it from disk
		logSettingsPath := filepath.Join(ret.StagingDir, pathLogSettings)
		logSettingsBytes, err := readFile(logSettingsPath)
		if err != nil {
			return nil, fmt.Errorf("environment variable '%s' not set, and failed to read log settings from file '%s': %w", envNameLogSettings, logSettingsPath, err)
		}
		if err := json.Unmarshal(logSettingsBytes, &ret.LogSettings); err != nil {
			return nil, fmt.Errorf("environment variable '%s' not set, and failed to JSON decode log settings from file '%s': %w", envNameLogSettings, logSettingsPath, err)
		}
	} else {
		// environment variable is set, try to JSON decode the value from it
		if err := json.Unmarshal([]byte(logSettingsJSONString), &ret.LogSettings); err != nil {
			return nil, fmt.Errorf("failed to JSON decode log settings from environment variable '%s' (value: '%s'): %w", envNameLogSettings, logSettingsJSONString, err)
		}
	}

	onieHeadersJSONString, ok := os.LookupEnv(envNameOnieHeaders)
	if !ok {
		// environment variable not set, so we'll try to read it from disk
		onieHeadersPath := filepath.Join(ret.StagingDir, pathOnieHeaders)
		onieHeadersBytes, err := readFile(onieHeadersPath)
		if err != nil {
			return nil, fmt.Errorf("environment variable '%s' not set, and failed to read ONIE headers from file '%s': %w", envNameOnieHeaders, onieHeadersPath, err)
		}
		var oh config.OnieHeaders
		if err := json.Unmarshal(onieHeadersBytes, &oh); err != nil {
			return nil, fmt.Errorf("environment variable '%s' not set, and failed to JSON decode log settings from file '%s': %w", envNameOnieHeaders, onieHeadersPath, err)
		}
		ret.OnieHeaders = &oh
	} else {
		// environment variable is set, try to JSON decode the value from it
		var oh config.OnieHeaders
		if err := json.Unmarshal([]byte(onieHeadersJSONString), &oh); err != nil {
			return nil, fmt.Errorf("failed to JSON decode ONIE headers from environment variable '%s' (value: '%s'): %w", envNameOnieHeaders, onieHeadersJSONString, err)
		}
		ret.OnieHeaders = &oh
	}

	ret.DeviceID, ok = os.LookupEnv(envNameDeviceID)
	if !ok {
		// environment variable not set, so we'll run the Device ID algorithm again
		ret.DeviceID = devid.ID()
		if ret.DeviceID == "" {
			return nil, fmt.Errorf("environment variable '%s' not set, and failed to determine the device ID again", envNameDeviceID)
		}
	} else {
		if ret.DeviceID == "" {
			ret.DeviceID = devid.ID()
			if ret.DeviceID == "" {
				return nil, fmt.Errorf("environment variable '%s' was empty, and failed to determine the device ID again", envNameDeviceID)
			}
		}
	}

	return ret, nil
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

var ErrValueNotSet = errors.New("staging info: value not set")

func valueNotSetError(s string) error {
	return fmt.Errorf("%w: %s", ErrValueNotSet, s)
}

func (si *StagingInfo) ServerCAPool() (*x509.CertPool, error) {
	if si != nil && len(si.ServerCA) > 0 {
		cert, err := x509.ParseCertificate(si.ServerCA)
		if err != nil {
			return nil, fmt.Errorf("staging info: parsing Server CA certificate: %w", err)
		}
		ret := x509.NewCertPool()
		ret.AddCert(cert)
		return ret, nil
	}
	return nil, valueNotSetError("ServerCA")
}

func (si *StagingInfo) ConfigSignatureCAPool() (*x509.CertPool, error) {
	if si != nil && len(si.ConfigSignatureCA) > 0 {
		cert, err := x509.ParseCertificate(si.ConfigSignatureCA)
		if err != nil {
			return nil, fmt.Errorf("staging info: parsing Config Signature CA certificate: %w", err)
		}
		ret := x509.NewCertPool()
		ret.AddCert(cert)
		return ret, nil
	}
	return nil, valueNotSetError("ConfigSignatureCA")
}
