package embedded

import (
	"embed"
	"io"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.uber.org/zap"
)

var l = log.L()

func Provider() artifacts.Provider {
	return &embeddedProvider{}
}

//go:embed artifacts/stage0-*
//go:embed artifacts/stage1-*
//go:embed artifacts/stage2-*
var content embed.FS

type embeddedProvider struct{}

// Get implements artifacts.Provider
func (*embeddedProvider) Get(artifact string) io.ReadCloser {
	switch artifact {
	case artifacts.Stage0X8664:
		f, err := content.Open("artifacts/stage0-amd64")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage1X8664:
		f, err := content.Open("artifacts/stage1-amd64")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage2X8664:
		f, err := content.Open("artifacts/stage2-amd64")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f

	case artifacts.Stage0Arm64:
		f, err := content.Open("artifacts/stage0-arm64")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage1Arm64:
		f, err := content.Open("artifacts/stage1-arm64")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage2Arm64:
		f, err := content.Open("artifacts/stage2-arm64")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage0Arm:
		f, err := content.Open("artifacts/stage0-arm")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage1Arm:
		f, err := content.Open("artifacts/stage1-arm")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage2Arm:
		f, err := content.Open("artifacts/stage2-arm")
		if err != nil {
			l.Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	default:
		l.Debug("no such artifact", zap.String("provider", "embedded"), zap.String("artifact", artifact))
		return nil
	}
}

var _ artifacts.Provider = &embeddedProvider{}