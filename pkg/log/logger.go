package log

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.githedgehog.com/dasboot/pkg/log/syslog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger Interface = NewZapWrappedLogger(zap.NewNop())
var loggerLock sync.RWMutex

type ReinitializeLoggerFunc func()

var registeredLoggers map[string]ReinitializeLoggerFunc
var registeredLoggersLock sync.RWMutex

// L returns the global logger. It can either be used directly in other
// packages, or they can create a sublogger from this and register a
// function for reinitialization on global logger change with `RegisterLogger`.
func L() Interface {
	loggerLock.RLock()
	l := logger
	loggerLock.RUnlock()
	return l
}

// RegisterLogger allows sub-loggers to register a callback function
// to get notified when the global logger changed. This allows sub-loggers
// to reinitialize their sub-logger.
func RegisterLogger(name string, f ReinitializeLoggerFunc) {
	registeredLoggersLock.Lock()
	registeredLoggers[name] = f
	registeredLoggersLock.Unlock()
}

// ReplaceGlobals allows to reinitialize the global logger exactly like
// the zap.ReplaceGlobals. It also returns a function to restore the
// previous logger. However, it takes the interface of this pacakge as
// an argument. Use `NewZapWrappedLogger` to create a zap logger which
// fulfills this interface.
func ReplaceGlobals(newLogger Interface) func() {
	// replace the new logger
	loggerLock.Lock()
	prevLogger := logger
	logger = newLogger
	loggerLock.Unlock()

	// call all registered loggers callback to let them do their piece
	registeredLoggersLock.RLock()
	for _, reglogReinitFunc := range registeredLoggers {
		reglogReinitFunc()
		zap.L()
	}
	registeredLoggersLock.RUnlock()

	// return like the zap function with a restore function for the previous logger
	// although I doubt we will use this
	return func() { ReplaceGlobals(prevLogger) }
}

func NewSerialConsole(level zapcore.Level, format string, development bool) (*zap.Logger, error) {
	// we enable callers, stacktraces and functions in development mode only
	disableCaller := true
	disableStacktrace := true
	functionKey := zapcore.OmitKey
	if development {
		disableCaller = false
		disableStacktrace = false
		functionKey = "F"
	}

	// these settings will be dependent on the format
	encoding := "console"
	encodeLevel := zapcore.CapitalColorLevelEncoder
	keyConvert := func(s string) string { return s }
	if format == "json" {
		encoding = "json"
		encodeLevel = zapcore.LowercaseLevelEncoder
		keyConvert = func(s string) string { return strings.ToLower(s) }
	}

	cfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       development,
		DisableCaller:     disableCaller,
		DisableStacktrace: disableStacktrace,
		Encoding:          encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        keyConvert("T"),
			LevelKey:       keyConvert("L"),
			NameKey:        keyConvert("N"),
			CallerKey:      keyConvert("C"),
			FunctionKey:    keyConvert(functionKey),
			MessageKey:     keyConvert("M"),
			StacktraceKey:  keyConvert("S"),
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    encodeLevel,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	return cfg.Build()
}

func NewSyslog(ctx context.Context, level zapcore.Level, development bool, facility syslog.Priority, server string, writerOptions ...syslog.WriterOption) (*zap.Logger, error) {
	// we enable callers, stacktraces and functions in development mode only
	callerKey := zapcore.OmitKey
	stacktraceKey := zapcore.OmitKey
	functionKey := zapcore.OmitKey
	// stacktraces aren't very pleasant in production - neither on the console nor in syslog
	// so we essentially disable them except for panics and above
	stackLevel := zapcore.PanicLevel
	if development {
		stackLevel = zapcore.WarnLevel
		callerKey = "c"
		stacktraceKey = "s"
		functionKey = "f"
	}

	// hostname will be unknown if we cannot resolve our hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// PID will simply be the PID of the running process
	pid := os.Getpid()

	// app will be set to the name of the calling binary
	// NOTE: as this is not resolving symlinks, this is perfect to do justice
	// even for busybox-style executables
	app := filepath.Base(os.Args[0])

	enc := syslog.NewSyslogEncoder(syslog.SyslogEncoderConfig{
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "t",
			LevelKey:       "l",
			NameKey:        "n",
			CallerKey:      callerKey,
			FunctionKey:    functionKey,
			MessageKey:     "m",
			StacktraceKey:  stacktraceKey,
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		Framing:  syslog.DefaultFraming,
		Facility: facility,
		Hostname: hostname,
		PID:      pid,
		App:      app,
	})

	sink := syslog.NewWriter(ctx, server, writerOptions...)
	out := zapcore.Lock(sink)

	logger := zap.New(
		zapcore.NewCore(
			enc,
			out,
			zap.NewAtomicLevelAt(level),
		),
		zap.ErrorOutput(out),
		zap.WithCaller(development),
		zap.AddStacktrace(stackLevel),
	)

	return logger, nil
}
