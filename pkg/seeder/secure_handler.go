package seeder

import (
	"bufio"
	"bytes"
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
	r.Get(path.Join(stage1PathBase, "{arch}"), s.getStageArtifact("stage1", s.embedStage1Config))
	r.Get(path.Join(stage2PathBase, "{arch}"), s.getStageArtifact("stage2", s.embedStage2Config))
	return r
}

func (s *seeder) getStageArtifact(artifact string, embedConfig func(*http.Request, string, []byte) ([]byte, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

func (s *seeder) embedStage1Config(_ *http.Request, arch string, artifactBytes []byte) ([]byte, error) {
	return s.ecg.Stage1(artifactBytes, &config1.Stage1{
		RegisterURL: s.installerSettings.registerURL(),
		Stage2URL:   s.installerSettings.stage2URL(arch),
	})
}

func (s *seeder) embedStage2Config(_ *http.Request, arch string, artifactBytes []byte) ([]byte, error) {
	return nil, nil
}
