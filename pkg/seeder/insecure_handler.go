package seeder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"go.githedgehog.com/dasboot/pkg/seeder/ipam"
	config0 "go.githedgehog.com/dasboot/pkg/stage0/config"
)

const (
	ipamPath = "/stage0/ipam"
)

func (s *seeder) insecureHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(RequestLogger(log.L()))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(AddResponseRequestID())
	r.Use(middleware.Heartbeat("/healthz"))
	r.Get("/onie-installer-{arch}", s.getStage0Artifact)
	r.Get("/onie-installer", s.getStage0Artifact)
	r.Get("/onie-updater-{arch}", s.getStage0Artifact)
	r.Get("/onie-updater", s.getStage0Artifact)
	r.Get("/stage0/{arch}", s.getStage0Artifact)
	r.Route(ipamPath, func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/", s.processIPAMRequest)
	})
	return r
}

func (s *seeder) getStage0Artifact(w http.ResponseWriter, r *http.Request) {
	// if this hit a fallback URL, we serve the bash script saying that this is not supported on this device
	archParam := chi.URLParam(r, "arch")
	if archParam == "" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(stage0Fallback) //nolint: errcheck
		return
	}

	// execute the "standard" getStageArtifact handler now
	s.getStageArtifact("stage0", s.stage0Authz, s.embedStage0Config)(w, r)
}

var stage0Fallback = []byte(`#!/bin/sh

source /etc/machine.conf
echo "FATAL: If you have not seen any previous installers failing, then this means that Hedgehog SONiC is not supported on this platform ($onie_platform)" 1>&2

exit 1
`)

func (s *seeder) stage0Authz(*http.Request) error {
	// stage 0 is literally the *only* artifact which does *not* require any other
	// additional authorization
	return nil
}

func (s *seeder) embedStage0Config(r *http.Request, _ string, artifactBytes []byte) ([]byte, error) {
	// build IPAM URL
	// we are going to send back the same host
	// that we are using for serving this stage 0 artifact
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	ipamURL := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   ipamPath,
	}
	parseUint := func(s string) uint {
		n, err := strconv.ParseUint(s, 0, 0)
		if err != nil {
			return 0
		}
		return uint(n)
	}

	// this is being requested from a link-local address
	// we are going to discover the neighbour and also serve
	// the location information for the configured neighbour
	var loc *location.Info
	if strings.HasPrefix(r.Host, "[fe80:") {
		host := strings.TrimSuffix(strings.TrimPrefix(r.Host, "["), "]")
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*30)
		defer cancel()
		sw, _, err := s.cpc.GetNeighbourSwitchByAddr(ctx, host)
		if err != nil {
			log.L().Error("failed to discover neighbouring switch", zap.String("addr", host), zap.Error(err))
		} else {
			md, err := json.Marshal(&sw.Spec.Location)
			if err != nil {
				log.L().Error("failed to marshal location information of neighbouring switch", zap.Error(err))
			} else {
				locationUUID, _ := sw.Spec.Location.GenerateUUID()
				loc = &location.Info{
					UUID:        locationUUID,
					UUIDSig:     []byte(sw.Spec.LocationSig.UUIDSig),
					Metadata:    string(md),
					MetadataSig: []byte(sw.Spec.LocationSig.Sig),
				}
				log.L().Info("Serving location information for request", zap.Reflect("loc", loc))
			}
		}
	}

	return s.ecg.Stage0(artifactBytes, &config0.Stage0{
		CA:          s.installerSettings.serverCADER,
		SignatureCA: s.installerSettings.configSignatureCADER,
		IPAMURL:     ipamURL.String(),
		Location:    loc,
		OnieHeaders: &config0.OnieHeaders{
			SerialNumber: r.Header.Get("ONIE-SERIAL-NUMBER"),
			EthAddr:      r.Header.Get("ONIE-ETH-ADDR"),
			VendorID:     parseUint(r.Header.Get("ONIE-VENDOR-ID")),
			Machine:      r.Header.Get("ONIE-MACHINE"),
			MachineRev:   parseUint(r.Header.Get("ONIE-MACHINE-REV")),
			Arch:         r.Header.Get("ONIE-ARCH"),
			SecurityKey:  r.Header.Get("ONIE-SECURITY-KEY"),
			Operation:    r.Header.Get("ONIE-OPERATION"),
		},
	})
}

func (s *seeder) processIPAMRequest(w http.ResponseWriter, r *http.Request) {
	// our response will always be JSON
	w.Header().Set("Content-Type", "application/json")

	if r.ContentLength == 0 {
		errorWithJSON(w, r, http.StatusBadRequest, "no request data")
		return
	}

	var req ipam.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorWithJSON(w, r, http.StatusBadRequest, "failed to decode JSON request: %s", err)
		return
	}
	if err := req.Validate(); err != nil {
		errorWithJSON(w, r, http.StatusBadRequest, "request validation: %s", err)
		return
	}

	// try to see if we can find the adjacent switch port
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*30)
	defer cancel()
	host := strings.TrimSuffix(strings.TrimPrefix(r.Host, "["), "]")
	adjacentSwitch, adjacentPort, err := s.cpc.GetNeighbourSwitchByAddr(ctx, host)
	if err != nil {
		log.L().Error("failed to discover switch port by address", zap.String("addr", host), zap.Error(err))
	}
	// TODO: the location UUID should match

	set := &ipam.Settings{
		DNSServers:    s.installerSettings.dnsServers,
		NTPServers:    s.installerSettings.ntpServers,
		SyslogServers: s.installerSettings.syslogServers,
		// as the architecture has been validated by this point, we can rely on this value
		Stage1URL: s.installerSettings.stage1URL(req.Arch),
	}
	for _, setRoute := range s.installerSettings.routes {
		r := &ipam.Route{
			Gateway:      setRoute.gateway,
			Destinations: make([]string, len(setRoute.destinations)),
		}
		copy(r.Destinations, setRoute.destinations)
		set.Routes = append(set.Routes, r)
	}
	resp, err := ipam.ProcessRequest(r.Context(), set, s.cpc, &req, adjacentSwitch, adjacentPort)
	if err != nil {
		errorWithJSON(w, r, http.StatusInternalServerError, "failed to process IPAM request: %s", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.L().Error("processIPAMRequest: failed to encode JSON response",
			zap.String("request", middleware.GetReqID(r.Context())),
			zap.Error(err),
		)
	}
}
