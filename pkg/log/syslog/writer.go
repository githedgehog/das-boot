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

package syslog

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultBufferMsgs        = 100
	DefaultConnectionTimeout = time.Second * 5
	DefaultWriteTimeout      = time.Second * 5
	DefaultSyncTimeout       = time.Second * 10
)

var (
	ErrInvalidIPAddressPort = errors.New("syslog: invalid IP address or port")
	ErrBufferFull           = errors.New("syslog: buffer full")
	ErrWriterClosed         = errors.New("syslog: writer closed")
	ErrSyncTimeout          = errors.New("syslog: sync timeout expired")
)

func writerClosedError(e any) error {
	return fmt.Errorf("%w: %v", ErrWriterClosed, e)
}

// ConnectFunc is the connect function type which is being called by a `*Writer` internally when connecting to a
// syslog server.
//
//go:generate mockgen -destination ../../../test/mock/mocknet/net_conn.go -package mocknet "net" Conn
type ConnectFunc func(ctx context.Context, connTimeout time.Duration, addr string, internalLogger *zap.Logger) net.Conn

// WriterOption is an option for the writer. Use those when creating the writer.
type WriterOption func(*Writer)

// BufferMsgs allows to change the buffer size. The buffer messages queued syslog messages - NOT message sizes
func BufferMsgs(no int) WriterOption {
	return func(w *Writer) {
		w.recvCh = make(chan []byte, no)
	}
}

// InternalLogger allows to pass a logger for the writer, to allow it to inform over writer internal errors
func InternalLogger(logger *zap.Logger) WriterOption {
	return func(w *Writer) {
		w.internalLogger = logger
	}
}

// ConnectionTimeout allows to change the default connection timeout to the server. Note that this is rather pointless
// for the default implementation which connects to a UDP server (and therefore simply binds the address).
func ConnectionTimeout(d time.Duration) WriterOption {
	return func(w *Writer) {
		w.connTimeout = d
	}
}

// WriteTimeout allows to change the default write timeout to the server. Note that this is rather pointless for the
// dfeault implementation which is UDP based. This is useful though if one passes one's own `ConnectFunction`.
func WriteTimeout(d time.Duration) WriterOption {
	return func(w *Writer) {
		w.writeTimeout = d
	}
}

// SyncTimeout allows to change the maximum amount of time that an internal call to `Sync()` can take before it will
// return with an error. This allows to gracefully sync all buffered messages, but still provide that functionality
// with a reasonable timeout.
func SyncTimeout(d time.Duration) WriterOption {
	return func(w *Writer) {
		w.syncTimeout = d
	}
}

// ConnectFunction allows to replace the default connection function which is using syslog UDP. The passed connection
// timeout is derived from ConnectionTimeout option. It is up to the implementor to decide what arguments to use
// reuse.
func ConnectFunction(t ConnectFunc) WriterOption {
	return func(w *Writer) {
		w.connect = t
	}
}

var _ zapcore.WriteSyncer = &Writer{}

// Writer is a network-based zap WriteSyncer. It buffers writes and writes them to a network destination on its own
// speed. Note that write failures or partially written messages are *NOT* being retried, and will simply be discarded.
// Similarly to an overflowing buffer which will fail to the zap API with an error, but it will not recover log messages
// that have failed to being queued.
type Writer struct {
	addr           string
	connect        ConnectFunc
	connTimeout    time.Duration
	writeTimeout   time.Duration
	syncTimeout    time.Duration
	recvCh         chan []byte
	internalLogger *zap.Logger
	// we're making use of a RWLock here even though this has nothing to do with ReadWrite
	// however, the use-case fits exactly what we need a RWLock for:
	// - multiple `Write()` calls are read locked
	// - while a `Sync()` call needs to acquire a write lock
	syncLock sync.RWMutex
}

