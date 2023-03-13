package syslog

import (
	"context"
	"errors"
	"fmt"
	"net"
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

type WriterOption func(*UDPWriter)

func BufferMsgs(no int) WriterOption {
	return func(w *UDPWriter) {
		w.recvCh = make(chan []byte, no)
	}
}

func InternalLogger(logger *zap.Logger) WriterOption {
	return func(w *UDPWriter) {
		w.internalLogger = logger
	}
}

func ConnectionTimeout(d time.Duration) WriterOption {
	return func(w *UDPWriter) {
		w.connTimeout = d
	}
}

func WriteTimeout(d time.Duration) WriterOption {
	return func(w *UDPWriter) {
		w.writeTimeout = d
	}
}

var _ zapcore.WriteSyncer = &UDPWriter{}

type UDPWriter struct {
	addr           string
	connTimeout    time.Duration
	writeTimeout   time.Duration
	recvCh         chan []byte
	internalLogger *zap.Logger
}

func NewUDPWriter(ctx context.Context, dialAddr string, options ...WriterOption) (*UDPWriter, error) {
	ret := &UDPWriter{
		addr:         dialAddr,
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

	return ret, nil
}

// Write implements zapcore.WriteSyncer
func (w *UDPWriter) Write(p []byte) (n int, err error) {
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
func (*UDPWriter) Sync() error {
	// TODO: should get a draining logic which is what this function is good for
	return nil
}

func (w *UDPWriter) loop(ctx context.Context) {
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
				conn = w.connect(ctx)
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

func (w *UDPWriter) connect(ctx context.Context) net.Conn {
	d := &net.Dialer{}
	subctx, cancel := context.WithTimeout(ctx, w.connTimeout)
	defer cancel()
	conn, err := d.DialContext(subctx, "udp", w.addr)
	if err != nil && w.internalLogger != nil {
		w.internalLogger.Error("connecting to syslog server", zap.Error(err))
	}
	return conn
}
