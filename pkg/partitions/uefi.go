package partitions

import (
	"errors"
	"strings"

	"github.com/0x5a17ed/uefi/efi/efireader"
	"github.com/0x5a17ed/uefi/efi/efivario"
	"github.com/0x5a17ed/uefi/efi/efivars"
)

var efiCtx = efivario.NewDefaultContext()

var (
	ErrNotBootedIntoONIE = errors.New("uefi: not booted into ONIE")
	ErrEmptyBootOrder    = errors.New("uefi: boot order is empty")
)

func MakeONIEDefaultBootEntry() error {
	// get current boot entry number variable
	_, bootCurrentNumber, err := efivars.BootCurrent.Get(efiCtx)
	if err != nil {
		return err
	}

	// and read that boot variable entry
	entry := efivars.Boot(bootCurrentNumber)
	_, bootCurrent, err := entry.Get(efiCtx)
	if err != nil {
		return err
	}

	// we're interested in the description because this will tell us if this is ONIE or not
	// the description is UTF-16 encoded in these EFI variables
	bootCurrentDescription := efireader.UTF16ZBytesToString(bootCurrent.Description)

	// compare and see if this is ONIE
	// we're assuming that we're running ONIE, so the current boot entry must be ONIE
	if !strings.Contains(bootCurrentDescription, "ONIE") {
		return ErrNotBootedIntoONIE
	}

	// get the boot order variable now
	_, bootOrder, err := efivars.BootOrder.Get(efiCtx)
	if err != nil {
		return err
	}
	if len(bootOrder) <= 0 {
		return ErrEmptyBootOrder
	}

	// see if this needs adjustment
	if bootOrder[0] == bootCurrentNumber {
		// ONIE is already the first entry, we can stop here
		return nil
	}

	// we need to move ONIE up to the front
	// build a new boot order
	newBootOrder := []uint16{bootCurrentNumber}
	for _, num := range bootOrder {
		if num != bootCurrentNumber {
			newBootOrder = append(newBootOrder, num)
		}
	}

	// and write the boot order to the EFI variable
	return efivars.BootOrder.Set(efiCtx, newBootOrder)
}
