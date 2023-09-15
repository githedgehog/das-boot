package ipam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"net"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	wiring1alpha2 "go.githedgehog.com/fabric/api/wiring/v1alpha2"
	"go.uber.org/zap"
)

// Settings needs to be passed in by the seeder to a ProcessRequest call
type Settings struct {
	ControlVIP    string
	DNSServers    []string
	SyslogServers []string
	NTPServers    []string
	KubeSubnets   []string
	Stage1URL     string
}

var (
	ErrUnsupportedArch = errors.New("ipam: unsupported architecture")
	ErrInvalidUUID     = errors.New("ipam: invalid uuid")
	ErrEmptyValue      = errors.New("ipam: empty value")
)

func unsupportedArchError(str string) error {
	return fmt.Errorf("%w: %s", ErrUnsupportedArch, str)
}

func invalidUUIDError(str string, err error) error {
	return fmt.Errorf("%w: %s: %w", ErrInvalidUUID, str, err)
}

func emptyValueError(str string) error {
	return fmt.Errorf("%w: %s", ErrEmptyValue, str)
}

// ProcessRequest processes an IPAM request and delivers back a response object.
func ProcessRequest(ctx context.Context, settings *Settings, cpc controlplane.Client, req *Request, adjacentSwitch *wiring1alpha2.Switch, adjacentConnection *wiring1alpha2.Connection) (*Response, error) {
	// ensure arch is supported
	var arch string
	switch req.Arch {
	case "x86_64":
		fallthrough
	case "arm64":
		fallthrough
	case "arm":
		arch = req.Arch
	default:
		return nil, unsupportedArchError(req.Arch)
	}

	if !strings.HasSuffix(settings.Stage1URL, arch) {
		return nil, fmt.Errorf("invalid Stage 1 URL '%s', must end in '%s'", settings.Stage1URL, arch)
	}

	// MOCKED VALUES
	// ips := mockedIPAddresses(req.Interfaces)

	// if the adjacent switch is filled, then we don't need to lookup the switch
	// otherwise we'll look it up by location first
	var err error
	var conns []wiring1alpha2.Connection
	if adjacentSwitch != nil {
		conns, err = cpc.GetSwitchConnections(ctx, adjacentSwitch.Name)
	} else {
		var sw *wiring1alpha2.Switch
		sw, err = cpc.GetSwitchByLocationUUID(ctx, req.LocationUUID)
		if err != nil {
			return nil, fmt.Errorf("finding switch: %w", err)
		}
		conns, err = cpc.GetSwitchConnections(ctx, sw.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("finding switch ports: %w", err)
	}

	reqIfs := make(map[string]any, len(req.Interfaces))
	for _, reqIf := range req.Interfaces {
		reqIfs[reqIf] = struct{}{}
	}

	ips := make(IPAddresses, len(req.Interfaces))
	for _, conn := range conns {
		if conn.Spec.Management != nil {
			// only return configuration if the port was in the requests
			if _, ok := reqIfs[conn.Spec.Management.Link.Switch.ONIEPortName]; !ok {
				log.L().Info("ipam: skipping connection for response as it was not in request", zap.String("conn", conn.Name), zap.String("oniePortName", conn.Spec.Management.Link.Switch.ONIEPortName))
				continue
			}

			// skip this port if it does not have ONIE configurations
			if conn.Spec.Management.Link.Switch.ONIEPortName == "" || conn.Spec.Management.Link.Switch.IP == "" {
				log.L().Info("ipam: skipping port for response as it is missing ONIE configuration", zap.String("conn", conn.Name))
				continue
			}

			// build the routes that we need to set in ONIE
			serverIP, err := ensureIPHasNoCIDR(conn.Spec.Management.Link.Server.IP)
			if err != nil {
				return nil, fmt.Errorf("extracting IP from CIDR notation failed for server IP: %w", err)
			}
			controlVIP, err := ensureIPHasCIDR(settings.ControlVIP)
			if err != nil {
				return nil, fmt.Errorf("ensuring control VIP has CIDR notation: %w", err)
			}
			routes := []*Route{
				{
					// the route to the controller over the server IP
					Destinations: []string{controlVIP},
					Gateway:      serverIP,
				},
				{
					// the route to access Kubernetes pods and services
					// NOTE: subject to change
					Destinations: settings.KubeSubnets,
					Gateway:      serverIP,
				},
			}

			// build the response for this port
			netif := conn.Spec.Management.Link.Switch.ONIEPortName
			ipa := IPAddress{
				IPAddresses: []string{conn.Spec.Management.Link.Switch.IP},
				Routes:      routes,
			}

			// if the adjacent port was passed in, then we'll let the
			// client know that this is the preferred connection to
			// try first before any other
			if adjacentConnection != nil && conn.Name == adjacentConnection.Name {
				ipa.Preferred = true
			}

			// last but not least, add it to the returned addresses
			ips[netif] = ipa
		}
	}

	// see if we built responses for all requested ports
	for _, reqIf := range req.Interfaces {
		if _, ok := ips[reqIf]; !ok {
			log.L().Info("ipam: missing IP address response for requested interface", zap.String("netif", reqIf))
		}
	}

	return &Response{
		IPAddresses:   ips,
		DNSServers:    settings.DNSServers,
		NTPServers:    settings.NTPServers,
		SyslogServers: settings.SyslogServers,
		Stage1URL:     settings.Stage1URL,
	}, nil
}

func ensureIPHasCIDR(ip string) (string, error) {
	// we assume IPv4 by default
	cidr := "32"

	// check if this is CIDR notation
	parsedIP, _, err := net.ParseCIDR(ip)
	if err == nil {
		if parsedIP.To4() == nil {
			cidr = "128"
		}
		// we ensure to return it with the given CIDR and throw out whatever other CIDR was there
		return fmt.Sprintf("%s/%s", parsedIP, cidr), nil
	}

	// if not, check if this is an IP at least
	parsedIP = net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("not a valid IP address: '%s'", ip)
	}
	if parsedIP.To4() == nil {
		cidr = "128"
	}

	// and we return this together with the provided CIDR
	return fmt.Sprintf("%s/%s", ip, cidr), nil
}

func ensureIPHasNoCIDR(ip string) (string, error) {
	parsedIP, _, err := net.ParseCIDR(ip)
	if err != nil {
		parsedIP = net.ParseIP(ip)
		if parsedIP == nil {
			return "", fmt.Errorf("not a valid IP/CIDR or IP address: '%s'", ip)
		}
	}
	return parsedIP.String(), nil
}
