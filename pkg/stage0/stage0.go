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
	"strings"
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

	// now issue the IPAM request
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

	// we can configure DNS just once
	if err := dns.SetSystemResolvers(ipamResp.DNSServers); err != nil {
		l.Error("Configuring system DNS resolver failed", zap.String("systemDNSResolverFile", "/etc/resolv.conf"), zap.Error(err))
		return executionError(err)
	}
	l.Info("System DNS resolver successfully configured", zap.String("systemDNSResolverFile", "/etc/resolv.conf"))

	// for the rest until we finished downloading stage 1, we iterate over all IP addresses that we got back
	// and essentially retry the rest of stage 0 until it works
	var stage1Path string
	for netdev, ipa := range ipamResp.IPAddresses {
		var err error
		stage1Path, err = runWith(ctx, stagingInfo, logSettings, httpClient, ipamResp, netdev, ipa)
		if err != nil {
			l.Error("System network configuration failed for netdev", zap.String("netdev", netdev), zap.Reflect("ipa", ipa), zap.Error(err))
			if err := net.DeleteVLANDevice(vlanName); err != nil {
				l.Warn("Deleting VLAN device failed", zap.String("vlanDevice", vlanName), zap.Error(err))
			}
			continue
		}
		l.Info("System network configured", zap.String("netdev", netdev), zap.Reflect("ipa", ipa))
		break
	}
	if stage1Path == "" {
		l.Error("System network configuration failed for all network devices")
		return ErrExecution
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

func runWith(ctx context.Context, stagingInfo *stage.StagingInfo, logSettings *stage.LogSettings, httpClient *http.Client, ipamResp *ipam.Response, netdev string, ipa ipam.IPAddress) (string, error) {
	// first things first: configure network interface
	ipaddrnets, err := net.StringsToIPNets(ipa.IPAddresses)
	if err != nil {
		l.Error("Conversion of IP addresses to IPNets failed", zap.String("netdev", netdev), zap.Reflect("ipAddresses", ipa.IPAddresses))
		return "", fmt.Errorf("converting IP addresses to IPNets: %w", err)
	}
	var routes []*net.Route
	if len(ipa.Routes) > 0 {
		for _, route := range ipa.Routes {
			dests, err := net.StringsToIPNets(route.Destinations)
			if err != nil {
				l.Error("Conversion of IP routes destinations to IPNets failed", zap.String("netdev", netdev), zap.Reflect("routes", ipa.Routes), zap.Error(err))
				return "", fmt.Errorf("converting routes destinations to IPNets: %w", err)
			}
			gw := gonet.ParseIP(route.Gateway)
			if gw == nil {
				l.Error("Conversion of IP gateway string to IP failed", zap.String("netdev", netdev), zap.String("gw", route.Gateway))
				return "", fmt.Errorf("converting routes gateway '%s' to IP failed", route.Gateway)
			}
			routes = append(routes, &net.Route{
				Dests: dests,
				Gw:    gw,
			})
		}
	}
	if err := net.AddVLANDeviceWithIP(netdev, ipa.VLAN, vlanName, ipaddrnets, routes); err != nil {
		l.Error("VLAN interface creation and configuration failed",
			zap.String("netdev", netdev),
			zap.String("vlanInterface", vlanName),
			zap.Reflect("ipa", ipa),
			zap.Error(err),
		)
		return "", fmt.Errorf("add vlan device with IP: %w", err)
	}
	l.Info("VLAN interface successfully created and configured",
		zap.String("netdev", netdev),
		zap.String("vlanInterface", vlanName),
		zap.Reflect("ipa", ipa),
	)

	// configure the syslog logger so that we're not blind anymore
	logSettings.SyslogServers = ipamResp.SyslogServers
	if err := stage.InitializeGlobalLogger(ctx, logSettings); err != nil {
		l.Warn("Reinitializing global logger with new settings including syslog servers failed", zap.Strings("syslogServers", ipamResp.SyslogServers), zap.Error(err))
	} else {
		l = log.L()
		l.Info("Reinitialized global logger with new settings including syslog servers",
			zap.Strings("syslogServers", ipamResp.SyslogServers),
		)
	}

	// now run NTP - we only fail if NTP fails, not if hardware clock sync fails
	if err := ntp.SyncClock(ctx, ipamResp.NTPServers); err != nil && !errors.Is(err, ntp.ErrHWClockSync) {
		l.Error("Syncing system clock with NTP failed", zap.Error(err))
		return "", fmt.Errorf("syncing clock with NTP: %w", err)
	}
	l.Info("System clock successfully synchronized with NTP", zap.Strings("ntpServers", ipamResp.NTPServers))

	// now try to download stage 1
	stage1Path := filepath.Join(stagingInfo.StagingDir, "stage1")
	if err := stage.DownloadExecutable(ctx, httpClient, ipamResp.Stage1URL, stage1Path, 60*time.Second); err != nil {
		l.Error("Downloading stage 1 installer failed", zap.String("url", ipamResp.Stage1URL), zap.String("dest", stage1Path), zap.Error(err))
		return "", fmt.Errorf("downloading stage 1: %w", err)
	}
	l.Info("Downloading stage 1 installer completed", zap.String("url", ipamResp.Stage1URL), zap.String("dest", stage1Path))

	// these are all the pieces which are dependent on the "right" network to work
	// we'll continue execution in the main function
	return stage1Path, nil
}

func ipamClient(ctx context.Context, hc *http.Client, ipamURL string, req *ipam.Request, onieEnv *stage.OnieEnv) (*ipam.Response, error) {
	url, err := url.Parse(ipamURL)
	if err != nil {
		return nil, fmt.Errorf("IPAM URL validation error: %w", err)
	}

	// if the IPAM URL is not a link-local address host, we can short-circuit here
	if !strings.HasPrefix(url.Host, "[fe80:") {
		l.Debug("IPAM URL does not have a link-local host", zap.String("host", url.Host))
		return ipam.DoRequest(ctx, hc, req, ipamURL)
	}

	// check if this is from within an ONIE installer
	// because then we are going to assume that we want to use the same interface that
	// was used to download the stage 0 installer
	// that is of course only the case if this was downloaded from a link-local address URL
	if strings.Contains(onieEnv.ExecURL, "[fe80:") {
		l.Warn("IPAM URL is on a link-local host, as was the stage 0 installer. We are trying to reuse the same interface for the request.", zap.String("ExecURL", onieEnv.ExecURL))

		// ONIE doesn't get URL encoding right for the host
		// so we try to account for this here before we parse the URL
		// we know that our URL path will not have any '%' characters
		// so if ONIE got the encoding right, then there will be a '%25'
		// otherwise it will be a single '%' in the URL, and also the first one
		// as this is a link-local
		execURLStr := onieEnv.ExecURL
		if !strings.Contains(onieEnv.ExecURL, "%25") {
			execURLStr = strings.Replace(onieEnv.ExecURL, "%", "%25", 1)
		}
		execURL, err := url.Parse(execURLStr)
		if err != nil {
			return nil, fmt.Errorf("ONIE Exec URL validation error: %w", err)
		}
		host := strings.SplitN(execURL.Host, "%", 2)
		if len(host) != 2 {
			return nil, fmt.Errorf("ONIE Exec URL Host splitting issue: [Host: '%s', Splitted Parts: %d]", execURL.Host, len(host))
		}

		netdev := strings.TrimRight(host[1], "]")
		hostTrimmed := strings.TrimLeft(strings.TrimRight(url.Host, "]"), "[")

		// now adjust the URL, and use it
		url.Host = "[" + hostTrimmed + "%" + netdev + "]"
		return ipam.DoRequest(ctx, hc, req, url.String())
	}

	// otherwise this is probably being executed from the ONIE rescue system
	// we will simply try the request on all interfaces
	l.Warn("IPAM URL is on a link-local host, and failed to detect which network interface to use. We will try all of them", zap.Strings("netdevs", req.Interfaces))
	urlHost := url.Host
	for _, netdev := range req.Interfaces {
		url.Host = urlHost + "%" + netdev
		resp, err := ipam.DoRequest(ctx, hc, req, url.String())
		if err != nil {
			l.Error("IPAM request failure", zap.String("netdev", netdev), zap.String("url", url.String()), zap.Reflect("ipamRequest", req), zap.Error(err))
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("request failed on all network interfaces [%s]", strings.Join(req.Interfaces, ","))
}
