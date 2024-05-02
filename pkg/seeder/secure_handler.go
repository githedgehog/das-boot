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

package seeder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	confighhagentprov "go.githedgehog.com/dasboot/pkg/hhagentprov/config"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	"go.githedgehog.com/dasboot/pkg/seeder/registration"
	config1 "go.githedgehog.com/dasboot/pkg/stage1/config"
	config2 "go.githedgehog.com/dasboot/pkg/stage2/config"
	agentv1alpha2 "go.githedgehog.com/fabric/api/agent/v1alpha2"
	"gopkg.in/yaml.v2"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

const (
	stage1PathBase             = "/stage1/"
	stage2PathBase             = "/stage2/"
	nosInstallerPathBase       = "/nos/install/"
	onieUpdaterPathBase        = "/onie/update/"
	hhAgentProvisionerPathBase = "/provisioners/hedgehog-agent/"
	registerPath               = "/register"
)

func (s *seeder) secureHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(RequestLogger(log.L()))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(AddResponseRequestID())
	r.Use(middleware.Heartbeat("/healthz"))
	r.Get(path.Join(stage1PathBase, "{arch}"), s.getStageArtifact("stage1", s.stage1Authz, s.embedStage1Config))
	r.Get(path.Join(stage2PathBase, "{arch}"), s.getStageArtifact("stage2", s.stage2Authz, s.embedStage2Config))
	r.Post(registerPath, s.registerHandler)
	r.Get(path.Join(registerPath, "{devid}"), s.registerPollHandler)
	r.Get(path.Join(nosInstallerPathBase, "{platform}", "{devid}"), s.getNOSArtifact(s.stage2Authz))
	r.Get(path.Join(onieUpdaterPathBase, "{platform}"), s.getONIEArtifact(s.stage2Authz))
	// to lift the confusion: this is the route for the provisioner executable
	r.Get(path.Join(hhAgentProvisionerPathBase, "{arch}"), s.getStageArtifact("hedgehog-agent-provisioner", s.stage2Authz, s.embedStageHedgehogAgentProvisionerConfig))
	// and this is the route to the agent executable which the provisioner calls
	r.Get(path.Join(hhAgentProvisionerPathBase, "agent", "{devid}"), s.getAgentArtifact(s.stage2Authz))
	r.Get(path.Join(hhAgentProvisionerPathBase, "agent", "config", "{devid}"), s.getAgentConfig(s.stage2Authz))
	r.Get(path.Join(hhAgentProvisionerPathBase, "agent", "kubeconfig", "{devid}"), s.getAgentKubeconfig(s.stage2Authz))
	return r
}

func (s *seeder) getStageArtifact(artifact string, authz func(*http.Request) error, embedConfig func(*http.Request, string, []byte) ([]byte, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authz(r); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// if this hit a fallback URL, we serve the bash script saying that this is not supported on this device
		archParam := chi.URLParam(r, "arch")
		if archParam == "" {
			errorWithJSON(w, r, http.StatusNotFound, "missing architecture in request path")
			return
		}

		// get the artifact which is architecture dependent
		artifactArch := artifact + "-" + archParam
		f := s.artifactsProvider.Get(artifactArch)
		if f == nil {
			errorWithJSON(w, r, http.StatusNotFound, "artifact '%s' not found", artifactArch)
			return
		}
		defer f.Close()

		// we need to read it completely into memory because it needs to be signed
		// and get its config embedded
		artifactBytes, err := io.ReadAll(f)
		if err != nil {
			errorWithJSON(w, r, http.StatusInternalServerError, "failed to read artifact: %s", err)
			return
		}

		// generate an embedded config for it
		signedArtifactWithConfig, err := embedConfig(r, archParam, artifactBytes)
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
				zap.String("artifact", artifactArch),
				zap.Error(err),
			)
		}
	}
}

func (s *seeder) stage1Authz(r *http.Request) error {
	// stage1 just needs to ensure that the request was made with TLS
	// In the future that might get extended to require other details
	if r.TLS == nil {
		return fmt.Errorf("stage 1 artifact requires a TLS connection")
	}
	return nil
}

