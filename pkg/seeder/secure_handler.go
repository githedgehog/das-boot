package seeder

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"

	config1 "go.githedgehog.com/dasboot/pkg/stage1/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

const (
	stage1PathBase = "/stage1/"
	stage2PathBase = "/stage2/"
	registerPath   = "/register"
)

func (s *seeder) secureHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(AddResponseRequestID())
	r.Use(middleware.Heartbeat("/healthz"))
	r.Get(path.Join(stage1PathBase, "{arch}"), s.getStageArtifact("stage1", s.stage1Authz, s.embedStage1Config))
	r.Get(path.Join(stage2PathBase, "{arch}"), s.getStageArtifact("stage2", s.stage2Authz, s.embedStage2Config))
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
		return fmt.Errorf("device certificate missing")
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
	return nil, nil
}
