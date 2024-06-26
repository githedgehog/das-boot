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

package stage0

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	gonet "net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"go.githedgehog.com/dasboot/pkg/config"
	"go.githedgehog.com/dasboot/pkg/devid"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/net"
	onieurl "go.githedgehog.com/dasboot/pkg/net/url"
	"go.githedgehog.com/dasboot/pkg/ntp"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"go.githedgehog.com/dasboot/pkg/seeder/ipam"
	"go.githedgehog.com/dasboot/pkg/stage"
	configstage "go.githedgehog.com/dasboot/pkg/stage0/config"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

var l = log.L()

var ErrExecution = errors.New("unrecoverable execution error encountered")

func executionError(err error) error {
	return fmt.Errorf("%w: %w", ErrExecution, err)
}

const (
	vlanName = "control"
)

func ReadConfig() (*configstage.Stage0, error) {
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

	// now read embedded config for the first time - ignoring signature verification this time
	// an embedded config must exist, so even without signature validation, if this steps fail, we fail
	var cfg configstage.Stage0
	if err := config.ReadEmbeddedConfig(exeBytes, &cfg, nil, config.ReadOptionIgnoreSignature); err != nil {
		return nil, fmt.Errorf("reading embedded config ignoring signature: %w", err)
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
			return nil, fmt.Errorf("parsing signature CA cert: %w", err)
		}
		configSignatureCAPool := x509.NewCertPool()
		configSignatureCAPool.AddCert(signatureCACert)

		// now read embedded configuration again, but this time only ignore wrong times at certificate validation time
		// NOTE: this is okay, as we haven't run NTP yet, and the time might not be up-to-date
		if err := config.ReadEmbeddedConfig(exeBytes, &cfg, configSignatureCAPool, config.ReadOptionIgnoreExpiryTime); err != nil {
			return nil, fmt.Errorf("reading embedded config ignoring certificate expiry time: %w", err)
		}
	}

	// this completes reading the stage0 configuration
	return &cfg, nil
}

