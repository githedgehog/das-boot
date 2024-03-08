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

package dynll

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/net"
	seedernet "go.githedgehog.com/dasboot/pkg/net"
	"go.githedgehog.com/dasboot/pkg/seeder/config"
	seedererrors "go.githedgehog.com/dasboot/pkg/seeder/errors"
	"go.githedgehog.com/dasboot/pkg/seeder/server"
	"go.githedgehog.com/dasboot/pkg/seeder/server/generic"

	wiring1alpha2 "go.githedgehog.com/fabric/api/wiring/v1alpha2"
)

type DynLLServer struct {
	done        chan struct{}
	err         chan error
	httpServers map[string]*generic.HTTPServer
	k8sClient   client.WithWatch
}

var _ server.ControlInterface = &DynLLServer{}

func NewDynLLServer(ctx context.Context, k8sClient client.WithWatch, cfg *config.DynLL, handler http.Handler) (*DynLLServer, error) {
	ret := &DynLLServer{
		done:        make(chan struct{}),
		err:         make(chan error, 100),
		httpServers: make(map[string]*generic.HTTPServer),
		k8sClient:   k8sClient,
	}

	// if this setting was not set, we simply default to the default HTTP port
	listeningPort := cfg.ListeningPort
	if listeningPort == 0 {
		listeningPort = 80
	}

	// if the device name is empty, we use our host name
	selfHostname := cfg.DeviceName
	if selfHostname == "" {
		var err error
		selfHostname, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	// search Kubernetes for our neighbours
	var err error
	var nics []string
	switch cfg.DeviceType {
	case config.DeviceTypeAuto:
		// we try server first
		nics, err = getInterfacesForServerNeighbours(ctx, k8sClient, selfHostname)
		if err != nil {
			nics, err = getInterfacesForSwitchNeighbours(ctx, k8sClient, selfHostname)
			if err != nil {
				return nil, err
			}
		}
	case config.DeviceTypeServer:
		nics, err = getInterfacesForServerNeighbours(ctx, k8sClient, selfHostname)
		if err != nil {
			return nil, err
		}
	case config.DeviceTypeSwitch:
		nics, err = getInterfacesForSwitchNeighbours(ctx, k8sClient, selfHostname)
		if err != nil {
			return nil, err
		}
	default:
		return nil, seedererrors.InvalidConfigError("invalid device_type setting for dynll configuration")
	}

	// build listen addresses for all the interfaces that we need to listen on
	listenAddresses := make(map[string]string)
	for _, nic := range nics {
		// we expect a switch on this port as a neighbour, so we want to listen on this port
		addrs, err := seedernet.GetInterfaceAddresses(nic)
		if err != nil {
			log.L().Warn("Getting interface addresses failed. Not listening on this port.", zap.String("nic", nic), zap.Error(err))
			continue
		}
		for _, addr := range addrs {
			if addr.Is6() && addr.IsLinkLocalUnicast() {
				// listenAddresses = append(listenAddresses, "["+addr.String()+"%"+port.Spec.Unbundled.NicName+"]")
				listenAddresses[nic] = fmt.Sprintf("[%s%%%s]:%d", addr.String(), nic, listeningPort)
			}
		}
	}
	log.L().Info("DynLL detected listening addresses", zap.Reflect("addrs", listenAddresses))

	// now we can run them
	for nic, addr := range listenAddresses {
		ret.httpServers[nic] = generic.NewHttpServer(addr, "", "", "", handler)
	}
	return ret, nil
}

func getInterfacesForServerNeighbours(ctx context.Context, k8sClient client.Client, selfHostname string) ([]string, error) {
	// get our own object first based on the hostname
	// if this isn't there, then all bets are off
	obj := &wiring1alpha2.Server{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: selfHostname}, obj); err != nil {
		return nil, err
	}

	// retrieve all connections that belong to us
	connList := &wiring1alpha2.ConnectionList{}
	if err := k8sClient.List(ctx, connList, wiring1alpha2.MatchingLabelsForListLabelServer(selfHostname)); err != nil {
		return nil, err
	}
	if len(connList.Items) == 0 {
		return nil, fmt.Errorf("no connections configured for server '%s'", selfHostname)
	}

	// now build a list of all interfaces that belong to the server and ensure to deduplicate it
	retMap := make(map[string]struct{}, len(connList.Items))
	for _, conn := range connList.Items {
		// we are only interested in management connections
		if conn.Spec.Management == nil {
			continue
		}

		portName := conn.Spec.Management.Link.Server.LocalPortName()
		portMAC := conn.Spec.Management.Link.Server.MAC
		nic, err := net.GetInterface(portName, portMAC)
		if err != nil {
			log.L().Warn("Getting interface failed, skipping", zap.String("nic", portName), zap.String("mac", portMAC), zap.Error(err))
			continue
		}

		retMap[nic] = struct{}{}
	}
	ret := make([]string, 0, len(retMap))
	for intf := range retMap {
		ret = append(ret, intf)
	}
	return ret, nil
}

