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

package ntp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestSyncClock(t *testing.T) {
	canceledCtx, canceledCtxCancel := context.WithCancel(context.Background())
	canceledCtxCancel()
	errSettimeofday := errors.New("settimeofday() failed")
	tempFile, err := os.CreateTemp("", "TestSyncClock-")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tempFile.Name())

	type args struct {
		ctx     context.Context
		servers []string
	}
	tests := []struct {
		name                string
		args                args
		wantErr             bool
		wantErrToBe         error
		syscallSettimeofday func(tv *syscall.Timeval) error
		unixIoctlGetRTCTime func(fd int) (*unix.RTCTime, error)
		unixIoctlSetRTCTime func(fd int, value *unix.RTCTime) error
		osOpen              func(name string) (*os.File, error)
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				servers: []string{
					// TODO: maybe find a better way than requiring internet for this to pass
					"0.arch.pool.ntp.org",
				},
			},
			syscallSettimeofday: func(tv *syscall.Timeval) error {
				// simulate success here
				return nil
			},
			osOpen: func(name string) (*os.File, error) {
				return tempFile, nil
			},
			unixIoctlGetRTCTime: func(int) (*unix.RTCTime, error) {
				t := time.Now().Add(-60 * time.Second)
				return timeToRTCTime(&t), nil
			},
			unixIoctlSetRTCTime: func(fd int, value *unix.RTCTime) error {
				return nil
			},
		},
		{
			name: "successful NTP but failing settimeofday",
			args: args{
				ctx: context.Background(),
				servers: []string{
					// TODO: maybe find a better way than requiring internet for this to pass
					"0.arch.pool.ntp.org",
				},
			},
			syscallSettimeofday: func(tv *syscall.Timeval) error {
				// simulate failure here
				return errSettimeofday
			},
			osOpen: func(name string) (*os.File, error) {
				// should not reach here
				return nil, fmt.Errorf("in os.Open even though we should not be here")
			},
			unixIoctlGetRTCTime: func(fd int) (*unix.RTCTime, error) {
				// should not reach here
				return nil, fmt.Errorf("in unix.IoctlGetRTCTime even though we should not be here")
			},
			unixIoctlSetRTCTime: func(fd int, value *unix.RTCTime) error {
				return fmt.Errorf("in unix.IoctlSetRTCTime even though we should not be here")
			},
			wantErr:     true,
			wantErrToBe: errSettimeofday,
		},
		{
			name: "canceled context",
			args: args{
				ctx: canceledCtx,
				servers: []string{
					"0.arch.pool.ntp.org",
				},
			},
			wantErr:     true,
			wantErrToBe: ErrNTPQueriesUnsuccessful,
			// not necessary, but just a precaution
			syscallSettimeofday: func(tv *syscall.Timeval) error {
				return nil
			},
			osOpen: func(name string) (*os.File, error) {
				// should not reach here
				return nil, fmt.Errorf("in os.Open even though we should not be here")
			},
			unixIoctlGetRTCTime: func(fd int) (*unix.RTCTime, error) {
				// should not reach here
				return nil, fmt.Errorf("in unix.IoctlGetRTCTime even though we should not be here")
			},
			unixIoctlSetRTCTime: func(fd int, value *unix.RTCTime) error {
				return fmt.Errorf("in unix.IoctlSetRTCTime even though we should not be here")
			},
		},
		{
			name: "no servers",
			args: args{
				ctx:     context.Background(),
				servers: []string{},
			},
			wantErr:     true,
			wantErrToBe: ErrNoServers,
			// not necessary, but just a precaution
			syscallSettimeofday: func(tv *syscall.Timeval) error {
				return nil
			},
			osOpen: func(name string) (*os.File, error) {
				// should not reach here
				return nil, fmt.Errorf("in os.Open even though we should not be here")
			},
			unixIoctlGetRTCTime: func(fd int) (*unix.RTCTime, error) {
				// should not reach here
				return nil, fmt.Errorf("in unix.IoctlGetRTCTime even though we should not be here")
			},
			unixIoctlSetRTCTime: func(fd int, value *unix.RTCTime) error {
				return fmt.Errorf("in unix.IoctlSetRTCTime even though we should not be here")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(tt.args.ctx)
			defer cancel()
			if tt.syscallSettimeofday != nil {
				oldSyscallSettimeofday := syscallSettimeofday
				defer func() {
					syscallSettimeofday = oldSyscallSettimeofday
				}()
				syscallSettimeofday = tt.syscallSettimeofday
			}
			if tt.unixIoctlGetRTCTime != nil {
				oldUnixIoctlGetRTCTime := unixIoctlGetRTCTime
				defer func() {
					unixIoctlGetRTCTime = oldUnixIoctlGetRTCTime
				}()
				unixIoctlGetRTCTime = tt.unixIoctlGetRTCTime
			}
			if tt.unixIoctlSetRTCTime != nil {
				oldUnixIoctlSetRTCTime := unixIoctlSetRTCTime
				defer func() {
					unixIoctlSetRTCTime = oldUnixIoctlSetRTCTime
				}()
				unixIoctlSetRTCTime = tt.unixIoctlSetRTCTime
			}
			if tt.osOpen != nil {
				oldOsOpen := osOpen
				defer func() {
					osOpen = oldOsOpen
				}()
				osOpen = tt.osOpen
			}
			err := SyncClock(ctx, tt.args.servers)
			if (err != nil) != tt.wantErr {
				t.Errorf("SyncClock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("SetSystemResolvers() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}
