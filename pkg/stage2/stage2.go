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

package stage2

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.githedgehog.com/dasboot/pkg/config"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/stage"
	configstage "go.githedgehog.com/dasboot/pkg/stage2/config"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var l = log.L()

var ErrExecution = errors.New("unrecoverable execution error encountered")

func executionError(err error) error {
	return fmt.Errorf("%w: %w", ErrExecution, err)
}

func ReadConfig(caPool *x509.CertPool) (*configstage.Stage2, error) {
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
	var cfg configstage.Stage2
	if err := config.ReadEmbeddedConfig(exeBytes, &cfg, caPool); err != nil {
		return nil, fmt.Errorf("reading embedded config: %w", err)
	}

	// this completes reading the stage0 configuration
	return &cfg, nil
}

func Run(ctx context.Context, override *configstage.Stage2, logSettings *stage.LogSettings) (runErr error) {
	// setup some console logging first
	// NOTE: we'll throw this away immediately after we've read the staging info
	// so this is really just for until then
	// TODO: this essentially should never fail, so should be implemented differently I guess
	if err := stage.InitializeGlobalLogger(ctx, logSettings); err != nil {
		return fmt.Errorf("stage0: failed to initialize logger: %w", err)
	}
	l = log.L()
	defer func() {
		if err := l.Sync(); err != nil {
			l.Debug("Flushing logger failed", zap.Error(err))
		}
	}()
	l.Info("Stage 2 execution starting", zap.String("version", version.Version))
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

	// now mount (or create and mount) the identity partition
	// this step fully initializes and prepares the partition for our usage
	identityPartition, err := stage.MountIdentityPartition(l, devices, onieEnv.Platform)
	if err != nil {
		l.Error("Identity Partition could not be opened/mounted/created", zap.Error(err))
		return executionError(fmt.Errorf("opening identity partition: %w", err))
	}
	l.Info("Opened Hedgehog Identity Partition successfully")

	hc, err := stage.SeederHTTPClient(si.ServerCA, identityPartition)
	if err != nil {
		l.Error("Building HTTP client for downloading stage 2 failed", zap.Error(err))
		return executionError(err)
	}

	switch onieEnv.BootReason {
	case "install":
		if err := runNosInstall(ctx, hc, cfg, si, onieEnv); err != nil {
			l.Error("NOS installation failure", zap.Error(err))
			return executionError(fmt.Errorf("NOS installation: %w", err))
		}
	case "update":
		if err := runOnieUpdate(ctx, hc, cfg, si, onieEnv); err != nil {
			l.Error("ONIE update failure", zap.Error(err))
			return executionError(fmt.Errorf("NOS installation: %w", err))
		}
	default:
		l.Warn("Unrecognized ONIE boot reason, assuming NOS installation", zap.String("boot_reason", onieEnv.BootReason))
		if err := runNosInstall(ctx, hc, cfg, si, onieEnv); err != nil {
			l.Error("NOS installation failure", zap.Error(err))
			return executionError(fmt.Errorf("NOS installation: %w", err))
		}
	}

	// we are done here
	l.Info("Stage 2 completed successfully")
	return nil
}

