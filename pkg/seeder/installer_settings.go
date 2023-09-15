package seeder

import (
	"fmt"
	"net/url"
	"path"

	"go.githedgehog.com/dasboot/pkg/seeder/config"
)

type loadedInstallerSettings struct {
	serverCADER          []byte
	configSignatureCADER []byte
	secureServerName     string
	controlVIP           string
	dnsServers           []string
	ntpServers           []string
	syslogServers        []string
	kubeSubnets          []string
}

func (s *seeder) initializeInstallerSettings(cfg *config.InstallerSettings) error {
	// secure server name must not be empty
	if cfg.SecureServerName == "" {
		return fmt.Errorf("secure server name must be set")
	}

	// read server CA and store the DER bytes in the seeder
	_, serverCADER, err := readCertFromPath(cfg.ServerCAPath)
	if err != nil {
		return err
	}

	// read config signature CA if set
	var configSignatureCADER []byte
	if cfg.ConfigSignatureCAPath != "" {
		var err error
		_, configSignatureCADER, err = readCertFromPath(cfg.ConfigSignatureCAPath)
		if err != nil {
			return err
		}
	}
	s.installerSettings = &loadedInstallerSettings{
		serverCADER:          serverCADER,
		configSignatureCADER: configSignatureCADER,
		secureServerName:     cfg.SecureServerName,
		controlVIP:           cfg.ControlVIP,
		dnsServers:           cfg.DNSServers,
		ntpServers:           cfg.NTPServers,
		syslogServers:        cfg.SyslogServers,
		kubeSubnets:          cfg.KubeSubnets,
	}

	return nil
}

func (lis *loadedInstallerSettings) stage1URL(arch string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", stage1PathBase, arch),
	}).String()
}

func (lis *loadedInstallerSettings) stage2URL(arch string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", stage2PathBase, arch),
	}).String()
}

func (lis *loadedInstallerSettings) registerURL() string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", registerPath),
	}).String()
}

func (lis *loadedInstallerSettings) nosInstallerURL() string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", nosInstallerPathBase),
	}).String()
}

func (lis *loadedInstallerSettings) onieUpdaterURL() string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", onieUpdaterPathBase),
	}).String()
}

func (lis *loadedInstallerSettings) hhAgentProvisionerURL(arch string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", hhAgentProvisionerPathBase, arch),
	}).String()
}

func (lis *loadedInstallerSettings) agentURL(arch string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", hhAgentProvisionerPathBase, "agent", arch),
	}).String()
}

func (lis *loadedInstallerSettings) agentConfigURL() string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", hhAgentProvisionerPathBase, "agent", "config"),
	}).String()
}

func (lis *loadedInstallerSettings) agentKubeconfigURL() string {
	return (&url.URL{
		Scheme: "https",
		Host:   lis.secureServerName,
		Path:   path.Join("/", hhAgentProvisionerPathBase, "agent", "kubeconfig"),
	}).String()
}
