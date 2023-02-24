package devid

import "fmt"

// Tlv represents an ONIE TLV
type Tlv uint8

const (
	TlvSerial Tlv = 0x23
)

func (t Tlv) String() string {
	return fmt.Sprintf("0x%2x", uint8(t))
}
