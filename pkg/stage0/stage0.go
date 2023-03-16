package stage0

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
	vlanName = "mgmt"
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

func Run(ctx context.Context, override *configstage.Stage0, logSettings *stage.LogSettings) error {
	// setup logging first
	// TODO: this essentially should never fail, so should be implemented differently I guess
	if err := stage.InitializeGlobalLogger(ctx, logSettings); err != nil {
		return fmt.Errorf("stage0: failed to initialize logger: %w", err)
	}
	l = log.L()

	// read the embedded configuration first
	embedded, err := ReadConfig()
	if err != nil {
		return fmt.Errorf("stage0: reading embedded config: %w", err)
	}

	// Merge configs with override
	cfg := configstage.MergeConfigs(embedded, override)
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("stage0: config validation: %w", err)
	}

	// we need to do partition discovery for finding our location UUID
	devices := partitions.Discover()

	// retrieve location info
	locationPartition, err := location.Open(devices.GetHedgehogLocationPartition())
	if err != nil {
		l.Warn("No location partition found", zap.Error(err))
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

	// for the rest we iterate over all IP addresses that we got back
	// and essentially retry the rest of stage 0 until it works
	var success bool
	for netdev, ipa := range ipamResp.IPAddresses {
		if err := runWith(ctx, logSettings, httpClient, ipamResp, netdev, ipa); err != nil {
			l.Error("failed to run stage 0 to completion with network interface and IP address pair", zap.String("netdev", netdev), zap.Strings("ipAddresses", ipa), zap.Error(err))
			if err := net.DeleteVLANDevice(vlanName); err != nil {
				l.Warn("failed to delete VLAN device", zap.Error(err))
			}
			continue
		}
		success = true
		break
	}
	if !success {
		return fmt.Errorf("stage0: failed to run stage 0 to completion on any network interface and IP addresses pair")
	}
	return nil
}

func runWith(ctx context.Context, logSettings *stage.LogSettings, httpClient *http.Client, ipamResp *ipam.Response, netdev string, ipAddresses []string) (runWithErr error) {
	// first things first: configure network interface
	ipaddrnets, err := net.StringsToIPNets(ipAddresses)
	if err != nil {
		return fmt.Errorf("converting IP addresses to IPNets: %w", err)
	}
	if err := net.AddVLANDeviceWithIP(netdev, ipamResp.VLAN, vlanName, ipaddrnets); err != nil {
		return fmt.Errorf("failed to configure network interface: %w", err)
	}

	// configure the syslog logger so that we're not blind anymore
	localLogSettings := *logSettings
	localLogSettings.SyslogServers = ipamResp.SyslogServers
	if err := stage.InitializeGlobalLogger(ctx, &localLogSettings); err != nil {
		l.Warn("failed to reinitialize global logger with new settings", zap.Error(err))
	} else {
		l = log.L()
	}

	// now run NTP - we only fail if NTP fails, not if hardware clock sync fails
	if err := ntp.SyncClock(ctx, ipamResp.NTPServers); err != nil && !errors.Is(err, ntp.ErrHWClockSync) {
		return fmt.Errorf("failed to sync clock with NTP: %w", err)
	}

	// prepare staging area
	stagingDir, err := os.MkdirTemp("", "das-boot-")
	if err != nil {
		// we can only reuse /tmp at this point
		stagingDir = os.TempDir()
		l.Warn("failed to create temporary directory, reusing system temporary directory, and not mounting a tmpfs either", zap.String("stagingDir", stagingDir))
	} else {
		// otherwise we mount a dedicated tmpfs
		// and we will try to unmount it if this function returns with an error
		if err := unix.Mount("das-boot", stagingDir, "tmpfs", 0, ""); err != nil {
			l.Warn("failed to mount tmpfs onto dedicated temporary staging directory", zap.String("stagingDir", stagingDir), zap.Error(err))
			// we will try to clean up on error all files in here
			defer func() {
				if runWithErr != nil {
					os.RemoveAll(stagingDir)
				}
			}()
		} else {
			// unmount staging dir and remove it
			defer func() {
				if runWithErr != nil {
					if err := unix.Unmount(stagingDir, 0); err == nil {
						os.Remove(stagingDir)
					}
				}
			}()
		}
	}
	if err := os.Chdir(stagingDir); err != nil {
		// very silly that this could fail, but we cannot recover from this
		return fmt.Errorf("failed to change directory to staging area directory: %s", stagingDir)
	}
	if err := os.Setenv("dasboot_staging_area", stagingDir); err != nil {
		// very silly that this could fail, but we cannot recover from this
		return fmt.Errorf("failed to set 'dasboot_staging_area' environment variable: %w", err)
	}

	// now try to download stage 1
	stage1Path := filepath.Join(stagingDir, "stage1")
	if err := stage.DownloadExecutable(ctx, httpClient, ipamResp.Stage1URL, stage1Path, 60*time.Second); err != nil {
		return fmt.Errorf("downloading stage 1 failed: %w", err)
	}

	return nil
}
