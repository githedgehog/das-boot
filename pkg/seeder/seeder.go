package seeder

import (
	"context"
	"sync"
	"time"

	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.githedgehog.com/dasboot/pkg/seeder/config"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	"go.githedgehog.com/dasboot/pkg/seeder/errors"
	"go.githedgehog.com/dasboot/pkg/seeder/registration"
	"go.githedgehog.com/dasboot/pkg/seeder/server"
	"go.githedgehog.com/dasboot/pkg/seeder/server/dynll"
	"go.githedgehog.com/dasboot/pkg/seeder/server/generic"
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
	secureServer      server.ControlInterface
	insecureServer    server.ControlInterface
	artifactsProvider artifacts.Provider
	installerSettings *loadedInstallerSettings
	registry          *registration.Processor
}

var _ Interface = &seeder{}
var _ controlplane.Client = &seeder{}

func New(ctx context.Context, cfg *config.SeederConfig) (Interface, error) {
	if cfg == nil {
		return nil, errors.InvalidConfigError("empty config")
	}
	if cfg.InsecureServer == nil && cfg.SecureServer == nil {
		return nil, errors.InvalidConfigError("neither InsecureServer nor SecureServer are set")
	}
	if cfg.ArtifactsProvider == nil {
		return nil, errors.InvalidConfigError("no artifacts provider")
	}
	if cfg.InstallerSettings == nil {
		return nil, errors.InvalidConfigError("no installer settings provided")
	}

	ret := &seeder{
		done:              make(chan struct{}),
		artifactsProvider: cfg.ArtifactsProvider,
	}

	// load the embedded configuration generator
	if err := ret.intializeEmbeddedConfigGenerator(cfg.EmbeddedConfigGenerator); err != nil {
		return nil, errors.EmbeddedConfigGeneratorError(err.Error())
	}

	// load the installer settings
	if err := ret.initializeInstallerSettings(cfg.InstallerSettings); err != nil {
		return nil, errors.InstallerSettingsError(err)
	}

	// load the registry settings
	if err := ret.initializeRegistrySettings(ctx, cfg.RegistrySettings); err != nil {
		return nil, errors.RegistrySettingsError(err)
	}

	// this section sets up the servers
	errChLen := 0
	if cfg.InsecureServer != nil {
		if cfg.InsecureServer.DynLL != nil {
			var err error
			ret.insecureServer, err = dynll.NewDynLLServer(cfg.InsecureServer.DynLL, ret.insecureHandler())
			if err != nil {
				return nil, err
			}
			errChLen += 100
		} else if cfg.InsecureServer.Generic != nil {
			var err error
			ret.insecureServer, err = generic.NewGenericServer(cfg.InsecureServer.Generic, ret.insecureHandler())
			if err != nil {
				return nil, err
			}
			errChLen += len(cfg.InsecureServer.Generic.Address)
		}
	}

	if cfg.SecureServer != nil {
		var err error
		ret.secureServer, err = generic.NewGenericServer(cfg.SecureServer, ret.secureHandler())
		if err != nil {
			return nil, err
		}
		errChLen += len(cfg.SecureServer.Address)
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
