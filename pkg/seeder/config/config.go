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

import "go.githedgehog.com/dasboot/pkg/seeder/artifacts"

// SeederConfig is passed to a seeder instance. It will initialize the seeder based on this configuration.
type SeederConfig struct {
	// InsecureServer will instantiate an insecure server if it is not nil. The insecure server serves
	// all artifacts which are allowed to be served over an unsecured connection like the stage0 installer.
	InsecureServer *InsecureServer

	// SecureServer will instantiate a secure server if it is not nil. The secure server serves all artifacts
	// which must be served over a secure connection.
	SecureServer *BindInfo

	// ArtifactsProvider is used to retrieve installer images.
	ArtifactsProvider artifacts.Provider

	// EmbeddedConfigGenerator contains all settings which are necessary to generate embedded configuration for the
	// staged installer artifacts
	EmbeddedConfigGenerator *EmbeddedConfigGeneratorConfig

	// InstallerSettings are various settings that are being used in configurations that are being sent to clients through
	// embedded configurations.
	InstallerSettings *InstallerSettings

	// RegistrySettings are all settings that deal with registration requests that are being sent by clients.
	RegistrySettings *RegistrySettings
}

// BindInfo provides all the necessary information for binding to an address and configuring TLS as necessary.
type BindInfo struct {
	// Address is a set of addresses that the server should bind on. In practice multiple HTTP server instances
	// will be running, but all serving the same routes for the same purpose. At least one address must be
	// provided.
	Address []string

	// ClientCAPath points to a file containing one or more CA certificates that client certificates will be
	// validated against if a client certificate is provided. If this is empty, no client authentication will
	// be required on the TLS server. This setting is ignored if no server key and certificate were provided.
	ClientCAPath string

	// ServerKeyPath points to a file containing the server key used for the TLS server. If this is empty,
	// a plain HTTP server will be initiated.
	ServerKeyPath string

	// ServerCertPath points to a file containing the server certificate used for the TLS server. If `ServerKeyPath`
	// is set, this setting is required to be set.
	ServerCertPath string
}

type EmbeddedConfigGeneratorConfig struct {
	// KeyPath points to a file which contains the key which is being used to sign embedded configuration.
	KeyPath string

	// CertPath points to a certificate which is used to sign embedded configuration. Its public key must
	// match the key from `KeyPath`.
	CertPath string
}

// InstallerSettings are various settings that are being used in configurations that are being sent to clients through
// embedded configurations
type InstallerSettings struct {
	// ServerCAPath points to a file containing the CA certificate which signed the server certificate which is used
	// for the TLS server. This is necessary to provide it to clients in case they have not received it through an
	// alternative way.
	ServerCAPath string

	// ConfigSignatureCAPath points to a file containing the CA certificate which signed the signature certificate
	// which is used to sign the embedded configuration which is served with every staged installer.
	ConfigSignatureCAPath string

	// SecureServerName is the host name as it should match the TLS SAN for the server certificates that are used by clients to reach the seeder.
	// This server name will be used to generate various URLs which are going to be used in embedded configurations. If the service needs a
	// different port it needs to be included here (e.g. dasboot.example.com:8080).
	SecureServerName string

	// ControlVIP is the virtual IP of where to reach the control network services
	ControlVIP string

	// NTPServers are the NTP servers which will be configured on clients at installation time
	NTPServers []string

	// SyslogServers are the syslog servers which will be configured on clients at installation time
	SyslogServers []string
}

// RegistrySettings are all the settings that instruct the seeder on what to do for registration requests
// from clients.
type RegistrySettings struct {
	// CertPath is the path to a file containing a CA certificate which is used to sign client certificates
	// for registration requests. NOTE: This should be empty, and registration requests should be
	// handled by the registration controller instead. If this is set, it means that we will automatically
	// accept and approve all registration requests.
	CertPath string `json:"cert_path,omitempty" yaml:"cert_path,omitempty"`

	// CAKey is the path to a file containing a CA key which is used to sign client certificates for
	// registration requests. NOTE: This should be empty, and registration requests should be
	// handled by the registration controller instead. If this is set, it means that we will automatically
	// accept and approve all registration requests.
	KeyPath string `json:"key_path,omitempty" yaml:"key_path,omitempty"`
}

// InsecureServer are all settings on how to start the insecure server handler.
type InsecureServer struct {
	// DynLL uses the dynamic linklocal server detection based on Kubernetes configuration of this device
	// and its neighbours
	DynLL *DynLL

	// Generic can be used to start the insecure server simply on given listeners.
	// This is not the preferred way of operations and prevents some functionality from working.
	// For example the seeder will not be able to deduce neighbours based on configuration stored in Kubernetes.
	// You should always configure DynLL unless you have a very good reason not to.
	Generic *BindInfo
}

// DynLL holds configuration for the dynamic linklocal insecure server listeners configuration. This mode allows
// for detection of neighbours based on configuration in Kubernetes. It will then start linklocal listeners only
// for those interfaces. Additionally this allows for advanced features like providing the location information
// to the stage0 installer instead of relying on it of being provided by the client itself.
type DynLL struct {
	// DeviceType is used while trying to self-detect who we are. The device could be either a switch or a server.
	// By default it tries to detect itself from both.
	DeviceType DeviceType

	// DeviceName is used while trying to self-detect who we are. Depening on the device type it is trying to look
	// for itself as being either a fabric.githedgehog.com/Switch or a fabric.githedgehog.com/Server.
	DeviceName string

	// ListeningPort is the port that will be used for all discovered ports that we need to listen on.
	ListeningPort uint16
}

type DeviceType uint8

// DeviceTypeAuto means that the system is trying to detect itself as either being a switch or a server
const DeviceTypeAuto DeviceType = 0

// DeviceTypeServer means that the system is looking for an entry in fabric.githedgehog.com/Server
const DeviceTypeServer DeviceType = 1

// DeviceTypeSwitch means that the system is looking for an entry in fabric.githedgehog.com/Switch
const DeviceTypeSwitch DeviceType = 2
