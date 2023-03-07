package seeder

import "go.githedgehog.com/dasboot/pkg/seeder/artifacts"

// Config is passed to a seeder instance. It will initialize the seeder based on this configuration.
type Config struct {
	// InsecureServer will instantiate an insecure server if it is not nil. The insecure server serves
	// all artifacts which are allowed to be served over an unsecured connection like the stage0 installer.
	InsecureServer *BindInfo

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
	// This server name will be used to generate various URLs which are going to be used in embedded configurations.
	SecureServerName string
}
