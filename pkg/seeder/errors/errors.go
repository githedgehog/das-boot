package errors

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig           = errors.New("seeder: invalid config")
	ErrEmbeddedConfigGenerator = errors.New("seeder: embedded config generator")
	ErrInstallerSettings       = errors.New("seeder: installer settings")
	ErrRegistrySettings        = errors.New("seeder: registry settings")
)

func InvalidConfigError(str string) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, str)
}

func EmbeddedConfigGeneratorError(str string) error {
	return fmt.Errorf("%w: %s", ErrEmbeddedConfigGenerator, str)
}

func InstallerSettingsError(err error) error {
	return fmt.Errorf("%w: %w", ErrInstallerSettings, err)
}

func RegistrySettingsError(err error) error {
	return fmt.Errorf("%w: %w", ErrRegistrySettings, err)
}