func runNosInstall(ctx context.Context, hc *http.Client, cfg *configstage.Stage2, si *stage.StagingInfo, onie *stage.OnieEnv) (funcErr error) {
	// Build donwload URL: cfg URL + ONIE platform
	url, err := stage.BuildURL(cfg.NOSInstallerURL, onie.Platform)
	if err != nil {
		l.Error("Building NOS installer URL failed", zap.String("url", cfg.NOSInstallerURL), zap.String("platform", onie.Platform), zap.Error(err))
		return fmt.Errorf("building NOS installer URL: %w", err)
	}

	// NOS download
	nosPath := filepath.Join(si.StagingDir, "nos-install")
	l.Info("Downloading NOS installer now...", zap.String("url", url), zap.String("dest", nosPath))
	if err := stage.DownloadExecutable(ctx, hc, url, nosPath, time.Second*120); err != nil {
		l.Error("Downloading NOS installer failed", zap.String("url", url), zap.String("dest", nosPath), zap.Error(err))
		return fmt.Errorf("NOS download: %w", err)
	}
	l.Info("Downloading NOS installer completed", zap.String("url", url), zap.String("dest", nosPath))

	// for every following error we need to ensure that we make ONIE the default boot option again, because:
	// - the NOS installation might have worked, but not the agent installation which is still a fatal error
	// - the NOS installation half-assed, and we don't know what that means
	defer func() {
		if funcErr != nil {
			l.Info("Trying to ensure that ONIE stays the default boot option...")
			if err := partitions.MakeONIEDefaultBootEntryAndCleanup(); err != nil {
				l.Error("Making ONIE the default boot option failed", zap.Error(err))
			}
			l.Info("ONIE is the default boot option again")
		}
	}()

	// NOS install
	l.Info("Executing NOS installer now...")
	subctx, cancel := context.WithCancel(ctx)
	nosCmd := exec.CommandContext(ctx, nosPath)
	nosCmd.Env = append(nosCmd.Environ(), "ZTP=n")
	nosCmd.Stdin = os.Stdin
	nosCmd.Stderr = log.NewSinkWithLogger(subctx, l, zapcore.InfoLevel, zap.String("app", "nos-install"), zap.String("stream", "stderr"))
	nosCmd.Stdout = log.NewSinkWithLogger(subctx, l, zapcore.InfoLevel, zap.String("app", "nos-install"), zap.String("stream", "stdout"))
	if err := nosCmd.Run(); err != nil {
		l.Error("NOS installer execution failed", zap.String("bin", nosPath), zap.Error(err))
		cancel()
		return fmt.Errorf("NOS installer execution: %w", err)
	}
	l.Info("NOS installation completed")
	cancel()

	// if this is Hedgehog SONiC, we are going to run our additional provisioners as well
	if cfg.NOSType == "hedgehog_sonic" && len(cfg.HedgehogSonicProvisioners) > 0 {
		// building a list of names for logging
		names := make([]string, 0, len(cfg.HedgehogSonicProvisioners))
		for _, p := range cfg.HedgehogSonicProvisioners {
			names = append(names, p.Name)
		}

		l.Info("Hedgehog SONiC NOS installation detected. Running all additional Hedgehog SONiC Provisioners...", zap.String("nos_type", cfg.NOSType), zap.Strings("provisioners", names))
		for _, p := range cfg.HedgehogSonicProvisioners {
			// provisioner download
			provisionerPath := filepath.Join(si.StagingDir, p.Name)
			if err := stage.DownloadExecutable(ctx, hc, p.URL, provisionerPath, time.Second*60); err != nil {
				l.Error("Downloading provisioner failed", zap.String("provisioner", p.Name), zap.String("url", p.URL), zap.String("dest", provisionerPath), zap.Error(err))
				return fmt.Errorf("provisioner '%s' download: %w", p.Name, err)
			}

			// provisioner execution
			l.Info("Executing provisioner now...", zap.String("provisioner", p.Name))
			provisionerCmd := exec.CommandContext(ctx, provisionerPath)
			provisionerCmd.Stdin = os.Stdin
			provisionerCmd.Stderr = os.Stderr
			provisionerCmd.Stdout = os.Stdout
			if err := provisionerCmd.Run(); err != nil {
				l.Error("Provisioner execution failed", zap.String("bin", provisionerPath), zap.Error(err))
				return fmt.Errorf("provisioner '%s' execution: %w", p.Name, err)
			}
			l.Info("Provisioner execution completed", zap.String("provisioner", p.Name))
		}
		l.Info("Completed execution of all additional Hedgehog SONiC Provisioners", zap.Strings("provisioners", names))
	}
	return nil
}

func runOnieUpdate(ctx context.Context, hc *http.Client, cfg *configstage.Stage2, si *stage.StagingInfo, onie *stage.OnieEnv) (funcErr error) {
	// Build donwload URL: cfg URL + ONIE platform
	url, err := stage.BuildURL(cfg.ONIEUpdaterURL, onie.Platform)
	if err != nil {
		l.Error("Building ONIE updater URL failed", zap.String("url", cfg.ONIEUpdaterURL), zap.String("platform", onie.Platform), zap.Error(err))
		return fmt.Errorf("building ONIE updater URL: %w", err)
	}

	// ONIE download
	onieUpdaterPath := filepath.Join(si.StagingDir, "onie-update")
	l.Info("Downloading ONIE updater now...", zap.String("url", url), zap.String("dest", onieUpdaterPath))
	if err := stage.DownloadExecutable(ctx, hc, url, onieUpdaterPath, time.Second*120); err != nil {
		l.Error("Downloading ONIE updater failed", zap.String("url", url), zap.String("dest", onieUpdaterPath), zap.Error(err))
		return fmt.Errorf("ONIE updater download: %w", err)
	}
	l.Info("Downloading ONIE updater completed", zap.String("url", url), zap.String("dest", onieUpdaterPath))

	// for every following error we need to ensure that we make ONIE the default boot option again, because:
	// - the ONIE updater half-assed, and we don't know what that means, it might boot back into the NOS
	//   leaving the impression that the installation was successful
	// TODO: the reverse might actually exactly be what we want in this case
	defer func() {
		if funcErr != nil {
			l.Info("Trying to ensure that ONIE stays the default boot option...")
			if err := partitions.MakeONIEDefaultBootEntryAndCleanup(); err != nil {
				l.Error("Making ONIE the default boot option failed", zap.Error(err))
			}
			l.Info("ONIE is the default boot option again")
		}
	}()

	// ONIE install
	l.Info("Executing ONIE updater now...")
	subctx, cancel := context.WithCancel(ctx)
	defer cancel()
	onieCmd := exec.CommandContext(ctx, onieUpdaterPath)
	onieCmd.Stdin = os.Stdin
	onieCmd.Stderr = log.NewSinkWithLogger(subctx, l, zapcore.InfoLevel, zap.String("app", "onie-update"), zap.String("stream", "stderr"))
	onieCmd.Stdout = log.NewSinkWithLogger(subctx, l, zapcore.InfoLevel, zap.String("app", "onie-update"), zap.String("stream", "stdout"))
	if err := onieCmd.Run(); err != nil {
		l.Error("ONIE updater execution failed", zap.String("bin", onieUpdaterPath), zap.Error(err))
		return fmt.Errorf("ONIE updater execution: %w", err)
	}
	l.Info("ONIE update completed")

	return nil
}
