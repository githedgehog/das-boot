package partitions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/0x5a17ed/uefi/efi/efireader"
	"github.com/0x5a17ed/uefi/efi/efivario"
	"github.com/0x5a17ed/uefi/efi/efivars"
)

// One of the prerequisites is a test EFI Context.
// As this is an interface, we are going to build a mock for this
// by simply using go mock for this which we can easily replace
// then for the `efiCtx` variable in a test.
//
//go:generate mockgen -destination uefi_mock_efictx_test.go -package partitions github.com/0x5a17ed/uefi/efi/efivario Context
var efiCtx = efivario.NewDefaultContext()

var (
	ErrNotBootedIntoONIE = errors.New("uefi: not booted into ONIE")
	ErrEmptyBootOrder    = errors.New("uefi: boot order is empty")
)

// MakeONIEDefaultBootEntryAndCleanup will ensure that ONIE is the first boot
// entry in the EFI BootOrder variable. It assumes that the system is currently
// running ONIE, and it will determine the ONIE boot entry through the EFI
// BootCurrent variable and by checking that the boot entry description of the
// BootXXXX entry for BootCurrent actually contains the "ONIE" string, otherwise
// it will error.
//
// Furthermore it will consider all entries in the BootOrder variable that are
// located *before* ONIE as entries which need to be cleaned up / removed. It
// will remove them from the BootOrder, but it will also try to delete the EFI
// variables themselves in order not to accumulate bogus boot entries. An error
// deleting those variables is not considered an error, and only a warning log
// will be issued.
//
// If ONIE is not in the BootOrder list at all, this will simply prefix the
// BootOrder with ONIE.
//
// **NOTE:** This function is called by `Devices.DeletePartitions()`, and usually
// there should be no reason to call it byself.
func MakeONIEDefaultBootEntryAndCleanup() error {
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
	bootEntriesToDelete := []uint16{}
	var foundONIE bool
	for _, num := range bootOrder {
		if num == bootCurrentNumber {
			foundONIE = true
			continue
		}
		if !foundONIE {
			bootEntriesToDelete = append(bootEntriesToDelete, num)
			continue
		}
		newBootOrder = append(newBootOrder, num)
	}
	if !foundONIE {
		bootEntriesToDelete = []uint16{}
		newBootOrder = append(newBootOrder, bootOrder...)
	}

	// write the boot order to the EFI variable
	if err := efivars.BootOrder.Set(efiCtx, newBootOrder); err != nil {
		return err
	}

	// and now delete all entries which we need to delete
	for _, num := range bootEntriesToDelete {
		name := fmt.Sprintf("Boot%04X", num)
		if err := efiCtx.Delete(name, efivars.GlobalVariable); err != nil {
			Logger.Sugar().Warnf("uefi: deleting EFI var %s: %s", name, err)
		}
	}

	return nil
}
