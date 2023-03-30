package config

import "go.githedgehog.com/dasboot/pkg/config"

var _ config.EmbeddedConfig = &HedgehogAgentProvisioner{}

type HedgehogAgentProvisioner struct {
	// AgentURL is the download URL for the agent binary
	AgentURL string `json:"agent_url,omitempty" yaml:"agent_url,omitempty"`

	// AgentConfigURL
	AgentConfigURL string `json:"agent_config_url,omitempty" yaml:"agent_config_url,omitempty"`

	// SignatureCert holds the DER encoded X509 certificate with which the signature of the embedded config
	// can be validated
	SignatureCert []byte `json:"signature_cert,omitempty" yaml:"signature_cert,omitempty"`

	// Version is tracking the format of this structure itself
	Version config.ConfigVersion `json:"version,omitempty" yaml:"version,omitempty"`
}

// Cert implements config.EmbeddedConfig
func (c *HedgehogAgentProvisioner) Cert() []byte {
	return c.SignatureCert
}

// Validate implements config.EmbeddedConfig
func (c *HedgehogAgentProvisioner) Validate() error {
	// TODO: implement
	return nil
}

// ConfigVersion implements config.EmbeddedConfig
func (c *HedgehogAgentProvisioner) ConfigVersion() config.ConfigVersion {
	return c.Version
}

// IsSupportedConfigVersion implements config.EmbeddedConfig
func (*HedgehogAgentProvisioner) IsSupportedConfigVersion(v config.ConfigVersion) bool {
	return v == 1
}
