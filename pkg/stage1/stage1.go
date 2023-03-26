package stage1

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"

	"go.githedgehog.com/dasboot/pkg/config"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/identity"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"go.githedgehog.com/dasboot/pkg/seeder/registration"
	"go.githedgehog.com/dasboot/pkg/stage"
	configstage "go.githedgehog.com/dasboot/pkg/stage1/config"
	"go.githedgehog.com/dasboot/pkg/tpm"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"
)

var l = log.L()

var ErrExecution = errors.New("unrecoverable execution error encountered")

func executionError(err error) error {
	return fmt.Errorf("%w: %w", ErrExecution, err)
}

func ReadConfig(caPool *x509.CertPool) (*configstage.Stage1, error) {
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
	var cfg configstage.Stage1
	if err := config.ReadEmbeddedConfig(exeBytes, &cfg, caPool); err != nil {
		return nil, fmt.Errorf("reading embedded config ignoring signature: %w", err)
	}

	// this completes reading the stage0 configuration
	return &cfg, nil
}

func Run(ctx context.Context, override *configstage.Stage1, logSettings *stage.LogSettings) (runErr error) {
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
	l.Info("Stage 1 execution starting", zap.String("version", version.Version))
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

	// check if this device has a TPM, if yes, we will do hardware remote attestation
	if tpm.HasTPM() {
		// TODO: implement
	} else {
		l.Warn("This device is lacking a TPM 2.0 module. Skipping hardware remote attestation.")
	}

	// discover partitions
	devices := partitions.Discover()

	// retrieve location info
	locationPartition, err := stage.MountLocationPartition(l, devices)
	if err != nil {
		l.Warn("Location partition failed to open", zap.Error(err))
	} else {
		l.Info("Location partition mounted successfully")
	}
	var locationInfo *location.Info
	if locationPartition != nil {
		var err error
		locationInfo, err = locationPartition.GetLocation()
		if err != nil {
			l.Error("Retrieving location information from location partition failed", zap.Error(err))
			return ErrExecution
		}
		l.Info("Location information found on location partition", zap.Reflect("locationInfo", locationInfo))
	}

	// now mount (or create and mount) the identity partition
	// this step fully initializes and prepares the partition for our usage
	identityPartition, err := stage.MountIdentityPartition(l, devices, onieEnv.Platform)
	if err != nil {
		l.Error("Identity Partition could not be opened/mounted/created", zap.Error(err))
		return executionError(fmt.Errorf("opening identity partition: %w", err))
	}
	l.Info("Opened Hedgehog Identity Partition successfully")

	// first let's check if there is already location information stored
	// if it is, it must match the location information that we detected before
	// if not, we must start from scratch and delete potentially previously stored keys and certs
	var reinitialize bool
	if locationInfo != nil {
		ipLocationInfo, err := identityPartition.GetLocation()
		if err == nil {
			if !reflect.DeepEqual(locationInfo, ipLocationInfo) {
				l.Warn("Location information for this device has changed. Deleting previous keys and certs from identity partition", zap.Reflect("storedLocationInformation", ipLocationInfo), zap.Reflect("locationInformation", locationInfo))
				reinitialize = true
			}
		}
		if err != nil || reinitialize {
			l.Info("Storing location information onto identity partition", zap.Bool("reinitialize", reinitialize), zap.Bool("identityPartiionHasLocationInformation", err == nil))
			if err := identityPartition.StoreLocation(locationInfo); err != nil {
				l.Error("Storing location information onto identity partition failed", zap.Error(err))
				return executionError(fmt.Errorf("storing location information: %w", err))
			}
		}
	}

	// we need to recreate a key in the following situations:
	// - if the location info changed
	// - if there never was a key before (duh)
	// - if the certificate is not valid - the HasClientCert check will fail if the cert expired (NOTE: it will not check the certificate chain because we don't know that)
	var generateNewCSR bool
	hasClientKey := identityPartition.HasClientKey()
	hasValidClientCert := identityPartition.HasClientCert()
	if reinitialize || !hasClientKey || !hasValidClientCert {
		l.Info("Generating client key pair now...", zap.Bool("reinitialize", reinitialize), zap.Bool("hasClientKey", hasClientKey), zap.Bool("hasValidClientCert", hasValidClientCert))
		if err := identityPartition.GenerateClientKeyPair(); err != nil {
			l.Error("Generating client key pair failed", zap.Error(err))
			return executionError(fmt.Errorf("generating client key pair: %w", err))
		}

		// a regenerated client key needs to force to generate a new CSR
		generateNewCSR = true
	}

	// if we didn't need to generate a new key, then generateNewCSR is false
	// and we can directly load the key and cert from disk
	if !generateNewCSR {
		l.Info("Reusing existing client key pair and certificate from identity partition")
	} else {
		// otherwise we need to register now
		if err := registerDevice(ctx, cfg, identityPartition, si, locationInfo, generateNewCSR); err != nil {
			// no detailed error handling necessary here, done in registerDevice
			return err
		}
	}

	hc, err := stage.SeederHTTPClient(si.ServerCA, identityPartition)
	if err != nil {
		l.Error("Building HTTP client for downloading stage 2 failed", zap.Error(err))
		return executionError(err)
	}

	// now try to download stage 2
	stage2Path := filepath.Join(si.StagingDir, "stage2")
	if err := stage.DownloadExecutable(ctx, hc, cfg.Stage2URL, stage2Path, 60*time.Second); err != nil {
		l.Error("Downloading stage 2 installer failed", zap.String("url", cfg.Stage2URL), zap.String("dest", stage2Path), zap.Error(err))
		return executionError(fmt.Errorf("downloading stage 2: %w", err))
	}
	l.Info("Downloading stage 2 installer completed", zap.String("url", cfg.Stage2URL), zap.String("dest", stage2Path))

	// success
	l.Info("Stage 1 completed successfully")

	// execute stage 2 now
	l.Info("Executing stage 2 now...")
	stage2Cmd := exec.CommandContext(ctx, stage2Path)
	stage2Cmd.Stdin = os.Stdin
	stage2Cmd.Stderr = os.Stderr
	stage2Cmd.Stdout = os.Stdout
	if err := stage2Cmd.Run(); err != nil {
		l.Errorf("Stage 2 execution failed", zap.Error(err))
		return executionError(err)
	}

	// we are truly done
	return nil
}

