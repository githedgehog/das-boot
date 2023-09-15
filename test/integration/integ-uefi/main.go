package main

import (
	"fmt"
	"os"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/urfave/cli/v2"
)

var l = log.L()

func main() {
	var acknowledgeDanger bool
	app := &cli.App{
		Name:                 "integ-disk",
		Usage:                "integration test for disk partitioning and formating",
		UsageText:            "integ-disk --acknowledge-danger",
		Description:          "Should be running in ONIE, and will find the ONIE partition and remove any unknown partitions and create a Hedgehog Identity partition and format it",
		Version:              version.Version,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "acknowledge-danger",
				Destination: &acknowledgeDanger,
				Usage:       "acknowledge that this is a dangerous operation",
				Required:    true,
			},
		},
		Action: func(ctx *cli.Context) error {
			// prevent a hooman from doing something stupid
			if !acknowledgeDanger {
				return fmt.Errorf("You *MUST* pass the --acknowledge-danger flag set to true to execute this integration test! **NOTE:** This is going to DELETE partitions if executed successfully, and create new ones in the process. Be sure you run this on a dedicated test bed system!")
			}

			// run the test
			return integUefi(ctx)
		},
	}

	l = log.NewZapWrappedLogger(zap.Must(log.NewSerialConsole(zapcore.DebugLevel, "console", true)))
	log.ReplaceGlobals(l)

	if err := app.Run(os.Args); err != nil {
		l.Fatal("integ-disk failed", zap.Error(err))
	}
}

func integUefi(_ *cli.Context) error {
	l.Info("Making ONIE default boot entry...")
	if err := partitions.MakeONIEDefaultBootEntryAndCleanup(); err != nil {
		l.Error("Making ONIE default boot entry failed", zap.Error(err))
		return err
	}
	l.Info("Successfully made ONIE default boot entry")
	return nil
}
