package partitions

import (
	"errors"
	"fmt"
	"testing"

	efiguid "github.com/0x5a17ed/uefi/efi/efiguid"
	"github.com/0x5a17ed/uefi/efi/efivario"
	"github.com/0x5a17ed/uefi/efi/efivars"
	"github.com/golang/mock/gomock"
)

// contents of /sys/firmware/efi/efivars/Boot0007-8be4df61-93ca-11d2-aa0d-00e098032b8c
// which is the ONIE boot entry on that system where it was captured
var onieBootContents = []byte{
	0x07, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x5e, 0x00, 0x4f, 0x00, 0x4e, 0x00, 0x49, 0x00,
	0x45, 0x00, 0x3a, 0x00, 0x20, 0x00, 0x4f, 0x00, 0x70, 0x00, 0x65, 0x00, 0x6e, 0x00, 0x20, 0x00,
	0x4e, 0x00, 0x65, 0x00, 0x74, 0x00, 0x77, 0x00, 0x6f, 0x00, 0x72, 0x00, 0x6b, 0x00, 0x20, 0x00,
	0x49, 0x00, 0x6e, 0x00, 0x73, 0x00, 0x74, 0x00, 0x61, 0x00, 0x6c, 0x00, 0x6c, 0x00, 0x20, 0x00,
	0x45, 0x00, 0x6e, 0x00, 0x76, 0x00, 0x69, 0x00, 0x72, 0x00, 0x6f, 0x00, 0x6e, 0x00, 0x6d, 0x00,
	0x65, 0x00, 0x6e, 0x00, 0x74, 0x00, 0x00, 0x00, 0x04, 0x01, 0x2a, 0x00, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x9a, 0xfb, 0x1d, 0x69, 0x53, 0x47, 0x88, 0x43, 0xad, 0x51, 0xd4, 0xa1, 0xda, 0xac, 0x38, 0x6a,
	0x02, 0x02, 0x04, 0x04, 0x30, 0x00, 0x5c, 0x00, 0x45, 0x00, 0x46, 0x00, 0x49, 0x00, 0x5c, 0x00,
	0x6f, 0x00, 0x6e, 0x00, 0x69, 0x00, 0x65, 0x00, 0x5c, 0x00, 0x73, 0x00, 0x68, 0x00, 0x69, 0x00,
	0x6d, 0x00, 0x78, 0x00, 0x36, 0x00, 0x34, 0x00, 0x2e, 0x00, 0x65, 0x00, 0x66, 0x00, 0x69, 0x00,
	0x00, 0x00, 0x7f, 0xff, 0x04, 0x00,
}

