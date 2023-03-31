package hhagentprov

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"

	"go.githedgehog.com/dasboot/pkg/config"
	configstage "go.githedgehog.com/dasboot/pkg/hhagentprov/config"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/stage"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"
)

var l = log.L()

var ErrExecution = errors.New("unrecoverable execution error encountered")

func executionError(err error) error {
	return fmt.Errorf("%w: %w", ErrExecution, err)
}

func ReadConfig(caPool *x509.CertPool) (*configstage.HedgehogAgentProvisioner, error) {
	// open and read executable into memory
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("returning executable path: %w", err)
	}

	f, err := os.Open(exePath)
	if err != nil {
		return nil, fmt.Errorf("open executable '%s': %w", exePath, err)
	}
	defer f.Close()

	exeBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading executable '%s': %w", exePath, err)
	}

	// now read embedded config for the first time
	// compared to stage 0 we require signature verification at this stage
	var cfg configstage.HedgehogAgentProvisioner
	if err := config.ReadEmbeddedConfig(exeBytes, &cfg, caPool); err != nil {
		return nil, fmt.Errorf("reading embedded config ignoring signature: %w", err)
	}

	// this completes reading the configuration
	return &cfg, nil
}

func Run(ctx context.Context, override *configstage.HedgehogAgentProvisioner, logSettings *stage.LogSettings) (runErr error) {
	// setup some console logging first
	// NOTE: we'll throw this away immediately after we've read the staging info
	// so this is really just for until then
	// TODO: this essentially should never fail, so should be implemented differently I guess
	if err := stage.InitializeGlobalLogger(ctx, logSettings); err != nil {
		return fmt.Errorf("hedgehog-agent-provisioner: failed to initialize logger: %w", err)
	}
	l = log.L()
	defer func() {
		if err := l.Sync(); err != nil {
			l.Debug("Flushing logger failed", zap.Error(err))
		}
	}()
	l.Info("Hedgehog Agent Provisioner execution starting", zap.String("version", version.Version))
	l.Info("System environment", zap.Strings("env", os.Environ()))

	// read ONIE env information
	onieEnv := stage.GetOnieEnv()
	l.Info("ONIE environment", zap.Reflect("onieEnv", onieEnv))

	// Read the staging info first, otherwise we are lost anyways
	si, err := stage.ReadStagingInfo()
	if err != nil {
		l.Error("Reading staging info", zap.Error(err))
		return executionError(fmt.Errorf("reading staging info: %w", err))
	}

	// reinitialize global logger
	// TODO: merge log settings I guess? will figure out what constitutes a change from the program flags
	l.Debug("Reinitializing global logger again", zap.Reflect("logSettings", &si.LogSettings))
	if err := stage.InitializeGlobalLogger(ctx, &si.LogSettings); err != nil {
		l.Warn("Reinitializing global logger failed", zap.Error(err))
	} else {
		l = log.L()
		l.Info("Reinitialized global logger from staging info", zap.Reflect("logSettings", &si.LogSettings))
	}

	// get the config signature CA pool, without it we should not read and trust our embedded configuration
	configCAPool, err := si.ConfigSignatureCAPool()
	if err != nil {
		l.Error("Initializing Config Signature CA Pool failed", zap.Error(err))
		return executionError(fmt.Errorf("initializing config signature CA pool: %w", err))
	}

	// read embedded config now
	embedded, err := ReadConfig(configCAPool)
	if err != nil {
		l.Error("Reading embedded config failed", zap.Error(err))
		return executionError(err)
	}
	l.Info("Read embedded configuration", zap.Reflect("config", embedded))

	// Merge configs with override
	cfg := configstage.MergeConfigs(embedded, override)
	if err := cfg.Validate(); err != nil {
		l.Error("Merged config validation error", zap.Error(err))
		return executionError(fmt.Errorf("merged config validation: %w", err))
	}
	if override != nil {
		l.Info("Merged override configuration", zap.Reflect("config", cfg))
	}

	// discover partitions
	devices := partitions.Discover()

	// now mount the identity partition
	// this step fully initializes and prepares the partition for our usage
	identityPartition, err := stage.MountIdentityPartition(l, devices, onieEnv.Platform)
	if err != nil {
		l.Error("Identity Partition could not be opened/mounted/created", zap.Error(err))
		return executionError(fmt.Errorf("opening identity partition: %w", err))
	}
	l.Info("Opened Hedgehog Identity Partition successfully")

	_, err = stage.SeederHTTPClient(si.ServerCA, identityPartition)
	if err != nil {
		l.Error("Building HTTP client for downloading agent and agent config failed", zap.Error(err))
		return executionError(err)
	}

	// TODO: implement

	// we are done here
	l.Info("Hedgehog Agent Provisioner completed successfully")
	return nil
}
