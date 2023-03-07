package seeder

import (
	"fmt"
	"net/url"
)

type loadedInstallerSettings struct {
	serverCADER          []byte
	configSignatureCADER []byte
	secureServerName     string
	dnsServers           []string
	ntpServers           []string
	syslogServers        []string
}

func (s *seeder) initializeInstallerSettings(config *InstallerSettings) error {
	// secure server name must not be empty
	if config.SecureServerName == "" {
		return fmt.Errorf("secure server name must be set")
	}

	// read server CA and store the DER bytes in the seeder
	_, serverCADER, err := readCertFromPath(config.ServerCAPath)
	if err != nil {
		return err
	}

	// read config signature CA if set
	var configSignatureCADER []byte
	if config.ConfigSignatureCAPath != "" {
		var err error
		_, configSignatureCADER, err = readCertFromPath(config.ConfigSignatureCAPath)
		if err != nil {
			return err
		}
	}
	s.installerSettings = &loadedInstallerSettings{
		serverCADER:          serverCADER,
		configSignatureCADER: configSignatureCADER,
		secureServerName:     config.SecureServerName,
		dnsServers:           config.DNSServers,
		ntpServers:           config.NTPServers,
		syslogServers:        config.SyslogServers,
	}

	return nil
}

func (lis *loadedInstallerSettings) stage1URLBase() string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   stage1PathBase,
	}).String()
}
