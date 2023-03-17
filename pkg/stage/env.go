package stage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	DeviceID          string
}

const (
	envNameStagingDir        = "dasboot_staging_dir"
	envNameServerCA          = "dasboot_server_ca"
	envNameConfigSignatureCA = "dasboot_config_signature_ca"
	envNameLogSettings       = "dasboot_log_settings"
	envNameDeviceID          = "dasboot_hhdevid"
)

func (si *StagingInfo) Export() error {
	logSettingsBytes, err := json.Marshal(&si.LogSettings)
	if err != nil {
		return fmt.Errorf("failed to JSON encode log settings: %w", err)
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
			serverCAPath := filepath.Join(si.StagingDir, "server-ca.der")
			if err := writeFile(serverCAPath, si.ServerCA); err != nil {
				return fmt.Errorf("failed to write server CA to disk at '%s': %w", serverCAPath, err)
			}
		}

		if len(si.ConfigSignatureCA) > 0 {
			configSignatureCAPath := filepath.Join(si.StagingDir, "config-signature-ca.der")
			if err := writeFile(configSignatureCAPath, si.ConfigSignatureCA); err != nil {
				return fmt.Errorf("failed to write config signature CA to disk at '%s': %w", configSignatureCAPath, err)
			}
		}

		logSettingsPath := filepath.Join(si.StagingDir, "log-settings.json")

		if err := writeFile(logSettingsPath, logSettingsBytes); err != nil {
			return fmt.Errorf("failed to write log settings to disk at '%s': %w", logSettingsPath, err)
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