// TODO: this actually needs rework as this is meant for a switch-switch neighbour case, but the connection type does not exist yet
func getInterfacesForSwitchNeighbours(ctx context.Context, k8sClient client.Client, selfHostname string) ([]string, error) {
	// get our own object first based on the hostname
	// if this isn't there, then all bets are off
	obj := &wiring1alpha2.Switch{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: selfHostname}, obj); err != nil {
		return nil, err
	}

	// retrieve all connections that belong to us
	connList := &wiring1alpha2.ConnectionList{}
	if err := k8sClient.List(ctx, connList, wiring1alpha2.MatchingLabelsForListLabelSwitch(selfHostname)); err != nil {
		return nil, err
	}
	if len(connList.Items) == 0 {
		return nil, fmt.Errorf("no connections configured for server '%s'", selfHostname)
	}

	// now build a list of all interfaces that belong to the server and ensure to deduplicate it
	// for switches that list contains a lot more connections than whwat we are interested in at first
	retMap := make(map[string]struct{}, len(connList.Items))
	for _, conn := range connList.Items {
		// we are only interested in management connections
		if conn.Spec.Management == nil {
			continue
		}
		intf := conn.Spec.Management.Link.Switch.BasePortName.LocalPortName()
		retMap[intf] = struct{}{}
	}
	ret := make([]string, 0, len(retMap))
	for intf := range retMap {
		ret = append(ret, intf)
	}
	return ret, nil
}

func (s *DynLLServer) Done() <-chan struct{} {
	return s.done
}

func (s *DynLLServer) Err() <-chan error {
	return s.err
}

func (s *DynLLServer) Start() {
	var wg sync.WaitGroup
	wg.Add(len(s.httpServers))

	for i, hs := range s.httpServers {
		go func(_ string, hs *generic.HTTPServer) {
			hs.Start()
			<-hs.Done()
			// we filter out all ErrServerClosed which are generated by Shutdown or Closed calls
			if err := hs.Err(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.err <- fmt.Errorf("server on '%s': %w", hs.Srv().Addr, err)
			}
			wg.Done()
		}(i, hs)
	}

	go func() {
		wg.Wait()
		close(s.done)
		close(s.err)
	}()
}

func (s *DynLLServer) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var errs []error
	errch := make(chan error, len(s.httpServers))
	wg.Add(len(s.httpServers))

	// fan out shutdown commands to all servers
	for _, hs := range s.httpServers {
		go func(hs *generic.HTTPServer) {
			if err := hs.Shutdown(ctx); err != nil {
				errch <- fmt.Errorf("server on '%s': %w", hs.Srv().Addr, err)
			}
			wg.Done()
		}(hs)
	}

	// collect errors
	done := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-errch:
				errs = append(errs, err)
			case <-done:
				return
			}
		}
	}()

	// wait until all shutdowns have run
	wg.Wait()
	close(done)

	// return accordingly
	if len(errs) == 0 {
		return nil
	} else if len(errs) == 1 {
		return fmt.Errorf("shutdown error: %w", errs[0])
	} else {
		return fmt.Errorf("multiple shutdown errors:\n%w", errors.Join(errs...))
	}
}

func (s *DynLLServer) Close() error {
	var wg sync.WaitGroup
	var errs []error
	errch := make(chan error, len(s.httpServers))
	wg.Add(len(s.httpServers))

	// fan out close commands to all servers
	for _, hs := range s.httpServers {
		go func(hs *generic.HTTPServer) {
			if err := hs.Close(); err != nil {
				errch <- fmt.Errorf("server on '%s': %w", hs.Srv().Addr, err)
			}
			wg.Done()
		}(hs)
	}

	// collect errors
	done := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-errch:
				errs = append(errs, err)
			case <-done:
				return
			}
		}
	}()

	wg.Wait()
	close(done)

	// return accordingly
	if len(errs) == 0 {
		return nil
	} else if len(errs) == 1 {
		return fmt.Errorf("close error: %w", errs[0])
	} else {
		return fmt.Errorf("multiple close errors:\n%w", errors.Join(errs...))
	}
}
