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
