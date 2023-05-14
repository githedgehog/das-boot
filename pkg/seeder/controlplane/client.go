package controlplane

import (
	"context"
	"errors"
	"fmt"
	"os"

	seedernet "go.githedgehog.com/dasboot/pkg/net"
	"go.githedgehog.com/dasboot/pkg/seeder/config"
	fabricv1alpha1 "go.githedgehog.com/wiring/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	GetSwitchPorts(ctx context.Context, switchName string) (*fabricv1alpha1.SwitchPortList, error)
	GetInterfacesForNeighbours(ctx context.Context) (map[string]string, map[string]string, error)
	GetNeighbourSwitchByAddr(ctx context.Context, addr string) (*fabricv1alpha1.Switch, *fabricv1alpha1.SwitchPort, error)
	GetSwitchByLocationUUID(ctx context.Context, uuid string) (*fabricv1alpha1.Switch, error)
}

const (
	RackLabelKey     = "fabric.githedgehog.com/rack"
	ServerLabelKey   = "fabric.githedgehog.com/server"
	SwitchLabelKey   = "fabric.githedgehog.com/switch"
	LocationLabelKey = "fabric.githedgehog.com/location"
)

var (
	ErrNotFound              = errors.New("not found")
	ErrUnsupportedDeviceType = errors.New("unsupported device type")
	ErrNotUnique             = errors.New("not unique")
)

type KubernetesControlPlaneClient struct {
	client          client.WithWatch
	deviceType      config.DeviceType
	deviceHostname  string
	deviceNamespace string
	deviceRack      string
}

var _ Client = &KubernetesControlPlaneClient{}

func NewKubernetesControlPlaneClient(ctx context.Context, client client.WithWatch, selfHostname string, deviceType config.DeviceType) (*KubernetesControlPlaneClient, error) {
	// if the hostname is empty we'll use the OS hostname
	deviceHostname := selfHostname
	if deviceHostname == "" {
		var err error
		deviceHostname, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	cpc := &KubernetesControlPlaneClient{
		client:          client,
		deviceType:      deviceType,
		deviceHostname:  deviceHostname,
		deviceNamespace: "default",
	}

	// if the device type is auto, we need to detect it
	// we're relying in this package that we know it
	switch deviceType {
	case config.DeviceTypeAuto:
		cpc.deviceType = config.DeviceTypeServer
		if _, _, err := cpc.getInterfacesForServerNeighbours(ctx); err != nil {
			cpc.deviceType = config.DeviceTypeSwitch
			if _, _, err := cpc.getInterfacesForSwitchNeighbours(ctx); err != nil {
				return nil, fmt.Errorf("unable to automatically determine device type")
			}
		}
	case config.DeviceTypeServer:
	case config.DeviceTypeSwitch:
	default:
		return nil, ErrUnsupportedDeviceType
	}

	return cpc, nil
}

func (c *KubernetesControlPlaneClient) GetInterfacesForNeighbours(ctx context.Context) (map[string]string, map[string]string, error) {
	switch c.deviceType { //nolint: exhaustive
	case config.DeviceTypeServer:
		return c.getInterfacesForServerNeighbours(ctx)
	case config.DeviceTypeSwitch:
		return c.getInterfacesForSwitchNeighbours(ctx)
	default:
		return nil, nil, ErrUnsupportedDeviceType
	}
}

func (c *KubernetesControlPlaneClient) getInterfacesForServerNeighbours(ctx context.Context) (map[string]string, map[string]string, error) {
	obj := &fabricv1alpha1.Server{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
		return nil, nil, err
	}
	labels := client.MatchingLabels{ServerLabelKey: c.deviceHostname}
	if rack, ok := obj.GetLabels()[RackLabelKey]; ok {
		labels[RackLabelKey] = rack
		c.deviceRack = rack
	}

	// retrieve all of our ports that belong to us
	portList := &fabricv1alpha1.ServerPortList{}
	if err := c.client.List(ctx, portList, labels); err != nil {
		return nil, nil, err
	}
	if len(portList.Items) == 0 {
		return nil, nil, fmt.Errorf("no ports configured for server")
	}
	ret1 := make(map[string]string, len(portList.Items))
	ret2 := make(map[string]string, len(portList.Items))
	for _, port := range portList.Items {
		if port.Spec.Unbundled != nil && port.Spec.Unbundled.Neighbor.Switch != nil {
			// we expect a switch on this port as a neighbour, so we want to listen on this port
			nic := port.Spec.Unbundled.NicName
			addrs, err := seedernet.GetInterfaceAddresses(nic)
			if err != nil {
				return nil, nil, err
			}
			for _, addr := range addrs {
				if addr.Is6() && addr.IsLinkLocalUnicast() {
					ret1[nic] = addr.String()
					ret2[addr.String()] = addr.String()
				}
			}
		}
	}
	return ret1, ret2, nil
}

func (c *KubernetesControlPlaneClient) getInterfacesForSwitchNeighbours(ctx context.Context) (map[string]string, map[string]string, error) {
	obj := &fabricv1alpha1.Switch{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
		return nil, nil, err
	}
	labels := client.MatchingLabels{SwitchLabelKey: c.deviceHostname}
	if rack, ok := obj.GetLabels()[RackLabelKey]; ok {
		labels[RackLabelKey] = rack
		c.deviceRack = rack
	}

	// retrieve all of our ports that belong to us
	portList := &fabricv1alpha1.SwitchPortList{}
	if err := c.client.List(ctx, portList, labels); err != nil {
		return nil, nil, err
	}
	if len(portList.Items) == 0 {
		return nil, nil, fmt.Errorf("no ports configured for server")
	}
	ret1 := make(map[string]string, len(portList.Items))
	ret2 := make(map[string]string, len(portList.Items))
	for _, port := range portList.Items {
		if port.Spec.Neighbor.Switch != nil {
			// we expect a switch on this port as a neighbour, so we want to listen on this port
			nic := port.Spec.NOSPortName
			addrs, err := seedernet.GetInterfaceAddresses(nic)
			if err != nil {
				return nil, nil, err
			}
			for _, addr := range addrs {
				if addr.Is6() && addr.IsLinkLocalUnicast() {
					ret1[nic] = addr.String()
					ret2[addr.String()] = addr.String()
				}
			}
		}
	}
	return ret1, ret2, nil
}

// GetNeighbourSwitchByAddr finds the switch that is connected to this device by its link local IP address `addr`.
func (c *KubernetesControlPlaneClient) GetNeighbourSwitchByAddr(ctx context.Context, addr string) (*fabricv1alpha1.Switch, *fabricv1alpha1.SwitchPort, error) {
	switch c.deviceType { //nolint: exhaustive
	case config.DeviceTypeServer:
		return c.getNeighbourSwitchByAddrForServer(ctx, addr)
	case config.DeviceTypeSwitch:
		return c.getNeighbourSwitchByAddrForSwitch(ctx, addr)
	default:
		return nil, nil, ErrUnsupportedDeviceType
	}
}

func (c *KubernetesControlPlaneClient) getNeighbourSwitchByAddrForServer(ctx context.Context, addr string) (*fabricv1alpha1.Switch, *fabricv1alpha1.SwitchPort, error) {
	// find ourselves first
	obj := &fabricv1alpha1.Server{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
		return nil, nil, err
	}
	labels := client.MatchingLabels{ServerLabelKey: c.deviceHostname}
	if rack, ok := obj.GetLabels()[RackLabelKey]; ok {
		labels[RackLabelKey] = rack
		c.deviceRack = rack
	}

	// then retrieve all of our ports that belong to us
	portList := &fabricv1alpha1.ServerPortList{}
	if err := c.client.List(ctx, portList, labels); err != nil {
		return nil, nil, err
	}
	if len(portList.Items) == 0 {
		return nil, nil, fmt.Errorf("no ports configured for server")
	}
	for _, port := range portList.Items {
		// we are only interested in "unbundled" ports and where the neighbor is a switch
		if port.Spec.Unbundled != nil && port.Spec.Unbundled.Neighbor.Switch != nil {
			// get all addresses that belong to this port
			addrs, err := seedernet.GetInterfaceAddresses(port.Spec.Unbundled.NicName)
			if err != nil {
				return nil, nil, err
			}
			// iterate over them and find the match
			for _, a := range addrs {
				if a.Is6() && a.IsLinkLocalUnicast() && a.String() == addr {
					// we found our match
					// now retrieve the switch item
					ret1 := &fabricv1alpha1.Switch{}
					if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: port.Spec.Unbundled.Neighbor.Switch.Name}, ret1); err != nil {
						return nil, nil, err
					}
					ret2 := &fabricv1alpha1.SwitchPort{}
					if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: port.Spec.Unbundled.Neighbor.Switch.Port}, ret2); err != nil {
						return ret1, nil, err
					}
					return ret1, ret2, nil
				}
			}
		}
	}
	return nil, nil, ErrNotFound
}

