package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts/embedded"
	"go.githedgehog.com/dasboot/pkg/version"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var l = log.L()

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

			// load config
			cfg, err := loadConfig(ctx.Path("config"))
			if err != nil {
				return err
			}

			// create seeder
			// this is a bit stupid, and maybe we should just share the config structs
			// however, something told me that it is good to decouple those
			s, err := seeder.New(ctx.Context, &seeder.Config{
				InsecureServer: &seeder.BindInfo{
					Address:        cfg.Servers.ServerInsecure.Addresses,
					ClientCAPath:   cfg.Servers.ServerInsecure.ClientCAPath,
					ServerKeyPath:  cfg.Servers.ServerInsecure.ServerKeyPath,
					ServerCertPath: cfg.Servers.ServerInsecure.ServerCertPath,
				},
				SecureServer: &seeder.BindInfo{
					Address:        cfg.Servers.ServerSecure.Addresses,
					ClientCAPath:   cfg.Servers.ServerSecure.ClientCAPath,
					ServerKeyPath:  cfg.Servers.ServerSecure.ServerKeyPath,
					ServerCertPath: cfg.Servers.ServerSecure.ServerCertPath,
				},
				EmbeddedConfigGenerator: &seeder.EmbeddedConfigGeneratorConfig{
					KeyPath:  cfg.EmbeddedConfigGenerator.KeyPath,
					CertPath: cfg.EmbeddedConfigGenerator.CertPath,
				},
				InstallerSettings: &seeder.InstallerSettings{
					ServerCAPath:          cfg.InstallerSettings.ServerCAPath,
					ConfigSignatureCAPath: cfg.InstallerSettings.ConfigSignatureCAPath,
					SecureServerName:      cfg.InstallerSettings.SecureServerName,
					DNSServers:            cfg.InstallerSettings.DNSServers,
					NTPServers:            cfg.InstallerSettings.NTPServers,
					SyslogServers:         cfg.InstallerSettings.SyslogServers,
				},
				RegistrySettings: &seeder.RegistrySettings{
					CertPath: cfg.RegistrySettings.CertPath,
					KeyPath:  cfg.RegistrySettings.KeyPath,
				},
				// at last, something which can't get simply be passed from a config file
				ArtifactsProvider: artifacts.New(
					embedded.Provider(),
				),
			})
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
					ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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
		l.Fatal("integ-disk failed", zap.Error(err))
	}
}
