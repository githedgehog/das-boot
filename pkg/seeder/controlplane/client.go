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

package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	dasbootv1alpha1 "go.githedgehog.com/dasboot/pkg/k8s/api/v1alpha1"
	"go.githedgehog.com/dasboot/pkg/log"
	seedernet "go.githedgehog.com/dasboot/pkg/net"
	"go.githedgehog.com/dasboot/pkg/seeder/config"
	agentv1alpha2 "go.githedgehog.com/fabric/api/agent/v1alpha2"
	wiring1alpha2 "go.githedgehog.com/fabric/api/wiring/v1alpha2"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//go:generate mockgen -destination ../../../test/mock/seeder/mockcontrolplane/client.go -package mockcontrolplane go.githedgehog.com/dasboot/pkg/seeder/controlplane Client

type Client interface {
	DeviceHostname() string
	DeviceNamespace() string
	GetInterfacesForNeighbours(ctx context.Context) (map[string]string, map[string]string, error)
	GetSwitchConnections(ctx context.Context, switchName string) ([]wiring1alpha2.Connection, error)
	GetSwitchByAddr(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.Connection, error)
	GetNeighbourSwitchByAddr(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.Connection, error)
	GetSwitchByLocationUUID(ctx context.Context, uuid string) (*wiring1alpha2.Switch, error)
	GetDeviceRegistration(ctx context.Context, deviceID string) (*dasbootv1alpha1.DeviceRegistration, error)
	CreateDeviceRegistration(ctx context.Context, reg *dasbootv1alpha1.DeviceRegistration) (*dasbootv1alpha1.DeviceRegistration, error)
	GetSwitchByDeviceID(ctx context.Context, deviceID string) (*wiring1alpha2.Switch, error)
	GetAgentConfig(ctx context.Context, deviceID string) ([]byte, error)
	GetAgentKubeconfig(ctx context.Context, deviceID string) ([]byte, error)
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
	obj := &wiring1alpha2.Server{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
		return nil, nil, err
	}

	// retrieve all of our connections that belong to us
	connList := &wiring1alpha2.ConnectionList{}
	if err := c.client.List(ctx, connList, wiring1alpha2.MatchingLabelsForListLabelServer(c.deviceHostname)); err != nil {
		return nil, nil, err
	}
	if len(connList.Items) == 0 {
		return nil, nil, fmt.Errorf("no connections configured for server '%s'", c.deviceHostname)
	}
	ret1 := make(map[string]string, len(connList.Items))
	ret2 := make(map[string]string, len(connList.Items))
	for _, conn := range connList.Items {
		if conn.Spec.Management != nil {
			// we expect a switch on this port as a neighbour, so we want to listen on this port

			portName := conn.Spec.Management.Link.Server.LocalPortName()
			portMAC := conn.Spec.Management.Link.Server.MAC
			nic, err := seedernet.GetInterface(portName, portMAC)
			if err != nil {
				log.L().Warn("Getting interface failed, skipping", zap.String("nic", portName), zap.String("mac", portMAC), zap.Error(err))
				continue
			}

			addrs, err := seedernet.GetInterfaceAddresses(nic)
			if err != nil {
				return nil, nil, err
			}
			for _, addr := range addrs {
				if addr.Is6() && addr.IsLinkLocalUnicast() {
					ret1[nic] = addr.String()
					ret2[addr.String()] = nic
				}
			}
		}
	}
	return ret1, ret2, nil
}

// TODO: this actually needs rework as this is meant for a switch-switch neighbour case, but the connection type does not exist yet
func (c *KubernetesControlPlaneClient) getInterfacesForSwitchNeighbours(ctx context.Context) (map[string]string, map[string]string, error) {
	obj := &wiring1alpha2.Switch{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
		return nil, nil, err
	}

	// retrieve all of our ports that belong to us
	connList := &wiring1alpha2.ConnectionList{}
	if err := c.client.List(ctx, connList, wiring1alpha2.MatchingLabelsForListLabelSwitch(c.deviceHostname)); err != nil {
		return nil, nil, err
	}
	if len(connList.Items) == 0 {
		return nil, nil, fmt.Errorf("no connections configured for switch '%s'", c.deviceHostname)
	}
	ret1 := make(map[string]string, len(connList.Items))
	ret2 := make(map[string]string, len(connList.Items))
	for _, conn := range connList.Items {
		if conn.Spec.Management != nil {
			// we expect a switch on this port as a neighbour, so we want to listen on this port

			portName := conn.Spec.Management.Link.Server.LocalPortName()
			portMAC := conn.Spec.Management.Link.Server.MAC
			nic, err := seedernet.GetInterface(portName, portMAC)
			if err != nil {
				log.L().Warn("Getting interface failed, skipping", zap.String("nic", portName), zap.String("mac", portMAC), zap.Error(err))
				continue
			}

			addrs, err := seedernet.GetInterfaceAddresses(nic)
			if err != nil {
				return nil, nil, err
			}
			for _, addr := range addrs {
				if addr.Is6() && addr.IsLinkLocalUnicast() {
					ret1[nic] = addr.String()
					ret2[addr.String()] = nic
				}
			}
		}
	}
	return ret1, ret2, nil
}

