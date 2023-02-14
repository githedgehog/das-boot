package config

import "go.githedgehog.com/dasboot/pkg/config"

var _ config.EmbeddedConfig = &Stage2{}

// Stage2 represents the structure of the config for the stage 2 installer.
//
// Here is an example JSON:
//
//	{
//	  "platform":"x86_64-kvm_x86_64-r0",
//	  "nos_installer_url":"https://das-boot.hedgehog.svc.cluster.local/nos/installer",
//	  "onie_updater_url":"https://das-boot.hedgehog.svc.cluster.local/onie/update",
//	  "nos_type":"hedgehog_sonic",
//	  "hedgehog_sonic_provisioners":[
//	    {
//	      "name":"Keylime Agent",
//	      "url":"https://das-boot.hedgehog.svc.cluster.local/provisioners/keylime-agent-x86_64"
//	    },
//	    {
//	      "name":"Hedgehog Agent",
//	      "url":"https://das-boot.hedgehog.svc.cluster.local/provisioners/hedgehog-agent-x86_64"
//	    }
//	  ]
//	}
type Stage2 struct {
	// Platform is an override for the "onie_platform" environment variable. This field should usually be empty
	// as the platform value should be derived from the environment.
	Platform string `json:"platform,omitempty"`

	// NOSInstallerURL is the URL where the NOS image is located
	NOSInstallerURL string `json:"nos_installer_url,omitempty"`

	// ONIEUpdaterURL is the URL where the ONIE updater image is located
	ONIEUpdaterURL string `json:"onie_updater_url,omitempty"`

	// NOSType represents the NOS that will be installed from the image in `NOSInstallerURL`.
	NOSType string `json:"nos_type,omitempty"`

	// HedgehogSonicProvisioners is a list of provisioners that will be executed if the `NOSType` is `hedgehog_sonic`.
	HedgehogSonicProvisioners []HedgehogSonicProvisioner `json:"hedgehog_sonic_provisioners,omitempty"`

	// SignatureCert holds the DER encoded X509 certificate with which the signature of the embedded config
	// can be validated
	SignatureCert []byte `json:"signature_cert,omitempty"`

	// Version is tracking the format of this structure itself
	Version config.ConfigVersion `json:"version,omitempty"`
}

// NOSTypeHedgehogSonic is the value for the Hedgehog SONiC distribution that can be sent through the stage 2 configuration.
const NOSTypeHedgehogSonic = "hedgehog_sonic"

// HedgehogSonicProvisioner represents the name and URL of a provisioner which are being executed in stage 2
// if the NOS type is set to "hedgehog_sonic"
type HedgehogSonicProvisioner struct {
	Name string `json:"name"`
	URL  string `json:"URL"`
}

// Cert implements config.EmbeddedConfig
func (c *Stage2) Cert() []byte {
	return c.SignatureCert
}

// Validate implements config.EmbeddedConfig
func (c *Stage2) Validate() error {
	// TODO: implement
	return nil
}

// ConfigVersion implements config.EmbeddedConfig
func (c *Stage2) ConfigVersion() config.ConfigVersion {
	return c.Version
}

// IsSupportedConfigVersion implements config.EmbeddedConfig
func (*Stage2) IsSupportedConfigVersion(v config.ConfigVersion) bool {
	return v == 1
}