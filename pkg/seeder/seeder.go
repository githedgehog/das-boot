package seeder

import (
	"errors"
	"fmt"
)

// Interface interacts with a seeder instance.
type Interface interface {
	// Stop tells the seeder to gracefully shutdown all running servers.
	Stop()
}

type seeder struct {
	secureServer   *server
	insecureServer *server
}

var _ Interface = &seeder{}

var (
	ErrInvalidConfig = errors.New("seeder: invalid config")
)

func invalidConfigError(str string) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, str)
}

func New(config *Config) (Interface, error) {
	if config == nil {
		return nil, invalidConfigError("empty config")
	}
	if config.InsecureServer == nil && config.SecureServer == nil {
		return nil, invalidConfigError("neither InsecureServer nor SecureServer are set")
	}

	ret := &seeder{}

	if config.InsecureServer != nil {
		var err error
		ret.insecureServer, err = newServer(config.InsecureServer, insecureHandler())
		if err != nil {
			return nil, err
		}
	}

	if config.SecureServer != nil {
		var err error
		ret.secureServer, err = newServer(config.SecureServer, secureHandler())
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *seeder) Stop() {
}