func (s *seeder) embedStage1Config(_ *http.Request, arch string, artifactBytes []byte) ([]byte, error) {
	return s.ecg.Stage1(artifactBytes, &config1.Stage1{
		RegisterURL: s.installerSettings.registerURL(),
		Stage2URL:   s.installerSettings.stage2URL(arch),
	})
}

func (s *seeder) stage2Authz(r *http.Request) error {
	// must be a TLS request
	if r.TLS == nil {
		return fmt.Errorf("stage 2 artifact requires a TLS connection")
	}

	// If there were no client certificates provided (and verified),
	// then you don't have access to this route
	if len(r.TLS.PeerCertificates) < 1 {
		return fmt.Errorf("device certificate not presented")
	}

	// check if certificate is not revoked
	// This might be covered by golang, but probably not entirely, we need to check
	// TODO

	// get the UUID of the device from the cert
	deviceCert := r.TLS.PeerCertificates[0]
	uuid := deviceCert.Subject.CommonName
	if uuid == "" {
		return fmt.Errorf("device certificate missing its CN UUID")
	}

	// check if we have this in our registry
	// this essentially makes revoking certificates needless for basic use-cases:
	// a device which is deleted from the control plane will not be able
	// to gain access to images anymore. Only compromised devices/certificates
	// still require the additional check as the device ID will never change
	// TODO

	// check the keylime CV for the status of the client first
	// this essentially means that only devices with good hardware attestation
	// state are allowed to get an installation image
	// TODO

	return nil
}

func (s *seeder) embedStage2Config(_ *http.Request, arch string, artifactBytes []byte) ([]byte, error) {
	return s.ecg.Stage2(artifactBytes, &config2.Stage2{
		Platform:        "", // this should be empty, might only be useful in the future
		NOSInstallerURL: s.installerSettings.nosInstallerURL(),
		ONIEUpdaterURL:  s.installerSettings.onieUpdaterURL(),
		NOSType:         "hedgehog_sonic",
		HedgehogSonicProvisioners: []config2.HedgehogSonicProvisioner{
			{
				Name: "hedgehog-agent-provisioner",
				URL:  s.installerSettings.hhAgentProvisionerURL(arch),
			},
		},
	})
}

func (s *seeder) embedStageHedgehogAgentProvisionerConfig(_ *http.Request, arch string, artifactBytes []byte) ([]byte, error) {
	return s.ecg.HedgehogAgentProvisioner(artifactBytes, &confighhagentprov.HedgehogAgentProvisioner{
		AgentURL:           s.installerSettings.agentURL(),
		AgentConfigURL:     s.installerSettings.agentConfigURL(),
		AgentKubeconfigURL: s.installerSettings.agentKubeconfigURL(),
	})
}

func (s *seeder) registerHandler(w http.ResponseWriter, r *http.Request) {
	// must be a TLS request
	if r.TLS == nil {
		errorWithJSON(w, r, http.StatusBadRequest, "route requires a TLS connection")
		return
	}

	if r.ContentLength == 0 {
		errorWithJSON(w, r, http.StatusBadRequest, "no request data")
		return
	}

	var req registration.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorWithJSON(w, r, http.StatusBadRequest, "failed to decode JSON request: %s", err)
		return
	}

	// validation doesn't require a CSR but will validate it if it is there
	// however, on this route we require the CSR
	if len(req.CSR) == 0 {
		errorWithJSON(w, r, http.StatusBadRequest, "invalid request: missing CSR")
		return
	}

	// validate it now: this ensures the Device ID is a UUID, and the CSR is valid
	if err := req.Validate(); err != nil {
		errorWithJSON(w, r, http.StatusBadRequest, "invalid request: %s", err.Error())
		return
	}

	resp := s.registry.ProcessRequest(r.Context(), &req)
	writeRegistrationResponse(w, r, resp)
}