// NewWriter returns a new network-based zap WriteSyncer for syslog messages. If a `ConnectionFunction` is missing in
// the `options`, then this is trying to attempt to write UDP-based syslog messages to `dialAddr` which can be an IP
// address or a hostname. If `dialAddr` is not specifying a port by separating it with a colon, then the default
// implementation of the connect function will append `:514` to `dialAddr`.
// This function cannot fail, and all retry mechanisms are internally to the writer. For example, temporary write
// failures will try to reestablish a new connection to the same `dialAddr`. See the documentation for `Writer` on
// message delivery guarantees (which are essentially not in place on purpose).
func NewWriter(ctx context.Context, dialAddr string, options ...WriterOption) *Writer {
	ret := &Writer{
		addr:         dialAddr,
		connect:      defaultUDPConnect,
		recvCh:       make(chan []byte, DefaultBufferMsgs),
		connTimeout:  DefaultConnectionTimeout,
		writeTimeout: DefaultWriteTimeout,
		syncTimeout:  DefaultSyncTimeout,
	}

	// apply options
	for _, opt := range options {
		opt(ret)
	}

	// start the processor
	go ret.loop(ctx)

	return ret
}

// Write implements zapcore.WriteSyncer
func (w *Writer) Write(p []byte) (n int, err error) {
	w.syncLock.RLock()
	defer w.syncLock.RUnlock()
	defer func() {
		if e := recover(); e != nil {
			err = writerClosedError(e)
		}
	}()

	// we need to copy out the message
	// as the same pointer is being reused by zap
	// this is the only way to really preserve the message
	// because we are sending the pointers over a channel
	send := make([]byte, len(p))
	copy(send, p)

	select {
	case w.recvCh <- send:
		return len(send), nil
	default:
		return 0, ErrBufferFull
	}
}

const syncPollTimeout = time.Millisecond * 10

// Sync implements zapcore.WriteSyncer
func (w *Writer) Sync() error {
	w.syncLock.Lock()
	defer w.syncLock.Unlock()

	// short circuit if really nothing needs to happen
	if len(w.recvCh) == 0 {
		return nil
	}

	// otherwise fire a sync timeout
	// and regularly poll for updates
	ch := make(chan struct{})
	go func() {
		<-time.After(w.syncTimeout)
		close(ch)
	}()

	for {
		select {
		case <-ch:
			return ErrSyncTimeout
		case <-time.After(syncPollTimeout):
			if len(w.recvCh) == 0 {
				return nil
			}
		}
	}
}

func (w *Writer) loop(ctx context.Context) {
	defer func() {
		w.syncLock.Lock()
		close(w.recvCh)
		w.syncLock.Unlock()
	}()

	for {
		var conn net.Conn = nil
		defer func(c *net.Conn) {
			conn := *c
			if conn != nil {
				conn.Close()
			}
		}(&conn)

		// enter the connect loop
	connectLoop:
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// try to connect
				beforeConnect := time.Now()
				conn = w.connect(ctx, w.connTimeout, w.addr, w.internalLogger)
				if conn != nil {
					break connectLoop
				}

				// if the connect call returned faster (probably unrelated to a timeout)
				// then we wait for the rest of the time before we try again
				connDur := time.Since(beforeConnect)
				sleepDur := w.connTimeout - connDur
				time.Sleep(sleepDur)
			}
		}

		// once connected, enter the write loop
	writeLoop:
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-w.recvCh:
				if err := conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)); err != nil && w.internalLogger != nil {
					w.internalLogger.Debug("failed to set write deadline for write to syslog server", zap.Error(err))
				}
				n, err := conn.Write(msg)
				if err != nil {
					if w.internalLogger != nil {
						w.internalLogger.Error("writing to syslog server", zap.Error(err))
					}
					// we're treating any write errors
					// as reconnection events
					conn.Close()
					conn = nil
					break writeLoop
				}
				if n != len(msg) && w.internalLogger != nil {
					w.internalLogger.Warn("len(written) != len(msg)", zap.Int("msgLen", len(msg)), zap.Int("written", n))
				}
			}
		}
	}
}

func defaultUDPConnect(ctx context.Context, connTimeout time.Duration, addr string, internalLogger *zap.Logger) net.Conn {
	// check the address
	// if it doesn't has a port, we'll add the default UDP port
	if addr == "" {
		return nil
	}
	dialAddr := addr
	if !strings.Contains(addr, ":") {
		dialAddr = addr + ":514"
	}
	d := &net.Dialer{}
	subctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()
	conn, err := d.DialContext(subctx, "udp", dialAddr)
	if err != nil && internalLogger != nil {
		internalLogger.Error("connecting to syslog server", zap.Error(err))
	}
	return conn
}
