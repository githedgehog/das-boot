package seeder

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	"go.githedgehog.com/dasboot/pkg/seeder/registration"
	"go.uber.org/zap"
)

// Interface interacts with a seeder instance.
type Interface interface {
	// Start will start the seeder and its servers in the background. This function will return
	// probably even before listeners are start.
	Start()

	// Stop tells the seeder to stop all running servers. It is trying a graceful shutdown at first,
	// but will close the servers if the context timeouts or after 30 seconds if the context did not
	// timeout before that.
	Stop(context.Context)

	// Done returns a channel which will be closed once all servers that were started with `Start()`
	// have finished listening.
	Done() <-chan struct{}

	// Err returns a channel which will get errors of servers during startup pushed
	Err() <-chan error
}

type seeder struct {
	done              chan struct{}
	err               chan error
	ecg               *embeddedConfigGenerator
	secureServer      *server
	insecureServer    *server
	artifactsProvider artifacts.Provider
	installerSettings *loadedInstallerSettings
	registry          *registration.Processor
}

var _ Interface = &seeder{}
var _ controlplane.Client = &seeder{}

var (
	ErrInvalidConfig           = errors.New("seeder: invalid config")
	ErrEmbeddedConfigGenerator = errors.New("seeder: embedded config generator")
	ErrInstallerSettings       = errors.New("seeder: installer settings")
	ErrRegistrySettings        = errors.New("seeder: registry settings")
)

func invalidConfigError(str string) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, str)
}

func embeddedConfigGeneratorError(str string) error {
	return fmt.Errorf("%w: %s", ErrEmbeddedConfigGenerator, str)
}

func installerSettingsError(err error) error {
	return fmt.Errorf("%w: %w", ErrInstallerSettings, err)
}

func registrySettingsError(err error) error {
	return fmt.Errorf("%w: %w", ErrRegistrySettings, err)
}

func New(ctx context.Context, config *Config) (Interface, error) {
	if config == nil {
		return nil, invalidConfigError("empty config")
	}
	if config.InsecureServer == nil && config.SecureServer == nil {
		return nil, invalidConfigError("neither InsecureServer nor SecureServer are set")
	}
	if config.ArtifactsProvider == nil {
		return nil, invalidConfigError("no artifacts provider")
	}
	if config.InstallerSettings == nil {
		return nil, invalidConfigError("no installer settings provided")
	}

	ret := &seeder{
		done:              make(chan struct{}),
		artifactsProvider: config.ArtifactsProvider,
	}

	// load the embedded configuration generator
	if err := ret.intializeEmbeddedConfigGenerator(config.EmbeddedConfigGenerator); err != nil {
		return nil, embeddedConfigGeneratorError(err.Error())
	}

	// load the installer settings
	if err := ret.initializeInstallerSettings(config.InstallerSettings); err != nil {
		return nil, installerSettingsError(err)
	}

	// load the registry settings
	if err := ret.initializeRegistrySettings(ctx, config.RegistrySettings); err != nil {
		return nil, registrySettingsError(err)
	}

	// this section sets up the servers
	errChLen := 0
	if config.InsecureServer != nil {
		var err error
		ret.insecureServer, err = newServer(config.InsecureServer, ret.insecureHandler())
		if err != nil {
			return nil, err
		}
		errChLen += len(config.InsecureServer.Address)
	}

	if config.SecureServer != nil {
		var err error
		ret.secureServer, err = newServer(config.SecureServer, ret.secureHandler())
		if err != nil {
			return nil, err
		}
		errChLen += len(config.SecureServer.Address)
	}
	ret.err = make(chan error, errChLen)

	return ret, nil
}

func (s *seeder) Start() {
	// fire up our servers
	var wg sync.WaitGroup
	wg.Add(2)
	if s.insecureServer != nil {
		go s.insecureServer.Start()
		go func() {
			for {
				err, ok := <-s.insecureServer.Err()
				if !ok {
					wg.Done()
					return
				}
				s.err <- err
			}
		}()
	}

	if s.secureServer != nil {
		go s.secureServer.Start()
		go func() {
			for {
				err, ok := <-s.secureServer.Err()
				if !ok {
					wg.Done()
					return
				}
				s.err <- err
			}
		}()
	}

	// we're all done once the secure and insecure servers are done
	go func() {
		if s.insecureServer != nil {
			<-s.insecureServer.Done()
		}
		if s.secureServer != nil {
			<-s.secureServer.Done()
		}
		wg.Wait()
		close(s.done)
		close(s.err)
	}()
}

func (s *seeder) Done() <-chan struct{} {
	return s.done
}

func (s *seeder) Err() <-chan error {
	return s.err
}

func (s *seeder) Stop(pctx context.Context) {
	// whatever context we get passed in, we will definitely cancel after 30 seconds
	ctx, cancel := context.WithTimeout(pctx, time.Second*30)
	defer cancel()

	// try graceful shutdown first
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		if err := s.insecureServer.Shutdown(ctx); err != nil {
			l.Warn("insecure server: graceful shutdown failed", zap.Error(err))
		}
		wg.Done()
	}()
	go func() {
		if err := s.secureServer.Shutdown(ctx); err != nil {
			l.Warn("secure server: graceful shutdown failed", zap.Error(err))
		}
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(done)
	}()

	// if graceful shutdown fails, just tear it down
	select {
	case <-ctx.Done():
		if err := s.insecureServer.Close(); err != nil {
			l.Debug("insecure server: error on close", zap.Error(err))
		}
		if err := s.secureServer.Close(); err != nil {
			l.Debug("secure server: error on close", zap.Error(err))
		}
	case <-done:
		// graceful shutdown was successful
	}
}
