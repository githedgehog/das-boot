package net

import (
	"errors"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

var ErrNotAVlanDevice = errors.New("net: not a vlan device")

func notAVlanDeviceError(str string) error {
	return fmt.Errorf("%w: %s", ErrNotAVlanDevice, str)
}

// AddVLANDeviceWithIP will create a new VLAN network interface called `vlanName` with VLAN ID `vid` and add it to
// the parent network interface `device`. It will also add all IP addresses as given with `ipaddrnets`, and, last
// but not least, it will set the interface UP.
func AddVLANDeviceWithIP(device string, vid uint16, vlanName string, ipaddrnets []*net.IPNet) error {
	// get the parent device
	pl, err := netlink.LinkByName(device)
	if err != nil {
		return err
	}

	// create a vlan link
	la := netlink.NewLinkAttrs()
	la.Name = vlanName
	la.ParentIndex = pl.Attrs().Index
	vlan := &netlink.Vlan{
		LinkAttrs:    la,
		VlanId:       int(vid),
		VlanProtocol: netlink.VLAN_PROTOCOL_8021Q,
	}

	// add the vlan link
	if err := netlink.LinkAdd(vlan); err != nil {
		return err
	}

	// now add the IP address
	for _, ipaddrnet := range ipaddrnets {
		addr := &netlink.Addr{
			IPNet: ipaddrnet,
		}
		if err := netlink.AddrAdd(vlan, addr); err != nil {
			return err
		}
	}

	// set the interface up
	if err := netlink.LinkSetUp(vlan); err != nil {
		return err
	}

	// that's it - that was easy
	return nil
}

// DeleteVLANDevice will delete the network interface with name `device`. The interface must exist,
// or otherwise the function will error with a netlink error. The network interface must also be a
// VLAN interface or otherwise the function will return an error of type `ErrNotAVlanDevice`.
func DeleteVLANDevice(device string) error {
	// get the device
	l, err := netlink.LinkByName(device)
	if err != nil {
		return err
	}
	if l.Type() != "vlan" {
		return notAVlanDeviceError(device)
	}
	if err := netlink.LinkDel(l); err != nil {
		return err
	}
	return nil
}
