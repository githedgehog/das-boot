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

package hhagentprov

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.githedgehog.com/dasboot/pkg/config"
	configstage "go.githedgehog.com/dasboot/pkg/hhagentprov/config"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/stage"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
		return nil, fmt.Errorf("reading embedded config: %w", err)
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
	l.Info("Staging information", zap.Reflect("si", si))

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

	hc, err := stage.SeederHTTPClient(si.ServerCA, identityPartition)
	if err != nil {
		l.Error("Building HTTP client for downloading agent and agent config failed", zap.Error(err))
		return executionError(err)
	}

	// now mount the SONiC partition
	sonicPart := devices.GetSONiCPartition()
	if sonicPart == nil {
		l.Error("SONiC Partition not found")
		return executionError(fmt.Errorf("SONiC partition not found"))
	}
	if err := sonicPart.Mount(); err != nil && !errors.Is(err, partitions.ErrAlreadyMounted) {
		l.Error("SONiC Partition could not be mounted", zap.String("device", sonicPart.Path), zap.String("mountPath", sonicPart.MountPath))
		return executionError(fmt.Errorf("SONiC partition mount: %w", err))
	}
	defer func() {
		l.Info("Unmounting SONiC Partition", zap.String("device", sonicPart.Path), zap.String("mountPath", sonicPart.MountPath))
		if err := sonicPart.Unmount(); err != nil {
			l.Error("Unmounting SONiC Partition failed", zap.String("device", sonicPart.Path), zap.String("mountPath", sonicPart.MountPath), zap.Error(err))
		}
	}()
	l.Info("Mounted SONiC Partition", zap.String("device", sonicPart.Path), zap.String("mountPath", sonicPart.MountPath))

	// determine SONiC root path on mounted partition
	sonicRootPath, err := determineSonicRootPath(sonicPart.MountPath)
	if err != nil {
		l.Error("Determining SONiC image directory failed", zap.String("mountPath", sonicPart.MountPath), zap.Error(err))
		return executionError(fmt.Errorf("determining SONiC image dir: %w", err))
	}
	l.Info("Found SONiC installation on SONiC partition", zap.String("sonicRootPath", sonicRootPath))

	// prepare several directories now which we need for installing the agent
	agentConfigTargetDir := filepath.Join(sonicRootPath, "/rw/etc/sonic/hedgehog/")
	if err := os.MkdirAll(agentConfigTargetDir, 0755); err != nil {
		l.Error("Preparing Hedgehog Agent config target directory failed", zap.String("agentConfigTargetDir", agentConfigTargetDir), zap.Error(err))
		return executionError(fmt.Errorf("creating agent config target dir '%s': %w", agentConfigTargetDir, err))
	}
	sonicAgentBinDir := "/opt/hedgehog/bin"
	agentBinTargetDir := filepath.Join(sonicRootPath, "rw", sonicAgentBinDir)
	if err := os.MkdirAll(agentBinTargetDir, 0755); err != nil {
		l.Error("Preparing Hedgehog Agent bin target directory failed", zap.String("agentBinTargetDir", agentBinTargetDir), zap.Error(err))
		return executionError(fmt.Errorf("creating agent bin target dir '%s': %w", agentBinTargetDir, err))
	}
	systemdMultiUserTargetDir := filepath.Join(sonicRootPath, "/rw/etc/systemd/system/multi-user.target.wants")
	if err := os.MkdirAll(systemdMultiUserTargetDir, 0755); err != nil {
		l.Error("Preparing systemd multi-user.target.wants dir failed", zap.String("systemdMultiUserTargetDir", systemdMultiUserTargetDir), zap.Error(err))
		return executionError(fmt.Errorf("creating systemd multi-user.target.wants dir '%s': %w", systemdMultiUserTargetDir, err))
	}
	l.Info("Created basic directory layout for Hedgehog agent installation",
		zap.String("agentConfigTargetDir", agentConfigTargetDir),
		zap.String("agentBinTargetDir", agentBinTargetDir),
		zap.String("systemdMultiUserTargetDir", systemdMultiUserTargetDir),
	)

	// populate it with
	// - agent
	// - agent config
	// - agent kubeconfig
	// by downloading it from the seeder
	agentBinPath := filepath.Join(agentBinTargetDir, "agent")
	agentConfigPath := filepath.Join(agentConfigTargetDir, "agent-config.yaml")
	agentKubeconfigPath := filepath.Join(agentConfigTargetDir, "agent-kubeconfig")

	cfg.AgentURL, err = url.JoinPath(cfg.AgentURL, si.DeviceID)
	if err != nil {
		l.Error("Joining agent URL with device ID failed", zap.String("url", cfg.AgentURL), zap.String("deviceID", si.DeviceID), zap.Error(err))
		return executionError(fmt.Errorf("joining agent URL with device ID '%s': %w", si.DeviceID, err))
	}

	if err := stage.DownloadExecutable(ctx, hc, cfg.AgentURL, agentBinPath, time.Second*60); err != nil {
		l.Error("Downloading agent binary failed", zap.String("url", cfg.AgentURL), zap.String("dest", agentBinPath), zap.Error(err))
		return executionError(fmt.Errorf("downloading agent binary: %w", err))
	}
	l.Info("Downloaded agent binary", zap.String("url", cfg.AgentURL), zap.String("dest", agentBinPath))

	agentConfigURL, err := url.Parse(cfg.AgentConfigURL)
	if err != nil {
		l.Error("Parsing agent config URL failed", zap.String("url", cfg.AgentConfigURL), zap.Error(err))
		return executionError(fmt.Errorf("parsing agent config URL '%s': %w", cfg.AgentConfigURL, err))
	}
	agentConfigURL.Path = path.Join(agentConfigURL.Path, si.DeviceID)
	if err := stage.Download(ctx, hc, agentConfigURL.String(), agentConfigPath, 0640, time.Second*60); err != nil {
		l.Error("Downloading agent config failed", zap.String("url", agentConfigURL.String()), zap.String("dest", agentConfigPath), zap.Error(err))
		return executionError(fmt.Errorf("downloading agent config: %w", err))
	}
	l.Info("Downloaded agent config for this device", zap.String("url", agentConfigURL.String()), zap.String("dest", agentConfigPath))

	agentKubeconfigURL, err := url.Parse(cfg.AgentKubeconfigURL)
	if err != nil {
		l.Error("Parsing agent kubeconfig URL failed", zap.String("url", cfg.AgentKubeconfigURL), zap.Error(err))
		return executionError(fmt.Errorf("parsing agent kubeconfig URL '%s': %w", cfg.AgentKubeconfigURL, err))
	}
	agentKubeconfigURL.Path = path.Join(agentKubeconfigURL.Path, si.DeviceID)
	if err := stage.Download(ctx, hc, agentKubeconfigURL.String(), agentKubeconfigPath, 0600, time.Second*60); err != nil {
		l.Error("Downloading agent kubeconfig failed", zap.String("url", agentKubeconfigURL.String()), zap.String("dest", agentKubeconfigPath), zap.Error(err))
		return executionError(fmt.Errorf("downloading agent kubeconfig: %w", err))
	}
	l.Info("Downloaded agent kubeconfig for this device", zap.String("url", agentKubeconfigURL.String()), zap.String("dest", agentKubeconfigPath))

	// now write systemd unit
	// we'll do this by calling the agent with the "generate systemd-unit" commands which will just do that
	// and we'll write the stdout of the command to the systemd service file
	systemdUnitPath := "/etc/systemd/system/hedgehog-agent.service"
	systemdUnitTargetPath := filepath.Join(sonicRootPath, "rw", systemdUnitPath)
	systemdUnitTargetFile, err := os.OpenFile(systemdUnitTargetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		l.Error("Opening hedgehog-agent.service file failed", zap.String("systemdUnitTargetPath", systemdUnitTargetPath), zap.Error(err))
		return executionError(fmt.Errorf("opening hedgehog agent service file '%s': %w", systemdUnitTargetPath, err))
	}
	subctx, cancel := context.WithCancel(ctx)
	cmdStrings := []string{agentBinPath, "generate", "systemd-unit", "--agent-path", filepath.Join(sonicAgentBinDir, "agent"), "--user", "root"}
	cmd := exec.CommandContext(ctx, cmdStrings[0], cmdStrings[1:]...) //#nosec G204
	cmd.Stdout = systemdUnitTargetFile
	cmd.Stderr = log.NewSinkWithLogger(subctx, l, zapcore.InfoLevel, zap.String("app", "agent-generate-systemd-unit"), zap.String("stream", "stderr"))
	if err := cmd.Run(); err != nil {
		systemdUnitTargetFile.Close()
		cancel()
		l.Error("Generating hedgehog-agent.service systemd unit with agent binary failed", zap.Strings("cmd", cmdStrings), zap.Error(err))
		return executionError(fmt.Errorf("generating systemd unit with agent binary: %w", err))
	}
	cancel()
	systemdUnitTargetFile.Close()
	l.Info("Generated hedgehog-agent.service systemd unit file using agent binary", zap.Strings("cmd", cmdStrings), zap.String("systemdUnitTargetPath", systemdUnitTargetPath))

	// and link systemd unit to multi-user target
	// TODO: we should find the right target
	symlinkPath := filepath.Join(sonicRootPath, "/rw/etc/systemd/system/multi-user.target.wants/hedgehog-agent.service")
	if err := os.Symlink(systemdUnitPath, symlinkPath); err != nil {
		l.Error("Creating symlink for systemd service failed", zap.String("symlinkPath", symlinkPath), zap.String("targetPath", systemdUnitPath), zap.Error(err))
		return executionError(fmt.Errorf("symlinking agent systemd unit '%s' -> '%s': %w", symlinkPath, systemdUnitPath, err))
	}
	l.Info("Created symlink for Hedgehog agent to enable hedgehog-agent.service unit on startup", zap.String("symlinkPath", symlinkPath), zap.String("targetPath", systemdUnitPath))

	// we are done here
	l.Info("Hedgehog Agent Provisioner completed successfully")
	return nil
}

func determineSonicRootPath(path string) (string, error) {
	// get all the files from path which we assume is the root of the SONiC partiton
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("reading dir entries at '%s': %w", path, err)
	}

	// iterate over the entries until we find the SONiC installation folder
	for _, dirEntry := range dirEntries {
		if strings.HasPrefix(dirEntry.Name(), "image-") {
			// as we are provisioning from scratch
			// we can rightfully assume (at the moment)
			// that there are no other SONiC images installed in this partition
			// so we'll assume that this is what we need
			return filepath.Join(path, dirEntry.Name()), nil
		}
	}

	// no SONiC installation found - truly irrecoverable at this point
	return "", fmt.Errorf("no SONiC image installation found")
}
