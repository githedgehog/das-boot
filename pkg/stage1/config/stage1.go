// Copyright 2023 Hedgehog
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	Keylime *KeylimeConfig `json:"keylime,omitempty" yaml:"keylime,omitempty"`

	// RegisterURL will be called by stage 1 to register the device (and receive its client certificate)
	RegisterURL string `json:"register_url,omitempty" yaml:"register_url,omitempty"`

	// Stage2URL is the URL to the stage 2 installer
	Stage2URL string `json:"stage2_url,omitempty" yaml:"stage2_url,omitempty"`

	// SignatureCert holds the DER encoded X509 certificate with which the signature of the embedded config
	// can be validated
	SignatureCert []byte `json:"signature_cert,omitempty" yaml:"signature_cert,omitempty"`

	// Version is tracking the format of this structure itself
	Version config.ConfigVersion `json:"version,omitempty" yaml:"version,omitempty"`
}

// KeylimeConfig is the keylime configuration as it is embedded in the stage 1 configuration.
type KeylimeConfig struct {
	// CVCAURL is the URL to the CA certificate of the Keylime Verifier (CV)
	CVCAURL string `json:"cvca_url,omitempty" yaml:"cvca_url,omitempty"`

	// RegistrarIP is the IP address of the Keylime registrar service
	RegistrarIP string `json:"registrar_ip,omitempty" yaml:"registrar_ip,omitempty"`

	// RegistrarPort is the port number of the Keylime registrar service
	RegistrarPort uint16 `json:"registrar_port,omitempty" yaml:"registrar_port,omitempty"`

	// RevocationNotificationIP is the IP address of the Keylime revocation notification queue system
	RevocationNotificationIP string `json:"revocation_notification_ip,omitempty" yaml:"revocation_notification_ip,omitempty"`

	// RevocationNotificationPort is the port number of the Keylime revocation notification queue system
	RevocationNotificationPort uint16 `json:"revocation_notification_port,omitempty" yaml:"revocation_notification_port,omitempty"`

	// TenantTriggerURL is the URL which notifies the Keylime tenant controller to add the device to the Keylime Verifier (CV)
	TenantTriggerURL string `json:"tenant_trigger_url,omitempty" yaml:"tenant_trigger_url,omitempty"`
}

// Cert implements config.EmbeddedConfig
func (c *Stage1) Cert() []byte {
	return c.SignatureCert
}

// Validate implements config.EmbeddedConfig
func (c *Stage1) Validate() error {
	// TODO: implement
	return nil
}

// ConfigVersion implements config.EmbeddedConfig
func (c *Stage1) ConfigVersion() config.ConfigVersion {
	return c.Version
}

// IsSupportedConfigVersion implements config.EmbeddedConfig
func (*Stage1) IsSupportedConfigVersion(v config.ConfigVersion) bool {
	return v == 1
}
