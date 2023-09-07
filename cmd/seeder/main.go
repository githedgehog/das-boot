package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts/embedded"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts/file"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts/oras"
	seederconfig "go.githedgehog.com/dasboot/pkg/seeder/config"
	"go.githedgehog.com/dasboot/pkg/version"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultLogLevel = zapcore.InfoLevel
)

var l = log.NewZapWrappedLogger(zap.Must(log.NewSerialConsole(zapcore.DebugLevel, "console", true)))

var description = `
This is the Hedgehog SONiC devic provisioning server. It needs to be running on
a dedicated VLAN with access to all management capabilities, and it needs to
run on untagged ports on link-local IP addresses to serve the initial staged
installer.

There are several components that need to be configured:
- bind info / listeners for the insecure server (serving stage0 and IPAM only)
- bind info / listeners for the secure server
- the artifacts provider which can make installers available from different
  sources
- the embedded config generator
- general installer settings which will be relayed to clients at the right time

More than one instance of the seeder should be running. And a seeder which
serves at least an insecure server should be running on all switch interconnect
ports.
`

func main() {
	app := &cli.App{
		Name:        "seeder",
		Usage:       "network device provisioning tool",
		UsageText:   "seeder",
		Description: description[1 : len(description)-1],
		Version:     version.Version,
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:  "log-level",
				Usage: "minimum log level to log at",
				Value: &defaultLogLevel,
			},
			&cli.StringFlag{
				Name:  "log-format",
				Usage: "log format to use: json or console",
				Value: "json",
			},
			&cli.BoolFlag{
				Name:  "log-development",
				Usage: "enables development log settings",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "reference-config",
				Usage: "prints a reference config to stdout and exits",
			},
			&cli.PathFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "load configuration from `FILE`",
				Value:   "/etc/hedgehog/seeder/config.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			// display reference config if requested
			if ctx.Bool("reference-config") {
				b, err := marshalReferenceConfig()
				if err != nil {
					return err
				}
				_, err = os.Stdout.Write(append(b, []byte("\n")...))
				return err
			}

			// initialize logger
			l = log.NewZapWrappedLogger(zap.Must(log.NewSerialConsole(
				*ctx.Generic("log-level").(*zapcore.Level),
				ctx.String("log-format"),
				ctx.Bool("log-development"),
			)))
			defer func() {
				if err := l.Sync(); err != nil {
					l.Debug("Flushing logger failed", zap.Error(err))
				}
			}()
			log.ReplaceGlobals(l)

			// load config
			cfg, err := loadConfig(ctx.Path("config"))
			if err != nil {
				return err
			}
			l.Info("Successfully loaded configuration", zap.String("path", ctx.Path("config")), zap.Reflect("config", cfg))

			// create seeder

			// this is a bit stupid, and maybe we should just share the config structs
			// however, something told me that it is good to decouple those
			// so translate the configs
			c := &seederconfig.SeederConfig{}
			if cfg.Servers != nil {
				if cfg.Servers.ServerInsecure != nil {
					c.InsecureServer = &seederconfig.InsecureServer{}
					if cfg.Servers.ServerInsecure.DynLL != nil {
						c.InsecureServer.DynLL = &seederconfig.DynLL{
							DeviceType:    seederconfig.DeviceType(cfg.Servers.ServerInsecure.DynLL.DeviceType),
							DeviceName:    cfg.Servers.ServerInsecure.DynLL.DeviceName,
							ListeningPort: cfg.Servers.ServerInsecure.DynLL.ListeningPort,
						}
					} else if cfg.Servers.ServerInsecure.Generic != nil {
						c.InsecureServer.Generic = &seederconfig.BindInfo{
							Address:        cfg.Servers.ServerInsecure.Generic.Addresses,
							ClientCAPath:   cfg.Servers.ServerInsecure.Generic.ClientCAPath,
							ServerKeyPath:  cfg.Servers.ServerInsecure.Generic.ServerKeyPath,
							ServerCertPath: cfg.Servers.ServerInsecure.Generic.ServerCertPath,
						}
					}
				}
				if cfg.Servers.ServerSecure != nil {
					c.SecureServer = &seederconfig.BindInfo{
						Address:        cfg.Servers.ServerSecure.Addresses,
						ClientCAPath:   cfg.Servers.ServerSecure.ClientCAPath,
						ServerKeyPath:  cfg.Servers.ServerSecure.ServerKeyPath,
						ServerCertPath: cfg.Servers.ServerSecure.ServerCertPath,
					}
				}
			}
			if cfg.EmbeddedConfigGenerator != nil {
				c.EmbeddedConfigGenerator = &seederconfig.EmbeddedConfigGeneratorConfig{
					KeyPath:  cfg.EmbeddedConfigGenerator.KeyPath,
					CertPath: cfg.EmbeddedConfigGenerator.CertPath,
				}
			}
			if cfg.InstallerSettings != nil {
				c.InstallerSettings = &seederconfig.InstallerSettings{
					ServerCAPath:          cfg.InstallerSettings.ServerCAPath,
					ConfigSignatureCAPath: cfg.InstallerSettings.ConfigSignatureCAPath,
					SecureServerName:      cfg.InstallerSettings.SecureServerName,
					DNSServers:            cfg.InstallerSettings.DNSServers,
					NTPServers:            cfg.InstallerSettings.NTPServers,
					SyslogServers:         cfg.InstallerSettings.SyslogServers,
				}
				if len(cfg.InstallerSettings.Routes) > 0 {
					routes := make([]*seederconfig.Route, 0, len(cfg.InstallerSettings.Routes))
					for _, route := range cfg.InstallerSettings.Routes {
						r := &seederconfig.Route{
							Gateway:      route.Gateway,
							Destinations: make([]string, len(route.Destinations)),
						}
						copy(r.Destinations, route.Destinations)
						routes = append(routes, r)
					}
					c.InstallerSettings.Routes = routes
				}
			}
			if cfg.RegistrySettings != nil {
				c.RegistrySettings = &seederconfig.RegistrySettings{
					CertPath: cfg.RegistrySettings.CertPath,
					KeyPath:  cfg.RegistrySettings.KeyPath,
				}
			}

			// we always add the embedded provider
			artifactProviders := []artifacts.Provider{embedded.Provider()}
			if cfg.ArtifactProviders != nil {
				if len(cfg.ArtifactProviders.Directories) > 0 {
					for _, dir := range cfg.ArtifactProviders.Directories {
						artifactProviders = append(artifactProviders, file.Provider(dir))
					}
				}
				if len(cfg.ArtifactProviders.OCIRegistries) > 0 {
					for _, ociReg := range cfg.ArtifactProviders.OCIRegistries {
						var opts []oras.ProviderOption
						if ociReg.AccessToken != "" {
							opts = append(opts, oras.ProviderOptionAccessToken(ociReg.AccessToken))
						}
						if ociReg.RefreshToken != "" {
							opts = append(opts, oras.ProviderOptionRefreshToken(ociReg.RefreshToken))
						}
						if ociReg.Username != "" && ociReg.Password != "" {
							opts = append(opts, oras.ProviderOptionBasicAuth(ociReg.Username, ociReg.Password))
						}
						if ociReg.ClientCertPath != "" && ociReg.ClientKeyPath != "" {
							opts = append(opts, oras.ProviderOptionTLSClientAuth(ociReg.ClientCertPath, ociReg.ClientKeyPath))
						}
						if ociReg.ServerCAPath != "" {
							opts = append(opts, oras.ProviderOptionServerCA(ociReg.ServerCAPath))
						}
						prov, err := oras.Provider(ctx.Context, ociReg.URL, opts...)
						if err != nil {
							return fmt.Errorf("oras provider: %w", err)
						}
						artifactProviders = append(artifactProviders, prov)
					}
				}
			}

			// the artifacts provider
			c.ArtifactsProvider = artifacts.New(
				artifactProviders...,
			)

			// now create the seeder
			l.Debug("Translated seeder config", zap.Reflect("seederConfig", c))
			s, err := seeder.New(ctx.Context, c)
			if err != nil {
				return err
			}

			// register TERM and INT signals
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

			// now start the seeder - and wait for things to happen
			l.Info("Seeder starting...")
			s.Start()
			var wg sync.WaitGroup
			var signalReceived bool
		mainLoop:
			for {
				select {
				case sig := <-signals:
					if signalReceived {
						l.Info("received additional signal, ignoring...", zap.String("signal", sig.String()))
						break
					}
					l.Info("received signal, stopping seeder...", zap.String("signal", sig.String()))
					signalReceived = true
					wg.Add(1)
					ctx, cancel := context.WithTimeout(ctx.Context, time.Minute)
					go func(ctx context.Context, cancel context.CancelFunc) {
						defer cancel()
						s.Stop(ctx)
						l.Info("seeder shutdown complete")
						wg.Done()
					}(ctx, cancel)
				case err, ok := <-s.Err():
					if ok {
						l.Error("error from seeder", zap.Error(err))
					}
				case <-s.Done():
					l.Info("Seeder stopped")
					break mainLoop
				}
			}
			l.Debug("Waiting for seeder shutdown to complete...")
			wg.Wait()
			l.Debug("Finished waiting for seeder shutdown")
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		l.Fatal("seeder failed", zap.Error(err))
	}
}
