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

package exec

import (
	"context"
	"os"
	"strings"
	"testing"
)

func Test_Command(t *testing.T) {
	// just to make sure that without switching out execCommand, it really calls a command
	t.Run("run", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		out, err := Command("pwd").Output()
		if err != nil {
			panic(err)
		}
		got := strings.TrimSpace(string(out))
		if wd != got {
			t.Errorf("working directories did not match: got = %s, want = %s", got, wd)
		}
	})
}

func Test_CommandContext(t *testing.T) {
	// just to make sure that without switching out execCommand, it really calls a command
	t.Run("run", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		out, err := CommandContext(context.Background(), "pwd").Output()
		if err != nil {
			panic(err)
		}
		got := strings.TrimSpace(string(out))
		if wd != got {
			t.Errorf("working directories did not match: got = %s, want = %s", got, wd)
		}
	})
}
