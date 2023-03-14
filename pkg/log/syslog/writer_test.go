package syslog

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.githedgehog.com/dasboot/test/mock/mocknet"
	"go.uber.org/zap"
)

func startServer(ctx context.Context, wg *sync.WaitGroup, dialAddr string, ch chan<- []byte) {
	lc := net.ListenConfig{}
	l, err := lc.ListenPacket(ctx, "udp", dialAddr)
	if err != nil {
		wg.Done()
		panic(err)
	}

	go func(ctx context.Context, l net.PacketConn) {
		const readTimeout = time.Millisecond * 10
		defer func() {
			l.Close()
			wg.Done()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				l.SetReadDeadline(time.Now().Add(readTimeout)) //nolint:errcheck
				buf := make([]byte, 4096)
				before := time.Now()
				n, _, _ := l.ReadFrom(buf)
				if n > 0 && ch != nil {
					ch <- buf[:n]
				}
				readDur := time.Since(before)
				time.Sleep(readTimeout - readDur)
			}
		}
	}(ctx, l)
}

func Test_defaultUDPConnect(t *testing.T) {
	timedOutCtx, cancel := context.WithCancel(context.Background())
	cancel()
	type args struct {
		ctx            context.Context
		addr           string
		connTimeout    time.Duration
		internalLogger *zap.Logger
	}
	tests := []struct {
		name       string
		args       args
		wantConn   bool
		serverAddr string
	}{
		{
			name: "success",
			args: args{
				ctx:            context.Background(),
				addr:           "[::1]:10514",
				connTimeout:    DefaultConnectionTimeout,
				internalLogger: zap.NewNop(),
			},
			wantConn:   true,
			serverAddr: "[::1]:10514",
		},
		{
			name: "context timeout",
			args: args{
				ctx:            timedOutCtx,
				addr:           "localhost",
				connTimeout:    DefaultConnectionTimeout,
				internalLogger: zap.NewNop(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(tt.args.ctx)
			var wg sync.WaitGroup
			defer func() {
				cancel()
				wg.Wait()
			}()
			if tt.serverAddr != "" {
				wg.Add(1)
				startServer(ctx, &wg, tt.serverAddr, nil)
			}
			got := defaultUDPConnect(ctx, tt.args.connTimeout, tt.args.addr, tt.args.internalLogger)
			if got != nil {
				defer got.Close()
			}
			if (got != nil) != tt.wantConn {
				t.Errorf("defaultUDPConnect() = %v, want %v", got != nil, tt.wantConn)
			}
		})
	}
}

func TestWriter_Write(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	type fields struct {
		ctx        context.Context
		addr       string
		options    []WriterOption
		serverAddr string
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		pre         func(t *testing.T, ctrl *gomock.Controller) WriterOption
		wantN       int
		wantErr     bool
		wantErrToBe error
		want        []byte
	}{
		{
			name: "success",
			fields: fields{
				ctx:        context.Background(),
				addr:       "[::1]:10514",
				serverAddr: "[::1]:10514",
				options: []WriterOption{
					InternalLogger(zap.NewNop()),
					ConnectionTimeout(DefaultConnectionTimeout),
					WriteTimeout(DefaultWriteTimeout),
				},
			},
			args:  args{p: []byte("unit test")},
			wantN: 9,
			want:  []byte("unit test"),
		},
		{
			name: "failed to queue",
			fields: fields{
				ctx: context.Background(),
				options: []WriterOption{
					BufferMsgs(0),
				},
			},
			args:        args{p: []byte("unit test")},
			wantErr:     true,
			wantErrToBe: ErrBufferFull,
		},
		{
			name: "failed to queue for closed server",
			fields: fields{
				ctx: canceledCtx,
				options: []WriterOption{
					BufferMsgs(0),
				},
			},
			args:        args{p: []byte("unit test")},
			wantErr:     true,
			wantErrToBe: ErrWriterClosed,
		},
		{
			name: "write to destination server fails swallowing message",
			fields: fields{
				ctx: context.Background(),
				options: []WriterOption{
					InternalLogger(zap.NewNop()),
				},
			},
			args: args{p: []byte("unit test")},
			pre: func(t *testing.T, ctrl *gomock.Controller) WriterOption {
				conn := mocknet.NewMockConn(ctrl)
				conn.EXPECT().Close().AnyTimes()
				conn.EXPECT().SetWriteDeadline(gomock.Any()).Times(1).Return(fmt.Errorf("set write deadline error"))
				conn.EXPECT().Write(gomock.Eq([]byte("unit test"))).Times(1).Return(0, fmt.Errorf("write error"))
				connect := func(context.Context, time.Duration, string, *zap.Logger) net.Conn {
					return conn
				}
				return func(w *Writer) {
					w.connect = connect
				}
			},
			wantN: 9,
		},
		{
			name: "write to destination server writes less bytes",
			fields: fields{
				ctx: context.Background(),
				options: []WriterOption{
					InternalLogger(zap.NewNop()),
				},
			},
			args: args{p: []byte("unit test")},
			pre: func(t *testing.T, ctrl *gomock.Controller) WriterOption {
				conn := mocknet.NewMockConn(ctrl)
				conn.EXPECT().Close().AnyTimes()
				conn.EXPECT().SetWriteDeadline(gomock.Any()).Times(1)
				conn.EXPECT().Write(gomock.Eq([]byte("unit test"))).Times(1).Return(8, nil)
				connect := func(context.Context, time.Duration, string, *zap.Logger) net.Conn {
					return conn
				}
				return func(w *Writer) {
					w.connect = connect
				}
			},
			wantN: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(tt.fields.ctx)
			var wg sync.WaitGroup
			defer func() {
				cancel()
				wg.Wait()
			}()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			opts := tt.fields.options
			if tt.pre != nil {
				opts = append(opts, tt.pre(t, ctrl))
			}
			w := NewWriter(ctx, tt.fields.addr, opts...)
			ch := make(chan []byte)
			if tt.fields.serverAddr != "" {
				wg.Add(1)
				startServer(ctx, &wg, tt.fields.serverAddr, ch)
			}
			time.Sleep(time.Millisecond * 10) // this might need tweaking
			gotN, err := w.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("Writer.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Writer.Write() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if gotN != tt.wantN {
				t.Errorf("Writer.Write() = %v, want %v", gotN, tt.wantN)
				return
			}

			var b []byte
			select {
			case b = <-ch:
			case <-time.After(time.Millisecond * 10): // this might need tweaking
			}
			if !reflect.DeepEqual(b, tt.want) {
				t.Errorf("Writer.Write() = %v, want %v", b, tt.want)
				return
			}
		})
	}
}