func (s *seeder) registerPollHandler(w http.ResponseWriter, r *http.Request) {
	// must be a TLS request
	if r.TLS == nil {
		errorWithJSON(w, r, http.StatusBadRequest, "route requires a TLS connection")
		return
	}

	// get the device ID from the URL paramater
	devidParam := chi.URLParam(r, "devid")
	if devidParam == "" {
		errorWithJSON(w, r, http.StatusBadRequest, "no device ID in URL")
		return
	}

	// build request and validate it before we send it to the processor
	req := &registration.Request{DeviceID: devidParam}
	if err := req.Validate(); err != nil {
		errorWithJSON(w, r, http.StatusBadRequest, "invalid request: %s", err.Error())
		return
	}

	resp := s.registry.ProcessRequest(r.Context(), req)
	writeRegistrationResponse(w, r, resp)
}

func writeRegistrationResponse(w http.ResponseWriter, r *http.Request, resp *registration.Response) {
	b, err := json.Marshal(resp)
	if err != nil {
		errorWithJSON(w, r, http.StatusInternalServerError, "JSON marshalling for registration response failed: %s", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch resp.Status { //nolint: exhaustive
	case registration.RegistrationStatusNotFound:
		w.WriteHeader(registration.HTTPRegistrationRequestNotFound)
	case registration.RegistrationStatusApproved:
		w.WriteHeader(http.StatusOK)
	case registration.RegistrationStatusRejected:
		w.WriteHeader(http.StatusOK)
	case registration.RegistrationStatusPending:
		w.WriteHeader(http.StatusAccepted)
	case registration.RegistrationStatusError:
		w.WriteHeader(registration.HTTPProcessError)
	default:
		// this shouldn't happen, so this status code is indeed appropriate
		sd := ""
		if resp.StatusDescription != "" {
			sd = " (" + resp.StatusDescription + ")"
		}
		errorWithJSON(w, r, http.StatusNotImplemented, "unknown registration response status: %s%s", resp.Status, sd)
		return
	}

	if n, err := w.Write(b); err != nil || n != len(b) {
		l.DPanic("writeRegistrationResponse failed to write response", zap.Error(err), zap.Int("written", n), zap.Int("len", len(b)))
	}
}

func (s *seeder) getNOSArtifact(authz func(*http.Request) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authz(r); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// we require the platform so that we can fetch the artifact
		platformParam := chi.URLParam(r, "platform")
		if platformParam == "" {
			errorWithJSON(w, r, http.StatusNotFound, "missing platform in request path")
			return
		}

		// get the device ID from the URL paramater
		devidParam := chi.URLParam(r, "devid")
		if devidParam == "" {
			errorWithJSON(w, r, http.StatusBadRequest, "no device ID in URL")
			return
		}

		// the device ID parameter and the CN of the peer cert need to match
		if err := s.authzMatchDevice(r, devidParam); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// get agent config from control plane
		agentCfg, err := s.cpc.GetAgentConfig(r.Context(), devidParam)
		if err != nil {
			if errors.Is(err, controlplane.ErrNotFound) {
				errorWithJSON(w, r, http.StatusNotFound, "agent config not found: %s", err)
				return
			}
			errorWithJSON(w, r, http.StatusInternalServerError, "fetching agent config: %s", err)
			return
		}

		var agent *agentv1alpha2.Agent
		if err := yaml.Unmarshal(agentCfg, &agent); err != nil {
			errorWithJSON(w, r, http.StatusInternalServerError, "unmarshalling agent config: %s", err)
			return
		}
		sonicVersion := agent.Spec.Version.NOSVersion

		artifact := fmt.Sprintf("sonic/%s", platformParam)
		if sonicVersion != "" {
			artifact += ":" + sonicVersion
		}
		s.getArtifact(artifact)(w, r)
	}
}

func (s *seeder) getONIEArtifact(authz func(*http.Request) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authz(r); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// we require the platform so that we can fetch the artifact
		platformParam := chi.URLParam(r, "platform")
		if platformParam == "" {
			errorWithJSON(w, r, http.StatusNotFound, "missing platform in request path")
			return
		}

		artifact := fmt.Sprintf("onie/%s", platformParam)
		s.getArtifact(artifact)(w, r)
	}
}

