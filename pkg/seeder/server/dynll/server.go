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
	obj := &wiring1alpha2.Server{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: selfHostname}, obj); err != nil {
		return nil, err
	}
	labels := client.MatchingLabels{"fabric.githedgehog.com/server": selfHostname}
	if rack, ok := obj.GetLabels()["fabric.githedgehog.com/rack"]; ok {
		labels["fabric.githedgehog.com/rack"] = rack
	}

	// retrieve all of our ports that belong to us
	portList := &wiring1alpha2.ServerPortList{}
	if err := k8sClient.List(ctx, portList, labels); err != nil {
		return nil, err
	}
	if len(portList.Items) == 0 {
		return nil, fmt.Errorf("no ports configured for server")
	}
	ret := make([]string, 0, len(portList.Items))
	for _, port := range portList.Items {
		if port.Spec.Unbundled != nil && port.Spec.Unbundled.Neighbor.Switch != nil {
			// we expect a switch on this port as a neighbour, so we want to listen on this port
			nic := port.Spec.Unbundled.NicName
			ret = append(ret, nic)
		}
	}
	return ret, nil
}

func getInterfacesForSwitchNeighbours(ctx context.Context, k8sClient client.Client, selfHostname string) ([]string, error) {
	obj := &wiring1alpha2.Switch{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: selfHostname}, obj); err != nil {
		return nil, err
	}
	labels := client.MatchingLabels{"fabric.githedgehog.com/switch": selfHostname}
	if rack, ok := obj.GetLabels()["fabric.githedgehog.com/rack"]; ok {
		labels["fabric.githedgehog.com/rack"] = rack
	}

	// retrieve all of our ports that belong to us
	portList := &wiring1alpha2.SwitchPortList{}
	if err := k8sClient.List(ctx, portList, labels); err != nil {
		return nil, err
	}
	if len(portList.Items) == 0 {
		return nil, fmt.Errorf("no ports configured for server")
	}
	ret := make([]string, 0, len(portList.Items))
	for _, port := range portList.Items {
		if port.Spec.Neighbor.Switch != nil {
			// we expect a switch on this port as a neighbour, so we want to listen on this port
			nic := port.Spec.NOSPortName
			ret = append(ret, nic)
		}
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
