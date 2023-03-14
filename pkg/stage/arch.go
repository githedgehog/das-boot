package stage

import "runtime"

// Arch will return an ONIE/SONiC compatbile architecture string based on
// the Go runtime which is executing this function. Any architecture which
// we do not support at this point in time will return "unsupported".
func Arch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm"
	default:
		return "unsupported"
	}
}