func Run(ctx context.Context, override *configstage.Stage0, logSettings *stage.LogSettings) (runErr error) {
	// we'll set things into this variable and export them before we execute the next stage
	stagingInfo := &stage.StagingInfo{}

	var resetNetwork func()
	resetNetworkLogSettings := *logSettings
	// In case of installation success which means that we were successful at setting up the network
	// we want to revert it again after we are done here.
	// NOTE: we leave it in the error case because it might help when we need to debug things, and the
	// installer is able to deal with previously existing network configuration
	defer func() {
		if runErr == nil && resetNetwork != nil {
			// reset the logger to one without syslog servers, otherwise this can hang
			stage.InitializeGlobalLogger(ctx, &resetNetworkLogSettings) //nolint: errcheck
			l = log.L()
			resetNetwork()
		}
	}()

	// setup logging first
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
	stagingInfo.LogSettings = *logSettings
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}
	l.Info("Stage 0 execution starting", zap.String("version", version.Version))
	l.Info("System environment", zap.Strings("env", os.Environ()))

	// read ONIE env information
	onieEnv := stage.GetOnieEnv()
	l.Info("ONIE environment", zap.Reflect("onieEnv", onieEnv))

	// read the embedded configuration first
	embedded, err := ReadConfig()
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
	stagingInfo.OnieHeaders = cfg.OnieHeaders
	stagingInfo.ServerCA = make([]byte, len(cfg.CA))
	stagingInfo.ConfigSignatureCA = make([]byte, len(cfg.SignatureCA))
	copy(stagingInfo.ServerCA, cfg.CA)
	copy(stagingInfo.ConfigSignatureCA, cfg.SignatureCA)
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}

	// cleanup potentially previous staging areas and SONiC installers
	// we want to do this on start of a new installation, and not on a failing installation
	// so that the previously failing installer leaves their things around for debugging
	tmpDir := os.TempDir()
	tmpDirEntries, err := os.ReadDir(tmpDir)
	if err != nil {
		l.Warn("Failed to read directory entries from OS temp dir. We will not be able to cleanup from previous installation attempts", zap.String("tmpDir", tmpDir), zap.Error(err))
	} else {
		for _, tmpDirEntry := range tmpDirEntries {
			name := tmpDirEntry.Name()
			if !tmpDirEntry.IsDir() {
				continue
			}
			// check for a DAS BOOT staging directory
			if strings.HasPrefix(name, "das-boot-") {
				dir := filepath.Join(tmpDir, name)
				// unmount it first, if it is mounted
				if ok, _ := stage.IsMountPoint(dir); ok {
					if err := unix.Unmount(dir, 0); err != nil {
						l.Warn("Failed to unmount previously used DAS BOOT staging directory", zap.String("stagingDir", dir), zap.Error(err))
					} else {
						l.Info("Unmounted previously existing DAS BOOT staging directory", zap.String("stagingDir", dir))
					}
				}
				// remove it and everything in there
				if err := os.RemoveAll(dir); err != nil {
					l.Warn("Failed to remove previously used DAS BOOT staging directory", zap.String("stagingDir", dir), zap.Error(err))
				} else {
					l.Info("Removed previously existing DAS BOOT staging directory", zap.String("stagingDir", dir))
				}
				continue
			}

			// check for a previous SONiC installer which definitely does not clean up after itself
			// NOTE: we don't know about any other installers besides from SONiC. If we ever want to support others, we need to revisit this
			// because "tmp." is not a very creative prefix
			if strings.HasPrefix(name, "tmp.") {
				dir := filepath.Join(tmpDir, name)
				// unmount it first, if it is mounted
				if ok, _ := stage.IsMountPoint(dir); ok {
					if err := unix.Unmount(dir, 0); err != nil {
						l.Warn("Failed to unmount previously used SONiC installer directory", zap.String("stagingDir", dir), zap.Error(err))
					} else {
						l.Info("Unmounted previously existing SONiC installer directory", zap.String("stagingDir", dir))
					}
				}
				// remove it and everything in there
				if err := os.RemoveAll(dir); err != nil {
					l.Warn("Failed to remove previously used SONiC installer directory", zap.String("stagingDir", dir), zap.Error(err))
				} else {
					l.Info("Removed previously existing SONiC installer directory", zap.String("stagingDir", dir))
				}
				continue
			}
		}
	}

	// prepare staging area
	stagingDir, err := os.MkdirTemp("", "das-boot-")
	if err != nil {
		// we can only reuse /tmp at this point
		stagingDir = os.TempDir()
		l.Warn("Failed to create temporary directory, reusing system temporary directory, and not mounting a tmpfs either", zap.String("stagingDir", stagingDir))
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
		l.Error("Changing directory to staging area directory failed", zap.String("stagingDir", stagingDir), zap.Error(err))
		return executionError(err)
	}
	stagingInfo.StagingDir = stagingDir
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}
	l.Info("Staging area directory prepared", zap.String("stagingDir", stagingDir))

	// we need to do partition discovery for finding our location UUID
	devices := partitions.Discover()

	// retrieve location info
	// - location info from partition has priority
	// - if it also found in configuration (either manually added, or served through link-local discovery), then it must match, or we must abort otherwise
	// - location info is not mandatory necessarily (TODO: IPAM needs work though for that)
	// - also export it to staging info
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
		if cfg.Location != nil {
			l.Warn("Location information was also provided through configuration. You should not provide location information through configuration if you are using the location partition feature.")
			if !reflect.DeepEqual(locationInfo, cfg.Location) {
				err := fmt.Errorf("location information form partition does not match location information from configuration (fix this setup)")
				l.Error("Location information mismatch", zap.Error(err), zap.Reflect("locationInfoPartition", locationInfo), zap.Reflect("locationInfoConfig", cfg.Location))
				return executionError(err)
			}
		}
	} else if cfg.Location != nil {
		locationInfo = cfg.Location
		l.Info("Location information provided through configuration", zap.Reflect("locationInfo", locationInfo))
	} else {
		l.Warn("No location information was detected")
	}

	if locationInfo != nil {
		stagingInfo.LocationInfo = locationInfo
		if err := stagingInfo.Export(); err != nil {
			l.Warn("Failed to export staging area information", zap.Error(err))
		}
	}

	// retrieve device ID
	hhdevid := devid.ID()
	if hhdevid == "" {
		l.Error("Determining device ID failed (hhdevid)")
		return ErrExecution
	}
	stagingInfo.DeviceID = hhdevid
	if err := stagingInfo.Export(); err != nil {
		l.Warn("Failed to export staging area information", zap.Error(err))
	}
	l.Info("Device ID determined successfully (hhdevid)", zap.String("hhdevid", hhdevid))

	// retrieve network interface list
	netdevs, err := net.GetInterfaces()
	if err != nil {
		l.Error("Retrieving network interface list failed", zap.Error(err))
		return executionError(err)
	}
	l.Info("Capable network interface list retrieved", zap.Strings("netdevs", netdevs))

	// build HTTP client
	httpClient, err := stage.SeederHTTPClient(cfg.CA, nil, stage.HTTPClientOptionServerCertificateIgnoreExpiryTime)
	if err != nil {
		l.Error("Building HTTP client failed", zap.Error(err))
		return executionError(err)
	}

	// now issue the IPAM request if we need to
	// NOTE: the seeder will decide if we need to do IPAM or not
	var stage1Path string
	if cfg.IPAMURL != "" {
		locationUUID := ""
		var locationUUIDSig []byte
		if locationInfo != nil {
			locationUUID = locationInfo.UUID
			locationUUIDSig = locationInfo.UUIDSig
		}
		ipamReq := &ipam.Request{
			Arch:                  stage.Arch(),
			DevID:                 hhdevid,
			LocationUUID:          locationUUID,
			LocationUUIDSignature: locationUUIDSig,
			Interfaces:            netdevs,
		}
		ipamResp, err := ipamClient(ctx, httpClient, cfg.IPAMURL, ipamReq, onieEnv)
		if err != nil {
			l.Error("IPAM request failure", zap.Reflect("ipamRequest", ipamReq), zap.Error(err))
			return executionError(err)
		}
		l.Info("IPAM response received", zap.Reflect("ipamRequest", ipamReq), zap.Reflect("ipamResp", ipamResp))

		// for the rest until we finished downloading stage 1, we iterate over all IP addresses that we got back
		// and essentially retry the rest of stage 0 until it works
		// first we try with "preferred" entries that we got back
		for netdev, ipa := range ipamResp.IPAddresses {
			if !ipa.Preferred {
				continue
			}
			var err error
			stage1Path, resetNetwork, err = runWith(ctx, stagingInfo, logSettings, httpClient, ipamResp, netdev, ipa)
			if err != nil {
				l.Error("System network configuration failed for netdev", zap.String("netdev", netdev), zap.Reflect("ipa", ipa), zap.Error(err))
				continue
			}
			l.Info("System network configured", zap.String("netdev", netdev), zap.Reflect("ipa", ipa))
			break
		}
		// if preferred responses did not work, we will try all other responses
		if stage1Path == "" {
			for netdev, ipa := range ipamResp.IPAddresses {
				if ipa.Preferred {
					continue
				}
				var err error
				stage1Path, resetNetwork, err = runWith(ctx, stagingInfo, logSettings, httpClient, ipamResp, netdev, ipa)
				if err != nil {
					l.Error("System network configuration failed for netdev", zap.String("netdev", netdev), zap.Reflect("ipa", ipa), zap.Error(err))
					continue
				}
				l.Info("System network configured", zap.String("netdev", netdev), zap.Reflect("ipa", ipa))
				break
			}
		}
		if stage1Path == "" {
			l.Error("System network configuration failed for all network devices")
			return ErrExecution
		}
	} else {
		// if we don't need to do IPAM, then this means that we were configured with LLDP (hopefully)
		// this means that we are going to setup NTP and Syslog servers from the configuration
		var err error
		stage1Path, err = runWithoutIPAM(ctx, stagingInfo, logSettings, httpClient, cfg)
		if err != nil {
			l.Error("System configuration failed", zap.Error(err))
			return executionError(err)
		}
		l.Info("System configuration successful")
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
		l.Error("Stage 1 execution failed", zap.Error(err))
		return executionError(err)
	}

	// as all installers are forked and execed, this is really the end of everything :)
	l.Info("Installation complete")
	return nil
}

