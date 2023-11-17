package main

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is passed to a seeder instance. It will initialize the seeder based on this configuration.
type Config struct {
	// Servers holds all HTTP server settings
	Servers *Servers `json:"servers,omitempty" yaml:"servers,omitempty"`

	// EmbeddedConfigGenerator contains all settings which are necessary to generate embedded configuration for the
	// staged installer artifacts
	EmbeddedConfigGenerator *EmbeddedConfigGeneratorConfig `json:"embedded_config_generator,omitempty" yaml:"embedded_config_generator,omitempty"`

	// InstallerSettings are various settings that are being used in configurations that are being sent to clients through
	// embedded configurations.
	InstallerSettings *InstallerSettings `json:"installer_settings,omitempty" yaml:"installer_settings,omitempty"`

	// RegistrySettings are all settings that deal with registration requests that are being sent by clients.
	RegistrySettings *RegistrySettings `json:"registry_settings,omitempty" yaml:"registry_settings,omitempty"`

	ArtifactProviders *ArtifactProviders `json:"artifact_providers,omitempty" yaml:"artifact_providers,omitempty"`
}

type Servers struct {
	// ServerInsecure will instantiate an insecure server if it is not nil. The insecure server serves
	// all artifacts which are allowed to be served over an unsecured connection like the stage0 installer.
	ServerInsecure *InsecureServer `json:"insecure,omitempty" yaml:"insecure,omitempty"`

	// ServerSecure will instantiate a secure server if it is not nil. The secure server serves all artifacts
	// which must be served over a secure connection.
	ServerSecure *BindInfo `json:"secure,omitempty" yaml:"secure,omitempty"`
}

type InsecureServer struct {
	DynLL   *DynLL    `json:"dynll,omitempty" yaml:"dynll,omitempty"`
	Generic *BindInfo `json:"generic,omitempty" yaml:"generic,omitempty"`
}

// DynLL holds configuration for the dynamic linklocal insecure server listeners configuration. This mode allows
// for detection of neighbours based on configuration in Kubernetes. It will then start linklocal listeners only
// for those interfaces. Additionally this allows for advanced features like providing the location information
// to the stage0 installer instead of relying on it of being provided by the client itself.
type DynLL struct {
	// DeviceType is used while trying to self-detect who we are. The device could be either a switch or a server.
	// By default it tries to detect itself from both.
	DeviceType DeviceType `json:"device_type" yaml:"device_type"` // do not use omitempty here

	// DeviceName is used while trying to self-detect who we are. Depening on the device type it is trying to look
	// for itself as being either a fabric.githedgehog.com/Switch or a fabric.githedgehog.com/Server.
	// If this is empty the current OS hostname is used.
	DeviceName string `json:"device_name" yaml:"device_name"` // do not use omitempty here

	// ListeningPort is the port that will be used for all discovered ports that we need to listen on.
	ListeningPort uint16 `json:"listening_port,omitempty" yaml:"listening_port,omitempty"`
}

type DeviceType uint8

// DeviceTypeAuto means that the system is trying to detect itself as either being a switch or a server
const DeviceTypeAuto DeviceType = 0

// DeviceTypeServer means that the system is looking for an entry in fabric.githedgehog.com/Server
const DeviceTypeServer DeviceType = 1

// DeviceTypeSwitch means that the system is looking for an entry in fabric.githedgehog.com/Switch
const DeviceTypeSwitch DeviceType = 2

// BindInfo provides all the necessary information for binding to an address and configuring TLS as necessary.
type BindInfo struct {
	// Address is a set of addresses that the server should bind on. In practice multiple HTTP server instances
	// will be running, but all serving the same routes for the same purpose. At least one address must be
	// provided.
	Addresses []string `json:"addresses,omitempty" yaml:"addresses,omitempty"`

	// ClientCAPath points to a file containing one or more CA certificates that client certificates will be
	// validated against if a client certificate is provided. If this is empty, no client authentication will
	// be required on the TLS server. This setting is ignored if no server key and certificate were provided.
	ClientCAPath string `json:"client_ca,omitempty" yaml:"client_ca,omitempty"`

	// ServerKeyPath points to a file containing the server key used for the TLS server. If this is empty,
	// a plain HTTP server will be initiated.
	ServerKeyPath string `json:"server_key,omitempty" yaml:"server_key,omitempty"`

	// ServerCertPath points to a file containing the server certificate used for the TLS server. If `ServerKeyPath`
	// is set, this setting is required to be set.
	ServerCertPath string `json:"server_cert,omitempty" yaml:"server_cert,omitempty"`
}

type EmbeddedConfigGeneratorConfig struct {
	// KeyPath points to a file which contains the key which is being used to sign embedded configuration.
	KeyPath string `json:"config_signature_key,omitempty" yaml:"config_signature_key,omitempty"`

	// CertPath points to a certificate which is used to sign embedded configuration. Its public key must
	// match the key from `KeyPath`.
	CertPath string `json:"config_signature_cert,omitempty" yaml:"config_signature_cert,omitempty"`
}

