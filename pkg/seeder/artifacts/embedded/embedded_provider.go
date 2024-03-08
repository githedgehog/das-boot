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

package embedded

import (
	"embed"
	"io"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.uber.org/zap"
)

func Provider() artifacts.Provider {
	return &embeddedProvider{}
}

//go:embed artifacts/stage0-*
//go:embed artifacts/stage1-*
//go:embed artifacts/stage2-*
//go:embed artifacts/hedgehog-agent-provisioner-*
var content embed.FS

type embeddedProvider struct{}

// Get implements artifacts.Provider
func (*embeddedProvider) Get(artifact string) io.ReadCloser {
	switch artifact {
	case artifacts.Stage0X8664:
		f, err := content.Open("artifacts/stage0-amd64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage1X8664:
		f, err := content.Open("artifacts/stage1-amd64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage2X8664:
		f, err := content.Open("artifacts/stage2-amd64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f

	case artifacts.Stage0Arm64:
		f, err := content.Open("artifacts/stage0-arm64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage1Arm64:
		f, err := content.Open("artifacts/stage1-arm64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage2Arm64:
		f, err := content.Open("artifacts/stage2-arm64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage0Arm:
		f, err := content.Open("artifacts/stage0-arm")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage1Arm:
		f, err := content.Open("artifacts/stage1-arm")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.Stage2Arm:
		f, err := content.Open("artifacts/stage2-arm")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.HHAgentProvX8664:
		f, err := content.Open("artifacts/hedgehog-agent-provisioner-amd64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.HHAgentProvArm64:
		f, err := content.Open("artifacts/hedgehog-agent-provisioner-arm64")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	case artifacts.HHAgentProvArm:
		f, err := content.Open("artifacts/hedgehog-agent-provisioner-arm")
		if err != nil {
			log.L().Error("open failed", zap.String("provider", "embedded"), zap.String("artifact", artifact), zap.Error(err))
			return nil
		}
		return f
	default:
		log.L().Debug("no such artifact", zap.String("provider", "embedded"), zap.String("artifact", artifact))
		return nil
	}
}

var _ artifacts.Provider = &embeddedProvider{}
