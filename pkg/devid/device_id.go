package devid

import (
	"bufio"
	"bytes"
	"crypto/x509/pkix"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"go.githedgehog.com/dasboot/pkg/exec"
	dbfilepath "go.githedgehog.com/dasboot/pkg/filepath"
	"go.githedgehog.com/dasboot/pkg/log"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var l = log.L()

// for unit testing
var (
	arch      = runtime.GOARCH
	rootPath  = "/"
	ioReadAll = io.ReadAll
)

const (
	onieSysinfo   = "onie-sysinfo"
	onieSyseeprom = "onie-syseeprom"
)

var (
	ErrBogusCPUSerial           = errors.New("devid: bogus CPU Serial number")
	ErrCPUSerialNotFound        = errors.New("devid: CPU Serial number not found")
	ErrNoNetdevs                = errors.New("devid: No network devices found")
	ErrNoMACAddressesForNetdevs = errors.New("devid: no MAC addresses for any network devices found")
)

// ID will return a unique Device ID for the device that the function is running on.
func ID() string {
	// see https://docs.x.githedgehog.com/dasboot/device-identification.html
	// for details on the algorithm used
	//
	// 1. ONIE vendor ID + serial
	ret, err := idFromVendorIDAndSerial()
	if err == nil {
		return ret
	}
	l.Warn("unable to determine device ID through vendor ID and device serial number using ONIE commands", zap.Error(err))

	// 2.1 on x86_64: System UUID - cat /sys/class/dmi/id/product_uuid if not a list of known bad system UUIDs
	if arch == "amd64" || arch == "386" {
		ret, err = idFromSystemUUID()
		if err == nil {
			return ret
		}
		l.Warn("unable to determine device ID through System UUID", zap.Error(err))
	}

	// 2.2 on ARM: Serial of CPU - grep Serial /proc/cpuinfo if set, and not a bogus serial (like all zeros)
	if arch == "arm64" || arch == "arm" {
		ret, err = idFromCPUInfo()
		if err == nil {
			return ret
		}
		l.Warn("unable to determine device ID through CPU serial number", zap.Error(err))
	}

	// 3. all NIC MAC addresses
	ret, err = idFromMACAddresses()
	if err == nil {
		return ret
	}
	l.Error("unable to determine device ID through MAC addresses", zap.Error(err))

	// you really have a problem if you get down here
	// nothing more we can do
	return ""
}

func idFromVendorIDAndSerial() (string, error) {
	// calling onie-sysinfo for the vendor ID
	out, err := exec.Command(onieSysinfo, "-i").Output()
	if err != nil {
		return "", err
	}
	vendorID := string(out)

	// calling onie-syseeprom for the serial
	out, err = exec.Command(onieSyseeprom, "-g", TlvSerial.String()).Output()
	if err != nil {
		return "", err
	}
	serial := string(out)

	dn := pkix.Name{
		Organization: []string{vendorID},
		CommonName:   serial,
	}
	return uuid.NewSHA1(uuid.NameSpaceX500, []byte(dn.String())).String(), nil
}

func idFromSystemUUID() (string, error) {
	// simply read /sys/class/dmi/id/product_uuid
	path := filepath.Join(rootPath, "sys", "class", "dmi", "id", "product_uuid")
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	b, err := ioReadAll(f)
	if err != nil {
		return "", err
	}

	// we'll parse it to ensure the vendors didn't put silly things in there
	id, err := uuid.ParseBytes(bytes.TrimSpace(b))
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func idFromCPUInfo() (string, error) {
	// read from /proc/cpuinfo
	// try to find a Serial entry
	path := filepath.Join(rootPath, "proc", "cpuinfo")
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.SplitN(line, ":", 2)
		if len(split) != 2 {
			continue
		}
		key := strings.TrimSpace(split[0])
		if key == "Serial" {
			val := strings.TrimSpace(split[1])

			// ensure it's not bogus
			onlyZero := true
			for _, v := range val {
				if v == '0' {
					continue
				}
				onlyZero = false
				break
			}
			if onlyZero {
				return "", ErrBogusCPUSerial
			}

			// now create a UUID from it
			dn := pkix.Name{
				Organization: []string{arch},
				CommonName:   val,
			}
			return uuid.NewSHA1(uuid.NameSpaceX500, []byte(dn.String())).String(), nil
		}
	}
	return "", ErrCPUSerialNotFound
}

func idFromMACAddresses() (string, error) {
	path := filepath.Join(rootPath, "sys", "class", "net")

	// first find all physical hardware devices
	// they are the ones which will have a symbolic "device" link in their folder
	devs := []string{}
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Name() == "device" {
			// we now konw that this is a *real* physical device
			// we will build paths from here
			devs = append(devs, filepath.Base(filepath.Dir(path)))
		}
		return nil
	}
	_ = dbfilepath.WalkDir(path, walkFunc, 1, "subsystem", "device")
	if len(devs) == 0 {
		return "", ErrNoNetdevs
	}

	// now read all the mac addresses
	macs := make([]string, 0, len(devs))
	for _, dev := range devs {
		f, err := os.Open(filepath.Join(path, dev, "address"))
		if err != nil {
			return "", err
		}
		b, err := ioReadAll(f)
		if err != nil {
			return "", err
		}
		mac := string(bytes.TrimSpace(b))
		if len(mac) > 0 {
			// there might be a chance that there is no HW address
			// seen happening for wwan devices which aren't unlocked for example
			macs = append(macs, mac)
		}
	}
	if len(macs) == 0 {
		return "", ErrNoMACAddressesForNetdevs
	}

	// make this more deterministic by sorting the macs
	// this way different device naming schemes don't have an impact on the
	// calculations here
	sort.Strings(macs)

	// build a UUID from them
	dn := pkix.Name{
		Organization:       []string{"MAC"},
		OrganizationalUnit: macs,
	}
	return uuid.NewSHA1(uuid.NameSpaceX500, []byte(dn.String())).String(), nil
}