func (s *seeder) getAgentArtifact(authz func(*http.Request) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authz(r); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// get the device ID from the URL paramater
		devidParam := chi.URLParam(r, "devid")
		if devidParam == "" {
			errorWithJSON(w, r, http.StatusBadRequest, "no device ID in URL")
			return
		}

		// get agent config from control plane
		agentCfg, err := s.cpc.GetAgentConfig(r.Context(), devidParam)
		if err != nil {
			if errors.Is(err, controlplane.ErrNotFound) {
				errorWithJSON(w, r, http.StatusNotFound, "agent config not found: %s", err)
				return
			}
			errorWithJSON(w, r, http.StatusInternalServerError, "fetching agent config: %s", err)
			return
		}

		var agent *agentv1alpha2.Agent
		if err := yaml.Unmarshal(agentCfg, &agent); err != nil {
			errorWithJSON(w, r, http.StatusInternalServerError, "unmarshalling agent config: %s", err)
			return
		}
		agentVersion := agent.Spec.Version.Default
		if agent.Spec.Version.Override != "" {
			agentVersion = agent.Spec.Version.Override
		}

		artifact := "fabric/agent"
		if agentVersion != "" {
			artifact += ":" + agentVersion
		}
		s.getArtifact(artifact)(w, r)
	}
}

func (s *seeder) getArtifact(artifact string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		f := s.artifactsProvider.Get(artifact)
		if f == nil {
			errorWithJSON(w, r, http.StatusNotFound, "artifact '%s' not found", artifact)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			l.Error("failed to write artifact to HTTP response",
				zap.String("request", middleware.GetReqID(r.Context())),
				zap.String("artifact", artifact),
				zap.Error(err),
			)
		}
	}
}

func (s *seeder) authzMatchDevice(r *http.Request, deviceID string) error {
	// TODO: this is redundant, needs better treatment somehow
	// must be a TLS request
	if r.TLS == nil {
		return fmt.Errorf("stage 2 artifact requires a TLS connection")
	}

	// If there were no client certificates provided (and verified),
	// then you don't have access to this route
	if len(r.TLS.PeerCertificates) < 1 {
		return fmt.Errorf("device certificate not presented")
	}

	// get the UUID of the device from the cert
	deviceCert := r.TLS.PeerCertificates[0]
	uuid := deviceCert.Subject.CommonName

	if uuid != deviceID {
		return fmt.Errorf("device ID mismatch")
	}
	return nil
}

func (s *seeder) getAgentConfig(authz func(*http.Request) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authz(r); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// get the device ID from the URL paramater
		devidParam := chi.URLParam(r, "devid")
		if devidParam == "" {
			errorWithJSON(w, r, http.StatusBadRequest, "no device ID in URL")
			return
		}

		// the device ID parameter and the CN of the peer cert need to match
		if err := s.authzMatchDevice(r, devidParam); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// get agent config from control plane
		agentCfg, err := s.cpc.GetAgentConfig(r.Context(), devidParam)
		if err != nil {
			if errors.Is(err, controlplane.ErrNotFound) {
				errorWithJSON(w, r, http.StatusNotFound, "agent config not found: %s", err)
				return
			}
			errorWithJSON(w, r, http.StatusInternalServerError, "fetching agent config: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(agentCfg); err != nil {
			l.Error("failed to write agent config to HTTP response", zap.Error(err))
		}
	}
}

func (s *seeder) getAgentKubeconfig(authz func(*http.Request) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authz(r); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// get the device ID from the URL paramater
		devidParam := chi.URLParam(r, "devid")
		if devidParam == "" {
			errorWithJSON(w, r, http.StatusBadRequest, "no device ID in URL")
			return
		}

		// the device ID parameter and the CN of the peer cert need to match
		if err := s.authzMatchDevice(r, devidParam); err != nil {
			errorWithJSON(w, r, http.StatusForbidden, "unauthorized access to artifact: %s", err)
			return
		}

		// get agent kubeconfig from control plane
		agentKubeconfigBytes, err := s.cpc.GetAgentKubeconfig(r.Context(), devidParam)
		if err != nil {
			if errors.Is(err, controlplane.ErrNotFound) {
				errorWithJSON(w, r, http.StatusNotFound, "agent kubeconfig not found: %s", err)
				return
			}
			errorWithJSON(w, r, http.StatusInternalServerError, "fetching agent config: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(agentKubeconfigBytes); err != nil {
			l.Error("failed to write agent config to HTTP response", zap.Error(err))
		}
	}
}
