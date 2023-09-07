package ipam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	wiring1alpha2 "go.githedgehog.com/fabric/api/wiring/v1alpha2"
	"go.uber.org/zap"
)

// Settings needs to be passed in by the seeder to a ProcessRequest call
type Settings struct {
	DNSServers    []string
	SyslogServers []string
	NTPServers    []string
	Stage1URL     string
	Routes        []*Route
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

			// build the response for this port
			netif := conn.Spec.Management.Link.Switch.ONIEPortName
			ipa := IPAddress{
				IPAddresses: []string{conn.Spec.Management.Link.Switch.IP},
				VLAN:        conn.Spec.Management.Link.Switch.VLAN,
				Routes:      settings.Routes,
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

// func mockedIPAddresses(interfaces []string) IPAddresses {
// 	ret := make(IPAddresses, len(interfaces))

// 	for _, netif := range interfaces {
// 		ret[netif] = IPAddress{
// 			IPAddresses: nextIP(),
// 			VLAN:        mockedVLAN(),
// 			Routes:      mockedRoutes(),
// 		}
// 	}

// 	return ret
// }

// func mockedVLAN() uint16 {
// 	return 42
// }

// func mockedRoutes() []*Route {
// 	return []*Route{
// 		{
// 			Destinations: []string{
// 				"10.42.0.0/16",
// 				"10.43.0.0/16",
// 			},
// 			Gateway: "192.168.42.1",
// 		},
// 	}
// }

// var curIP byte = 0

// func nextIP() []string {
// 	if curIP < 100 || curIP > 254 {
// 		curIP = 100
// 	} else {
// 		curIP += 1
// 	}

// 	ip := net.IPv4(192, 168, 42, curIP)

// 	return []string{ip.String() + "/24"}
// }