// registers the device with the
func registerDevice(ctx context.Context, cfg *configstage.Stage1, identityPartition identity.IdentityPartition, si *stage.StagingInfo, locationInfo *location.Info, generateNewCSR bool) error {
	var clientCSRBytes []byte
	hasClientCSR := identityPartition.HasClientCSR()
	if generateNewCSR || !hasClientCSR {
		l.Info("Generating CSR from client key pair now...", zap.Bool("generateNewCSR", generateNewCSR), zap.Bool("hasClientCSR", hasClientCSR))
		var err error
		clientCSRBytes, err = identityPartition.GenerateClientCSR()
		if err != nil {
			l.Error("Generating CSR from client key pair failed", zap.Error(err))
			return executionError(fmt.Errorf("generating CSR: %w", err))
		}
	}
	if len(clientCSRBytes) == 0 {
		l.Info("Reading existing CSR from disk")
		var err error
		clientCSRBytes, err = identityPartition.ReadClientCSR()
		if err != nil {
			l.Error("Reading existing CSR from disk failed", zap.Error(err))
			return executionError(fmt.Errorf("reading CSR: %w", err))
		}
	}

	// build an HTTP client for the register requests
	hc, err := stage.SeederHTTPClient(si.ServerCA, nil)
	if err != nil {
		l.Error("Building HTTP client for registration failed", zap.Error(err))
		return executionError(err)
	}

	l.Info("Performing device registration now...")
	// TODO: needs all the details - this is truly the bare minimum
	req := &registration.Request{
		DeviceID:     si.DeviceID,
		CSR:          clientCSRBytes,
		LocationInfo: locationInfo,
	}
	resp, err := registration.DoRequest(ctx, hc, req, cfg.RegisterURL)
	if err != nil {
		l.Error("Device registration failed", zap.Error(err))
		return executionError(fmt.Errorf("device registration: %w", err))
	}

	// store returned certificate onto identity partition
	l.Info("Storing client certificate to identity partition...")
	if err := identityPartition.StoreClientCert(resp.ClientCertificate); err != nil {
		l.Error("Storing client certificate to identity partition failed", zap.Error(err))
		return executionError(fmt.Errorf("storing client certificate: %w", err))
	}

	return nil
}
