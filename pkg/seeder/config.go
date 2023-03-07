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

	ArtifactsProvider artifacts.Provider
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
