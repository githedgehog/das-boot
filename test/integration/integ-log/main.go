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

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/log/syslog"
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
		Name:                 "integ-log",
		Usage:                "integration test for zap logger for syslog and console combined",
		UsageText:            "integ-log --log-level debug --syslog-server 192.168.42.1",
		Description:          "Should be running in ONIE, needs networking configured, and should reconfigure network during logging",
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
				Usage: "NTP server IP addresses or hostnames or FQDNs",
				Value: "192.168.42.1",
			},
			&cli.GenericFlag{
				Name:  "syslog-facility",
				Usage: "syslog facility to use within syslog messages",
				Value: &defaultFacility,
			},
			&cli.UintFlag{
				Name:  "generate-messages",
				Usage: "number of messages to generate, 0 means indefinite number of messages",
				Value: 0,
			},
			&cli.DurationFlag{
				Name:  "generate-sleep",
				Usage: "duration to sleep between generated messages",
				Value: time.Second,
			},
		},
		Action: func(ctx *cli.Context) error {
			// run the test
			return integLog(ctx)
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to run integ-log: %s\n", err)
		os.Exit(1)
	}
}

func initLoggers(ctx context.Context, logDevelopment bool, logLevel zapcore.Level, logFormat, syslogServer string, syslogFacility syslog.Priority) (log.Interface, error) {
	// initialize zap serial logger
	var logger log.Interface
	serialLogger, err := log.NewSerialConsole(logLevel, logFormat, logDevelopment)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize serial logger: %w", err)
	}
	serialLogger.Info("Initialized serial logger from command-line settings", zap.Bool("logDevelopment", logDevelopment), zap.String("logLevel", logLevel.String()), zap.String("logFormat", logFormat))
	logger = log.NewZapWrappedLogger(serialLogger)

	// initialize zap syslog logger
	var syslogLogger *zap.Logger
	if syslogServer != "" {
		var err error
		syslogLogger, err = log.NewSyslog(ctx, logLevel, logDevelopment, syslogFacility, syslogServer, syslog.InternalLogger(serialLogger))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize syslog logger: %w", err)
		}
		serialLogger.Info("Initialized syslog logger from command-line settings", zap.String("syslogServer", syslogServer), zap.String("syslogFacility", syslogFacility.String()))

		// now create a "tee" logger for both serial and syslog destinations
		logger = log.NewZapWrappedLogger(serialLogger, syslogLogger)
	}
	return logger, nil
}

func integLog(ctx *cli.Context) error {
	// CLI flags
	logDevelopment := ctx.Bool("log-development")
	logLevel := *ctx.Generic("log-level").(*zapcore.Level)
	logFormat := ctx.String("log-format")
	syslogServer := ctx.String("syslog-server")
	syslogFacility := *ctx.Generic("syslog-facility").(*syslog.Priority)
	generateMessages := ctx.Uint("generate-messages")
	generateSleep := ctx.Duration("generate-sleep")

	// init loggers
	logger, err := initLoggers(ctx.Context, logDevelopment, logLevel, logFormat, syslogServer, syslogFacility)
	if err != nil {
		return err
	}

	// now replace the global logger
	log.ReplaceGlobals(logger)

	// now generate log messages
	if generateMessages > 0 {
		for i := uint(0); i < generateMessages; i++ {
			log.L().Info("generated log message", zap.Uint("i", i))
			time.Sleep(generateSleep)
		}
		return nil
	}

	// indefinitive log messages case
	i := uint(0)
	for {
		log.L().Info("generated log message", zap.Uint("i", i))
		time.Sleep(generateSleep)
		i++
	}
}
