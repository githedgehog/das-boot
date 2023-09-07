package controlplane

import (
	"context"
	"errors"
	"fmt"
	"os"

	dasbootv1alpha1 "go.githedgehog.com/dasboot/pkg/k8s/api/v1alpha1"
	seedernet "go.githedgehog.com/dasboot/pkg/net"
	"go.githedgehog.com/dasboot/pkg/seeder/config"
	agentv1alpha2 "go.githedgehog.com/fabric/api/agent/v1alpha2"
	wiring1alpha2 "go.githedgehog.com/fabric/api/wiring/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	DeviceHostname() string
	DeviceNamespace() string

	GetSwitchPorts(ctx context.Context, switchName string) (*wiring1alpha2.SwitchPortList, error)
	GetNeighbourSwitchByAddr(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.SwitchPort, error)

	GetSwitchByLocationUUID(ctx context.Context, uuid string) (*wiring1alpha2.Switch, error)
	GetDeviceRegistration(ctx context.Context, deviceID string) (*dasbootv1alpha1.DeviceRegistration, error)
	CreateDeviceRegistration(ctx context.Context, reg *dasbootv1alpha1.DeviceRegistration) (*dasbootv1alpha1.DeviceRegistration, error)
	GetSwitchByDeviceID(ctx context.Context, deviceID string) (*wiring1alpha2.Switch, error)
	GetAgentConfig(ctx context.Context, deviceID string) (*agentv1alpha2.Agent, error)
	GetAgentKubeconfig(ctx context.Context, deviceID string) ([]byte, error)
	// GetInterfacesForNeighbours(ctx context.Context) (map[string]string, map[string]string, error)
}

const (
	RackLabelKey             = "fabric.githedgehog.com/rack"
	ServerLabelKey           = "fabric.githedgehog.com/server"
	SwitchLabelKey           = "fabric.githedgehog.com/switch"
	LocationLabelKey         = "fabric.githedgehog.com/location"
	KubeconfigAgentSecretKey = "kubeconfig"
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

func (c *KubernetesControlPlaneClient) DeviceHostname() string {
	return c.deviceHostname
}

func (c *KubernetesControlPlaneClient) DeviceNamespace() string {
	return c.deviceNamespace
}

// func (c *KubernetesControlPlaneClient) GetInterfacesForNeighbours(ctx context.Context) (map[string]string, map[string]string, error) {
// 	switch c.deviceType { //nolint: exhaustive
// 	case config.DeviceTypeServer:
// 		return c.getInterfacesForServerNeighbours(ctx)
// 	case config.DeviceTypeSwitch:
// 		return c.getInterfacesForSwitchNeighbours(ctx)
// 	default:
// 		return nil, nil, ErrUnsupportedDeviceType
// 	}
// }

// func (c *KubernetesControlPlaneClient) getInterfacesForServerNeighbours(ctx context.Context) (map[string]string, map[string]string, error) {
// 	obj := &wiring1alpha2.Server{}
// 	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
// 		return nil, nil, err
// 	}
// 	labels := client.MatchingLabels{ServerLabelKey: c.deviceHostname}
// 	if rack, ok := obj.GetLabels()[RackLabelKey]; ok {
// 		labels[RackLabelKey] = rack
// 		c.deviceRack = rack
// 	}

// 	// retrieve all of our ports that belong to us
// 	portList := &wiring1alpha2.ServerPortList{}
// 	if err := c.client.List(ctx, portList, labels); err != nil {
// 		return nil, nil, err
// 	}
// 	if len(portList.Items) == 0 {
// 		return nil, nil, fmt.Errorf("no ports configured for server")
// 	}
// 	ret1 := make(map[string]string, len(portList.Items))
// 	ret2 := make(map[string]string, len(portList.Items))
// 	for _, port := range portList.Items {
// 		if port.Spec.Unbundled != nil && port.Spec.Unbundled.Neighbor.Switch != nil {
// 			// we expect a switch on this port as a neighbour, so we want to listen on this port
// 			nic := port.Spec.Unbundled.NicName
// 			addrs, err := seedernet.GetInterfaceAddresses(nic)
// 			if err != nil {
// 				return nil, nil, err
// 			}
// 			for _, addr := range addrs {
// 				if addr.Is6() && addr.IsLinkLocalUnicast() {
// 					ret1[nic] = addr.String()
// 					ret2[addr.String()] = addr.String()
// 				}
// 			}
// 		}
// 	}
// 	return ret1, ret2, nil
// }

// func (c *KubernetesControlPlaneClient) getInterfacesForSwitchNeighbours(ctx context.Context) (map[string]string, map[string]string, error) {
// 	obj := &wiring1alpha2.Switch{}
// 	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
// 		return nil, nil, err
// 	}
// 	labels := client.MatchingLabels{SwitchLabelKey: c.deviceHostname}
// 	if rack, ok := obj.GetLabels()[RackLabelKey]; ok {
// 		labels[RackLabelKey] = rack
// 		c.deviceRack = rack
// 	}

// 	// retrieve all of our ports that belong to us
// 	portList := &wiring1alpha2.SwitchPortList{}
// 	if err := c.client.List(ctx, portList, labels); err != nil {
// 		return nil, nil, err
// 	}
// 	if len(portList.Items) == 0 {
// 		return nil, nil, fmt.Errorf("no ports configured for server")
// 	}
// 	ret1 := make(map[string]string, len(portList.Items))
// 	ret2 := make(map[string]string, len(portList.Items))
// 	for _, port := range portList.Items {
// 		if port.Spec.Neighbor.Switch != nil {
// 			// we expect a switch on this port as a neighbour, so we want to listen on this port
// 			nic := port.Spec.NOSPortName
// 			addrs, err := seedernet.GetInterfaceAddresses(nic)
// 			if err != nil {
// 				return nil, nil, err
// 			}
// 			for _, addr := range addrs {
// 				if addr.Is6() && addr.IsLinkLocalUnicast() {
// 					ret1[nic] = addr.String()
// 					ret2[addr.String()] = addr.String()
// 				}
// 			}
// 		}
// 	}
// 	return ret1, ret2, nil
// }

// GetNeighbourSwitchByAddr finds the switch that is connected to this device by its link local IP address `addr`.
func (c *KubernetesControlPlaneClient) GetNeighbourSwitchByAddr(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.SwitchPort, error) {
	switch c.deviceType { //nolint: exhaustive
	case config.DeviceTypeServer:
		return c.getNeighbourSwitchByAddrForServer(ctx, addr)
	case config.DeviceTypeSwitch:
		return c.getNeighbourSwitchByAddrForSwitch(ctx, addr)
	default:
		return nil, nil, ErrUnsupportedDeviceType
	}
}

func (c *KubernetesControlPlaneClient) getNeighbourSwitchByAddrForServer(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.SwitchPort, error) {
	// find ourselves first
	obj := &wiring1alpha2.Server{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
		return nil, nil, err
	}
	labels := client.MatchingLabels{ServerLabelKey: c.deviceHostname}
	if rack, ok := obj.GetLabels()[RackLabelKey]; ok {
		labels[RackLabelKey] = rack
		c.deviceRack = rack
	}

	// then retrieve all of our ports that belong to us
	portList := &wiring1alpha2.ServerPortList{}
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
					ret1 := &wiring1alpha2.Switch{}
					if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: port.Spec.Unbundled.Neighbor.Switch.Name}, ret1); err != nil {
						return nil, nil, err
					}
					ret2 := &wiring1alpha2.SwitchPort{}
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

