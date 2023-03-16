package stage0

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
	"go.githedgehog.com/dasboot/pkg/devid"
	"go.githedgehog.com/dasboot/pkg/dns"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/net"
	"go.githedgehog.com/dasboot/pkg/ntp"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"go.githedgehog.com/dasboot/pkg/seeder/ipam"
	"go.githedgehog.com/dasboot/pkg/stage"
	configstage "go.githedgehog.com/dasboot/pkg/stage0/config"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

var l = log.L()

const (
	vlanName = "control"
)

func ReadConfig() (*configstage.Stage0, error) {
	// open and read executable into memory
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(exePath)
	if err != nil {
		return nil, err
	}

	exeBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// now read embedded config for the first time - ignoring signature verification this time
	// an embedded config must exist, so even without signature validation, if this steps fail, we fail
	var cfg configstage.Stage0
	if err := config.ReadEmbeddedConfig(exeBytes, &cfg, nil, config.ReadOptionIgnoreSignature); err != nil {
		return nil, err
	}

	// perform signature validation if a CA is embedded at this point
	if len(cfg.SignatureCA) > 0 {
		// NOTE: this is a chicken & egg problem: using an embedded CA cert here
		// does not add any real security - with the exception of making things harder
		// for attackers :)
		// The Signature CA certificate should ideally come from the attached USB stick (location partition).
		// However, as the USB stick is most likely "downloadable" from the factory, I'm not sure we'll be
		// able to deal with this that way. Anyway, so this prepares things in the right way.
		//
		// Parse Signature CA cert and create a cert pool for it
		signatureCACert, err := x509.ParseCertificate(cfg.SignatureCA)
		if err != nil {
			return nil, err
		}
		configSignatureCAPool := x509.NewCertPool()
		configSignatureCAPool.AddCert(signatureCACert)

		// now read embedded configuration again, but this time only ignore wrong times at certificate validation time
		// NOTE: this is okay, as we haven't run NTP yet, and the time might not be up-to-date
		if err := config.ReadEmbeddedConfig(exeBytes, &cfg, configSignatureCAPool, config.ReadOptionIgnoreExpiryTime); err != nil {
			return nil, err
		}
	}

	// this completes reading the stage0 configuration
	return &cfg, nil
}

