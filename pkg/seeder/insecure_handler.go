package seeder

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"go.githedgehog.com/dasboot/pkg/seeder/ipam"
	config0 "go.githedgehog.com/dasboot/pkg/stage0/config"
)

const (
	ipamPath = "/stage0/ipam"
)

func (s *seeder) insecureHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
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
	s.getStageArtifact("stage0", s.embedStage0Config)(w, r)
}

var stage0Fallback = []byte(`#!/bin/sh

echo "ERROR: Hedgehog SONiC is not supported on this platform ($onie_platform)" 1>&2

exit 1
`)

func (s *seeder) embedStage0Config(r *http.Request, _ string, artifactBytes []byte) ([]byte, error) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	ipamURL := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   ipamPath,
	}
	return s.ecg.Stage0(artifactBytes, &config0.Stage0{
		CA:          s.installerSettings.serverCADER,
		SignatureCA: s.installerSettings.configSignatureCADER,
		IPAMURL:     ipamURL.String(),
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

	set := &ipam.Settings{
		DNSServers:    s.installerSettings.dnsServers,
		NTPServers:    s.installerSettings.ntpServers,
		SyslogServers: s.installerSettings.syslogServers,
		// as the architecture has been validated by this point, we can rely on this value
		Stage1URL: s.installerSettings.stage1URL(req.Arch),
	}
	resp, err := ipam.ProcessRequest(r.Context(), set, s, &req)
	if err != nil {
		errorWithJSON(w, r, http.StatusInternalServerError, "failed to process IPAM request: %s", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		l.Error("processIPAMRequest: failed to encode JSON response",
			zap.String("request", middleware.GetReqID(r.Context())),
			zap.Error(err),
		)
	}
}