func runWith(ctx context.Context, stagingInfo *stage.StagingInfo, logSettings *stage.LogSettings, httpClient *http.Client, ipamResp *ipam.Response, netdev string, ipa ipam.IPAddress) (funcRet string, funcResetNetwork func(), funcErr error) {
	// first things first: configure network interface, and we need to do some conversions first
	// if these fail, then there is no need to proceed with anything else
	ipaddrnets, err := net.StringsToIPNets(ipa.IPAddresses)
	if err != nil {
		l.Error("Conversion of IP addresses to IPNets failed", zap.String("netdev", netdev), zap.Reflect("ipAddresses", ipa.IPAddresses))
		return "", nil, fmt.Errorf("converting IP addresses to IPNets: %w", err)
	}
	var routes []*net.Route
	if len(ipa.Routes) > 0 {
		for _, route := range ipa.Routes {
			dests, err := net.StringsToIPNets(route.Destinations)
			if err != nil {
				l.Error("Conversion of IP routes destinations to IPNets failed", zap.String("netdev", netdev), zap.Reflect("routes", ipa.Routes), zap.Error(err))
				return "", nil, fmt.Errorf("converting routes destinations to IPNets: %w", err)
			}
			gw := gonet.ParseIP(route.Gateway)
			if gw == nil {
				l.Error("Conversion of IP gateway string to IP failed", zap.String("netdev", netdev), zap.String("gw", route.Gateway))
				return "", nil, fmt.Errorf("converting routes gateway '%s' to IP failed", route.Gateway)
			}
			routes = append(routes, &net.Route{
				Dests: dests,
				Gw:    gw,
				Flags: route.Flags,
			})
		}
	}

	// if anything goes wrong below, we are going to try to delete the VLAN interface and revert any network configuration
	// we are doing this already before we create the devices because we don't really know what failed, so it's essentially safe
	// to just try and delete / revert everything we are going to try to delete
	// NOTE: We will also pass on this function **on success**, so that if anything else fails down the line, this function can be called to
	// reset the network.
	resetNetwork := func() {
		if ipa.VLAN > 0 {
			if err := net.DeleteVLANDevice(vlanName, ipaddrnets, routes); err != nil {
				l.Warn("Deleting VLAN device or reverting its configuration failed", zap.String("vlanDevice", vlanName), zap.Error(err))
			} else {
				l.Info("Successfully deleted VLAN device and reverted its configuration", zap.String("vlanDevice", vlanName))
			}
		} else {
			if err := net.UnconfigureDeviceWithIP(netdev, ipaddrnets, routes); err != nil {
				l.Warn("Reverting network device configuration failed", zap.String("netdev", netdev), zap.Error(err))
			} else {
				l.Info("Successfully reverted network device configuration", zap.String("netdev", netdev))
			}
		}
	}
	defer func() {
		if funcErr != nil {
			resetNetwork()
		}
	}()

	// VLAN configuration is being considered optional when its value is `0`
	// otherwise we configure the IP and routes directly on netdev
	if ipa.VLAN > 0 {
		if err := net.AddVLANDeviceWithIP(netdev, ipa.VLAN, vlanName, ipaddrnets, routes); err != nil {
			l.Error("VLAN interface creation and configuration failed",
				zap.String("netdev", netdev),
				zap.String("vlanInterface", vlanName),
				zap.Reflect("ipa", ipa),
				zap.Reflect("ipaddrnets", ipaddrnets),
				zap.Reflect("routes", routes),
				zap.Error(err),
			)
			return "", nil, fmt.Errorf("add vlan device with IP: %w", err)
		}
		l.Info("VLAN interface successfully created and configured",
			zap.String("netdev", netdev),
			zap.String("vlanInterface", vlanName),
			zap.Reflect("ipa", ipa),
			zap.Reflect("ipaddrnets", ipaddrnets),
			zap.Reflect("routes", routes),
		)
	} else {
		if err := net.ConfigureDeviceWithIP(netdev, ipaddrnets, routes); err != nil {
			l.Error("Configuring network interface failed",
				zap.String("netdev", netdev),
				zap.Reflect("ipaddrnets", ipaddrnets),
				zap.Reflect("routes", routes),
				zap.Error(err),
			)
			return "", nil, fmt.Errorf("configure device with IP: %w", err)
		}
		l.Info("Network interface successfully configured",
			zap.String("netdev", netdev),
			zap.Reflect("ipaddrnets", ipaddrnets),
			zap.Reflect("routes", routes),
		)
	}

	// configure the syslog logger so that we're not blind anymore
	// this gets a special context so that if this function failed
	// we will essentially stop the underlying syslog client
	// however, we want to keep it running on success
	logSettings.SyslogServers = ipamResp.SyslogServers
	logCtx, logCtxCancel := context.WithCancel(ctx)
	defer func() {
		if funcErr != nil {
			logCtxCancel()
		}
	}()
	if err := stage.InitializeGlobalLogger(logCtx, logSettings); err != nil {
		l.Warn("Reinitializing global logger with new settings including syslog servers failed", zap.String("netdev", netdev), zap.Strings("syslogServers", ipamResp.SyslogServers), zap.Error(err))
	} else {
		l = log.L()
		l.Info("Reinitialized global logger with new settings including syslog servers",
			zap.String("netdev", netdev),
			zap.Strings("syslogServers", ipamResp.SyslogServers),
		)
	}

	// now run NTP - we only fail if NTP fails, not if hardware clock sync fails
	l.Info("Trying to query NTP servers now to synchronize system clock...", zap.String("netdev", netdev), zap.Strings("ntpServers", ipamResp.NTPServers))
	if err := ntp.SyncClock(ctx, ipamResp.NTPServers); err != nil && !errors.Is(err, ntp.ErrHWClockSync) {
		l.Error("Syncing system clock with NTP failed", zap.String("netdev", netdev), zap.Error(err))
		return "", nil, fmt.Errorf("syncing clock with NTP: %w", err)
	}
	l.Info("System clock successfully synchronized with NTP", zap.String("netdev", netdev), zap.Strings("ntpServers", ipamResp.NTPServers))

	// now try to download stage 1
	stage1Path := filepath.Join(stagingInfo.StagingDir, "stage1")
	if err := stage.DownloadExecutable(ctx, httpClient, ipamResp.Stage1URL, stage1Path, 60*time.Second); err != nil {
		l.Error("Downloading stage 1 installer failed", zap.String("netdev", netdev), zap.String("url", ipamResp.Stage1URL), zap.String("dest", stage1Path), zap.Error(err))
		return "", nil, fmt.Errorf("downloading stage 1: %w", err)
	}
	l.Info("Downloading stage 1 installer completed", zap.String("netdev", netdev), zap.String("url", ipamResp.Stage1URL), zap.String("dest", stage1Path))

	// these are all the pieces which are dependent on the "right" network to work
	// we'll continue execution in the main function
	return stage1Path, resetNetwork, nil
}