func (c *KubernetesControlPlaneClient) GetSwitchByAddr(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.Connection, error) {
	// get all connections
	// TODO: yes, it would be great if we could filter here
	connList := &wiring1alpha2.ConnectionList{}
	if err := c.client.List(ctx, connList); err != nil {
		return nil, nil, err
	}

	// find the connection by the passed address
	// existing Conn type=Management to define leaf-front-panel <> control-node connection, we don’t need any extra data
	// existing Conn type=Fabric to use spine <> leaf connections
	for _, conn := range connList.Items {
		var deviceName string
		if conn.Spec.Management != nil {
			if conn.Spec.Management.Link.Switch.IP == addr {
				deviceName = conn.Spec.Management.Link.Switch.DeviceName()
			}
			ip, _, err := net.ParseCIDR(conn.Spec.Management.Link.Switch.IP)
			if err == nil {
				if ip.String() == addr {
					deviceName = conn.Spec.Management.Link.Switch.DeviceName()
				}
			}
		} else if conn.Spec.Fabric != nil {
			for _, link := range conn.Spec.Fabric.Links {
				if link.Leaf.IP == addr {
					deviceName = link.Leaf.DeviceName()
				}
				ip, _, err := net.ParseCIDR(link.Leaf.IP)
				if err == nil {
					if ip.String() == addr {
						deviceName = link.Leaf.DeviceName()
					}
				}
				if link.Spine.IP == addr {
					deviceName = link.Spine.DeviceName()
				}
				ip, _, err = net.ParseCIDR(link.Spine.IP)
				if err == nil {
					if ip.String() == addr {
						deviceName = link.Spine.DeviceName()
					}
				}
			}
		}

		if deviceName != "" {
			// we found our match, now retrieve the switch item
			ret1 := &wiring1alpha2.Switch{}
			if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: deviceName}, ret1); err != nil {
				return nil, nil, err
			}
			ret2 := conn.DeepCopy()
			return ret1, ret2, nil
		}
	}
	return nil, nil, ErrNotFound
}

// GetNeighbourSwitchByAddr finds the switch that is connected to this device by its link local IP address `addr`.
func (c *KubernetesControlPlaneClient) GetNeighbourSwitchByAddr(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.Connection, error) {
	switch c.deviceType { //nolint: exhaustive
	case config.DeviceTypeServer:
		return c.getNeighbourSwitchByAddrForServer(ctx, addr)
	case config.DeviceTypeSwitch:
		return c.getNeighbourSwitchByAddrForSwitch(ctx, addr)
	default:
		return nil, nil, ErrUnsupportedDeviceType
	}
}

func (c *KubernetesControlPlaneClient) getNeighbourSwitchByAddrForServer(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.Connection, error) {
	// find ourselves first
	obj := &wiring1alpha2.Server{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: c.deviceHostname}, obj); err != nil {
		return nil, nil, err
	}

	// retrieve all of our connections that belong to us
	connList := &wiring1alpha2.ConnectionList{}
	if err := c.client.List(ctx, connList, wiring1alpha2.MatchingLabelsForListLabelServer(c.deviceHostname)); err != nil {
		return nil, nil, err
	}
	if len(connList.Items) == 0 {
		return nil, nil, fmt.Errorf("no connections configured for server '%s'", c.deviceHostname)
	}
	for _, conn := range connList.Items {
		// we are only interested in management connections at the moment
		if conn.Spec.Management != nil {
			// get all addresses that belong to this port

			portName := conn.Spec.Management.Link.Server.LocalPortName()
			portMAC := conn.Spec.Management.Link.Server.MAC
			nic, err := seedernet.GetInterface(portName, portMAC)
			if err != nil {
				log.L().Warn("Getting interface failed, skipping", zap.String("nic", portName), zap.String("mac", portMAC), zap.Error(err))
				continue
			}

			addrs, err := seedernet.GetInterfaceAddresses(nic)
			if err != nil {
				return nil, nil, err
			}
			// iterate over them and find the match
			for _, a := range addrs {
				if a.Is6() && a.IsLinkLocalUnicast() && a.String() == addr {
					// we found our match
					// now retrieve the switch item
					ret1 := &wiring1alpha2.Switch{}
					if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.deviceNamespace, Name: conn.Spec.Management.Link.Switch.DeviceName()}, ret1); err != nil {
						return nil, nil, err
					}
					// this is simply the connection we are on
					ret2 := conn.DeepCopy()
					return ret1, ret2, nil
				}
			}
		}
	}
	return nil, nil, ErrNotFound
}

func (c *KubernetesControlPlaneClient) getNeighbourSwitchByAddrForSwitch(ctx context.Context, addr string) (*wiring1alpha2.Switch, *wiring1alpha2.Connection, error) {
	// TODO
	return nil, nil, fmt.Errorf("TODO")
}

func (c *KubernetesControlPlaneClient) GetSwitchConnections(ctx context.Context, switchName string) ([]wiring1alpha2.Connection, error) {
	connList := &wiring1alpha2.ConnectionList{}
	if err := c.client.List(ctx, connList, wiring1alpha2.MatchingLabelsForListLabelSwitch(switchName)); err != nil {
		return nil, err
	}

	// and filter the list further down by connection types
	ret := make([]wiring1alpha2.Connection, 0, len(connList.Items))
	for _, conn := range connList.Items {
		if conn.Spec.Management != nil {
			ret = append(ret, *conn.DeepCopy())
		}
	}

	return ret, nil
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

func (c *KubernetesControlPlaneClient) GetAgentConfig(ctx context.Context, deviceID string) ([]byte, error) {
	// we will get the switch by device ID
	switchObj, err := c.GetSwitchByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("switch by deviceID: %w", err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(agentv1alpha2.GroupVersion.WithKind("Agent"))
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: switchObj.Namespace, Name: switchObj.Name}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("agent: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("agent: %w", err)
	}

	objBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("yaml encoding: %w", err)
	}

	return objBytes, nil
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
