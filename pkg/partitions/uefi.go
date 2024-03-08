// Copyright 2023 Hedgehog
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package partitions

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/0x5a17ed/uefi/efi/efireader"
	"github.com/0x5a17ed/uefi/efi/efivario"
	"github.com/0x5a17ed/uefi/efi/efivars"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.uber.org/zap"
)

// One of the prerequisites is a test EFI Context.
// As this is an interface, we are going to build a mock for this
// by simply using go mock for this which we can easily replace
// then for the `efiCtx` variable in a test.
//
//go:generate mockgen -destination ../../test/mock/mockuefi/uefi_mock_efictx.go -package mockuefi github.com/0x5a17ed/uefi/efi/efivario Context
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
	// Check first that we are booted into ONIE. We are making that assumption based on the /etc/os-release file at the moment.
	isBootedIntoONIE, err := IsBootedIntoONIE()
	if err != nil {
		return fmt.Errorf("%w: failed to detect if we are booted into ONIE: %w", ErrNotBootedIntoONIE, err)
	}
	if !isBootedIntoONIE {
		return ErrNotBootedIntoONIE
	}

	// get ONIE boot entry variable
	onieBootEntryNumber, err := FindONIEBootEntry()
	if err != nil {
		return fmt.Errorf("uefi: finding ONIE boot entry: %w", err)
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
	if bootOrder[0] == onieBootEntryNumber {
		// ONIE is already the first entry, we can stop here
		return nil
	}

	// we need to move ONIE up to the front
	// build a new boot order
	newBootOrder := []uint16{onieBootEntryNumber}
	bootEntriesToDelete := []uint16{}
	var foundONIE bool
	for _, num := range bootOrder {
		if num == onieBootEntryNumber {
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

	// prepare a string that we use for logging and errors
	newBootOrderStrings := make([]string, 0, len(newBootOrder))
	for _, num := range newBootOrder {
		newBootOrderStrings = append(newBootOrderStrings, fmt.Sprintf("%04X", num))
	}
	newBootOrderStr := strings.Join(newBootOrderStrings, ",")

	// write the boot order to the EFI variable
	if err := efivars.BootOrder.Set(efiCtx, newBootOrder); err != nil {
		return fmt.Errorf("uefi: setting BootOrder to '%s': %w", newBootOrderStr, err)
	}
	log.L().Info("uefi: successfully set EFI BootOrder variable", zap.String("BootOrder", newBootOrderStr))

	// and now delete all entries which we need to delete
	for _, num := range bootEntriesToDelete {
		name := fmt.Sprintf("Boot%04X", num)
		if err := efiCtx.Delete(name, efivars.GlobalVariable); err != nil {
			log.L().Warn("uefi: deleting stale EFI variable failed", zap.String("efivar", name), zap.Error(err))
		}
		log.L().Info("uefi: successfully deleted stale EFI variable", zap.String("efivar", name))
	}

	return nil
}

// osReleasePath points to /etc/os-release. It's a var instead of a const so that we can change it in unit tests.
var osReleasePath = "/etc/os-release"

// IsBootedIntoONIE checks the running OS to see if this truly is running ONIE
func IsBootedIntoONIE() (bool, error) {
	f, err := os.Open(osReleasePath)
	if err != nil {
		return false, fmt.Errorf("failed to open '%s': %w", osReleasePath, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.SplitN(line, "=", 2)
		if len(split) != 2 {
			continue
		}
		key := strings.TrimSpace(split[0])
		if strings.ToUpper(key) != "NAME" {
			continue
		}
		val := strings.ToLower(strings.Trim(strings.TrimSpace(split[1]), "\"'"))
		return strings.Contains(val, "onie"), nil
	}
	return false, nil
}

// FindONIEBootEntry will find the UEFI ONIE boot entry
func FindONIEBootEntry() (uint16, error) {
	bootIterator, err := efivars.BootIterator(efiCtx)
	if err != nil {
		return 0, fmt.Errorf("failed to get BootIterator: %w", err)
	}
	defer bootIterator.Close()

	for bootIterator.Next() {
		bootEntry := bootIterator.Value()
		_, bootEntryLoadOptions, err := bootEntry.Variable.Get(efiCtx)
		if err != nil {
			continue
		}
		desc := efireader.UTF16ZBytesToString(bootEntryLoadOptions.Description)
		if strings.Contains(desc, "ONIE") {
			return bootEntry.Index, nil
		}
	}
	if err := bootIterator.Err(); err != nil {
		return 0, fmt.Errorf("BootIterator aborted: %w", err)
	}
	return 0, fmt.Errorf("not found")
}