func (c *KubernetesControlPlaneClient) getNeighbourSwitchByAddrForSwitch(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.SwitchPort, error) {
	// TODO
	return nil, nil, fmt.Errorf("TODO")
}

// GetSwitchPorts retrieves all switch ports for a given switch
func (c *KubernetesControlPlaneClient) GetSwitchPorts(ctx context.Context, switchName string) (*wiring1alpha2.SwitchPortList, error) {
	// build filter with labels, this is how we expect the data in Kubernetes
	labels := client.MatchingLabels{SwitchLabelKey: switchName}
	if c.deviceRack != "" {
		labels[RackLabelKey] = c.deviceRack
	}

	// now simply retrieve them
	portList := &wiring1alpha2.SwitchPortList{}
	if err := c.client.List(ctx, portList, labels); err != nil {
		return nil, err
	}

	return portList, nil
}

func (c *KubernetesControlPlaneClient) GetSwitchByLocationUUID(ctx context.Context, uuid string) (*wiring1alpha2.Switch, error) {
	// build filter with labels, this is how we expect the data in Kubernetes
	labels := client.MatchingLabels{LocationLabelKey: uuid}
	if c.deviceRack != "" {
		labels[RackLabelKey] = c.deviceRack
	}

	switchList := &wiring1alpha2.SwitchList{}
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

func (c *KubernetesControlPlaneClient) GetDeviceRegistration(ctx context.Context, deviceID string) (*dasbootv1alpha1.DeviceRegistration, error) {
	obj := &dasbootv1alpha1.DeviceRegistration{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: deviceID}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return obj, nil
}

func (c *KubernetesControlPlaneClient) CreateDeviceRegistration(ctx context.Context, reg *dasbootv1alpha1.DeviceRegistration) (*dasbootv1alpha1.DeviceRegistration, error) {
	obj := reg.DeepCopy()
	if err := c.client.Create(ctx, reg); err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *KubernetesControlPlaneClient) GetSwitchByDeviceID(ctx context.Context, deviceID string) (*wiring1alpha2.Switch, error) {
	// the device registration will have the location information for this device
	devReg, err := c.GetDeviceRegistration(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device registration: %w", err)
	}

	// we will get the switch next by UUID
	switchObj, err := c.GetSwitchByLocationUUID(ctx, devReg.Spec.LocationUUID)
	if err != nil {
		return nil, fmt.Errorf("switch by location UUID: %w", err)
	}

	return switchObj, nil
}

func (c *KubernetesControlPlaneClient) GetAgentConfig(ctx context.Context, deviceID string) (*agentv1alpha2.Agent, error) {
	// we will get the switch by device ID
	switchObj, err := c.GetSwitchByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("switch by deviceID: %w", err)
	}

	obj := &agentv1alpha2.Agent{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: switchObj.Namespace, Name: switchObj.Name}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("agent: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("agent: %w", err)
	}

	obj.APIVersion = agentv1alpha2.GroupVersion.Identifier()
	obj.Kind = "Agent"

	return obj, nil
}

func (c *KubernetesControlPlaneClient) GetAgentKubeconfig(ctx context.Context, deviceID string) ([]byte, error) {
	// we will get the switch by device ID
	switchObj, err := c.GetSwitchByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("switch by deviceID: %w", err)
	}

	// retrieve the secret
	obj := &corev1.Secret{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: switchObj.Namespace, Name: switchObj.Name}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("secret: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("secret: %w", err)
	}

	kubeCfg, ok := obj.Data[KubeconfigAgentSecretKey]
	if !ok {
		return nil, fmt.Errorf("secret kubeconfig entry: %w", ErrNotFound)
	}

	return kubeCfg, nil
}