func Run(ctx context.Context, override *configstage.Stage0, logSettings *stage.LogSettings) (runErr error) {
	// we'll set things into this variable and export them before we execute the next stage
	stagingInfo := &stage.StagingInfo{}

	// setup logging first
	// TODO: this essentially should never fail, so should be implemented differently I guess
	if err := stage.InitializeGlobalLogger(ctx, logSettings); err != nil {
		return fmt.Errorf("stage0: failed to initialize logger: %w", err)
	}
	l = log.L()
	stagingInfo.LogSettings = *logSettings
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}

	// read ONIE env information
	onieEnv := stage.GetOnieEnv()
	l.Info("ONIE environment", zap.Reflect("onieEnv", onieEnv))

	// read the embedded configuration first
	embedded, err := ReadConfig()
	if err != nil {
		return fmt.Errorf("stage0: reading embedded config: %w", err)
	}
	l.Info("Read embedded configuration", zap.Reflect("config", embedded))

	// Merge configs with override
	cfg := configstage.MergeConfigs(embedded, override)
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("stage0: config validation: %w", err)
	}
	if override != nil {
		l.Info("Merged override configuration", zap.Reflect("config", cfg))
	}
	stagingInfo.ServerCA = make([]byte, len(cfg.CA))
	stagingInfo.ConfigSignatureCA = make([]byte, len(cfg.SignatureCA))
	copy(stagingInfo.ServerCA, cfg.CA)
	copy(stagingInfo.ConfigSignatureCA, cfg.SignatureCA)
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}

	// prepare staging area
	stagingDir, err := os.MkdirTemp("", "das-boot-")
	if err != nil {
		// we can only reuse /tmp at this point
		stagingDir = os.TempDir()
		l.Warn("failed to create temporary directory, reusing system temporary directory, and not mounting a tmpfs either", zap.String("stagingDir", stagingDir))
	} else {
		// otherwise we mount a dedicated tmpfs
		// and we will try to unmount it if this function returns successfully
		// otherwise we will keep things around for troubleshooting purposes
		if err := unix.Mount("das-boot", stagingDir, "tmpfs", 0, ""); err != nil {
			l.Warn("failed to mount tmpfs onto dedicated temporary staging directory", zap.String("stagingDir", stagingDir), zap.Error(err))
			// we will try to clean up on error all files in here
			defer func() {
				if runErr == nil {
					os.RemoveAll(stagingDir)
				}
			}()
		} else {
			// unmount staging dir and remove it
			defer func() {
				if runErr == nil {
					if err := unix.Unmount(stagingDir, 0); err != nil {
						return
					}
					os.Remove(stagingDir)
				}
			}()
		}
	}
	if err := os.Chdir(stagingDir); err != nil {
		// very silly that this could fail, but we cannot recover from this
		return fmt.Errorf("failed to change directory to staging area directory: %s", stagingDir)
	}
	stagingInfo.StagingDir = stagingDir
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}

	// we need to do partition discovery for finding our location UUID
	devices := partitions.Discover()

	// retrieve location info
	locationPartition, err := stage.MountLocationPartition(devices)
	if err != nil {
		l.Warn("Location partition failed to open", zap.Error(err))
	}
	var locationInfo *location.Info
	if locationPartition != nil {
		var err error
		locationInfo, err = locationPartition.GetLocation()
		if err != nil {
			return fmt.Errorf("stage0: retrieving location info: %w", err)
		}
	}

	// retrieve device ID
	hhdevid := devid.ID()
	if hhdevid == "" {
		return fmt.Errorf("stage0: failed to determine device ID")
	}
	stagingInfo.DeviceID = hhdevid
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}

	// retrieve network interface list
	netdevs, err := net.GetInterfaces()
	if err != nil {
		return fmt.Errorf("stage0: retrieving network interface list: %w", err)
	}

	// build HTTP client
	httpClient, err := stage.SeederHTTPClient(cfg.CA, nil, stage.HTTPClientOptionServerCertificateIgnoreExpiryTime)
	if err != nil {
		return fmt.Errorf("stage0: building HTTP client: %w", err)
	}

	// now issue the IPAM request
	ipamReq := &ipam.Request{
		Arch:                  stage.Arch(),
		DevID:                 hhdevid,
		LocationUUID:          locationInfo.UUID,
		LocationUUIDSignature: locationInfo.UUIDSig,
		Interfaces:            netdevs,
	}
	ipamResp, err := ipam.DoRequest(ctx, httpClient, ipamReq, cfg.IPAMURL)
	if err != nil {
		return fmt.Errorf("stage0: IPAM request: %w", err)
	}

	// we can configure DNS just once
	if err := dns.SetSystemResolvers(ipamResp.DNSServers); err != nil {
		return fmt.Errorf("stage0: configuring system DNS resolver: %w", err)
	}

	// for the rest until we finished downloading stage 1, we iterate over all IP addresses that we got back
	// and essentially retry the rest of stage 0 until it works
	var stage1Path string
	for netdev, ipa := range ipamResp.IPAddresses {
		var err error
		stage1Path, err = runWith(ctx, stagingInfo, logSettings, httpClient, ipamResp, netdev, ipa)
		if err != nil {
			l.Error("failed to run stage 0 to completion with network interface and IP address pair", zap.String("netdev", netdev), zap.Strings("ipAddresses", ipa), zap.Error(err))
			if err := net.DeleteVLANDevice(vlanName); err != nil {
				l.Warn("failed to delete VLAN device", zap.String("vlanDevice", vlanName), zap.Error(err))
			}
			continue
		}
		break
	}
	if stage1Path == "" {
		return fmt.Errorf("stage0: failed to run stage 0 to completion on any network interface and IP addresses pair")
	}

	// set the log settings which will now also have the right syslog servers
	stagingInfo.LogSettings = *logSettings
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}

	// success
	l.Info("Stage 0 completed successfully")

	// execute stage 1 now
	l.Info("Executing stage 1 now...")
	stage1Cmd := exec.CommandContext(ctx, stage1Path)
	stage1Cmd.Stdin = os.Stdin
	stage1Cmd.Stderr = os.Stderr
	stage1Cmd.Stdout = os.Stdout
	if err := stage1Cmd.Run(); err != nil {
		return fmt.Errorf("stage 1 execution failed: %w", err)
	}

	// as all installers are forked and execed, this is really the end of everything :)
	return nil
}

func runWith(ctx context.Context, stagingInfo *stage.StagingInfo, logSettings *stage.LogSettings, httpClient *http.Client, ipamResp *ipam.Response, netdev string, ipAddresses []string) (string, error) {
	// first things first: configure network interface
	ipaddrnets, err := net.StringsToIPNets(ipAddresses)
	if err != nil {
		return "", fmt.Errorf("converting IP addresses to IPNets: %w", err)
	}
	if err := net.AddVLANDeviceWithIP(netdev, ipamResp.VLAN, vlanName, ipaddrnets); err != nil {
		return "", fmt.Errorf("failed to configure network interface: %w", err)
	}

	// configure the syslog logger so that we're not blind anymore
	logSettings.SyslogServers = ipamResp.SyslogServers
	if err := stage.InitializeGlobalLogger(ctx, logSettings); err != nil {
		l.Warn("failed to reinitialize global logger with new settings", zap.Error(err))
	} else {
		l = log.L()
	}

	// now run NTP - we only fail if NTP fails, not if hardware clock sync fails
	if err := ntp.SyncClock(ctx, ipamResp.NTPServers); err != nil && !errors.Is(err, ntp.ErrHWClockSync) {
		return "", fmt.Errorf("failed to sync clock with NTP: %w", err)
	}

	// now try to download stage 1
	stage1Path := filepath.Join(stagingInfo.StagingDir, "stage1")
	if err := stage.DownloadExecutable(ctx, httpClient, ipamResp.Stage1URL, stage1Path, 60*time.Second); err != nil {
		return "", fmt.Errorf("downloading stage 1 failed: %w", err)
	}

	// these are all the pieces which are dependent on the "right" network to work
	// we'll continue execution in the main function
	return stage1Path, nil
}
