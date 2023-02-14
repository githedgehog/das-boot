package config

// ConfigVersion represents the version of the configuration structure.
// The zero value is an invalid version number.
//
// NOTE: Do **not** confuse this with `HeaderVersion` which tracks the
// header version format.
type ConfigVersion uint8

// EmbeddedConfigVersion should be implemented by all config structures.
// It tracks the version of the configuration structure itself.
// It is supposed to prevent inconsistencies between generator and the
// staged installers itself.
// For example, if the generator generates a version 2 of the configuration
// of the stage 0 installer, but the installer itself does not understand
// it, it can exit gracefully.
// Technically this should never happen because they are currently build
// together, however, this is good practice early on.
type EmbeddedConfigVersion interface {
	// Version returns the version of the configuration structure
	ConfigVersion() ConfigVersion

	// IsSupportedConfigVersion tests if the provided config version
	// is supported by the code of the configuration structure
	IsSupportedConfigVersion(v ConfigVersion) bool
}