func (c *KubernetesControlPlaneClient) getNeighbourSwitchByAddrForSwitch(ctx context.Context, addr string) (*fabricv1alpha1.Switch, *fabricv1alpha1.SwitchPort, error) {
	// TODO
	return nil, nil, fmt.Errorf("TODO")
}

// GetSwitchPorts retrieves all switch ports for a given switch
func (c *KubernetesControlPlaneClient) GetSwitchPorts(ctx context.Context, switchName string) (*fabricv1alpha1.SwitchPortList, error) {
	// build filter with labels, this is how we expect the data in Kubernetes
	labels := client.MatchingLabels{SwitchLabelKey: switchName}
	if c.deviceRack != "" {
		labels[RackLabelKey] = c.deviceRack
	}

	// now simply retrieve them
	portList := &fabricv1alpha1.SwitchPortList{}
	if err := c.client.List(ctx, portList, labels); err != nil {
		return nil, err
	}

	return portList, nil
}

func (c *KubernetesControlPlaneClient) GetSwitchByLocationUUID(ctx context.Context, uuid string) (*fabricv1alpha1.Switch, error) {
	// build filter with labels, this is how we expect the data in Kubernetes
	labels := client.MatchingLabels{LocationLabelKey: uuid}
	if c.deviceRack != "" {
		labels[RackLabelKey] = c.deviceRack
	}

	switchList := &fabricv1alpha1.SwitchList{}
	if err := c.client.List(ctx, switchList, labels); err != nil {
		return nil, err
	}

	num := len(switchList.Items)
	switch num {
	case 0:
		return nil, ErrNotFound
	case 1:
		return &switchList.Items[0], nil
	default:
		return nil, fmt.Errorf("%w: %d items found", ErrNotUnique, num)
	}
}