func ipamClient(ctx context.Context, hc *http.Client, ipamURLStr string, req *ipam.Request, onieEnv *stage.OnieEnv) (*ipam.Response, error) {
	ipamURL, err := url.Parse(ipamURLStr)
	if err != nil {
		return nil, fmt.Errorf("IPAM URL validation error: %w", err)
	}

	// if the IPAM URL is not a link-local address host, we can short-circuit here
	if !strings.HasPrefix(ipamURL.Host, "[fe80:") && !strings.HasPrefix(ipamURL.Host, "fe80:") {
		l.Debug("IPAM URL does not have a link-local host", zap.String("host", ipamURL.Host))
		return ipam.DoRequest(ctx, hc, req, ipamURLStr)
	}

	// check if this is from within an ONIE installer
	// because then we are going to assume that we want to use the same interface that
	// was used to download the stage 0 installer
	// that is of course only the case if this was downloaded from a link-local address URL
	if strings.Contains(onieEnv.ExecURL, "fe80:") {
		l.Warn("IPAM URL is on a link-local host, as was the stage 0 installer. We are trying to reuse the same interface for the request.", zap.String("ExecURL", onieEnv.ExecURL))

		// ONIE doesn't get URL encoding right for the host
		// Furthermore, some older ONIE versions are even not using brackets around the
		// IPv6 address
		// This is why we have a dedicated URL parser package which deals with broken URLs
		// However, we need to deal with the %netdev part beforehand, as it is impossible to get this
		// corrected generically for all use-cases.
		// NOTE: do *NOT* use this package for anything else than parsing the ONIE Exec URL
		execURLStr := onieEnv.ExecURL
		if !strings.Contains(onieEnv.ExecURL, "%25") {
			execURLStr = strings.Replace(onieEnv.ExecURL, "%", "%25", 1)
		}
		execURL, err := onieurl.Parse(execURLStr)
		if err != nil {
			return nil, fmt.Errorf("ONIE Exec URL validation error: %w", err)
		}
		host := strings.SplitN(execURL.Host, "%", 2)
		if len(host) != 2 {
			return nil, fmt.Errorf("ONIE Exec URL Host splitting issue: [Host: '%s', Splitted Parts: %d]", execURL.Host, len(host))
		}

		netdev := strings.TrimRight(host[1], "]")
		hostTrimmed := strings.TrimLeft(strings.TrimRight(ipamURL.Host, "]"), "[")

		// now adjust the URL, and use it
		ipamURL.Host = "[" + hostTrimmed + "%" + netdev + "]"
		return ipam.DoRequest(ctx, hc, req, ipamURL.String())
	}

	// otherwise this is probably being executed from the ONIE rescue system
	// we will simply try the request on all interfaces
	l.Warn("IPAM URL is on a link-local host, and failed to detect which network interface to use. We will try all of them", zap.Strings("netdevs", req.Interfaces))
	urlHost := ipamURL.Host
	for _, netdev := range req.Interfaces {
		ipamURL.Host = urlHost + "%" + netdev
		resp, err := ipam.DoRequest(ctx, hc, req, ipamURL.String())
		if err != nil {
			l.Error("IPAM request failure", zap.String("netdev", netdev), zap.String("url", ipamURL.String()), zap.Reflect("ipamRequest", req), zap.Error(err))
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("request failed on all network interfaces [%s]", strings.Join(req.Interfaces, ","))
}

func runWithoutIPAM(ctx context.Context, stagingInfo *stage.StagingInfo, logSettings *stage.LogSettings, httpClient *http.Client, cfg *configstage.Stage0) (funcRet string, funcErr error) {
	// configure the syslog logger so that we're not blind anymore
	// this gets a special context so that if this function failed
	// we will essentially stop the underlying syslog client
	// however, we want to keep it running on success
	logSettings.SyslogServers = cfg.Services.SyslogServers
	logCtx, logCtxCancel := context.WithCancel(ctx)
	defer func() {
		if funcErr != nil {
			logCtxCancel()
		}
	}()
	if err := stage.InitializeGlobalLogger(logCtx, logSettings); err != nil {
		l.Warn("Reinitializing global logger with new settings including syslog servers failed", zap.Strings("syslogServers", cfg.Services.SyslogServers), zap.Error(err))
	} else {
		l = log.L()
		l.Info("Reinitialized global logger with new settings including syslog servers",
			zap.Strings("syslogServers", cfg.Services.SyslogServers),
		)
	}

	// now run NTP - we only fail if NTP fails, not if hardware clock sync fails
	l.Info("Trying to query NTP servers now to synchronize system clock...", zap.Strings("ntpServers", cfg.Services.NTPServers))
	if err := ntp.SyncClock(ctx, cfg.Services.NTPServers); err != nil && !errors.Is(err, ntp.ErrHWClockSync) {
		l.Error("Syncing system clock with NTP failed", zap.Error(err))
		return "", fmt.Errorf("syncing clock with NTP: %w", err)
	}
	l.Info("System clock successfully synchronized with NTP", zap.Strings("ntpServers", cfg.Services.NTPServers))

	// now try to download stage 1
	stage1Path := filepath.Join(stagingInfo.StagingDir, "stage1")
	if err := stage.DownloadExecutable(ctx, httpClient, cfg.Stage1URL, stage1Path, 60*time.Second); err != nil {
		l.Error("Downloading stage 1 installer failed", zap.String("url", cfg.Stage1URL), zap.String("dest", stage1Path), zap.Error(err))
		return "", fmt.Errorf("downloading stage 1: %w", err)
	}
	l.Info("Downloading stage 1 installer completed", zap.String("url", cfg.Stage1URL), zap.String("dest", stage1Path))

	// these are all the pieces which are dependent on the "right" network to work
	// we'll continue execution in the main function
	return stage1Path, nil
}
