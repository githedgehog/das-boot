package main

import (
	"errors"
	"fmt"
	"os"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/log/syslog"
	"go.githedgehog.com/dasboot/pkg/stage"
	"go.githedgehog.com/dasboot/pkg/stage0"
	"go.githedgehog.com/dasboot/pkg/stage0/config"
	"go.githedgehog.com/dasboot/pkg/version"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultLogLevel = zapcore.InfoLevel
	defaultFacility = syslog.LOG_LOCAL0
)

func main() {
	app := &cli.App{
		Name:                 "stage0",
		Usage:                "configures system network and basic operating system functionalities",
		UsageText:            "stage0",
		Description:          "Should be running in ONIE, and is the first of a series of installer stages within DAS BOOT",
		Version:              version.Version,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:  "log-level",
				Usage: "minimum log level to log at",
				Value: &defaultLogLevel,
			},
			&cli.StringFlag{
				Name:  "log-format",
				Usage: "log format to use: json or console (only affects serial console)",
				Value: "console",
			},
			&cli.BoolFlag{
				Name:  "log-development",
				Usage: "enables development log settings",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "syslog-server",
				Usage: "syslog server IP addresses or hostnames or FQDNs",
			},
			&cli.GenericFlag{
				Name:  "syslog-facility",
				Usage: "syslog facility to use within syslog messages",
				Value: &defaultFacility,
			},
			&cli.PathFlag{
				Name:  "config",
				Usage: "optional configuration file to load which can override settings of the embedded configuration",
			},
		},
		Action: func(ctx *cli.Context) error {
			return runStage0(ctx)
		},
	}

	if err := app.Run(os.Args); err != nil {
		if errors.Is(err, stage0.ErrExecution) {
			log.L().Fatal("runtime error", zap.Error(err))
		}
		fmt.Fprintf(os.Stderr, "FATAL: failed to run stage 0: %s\n", err)
		os.Exit(1)
	}
}

func runStage0(ctx *cli.Context) error {
	// read optional configuration file first
	configPath := ctx.Path("config")
	var cfg *config.Stage0
	if configPath != "" {
		var err error
		cfg, err = config.ReadFromFile(configPath)
		if err != nil {
			return err
		}
	}

	// CLI flags for log settings
	var syslogServers []string
	syslogServer := ctx.String("syslog-server")
	if syslogServer != "" {
		syslogServers = append(syslogServers, syslogServer)
	}
	logSettings := &stage.LogSettings{
		Development:    ctx.Bool("log-development"),
		Level:          *ctx.Generic("log-level").(*zapcore.Level),
		Format:         ctx.String("log-format"),
		SyslogServers:  syslogServers,
		SyslogFacility: *ctx.Generic("syslog-facility").(*syslog.Priority),
	}
	return stage0.Run(ctx.Context, cfg, logSettings)
}
