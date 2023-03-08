package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is passed to a seeder instance. It will initialize the seeder based on this configuration.
type Config struct {
	// Servers holds all HTTP server settings
	Servers *Servers `json:"servers" yaml:"servers"`

	// EmbeddedConfigGenerator contains all settings which are necessary to generate embedded configuration for the
	// staged installer artifacts
	EmbeddedConfigGenerator *EmbeddedConfigGeneratorConfig `json:"embedded_config_generator" yaml:"embedded_config_generator"`

	// InstallerSettings are various settings that are being used in configurations that are being sent to clients through
	// embedded configurations.
	InstallerSettings *InstallerSettings `json:"installer_settings" yaml:"installer_settings"`
}

type Servers struct {
	// ServerInsecure will instantiate an insecure server if it is not nil. The insecure server serves
	// all artifacts which are allowed to be served over an unsecured connection like the stage0 installer.
	ServerInsecure *BindInfo `json:"insecure" yaml:"insecure"`

	// ServerSecure will instantiate a secure server if it is not nil. The secure server serves all artifacts
	// which must be served over a secure connection.
	ServerSecure *BindInfo `json:"secure" yaml:"secure"`
}

// BindInfo provides all the necessary information for binding to an address and configuring TLS as necessary.
type BindInfo struct {
	// Address is a set of addresses that the server should bind on. In practice multiple HTTP server instances
	// will be running, but all serving the same routes for the same purpose. At least one address must be
	// provided.
	Addresses []string `json:"addresses" yaml:"addresses"`

	// ClientCAPath points to a file containing one or more CA certificates that client certificates will be
	// validated against if a client certificate is provided. If this is empty, no client authentication will
	// be required on the TLS server. This setting is ignored if no server key and certificate were provided.
	ClientCAPath string `json:"client_ca" yaml:"client_ca"`

	// ServerKeyPath points to a file containing the server key used for the TLS server. If this is empty,
	// a plain HTTP server will be initiated.
	ServerKeyPath string `json:"server_key" yaml:"server_key"`

	// ServerCertPath points to a file containing the server certificate used for the TLS server. If `ServerKeyPath`
	// is set, this setting is required to be set.
	ServerCertPath string `json:"server_cert" yaml:"server_cert"`
}

type EmbeddedConfigGeneratorConfig struct {
	// KeyPath points to a file which contains the key which is being used to sign embedded configuration.
	KeyPath string `json:"config_signature_key" yaml:"config_signature_key"`

	// CertPath points to a certificate which is used to sign embedded configuration. Its public key must
	// match the key from `KeyPath`.
	CertPath string `json:"config_signature_cert" yaml:"config_signature_cert"`
}

// InstallerSettings are various settings that are being used in configurations that are being sent to clients through
// embedded configurations
type InstallerSettings struct {
	// ServerCAPath points to a file containing the CA certificate which signed the server certificate which is used
	// for the TLS server. This is necessary to provide it to clients in case they have not received it through an
	// alternative way.
	ServerCAPath string `json:"server_ca" yaml:"server_ca"`

	// ConfigSignatureCAPath points to a file containing the CA certificate which signed the signature certificate
	// which is used to sign the embedded configuration which is served with every staged installer.
	ConfigSignatureCAPath string `json:"config_signature_ca" yaml:"config_signature_ca"`

	// SecureServerName is the host name as it should match the TLS SAN for the server certificates that are used by clients to reach the seeder.
	// This server name will be used to generate various URLs which are going to be used in embedded configurations. If the service needs a
	// different port it needs to be included here (e.g. dasboot.example.com:8080).
	SecureServerName string `json:"secure_server_name" yaml:"secure_server_name"`

	// DNSServers are the DNS servers which will be configured on clients at installation time
	DNSServers []string `json:"dns_servers" yaml:"dns_servers"`

	// NTPServers are the NTP servers which will be configured on clients at installation time
	NTPServers []string `json:"ntp_servers" yaml:"ntp_servers"`

	// SyslogServers are the syslog servers which will be configured on clients at installation time
	SyslogServers []string `json:"syslog_servers" yaml:"syslog_servers"`
}

// ReferenceConfig will be displayed when requested through the CLI
var ReferenceConfig = Config{
	Servers: &Servers{
		ServerInsecure: &BindInfo{
			Addresses: []string{
				"fe80::808f:98ff:fe66:c45c",
			},
		},
		ServerSecure: &BindInfo{
			Addresses: []string{
				"192.168.42.11",
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
		DNSServers:            []string{"192.168.42.11", "192.168.42.12"},
		NTPServers:            []string{"192.168.42.11", "192.168.42.12"},
		SyslogServers:         []string{"192.168.42.11"},
	},
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
