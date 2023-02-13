package config

import (
	"go.githedgehog.com/dasboot/pkg/config"
)

var _ config.EmbeddedConfig = &Stage1{}

// Stage1 represents the structure of the config for the stage 1 installer.
//
// Here is an example JSON:
//
//	{
//	  "keylime":{
//	    "cvca_url":"https://das-boot.hedgehog.svc.cluster.local/keylime/cvca.pem",
//	    "registrar_ip":"10.255.1.113",
//	    "registrar_port":8890,
//	    "revocation_notification_ip":"10.255.1.113",
//	    "revocation_notification_port":8892,
//	    "tenant_trigger_url":"https://das-boot.hedgehog.svc.cluster.local/keylime/tenant_trigger/",
//	  },
//	  "register_url":"https://das-boot.hedgehog.svc.cluster.local/register",
//	  "stage2_url":"https://das-boot.hedgehog.svc.cluster.local/stage2-x86_64"
//	}
type Stage1 struct {
	// Keylime is the keylime configuration
	Keylime *KeylimeConfig `json:"keylime,omitempty"`

	// RegisterURL will be called by stage 1 to register the device (and receive its client certificate)
	RegisterURL string `json:"register_url,omitempty"`

	// Stage2URL is the URL to the stage 2 installer
	Stage2URL string `json:"stage2_url"`

	// SignatureCert holds the DER encoded X509 certificate with which the signature of the embedded config
	// can be validated
	SignatureCert []byte `json:"signature_cert,omitempty"`
}

// KeylimeConfig is the keylime configuration as it is embedded in the stage 1 configuration.
type KeylimeConfig struct {
	// CVCAURL is the URL to the CA certificate of the Keylime Verifier (CV)
	CVCAURL string `json:"cvca_url,omitempty"`

	// RegistrarIP is the IP address of the Keylime registrar service
	RegistrarIP string `json:"registrar_ip,omitempty"`

	// RegistrarPort is the port number of the Keylime registrar service
	RegistrarPort uint16 `json:"registrar_port,omitempty"`

	// RevocationNotificationIP is the IP address of the Keylime revocation notification queue system
	RevocationNotificationIP string `json:"revocation_notification_ip,omitempty"`

	// RevocationNotificationPort is the port number of the Keylime revocation notification queue system
	RevocationNotificationPort uint16 `json:"revocation_notification_port,omitempty"`

	// TenantTriggerURL is the URL which notifies the Keylime tenant controller to add the device to the Keylime Verifier (CV)
	TenantTriggerURL string `json:"tenant_trigger_url,omitempty"`
}

// Cert implements config.EmbeddedConfig
func (c *Stage1) Cert() []byte {
	return c.SignatureCert
}

// Validate implements config.EmbeddedConfig
func (c *Stage1) Validate() error {
	panic("unimplemented")
}
