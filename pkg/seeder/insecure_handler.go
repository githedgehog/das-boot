package seeder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	// get the stage0 artifact
	artifact := "stage0-" + archParam
	f := s.artifactsProvider.Get(artifact)
	if f == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer f.Close()

	// generate an embedded config for it
	artifactBytes, err := io.ReadAll(f)
	if err != nil {
		errorWithJSON(w, r, http.StatusInternalServerError, "failed to read artifact: %s", err)
		return
	}
	ipamURL := url.URL{
		Scheme: r.URL.Scheme,
		Host:   r.Host,
		Path:   ipamPath,
	}
	signedArtifactWithConfig, err := s.ecg.Stage0(artifactBytes, &config0.Stage0{
		CA:          s.installerSettings.serverCADER,
		SignatureCA: s.installerSettings.configSignatureCADER,
		IPAMURL:     ipamURL.String(),
	})
	if err != nil {
		errorWithJSON(w, r, http.StatusInternalServerError, "failed to embed configuration: %s", err)
		return
	}
	src := bufio.NewReader(bytes.NewBuffer(signedArtifactWithConfig))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, src); err != nil {
		l.Error("failed to write artifact to HTTP response",
			zap.String("request", middleware.GetReqID(r.Context())),
			zap.String("artifact", artifact),
			zap.Error(err),
		)
	}
}

var stage0Fallback = []byte(`#!/bin/sh

echo "ERROR: Hedgehog SONiC is not supported on this platform ($onie_platform)" 1>&2

exit 1
`)

func (s *seeder) processIPAMRequest(w http.ResponseWriter, r *http.Request) {
	// our response will always be JSON
	w.Header().Set("Content-Type", "application/json")

	var req ipam.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorWithJSON(w, r, http.StatusUnprocessableEntity, "failed to decode JSON request: %s", err)
		return
	}

	resp, err := ipam.ProcessRequest(r.Context(), s, &req)
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

func errorWithJSON(w http.ResponseWriter, r *http.Request, statusCode int, format string, a ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	v := struct {
		ReqID string `json:"request_id,omitempty"`
		Err   string `json:"error"`
	}{
		ReqID: middleware.GetReqID(r.Context()),
		Err:   fmt.Sprintf(format, a...),
	}
	b, err := json.Marshal(&v)
	if err == nil {
		w.Write(b) //nolint: errcheck
	}
}