// InstallerSettings are various settings that are being used in configurations that are being sent to clients through
// embedded configurations
type InstallerSettings struct {
	// ServerCAPath points to a file containing the CA certificate which signed the server certificate which is used
	// for the TLS server. This is necessary to provide it to clients in case they have not received it through an
	// alternative way.
	ServerCAPath string `json:"server_ca,omitempty" yaml:"server_ca,omitempty"`

	// ConfigSignatureCAPath points to a file containing the CA certificate which signed the signature certificate
	// which is used to sign the embedded configuration which is served with every staged installer.
	ConfigSignatureCAPath string `json:"config_signature_ca,omitempty" yaml:"config_signature_ca,omitempty"`

	// SecureServerName is the host name as it should match the TLS SAN for the server certificates that are used by clients to reach the seeder.
	// This server name will be used to generate various URLs which are going to be used in embedded configurations. If the service needs a
	// different port it needs to be included here (e.g. dasboot.example.com:8080).
	SecureServerName string `json:"secure_server_name,omitempty" yaml:"secure_server_name,omitempty"`

	// ControlVIP is the virtual IP of where to reach the control network services
	ControlVIP string `json:"control_vip,omitempty" yaml:"control_vip,omitempty"`

	// NTPServers are the NTP servers which will be configured on clients at installation time
	NTPServers []string `json:"ntp_servers,omitempty" yaml:"ntp_servers,omitempty"`

	// SyslogServers are the syslog servers which will be configured on clients at installation time
	SyslogServers []string `json:"syslog_servers,omitempty" yaml:"syslog_servers,omitempty"`

	// KubeSubnets are the subnets for which the seeder will generate routes that will be configured to access the management/control plane network
	// NOTE: subject to change in the future
	KubeSubnets []string `json:"kube_subnets,omitempty" yaml:"kube_subnets,omitempty"`
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

type ArtifactProviders struct {
	Directories   []string       `json:"directories,omitempty" yaml:"directories,omitempty"`
	OCITempDir    string         `json:"oci_temp_dir,omitempty" yaml:"oci_temp_dir,omitempty"`
	OCIRegistries []*OCIRegistry `json:"oci_registries,omitempty" yaml:"oci_registries,omitempty"`
}

type OCIRegistry struct {
	URL            string `json:"url,omitempty" yaml:"url,omitempty"`
	Username       string `json:"username,omitempty" yaml:"username,omitempty"`
	Password       string `json:"password,omitempty" yaml:"password,omitempty"`
	AccessToken    string `json:"access_token,omitempty" yaml:"access_token,omitempty"`
	RefreshToken   string `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
	ServerCAPath   string `json:"server_ca_path,omitempty" yaml:"server_ca_path,omitempty"`
	ClientCertPath string `json:"client_cert_path,omitempty" yaml:"cert_path,omitempty"`
	ClientKeyPath  string `json:"client_key_path,omitempty" yaml:"key_path,omitempty"`
}

// ReferenceConfig will be displayed when requested through the CLI
var ReferenceConfig = Config{
	Servers: &Servers{
		ServerInsecure: &InsecureServer{
			DynLL: &DynLL{
				DeviceType:    DeviceTypeAuto,
				DeviceName:    "",
				ListeningPort: 80,
			},
		},
		ServerSecure: &BindInfo{
			Addresses: []string{
				"192.168.42.1",
			},
			ClientCAPath:   "/etc/hedgehog/seeder/client-ca-cert.pem",
			ServerKeyPath:  "/etc/hedgehog/seeder/server-key.pem",
			ServerCertPath: "/etc/hedgehog/seeder/server-cert.pem",
		},
	},
	EmbeddedConfigGenerator: &EmbeddedConfigGeneratorConfig{
		KeyPath:  "/etc/hedgehog/seeder/embedded-config-generator-key.pem",
		CertPath: "/etc/hedgehog/seeder/embedded-config-generator-cert.pem",
	},
	InstallerSettings: &InstallerSettings{
		ServerCAPath:          "/etc/hedgehog/seeder/server-ca-cert.pem",
		ConfigSignatureCAPath: "/etc/hedgehog/seeder/embedded-config-generator-ca-cert.pem",
		SecureServerName:      "das-boot.hedgehog.svc.cluster.local",
		ControlVIP:            "192.168.42.1",
		NTPServers:            []string{"192.168.42.1", "192.168.42.2"},
		SyslogServers:         []string{"192.168.42.1"},
		KubeSubnets:           []string{"10.142.0.0/16", "10.143.0.0/16"},
	},
}

func marshalReferenceConfig() ([]byte, error) {
	b := &bytes.Buffer{}
	enc := yaml.NewEncoder(b)
	enc.SetIndent(2)
	if err := enc.Encode(&ReferenceConfig); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open '%s': %w", path, err)
	}
	defer f.Close()
	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config yaml decode: %w", err)
	}
	return &cfg, nil
}
