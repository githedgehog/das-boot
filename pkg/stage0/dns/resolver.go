package dns

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"text/template"
)

var (
	ErrNoServers        = errors.New("dns: empty server list")
	ErrInvalidIPAddress = errors.New("dns: invalid IP address")
)

func invalidIPAddressError(str string) error {
	return fmt.Errorf("%w: %s", ErrInvalidIPAddress, str)
}

const resolvconfTemplate = `# Hedgehog DAS BOOT
# This DNS resolver configuration was being derived by the stage0 installer.
{{ range . }}
nameserver {{.}}
{{- end }}

options edns0 trust-ad timeout:5 attempts:2 rotate
search .
`

const (
	etcResolvConfPath = "/etc/resolv.conf"
)

// for unit testing
var (
	osOpenFile func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) = func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
		return os.OpenFile(name, flag, perm)
	}
)

// SetSystemResolvers is going to program the system DNS servers in the /etc/resolv.conf.
// Any previous configuration in the file will be overwritten.
func SetSystemResolvers(servers []string) error {
	// validate servers
	if len(servers) == 0 {
		return ErrNoServers
	}
	for _, server := range servers {
		if net.ParseIP(server) == nil {
			return invalidIPAddressError(server)
		}
	}

	f, err := osOpenFile(etcResolvConfPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("dns: open '%s': %w", etcResolvConfPath, err)
	}
	defer f.Close()

	t := template.Must(template.New("resolvconf").Parse(resolvconfTemplate))
	if err := t.Execute(f, servers); err != nil {
		return fmt.Errorf("dns: template write to '%s': %w", etcResolvConfPath, err)
	}

	return nil
}
