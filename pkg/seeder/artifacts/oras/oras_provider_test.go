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

package oras

import (
	"context"
	"io"
	"os"
	"testing"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_orasProvider_Get(t *testing.T) {
	// TODO: this is not really a unit test at this point
	// However, it helps to check this manually
	l := log.NewZapWrappedLogger(zap.Must(log.NewSerialConsole(zapcore.DebugLevel, "json", false)))
	log.ReplaceGlobals(l)
	type args struct {
		artifact string
	}
	tests := []struct {
		name string
		args args
		skip bool
	}{
		{
			name: "SONiC image download",
			args: args{
				artifact: "sonic/x86_64-kvm_x86_64-r0",
			},
			skip: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileStoreBasePath, err := os.MkdirTemp(os.TempDir(), "oras-provider-*")
			if err != nil {
				t.Errorf("failed to create temporary directory: %s", err)
				return
			}
			defer func() {
				os.RemoveAll(fileStoreBasePath)
			}()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			op, err := Provider(ctx, "oci://registry.local:5000/githedgehog", fileStoreBasePath, ProviderOptionServerCA("/home/mheese/git/das-boot/dev/oci/oci-repo-ca-cert.pem"))
			if err != nil {
				t.Errorf("Provider: %s", err)
				return
			}
			if tt.skip {
				t.Skipf("skipping test '%s' as requested", tt.name)
				return
			}
			got := op.Get(tt.args.artifact)
			if got == nil {
				t.Errorf("no artifact found")
				return
			}
			defer got.Close()
			f, err := os.CreateTemp("", "oras-")
			if err != nil {
				t.Errorf("CreateTemp: %s", err)
				return
			}
			defer f.Close()
			if _, err := io.Copy(f, got); err != nil {
				t.Errorf("Copy: %s", err)
			}
			t.Logf("artifact written to: %s", f.Name())
		})
	}
}
