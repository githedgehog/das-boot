package ntp

import (
	"context"
	"errors"
	"syscall"
	"testing"
)

func TestSyncClock(t *testing.T) {
	canceledCtx, canceledCtxCancel := context.WithCancel(context.Background())
	canceledCtxCancel()
	errSettimeofday := errors.New("settimeofday() failed")
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
