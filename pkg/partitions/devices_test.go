package partitions

import (
	"reflect"
	"testing"
)

func TestDevices_GetEFIPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetEFIPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetEFIPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetONIEPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeONIE,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetONIEPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetONIEPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetDiagPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype:  UeventDevtypePartition,
						UeventPartname: "HH-DIAG",
					},
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype:  UeventDevtypePartition,
					UeventPartname: "HH-DIAG",
				},
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetDiagPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetDiagPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetHedgehogIdentityPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogLocation,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetHedgehogIdentityPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetHedgehogIdentityPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetHedgehogLocationPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogLocation,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogLocation,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetHedgehogLocationPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetHedgehogLocationPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}
