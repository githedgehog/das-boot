package net

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/vishvananda/netlink"
)

var ErrNotAVlanDevice = errors.New("net: not a vlan device")

func notAVlanDeviceError(str string) error {
	return fmt.Errorf("%w: %s", ErrNotAVlanDevice, str)
}

// StringsToIPNets is a convenience function to convert between the two formats
func StringsToIPNets(ipaddrs []string) ([]*net.IPNet, error) {
	var ipnets []*net.IPNet
	for _, ipaddrstr := range ipaddrs {
		ipaddr, ipnet, err := net.ParseCIDR(ipaddrstr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IP address and netmask: %w", err)
		}
		ipnet.IP = ipaddr
		ipnets = append(ipnets, ipnet)
	}
	return ipnets, nil
}

type Route struct {
	Dests []*net.IPNet
	Gw    net.IP
}

// AddVLANDeviceWithIP will create a new VLAN network interface called `vlanName` with VLAN ID `vid` and add it to
// the parent network interface `device`. It will also add all IP addresses as given with `ipaddrnets`, and, last
// but not least, it will set the interface UP.
func AddVLANDeviceWithIP(device string, vid uint16, vlanName string, ipaddrnets []*net.IPNet, routes []*Route) error {
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

	// add subnets to be routed over same interface
	// network needs to be up for this, so must come after we bring up the link
	if len(routes) > 0 {
		for _, route := range routes {
			for _, dest := range route.Dests {
				r := &netlink.Route{
					Dst:       dest,
					Gw:        route.Gw,
					LinkIndex: vlan.Index,
				}
				if err := netlink.RouteAdd(r); err != nil {
					return err
				}
			}
		}
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

// GetInterfaces will return a list of interface names for all network interfaces which are "real devices".
// Being a "real device" means that its netlink type is a "device" and its encapsulation type is "ether".
func GetInterfaces() ([]string, error) {
	ll, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, link := range ll {
		la := link.Attrs()
		if link.Type() == "device" && la.EncapType == "ether" {
			ret = append(ret, la.Name)
		}
	}
	return ret, nil
}

// GetInterfaceAddresses returns all IP addresses for an interface.
func GetInterfaceAddresses(device string) ([]netip.Addr, error) {
	link, err := netlink.LinkByName(device)
	if err != nil {
		return nil, err
	}

	addrs, err := netlink.AddrList(link, 0)
	if err != nil {
		return nil, err
	}
	ret := make([]netip.Addr, 0, len(addrs))
	for _, addr := range addrs {
		if ip, ok := netip.AddrFromSlice(addr.IP); ok {
			ret = append(ret, ip)
		}
	}
	return ret, nil
}
