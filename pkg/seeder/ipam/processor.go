package ipam

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
)

// Settings needs to be passed in by the seeder to a ProcessRequest call
type Settings struct {
	DNSServers    []string
	SyslogServers []string
	NTPServers    []string
	Stage1URL     string
}

var (
	ErrUnsupportedArch = errors.New("ipam: unsupported architecture")
)

func unsupportedArchError(str string) error {
	return fmt.Errorf("%w: %s", ErrUnsupportedArch, str)
}

// ProcessRequest processes an IPAM request and delivers back a response object.
func ProcessRequest(ctx context.Context, settings *Settings, cpc controlplane.Client, req *Request) (*Response, error) {
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

	if !strings.HasSuffix(settings.Stage1URL, "/") {
		settings.Stage1URL += "/"
	}

	// MOCKED VALUES
	vlan := mockedVLAN()
	ips := mockedIPAddresses(req.Interfaces)

	return &Response{
		VLAN:          vlan,
		IPAddresses:   ips,
		DNSServers:    settings.DNSServers,
		NTPServers:    settings.NTPServers,
		SyslogServers: settings.SyslogServers,
		Stage1URL:     settings.Stage1URL + arch,
	}, nil
}

func mockedIPAddresses(interfaces []string) IPAddresses {
	ret := make(IPAddresses, len(interfaces))

	for _, netif := range interfaces {
		ret[netif] = nextIP()
	}

	return ret
}

func mockedVLAN() uint16 {
	return 42
}

var curIP byte = 0

func nextIP() []string {
	if curIP < 100 || curIP > 254 {
		curIP = 100
	} else {
		curIP += 1
	}

	ip := net.IPv4(192, 168, 42, curIP)

	return []string{ip.String()}
}
