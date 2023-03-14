package syslog

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultBufferMsgs        = 100
	DefaultConnectionTimeout = time.Second * 5
	DefaultWriteTimeout      = time.Second * 5
)

var (
	ErrInvalidIPAddressPort = errors.New("syslog: invalid IP address or port")
	ErrBufferFull           = errors.New("syslog: buffer full")
	ErrWriterClosed         = errors.New("syslog: writer closed")
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
	recvCh         chan []byte
	internalLogger *zap.Logger
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
	defer func() {
		if e := recover(); e != nil {
			err = writerClosedError(e)
		}
	}()
	select {
	case w.recvCh <- p:
		return len(p), nil
	default:
		return 0, ErrBufferFull
	}
}

// Sync implements zapcore.WriteSyncer
func (*Writer) Sync() error {
	// TODO: should get a draining logic which is what this function is good for
	return nil
}

func (w *Writer) loop(ctx context.Context) {
	defer close(w.recvCh)

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
