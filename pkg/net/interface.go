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
	Flags int
}

// AddVLANDeviceWithIP will create a new VLAN network interface called `vlanName` with VLAN ID `vid` and add it to
// the parent network interface `device`. It will also add all IP addresses as given with `ipaddrnets`, add the additional
// routes in `routes`, and, last but not least, it will set the interface UP.
func AddVLANDeviceWithIP(device string, vid uint16, vlanName string, ipaddrnets []*net.IPNet, routes []*Route) error {
	// This is kind of desperate, but the easiest way to ensure that it's really not configured before we configure it
	// It has the advantage though that it will also work in cases when our installer crashed before it could reset the network
	DeleteVLANDevice(device, ipaddrnets, routes) //nolint: errcheck

	// get the parent device
	pl, err := netlink.LinkByName(device)
	if err != nil {
		return fmt.Errorf("netlink: link by name: %w", err)
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
		return fmt.Errorf("netlink: link add: %w", err)
	}

	// now add the IP address
	for _, ipaddrnet := range ipaddrnets {
		addr := &netlink.Addr{
			IPNet: ipaddrnet,
		}
		if err := netlink.AddrAdd(vlan, addr); err != nil {
			return fmt.Errorf("netlink: addr add '%s': %w", addr, err)
		}
	}

	// set the interface up
	if err := netlink.LinkSetUp(vlan); err != nil {
		return fmt.Errorf("netlink: link set up: %w", err)
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
					Flags:     route.Flags,
				}
				if err := netlink.RouteAdd(r); err != nil {
					return fmt.Errorf("netlink: route add '%s': %w", r, err)
				}
			}
		}
	}

	// that's it - that was easy
	return nil
}

// ConfigureDeviceWithIP will add all IP addresses as given with `ipaddrnets`, add the additional
// routes in `routes`, and, last but not least, it will ensure the interface is UP.
func ConfigureDeviceWithIP(device string, ipaddrnets []*net.IPNet, routes []*Route) error {
	// This is kind of desperate, but the easiest way to ensure that it's really not configured before we configure it
	// It has the advantage though that it will also work in cases when our installer crashed before it could reset the network
	UnconfigureDeviceWithIP(device, ipaddrnets, routes) //nolint: errcheck

	// get the device
	link, err := netlink.LinkByName(device)
	if err != nil {
		return fmt.Errorf("netlink: link by name: %w", err)
	}

	// now add the IP address
	for _, ipaddrnet := range ipaddrnets {
		addr := &netlink.Addr{
			IPNet: ipaddrnet,
		}
		if err := netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("netlink: addr add '%s': %w", addr, err)
		}
	}

	// ensure the interface is up
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("netlink: link set up: %w", err)
	}

	// add subnets to be routed over same interface
	// network needs to be up for this, so must come after we bring up the link
	if len(routes) > 0 {
		for _, route := range routes {
			for _, dest := range route.Dests {
				r := &netlink.Route{
					Dst:       dest,
					Gw:        route.Gw,
					LinkIndex: link.Attrs().Index,
					Flags:     route.Flags,
				}
				if err := netlink.RouteAdd(r); err != nil {
					return fmt.Errorf("netlink: route add '%s': %w", r, err)
				}
			}
		}
	}

	// that's it - that was easy
	return nil
}

// UnconfigureDeviceWithIP will remove all IP addresses as given with `ipaddrnets`, and remove the additional
// routes in `routes`.
func UnconfigureDeviceWithIP(device string, ipaddrnets []*net.IPNet, routes []*Route) error {
	var errs []error

	// get the device
	link, err := netlink.LinkByName(device)
	if err != nil {
		return fmt.Errorf("netlink: link by name: %w", err)
	}

	// remove routes
	if len(routes) > 0 {
		for _, route := range routes {
			for _, dest := range route.Dests {
				r := &netlink.Route{
					Dst:       dest,
					Gw:        route.Gw,
					LinkIndex: link.Attrs().Index,
					Flags:     route.Flags,
				}
				if err := netlink.RouteDel(r); err != nil {
					errs = append(errs, fmt.Errorf("netlink: route del '%s': %w", r, err))
				}
			}
		}
	}

	// now remove the IP addresses
	for _, ipaddrnet := range ipaddrnets {
		addr := &netlink.Addr{
			IPNet: ipaddrnet,
		}
		if err := netlink.AddrDel(link, addr); err != nil {
			errs = append(errs, fmt.Errorf("netlink: addr del '%s': %w", addr, err))
		}
	}

	// that's it - that was easy
	if len(errs) > 0 {
		var reterr error
		for _, err := range errs {
			if reterr == nil {
				reterr = err
			} else {
				reterr = fmt.Errorf("%w, %w", reterr, err)
			}
		}
		return reterr
	}
	return nil
}

// DeleteVLANDevice will delete the network interface with name `device`. The interface must exist,
// or otherwise the function will error with a netlink error. The network interface must also be a
// VLAN interface or otherwise the function will return an error of type `ErrNotAVlanDevice`.
// Before it does that it will delete all associated routes though as well as IP addresses.
func DeleteVLANDevice(device string, ipaddrnets []*net.IPNet, routes []*Route) error {
	// get the device
	l, err := netlink.LinkByName(device)
	if err != nil {
		return fmt.Errorf("netlink: link by name: %w", err)
	}
	if l.Type() != "vlan" {
		return notAVlanDeviceError(device)
	}

	var errs []error
	// remove routes
	if len(routes) > 0 {
		for _, route := range routes {
			for _, dest := range route.Dests {
				r := &netlink.Route{
					Dst:       dest,
					Gw:        route.Gw,
					LinkIndex: l.Attrs().Index,
					Flags:     route.Flags,
				}
				if err := netlink.RouteDel(r); err != nil {
					errs = append(errs, fmt.Errorf("netlink: route del '%s': %w", r, err))
				}
			}
		}
	}

	// now remove the IP addresses
	for _, ipaddrnet := range ipaddrnets {
		addr := &netlink.Addr{
			IPNet: ipaddrnet,
		}
		if err := netlink.AddrDel(l, addr); err != nil {
			errs = append(errs, fmt.Errorf("netlink: addr del '%s': %w", addr, err))
		}
	}

	// last but not least, delete the device
	if err := netlink.LinkDel(l); err != nil {
		errs = append(errs, fmt.Errorf("netlink: link del: %w", err))
	}

	if len(errs) > 0 {
		var reterr error
		for _, err := range errs {
			if reterr == nil {
				reterr = err
			} else {
				reterr = fmt.Errorf("%w, %w", reterr, err)
			}
		}
		return reterr
	}
	return nil
}

// GetInterfaces will return a list of interface names for all network interfaces which are "real devices".
// Being a "real device" means that its netlink type is a "device" and its encapsulation type is "ether".
func GetInterfaces() ([]string, error) {
	ll, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("netlink: link list: %w", err)
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
		return nil, fmt.Errorf("netlink: link by name: %w", err)
	}

	addrs, err := netlink.AddrList(link, 0)
	if err != nil {
		return nil, fmt.Errorf("netlink: addr list for '%s': %w", device, err)
	}
	ret := make([]netip.Addr, 0, len(addrs))
	for _, addr := range addrs {
		if ip, ok := netip.AddrFromSlice(addr.IP); ok {
			ret = append(ret, ip)
		}
	}
	return ret, nil
}
