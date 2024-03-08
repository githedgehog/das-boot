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

import "go.githedgehog.com/dasboot/pkg/config"

var _ config.EmbeddedConfig = &HedgehogAgentProvisioner{}

type HedgehogAgentProvisioner struct {
	// AgentURL is the download URL for the agent binary
	AgentURL string `json:"agent_url,omitempty" yaml:"agent_url,omitempty"`

	// AgentConfigURL is the download URL for the agent config yaml file
	AgentConfigURL string `json:"agent_config_url,omitempty" yaml:"agent_config_url,omitempty"`

	// AgentKubeconfigURL is the download URL for the kubeconfig for the agent
	AgentKubeconfigURL string `json:"agent_kubeconfig_url,omitempty" yaml:"agent_kubeconfig_url,omitempty"`

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
