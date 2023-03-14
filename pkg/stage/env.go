package stage

import "os"

// OnieEnv represents a set of environment variables that *should* always
// be set in any running ONIE installer
type OnieEnv struct {
	ExecURL   string
	Platform  string
	VendorID  string
	SerialNum string
	EthAddr   string
}

// GetOnieEnv returns the set of ONIE environment variables that *should* always
// bet in any running ONIE installer
func GetOnieEnv() *OnieEnv {
	return &OnieEnv{
		ExecURL:   os.Getenv("onie_exec_url"),
		Platform:  os.Getenv("onie_platform"),
		VendorID:  os.Getenv("onie_vendor_id"),
		SerialNum: os.Getenv("onie_serial_num"),
		EthAddr:   os.Getenv("onie_eth_addr"),
	}
}
