package config

import (
	"go.githedgehog.com/dasboot/pkg/config"
)

var _ config.EmbeddedConfig = &Stage0{}

// Stage0 represents the structure of the config for the stage 0 installer.
//
// Here is an example JSON:
//
//	{
//	  "ca":"PEM encoded CA certificate",
//	  "onie_headers":{
//	    "ONIE-SERIAL-NUMBER":"XYZ123004",
//	    "ONIE-ETH-ADDR":"02:42:9b:5d:de:14",
//	    "ONIE-VENDOR-ID":12345,
//	    "ONIE-MACHINE":"VENDOR_MACHINE",
//	    "ONIE-MACHINE-REV":0,
//	    "ONIE-ARCH":"x86_64",
//	    "ONIE-SECURITY-KEY":"d3b07384d-ac-6238ad5ff00",
//	    "ONIE-OPERATION":"os-install"
//	  },
//	  "ipam_url":"https://fe80::4638:39ff:fe00/stage0/ipam"
//	}
type Stage0 struct {
	// CA is a DER encoded root certificate with which server connections to the control plane must be validated.
	// This can be empty if it is being dictated to be derived from attached USB sticks.
	// Either must be present though.
	CA []byte `json:"ca,omitempty" yaml:"ca,omitempty"`

	// OnieHeaders are the ONIE request headers as they were made by ONIE when downloading the stage 0 installer
	OnieHeaders *OnieHeaders `json:"onie_headers,omitempty" yaml:"onie_headers,omitempty"`

	// IPAMURL is the URL where the installer is going to get its IP and VLAN configuration from.
	IPAMURL string `json:"ipam_url,omitempty" yaml:"ipam_url,omitempty"`

	// SignatureCA holds the optional DER encoded CA certificate which signed 'signature_cert'. This should better
	// be derived from a different place.
	SignatureCA []byte `json:"signature_ca,omitempty" yaml:"signature_ca,omitempty"`

	// SignatureCert holds the DER encoded X509 certificate with which the signature of the embedded config
	// can be validated
	SignatureCert []byte `json:"signature_cert,omitempty" yaml:"signature_cert,omitempty"`

	// Version is tracking the format of this structure itself
	Version config.ConfigVersion `json:"version,omitempty" yaml:"version,omitempty"`
}

// OnieHeaders is being included by the control plane (seeder) when generating the
type OnieHeaders struct {
	// SerialNumber is the serial number as stored in the EEPROM
	SerialNumber string `json:"ONIE-SERIAL-NUMBER,omitempty" yaml:"ONIE-SERIAL-NUMBER,omitempty"`

	// EthAddr is the management MAC address
	EthAddr string `json:"ONIE-ETH-ADDR,omitempty" yaml:"ONIE-ETH-ADDR,omitempty"`

	// VendorID corresponds to the IANA enterprise number
	VendorID uint `json:"ONIE-VENDOR-ID,omitempty" yaml:"ONIE-VENDOR-ID,omitempty"`

	// Machine represents vendor and machine as a string. The format is <vendor>_<machine>
	Machine string `json:"ONIE-MACHINE,omitempty" yaml:"ONIE-MACHINE,omitempty"`

	// MachineRev refers to the machine revision (<machine_revision> in ONIE docs). The number 0 is a valid machine revision.
	MachineRev uint `json:"ONIE-MACHINE-REV" yaml:"ONIE-MACHINE-REV"` // don't use omitempty here

	// Arch is the CPU architecture of the calling device. E.g. "x86_64"
	Arch string `json:"ONIE-ARCH,omitempty" yaml:"ONIE-ARCH,omitempty"`

	// SecurityKey is the security key as can be set in ONIE.
	SecurityKey string `json:"ONIE-SECURITY-KEY,omitempty" yaml:"ONIE-SECURITY-KEY,omitempty"`

	// Operation will be either "install" or "onie-update".
	Operation string `json:"ONIE-OPERATION,omitempty" yaml:"ONIE-OPERATION,omitempty"`
}

// Cert implements config.EmbeddedConfig
func (c *Stage0) Cert() []byte {
	return c.SignatureCert
}

// Validate implements config.EmbeddedConfig
func (c *Stage0) Validate() error {
	// TODO: implement
	return nil
}

// ConfigVersion implements config.EmbeddedConfig
func (c *Stage0) ConfigVersion() config.ConfigVersion {
	return c.Version
}

// IsSupportedConfigVersion implements config.EmbeddedConfig
func (*Stage0) IsSupportedConfigVersion(v config.ConfigVersion) bool {
	return v == 1
}