func TestMakeONIEDefaultBootEntryAndCleanup(t *testing.T) {
	// contents of /sys/firmware/efi/efivars/Boot0003-8be4df61-93ca-11d2-aa0d-00e098032b8c
	// which is the shim boot entry a local Arch Linux installation (definitely not ONIE)
	shimBootContents := []byte{
		0x07, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x5e, 0x00, 0x53, 0x00, 0x68, 0x00, 0x69, 0x00,
		0x6d, 0x00, 0x00, 0x00, 0x04, 0x01, 0x2a, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x87, 0x16, 0xc4, 0x47,
		0xfd, 0x67, 0xac, 0x4b, 0x99, 0x94, 0x81, 0x5d, 0x5c, 0x01, 0x6b, 0x65, 0x02, 0x02, 0x04, 0x04,
		0x30, 0x00, 0x5c, 0x00, 0x45, 0x00, 0x46, 0x00, 0x49, 0x00, 0x5c, 0x00, 0x73, 0x00, 0x68, 0x00,
		0x69, 0x00, 0x6d, 0x00, 0x5c, 0x00, 0x73, 0x00, 0x68, 0x00, 0x69, 0x00, 0x6d, 0x00, 0x78, 0x00,
		0x36, 0x00, 0x34, 0x00, 0x2e, 0x00, 0x65, 0x00, 0x66, 0x00, 0x69, 0x00, 0x00, 0x00, 0x7f, 0xff,
		0x04, 0x00,
	}

	errSetFailed := errors.New("EFI Set() failed")
	errBootOrderGetFailed := errors.New("EFI BootOrder.Get() failed")
	errGetBootCurrentFailed := errors.New("EFI BootCurrent.Get() failed")
	errGetBootXXXXFailed := errors.New("EFI BootXXXX.Get() failed")

	tests := []struct {
		name        string
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, c *MockContext)
	}{
		{
			name: "success without adjustments",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns `onieBootContents`
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(onieBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(onieBootContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range onieBootContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(onieBootContents) - 4, nil
					})

				// BootOrder - returns `bootOrderContents`
				bootOrderContents := []byte{
					0x07, 0x00, 0x00, 0x00, 0x07, 0x00, 0x0b, 0x00, 0x00, 0x00, 0x01, 0x00, 0x06, 0x00, 0x02, 0x00,
					0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
				}
				c.EXPECT().GetSizeHint(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(bootOrderContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(bootOrderContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range bootOrderContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(bootOrderContents) - 4, nil
					})
			},
			wantErr: false,
		},
		{
			name: "success needing adjustments",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns `onieBootContents`
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(onieBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(onieBootContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range onieBootContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(onieBootContents) - 4, nil
					})

				// BootOrder - returns `bootOrderContents`
				bootOrderContents := []byte{
					0x07, 0x00, 0x00, 0x00, 0x0b, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x07, 0x00, 0x02, 0x00,
					0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
				}
				// the above boot order contents will need to remove 0b, 01, 00, 06
				c.EXPECT().GetSizeHint(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(bootOrderContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(bootOrderContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range bootOrderContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(bootOrderContents) - 4, nil
					})

				// Set new expected BootOrder
				expectedBootOrderContents := []byte{
					0x07, 0x00, 0x0b, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x02, 0x00,
					0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
				}
				c.EXPECT().Set(
					gomock.Eq("BootOrder"),
					gomock.Eq(efivars.GlobalVariable),
					gomock.Eq(efivario.BootServiceAccess|efivario.RuntimeAccess|efivario.NonVolatile),
					gomock.Eq(expectedBootOrderContents),
				).Times(1).
					Return(nil)

				// now we expect four deletes: 0b, 01, 00, 06
				c.EXPECT().Delete(gomock.Eq("Boot000B"), gomock.Eq(efivars.GlobalVariable)).Times(1).Return(nil)
				c.EXPECT().Delete(gomock.Eq("Boot0001"), gomock.Eq(efivars.GlobalVariable)).Times(1).Return(nil)
				c.EXPECT().Delete(gomock.Eq("Boot0000"), gomock.Eq(efivars.GlobalVariable)).Times(1).Return(fmt.Errorf("ignored error"))
				c.EXPECT().Delete(gomock.Eq("Boot0006"), gomock.Eq(efivars.GlobalVariable)).Times(1).Return(nil)
			},
			wantErr: false,
		},
		{
			// no need to delete entries because ONIE is not part of the boot order
			name: "success and no need to delete entries",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns `onieBootContents`
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(onieBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(onieBootContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range onieBootContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(onieBootContents) - 4, nil
					})

				// BootOrder - returns `bootOrderContents`
				// this boot order is missing ONIE
				bootOrderContents := []byte{
					0x07, 0x00, 0x00, 0x00, 0x0b, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x02, 0x00,
					0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
				}
				// the above boot order contents do not conain ONIE, so there must nothing being deleted
				c.EXPECT().GetSizeHint(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(bootOrderContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(bootOrderContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range bootOrderContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(bootOrderContents) - 4, nil
					})

				// Set new expected BootOrder
				// this order will simply have ONIE prepended
				expectedBootOrderContents := []byte{
					0x07, 0x00, 0x0b, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x02, 0x00,
					0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
				}
				c.EXPECT().Set(
					gomock.Eq("BootOrder"),
					gomock.Eq(efivars.GlobalVariable),
					gomock.Eq(efivario.BootServiceAccess|efivario.RuntimeAccess|efivario.NonVolatile),
					gomock.Eq(expectedBootOrderContents),
				).Times(1).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "set BootOrder fails",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns `onieBootContents`
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(onieBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(onieBootContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range onieBootContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(onieBootContents) - 4, nil
					})

				// BootOrder - returns `bootOrderContents`
				// this boot order is missing ONIE
				bootOrderContents := []byte{
					0x07, 0x00, 0x00, 0x00, 0x0b, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x02, 0x00,
					0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
				}
				// the above boot order contents do not conain ONIE, so there must nothing being deleted
				c.EXPECT().GetSizeHint(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(bootOrderContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(bootOrderContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range bootOrderContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(bootOrderContents) - 4, nil
					})

				// Set new expected BootOrder
				// this order will simply have ONIE prepended
				expectedBootOrderContents := []byte{
					0x07, 0x00, 0x0b, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x02, 0x00,
					0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
				}
				c.EXPECT().Set(
					gomock.Eq("BootOrder"),
					gomock.Eq(efivars.GlobalVariable),
					gomock.Eq(efivario.BootServiceAccess|efivario.RuntimeAccess|efivario.NonVolatile),
					gomock.Eq(expectedBootOrderContents),
				).Times(1).
					Return(errSetFailed)
			},
			wantErr:     true,
			wantErrToBe: errSetFailed,
		},
		{
			name: "BootOrder returns empty",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns `onieBootContents`
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(onieBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(onieBootContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range onieBootContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(onieBootContents) - 4, nil
					})

				// BootOrder - returns empty
				// the above boot order contents do not conain ONIE, so there must nothing being deleted
				c.EXPECT().GetSizeHint(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(0), nil)
				c.EXPECT().Get(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, 0))).Times(1).
					Return(efivario.BootServiceAccess|efivario.RuntimeAccess|efivario.NonVolatile, 0, nil)
			},
			wantErr:     true,
			wantErrToBe: ErrEmptyBootOrder,
		},
		{
			name: "get BootOrder fails",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns `onieBootContents`
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(onieBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(onieBootContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range onieBootContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(onieBootContents) - 4, nil
					})

				// BootOrder - returns error
				// the above boot order contents do not conain ONIE, so there must nothing being deleted
				c.EXPECT().GetSizeHint(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(0), nil)
				c.EXPECT().Get(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, 0))).Times(1).
					Return(efivario.Attributes(0), 0, errBootOrderGetFailed)
			},
			wantErr:     true,
			wantErrToBe: errBootOrderGetFailed,
		},
		{
			name: "get BootOrder fails",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns `shimBootContents`
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(shimBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(shimBootContents)-4))).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						for i, b := range shimBootContents[4:] {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(shimBootContents) - 4, nil
					})
			},
			wantErr:     true,
			wantErrToBe: ErrNotBootedIntoONIE,
		},
		{
			name: "get BootXXXX fails",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns 7
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
						// contents of BootCurrent are 6 bytes:
						// - first 4 bytes are Attributes, just return them directly
						// - last two bytes are little endian int16
						// the Get function sets it to the `out` variable
						// []byte{0x06, 0x00, 0x00, 0x00, 0x07, 0x00}
						for i, b := range []byte{0x07, 0x00} {
							out[i] = b
						}
						return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
					})

				// Boot0007 - returns error
				c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(len(shimBootContents)-4), nil)
				c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(shimBootContents)-4))).Times(1).
					Return(efivario.Attributes(0), 0, errGetBootXXXXFailed)
			},
			wantErr:     true,
			wantErrToBe: errGetBootXXXXFailed,
		},
		{
			name: "get BootCurrent fails",
			pre: func(t *testing.T, c *MockContext) {
				// BootCurrent - returns error
				c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
					Return(int64(2), nil)
				c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
					Return(efivario.Attributes(0), 0, errGetBootCurrentFailed)
			},
			wantErr:     true,
			wantErrToBe: errGetBootCurrentFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			c := NewMockContext(ctrl)
			oldEfiCtx := efiCtx
			defer func() {
				efiCtx = oldEfiCtx
			}()
			efiCtx = c
			if tt.pre != nil {
				tt.pre(t, c)
			}
			err := MakeONIEDefaultBootEntryAndCleanup()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetONIEBootEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.Delete() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}
