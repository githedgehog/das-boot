package log

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapWrapperLogger struct {
	loggers        []*zap.Logger
	sugaredLoggers []*zap.SugaredLogger
}

var _ Interface = &zapWrapperLogger{}

func NewZapWrappedLogger(loggers ...*zap.Logger) Interface {
	var ls []*zap.Logger
	var sls []*zap.SugaredLogger
	for _, logger := range loggers {
		l := logger.WithOptions(zap.AddCallerSkip(1))
		ls = append(ls, l)
		sls = append(sls, l.Sugar())
	}
	return &zapWrapperLogger{
		loggers:        ls,
		sugaredLoggers: sls,
	}
}

// DPanic implements Interface
func (l *zapWrapperLogger) DPanic(msg string, fields ...zapcore.Field) {
	for _, logger := range l.loggers {
		logger.DPanic(msg, fields...)
	}
}

// DPanicf implements Interface
func (l *zapWrapperLogger) DPanicf(template string, args ...interface{}) {
	for _, sugaredLogger := range l.sugaredLoggers {
		sugaredLogger.DPanicf(template, args...)
	}
}

// Debug implements Interface
func (l *zapWrapperLogger) Debug(msg string, fields ...zapcore.Field) {
	for _, logger := range l.loggers {
		logger.Debug(msg, fields...)
	}
}

// Debugf implements Interface
func (l *zapWrapperLogger) Debugf(template string, args ...interface{}) {
	for _, sugaredLogger := range l.sugaredLoggers {
		sugaredLogger.Debugf(template, args...)
	}
}

// Error implements Interface
func (l *zapWrapperLogger) Error(msg string, fields ...zapcore.Field) {
	for _, logger := range l.loggers {
		logger.Error(msg, fields...)
	}
}

// Errorf implements Interface
func (l *zapWrapperLogger) Errorf(template string, args ...interface{}) {
	for _, sugaredLogger := range l.sugaredLoggers {
		sugaredLogger.Errorf(template, args...)
	}
}

// Fatal implements Interface
func (l *zapWrapperLogger) Fatal(msg string, fields ...zapcore.Field) {
	for _, logger := range l.loggers {
		logger.Fatal(msg, fields...)
	}
}

// Fatalf implements Interface
func (l *zapWrapperLogger) Fatalf(template string, args ...interface{}) {
	for _, sugaredLogger := range l.sugaredLoggers {
		sugaredLogger.Fatalf(template, args...)
	}
}

// Info implements Interface
func (l *zapWrapperLogger) Info(msg string, fields ...zapcore.Field) {
	for _, logger := range l.loggers {
		logger.Info(msg, fields...)
	}
}

// Infof implements Interface
func (l *zapWrapperLogger) Infof(template string, args ...interface{}) {
	for _, sugaredLogger := range l.sugaredLoggers {
		sugaredLogger.Infof(template, args...)
	}
}

// Panic implements Interface
func (l *zapWrapperLogger) Panic(msg string, fields ...zapcore.Field) {
	for _, logger := range l.loggers {
		logger.Panic(msg, fields...)
	}
}

// Panicf implements Interface
func (l *zapWrapperLogger) Panicf(template string, args ...interface{}) {
	for _, sugaredLogger := range l.sugaredLoggers {
		sugaredLogger.Panicf(template, args...)
	}
}

// Warn implements Interface
func (l *zapWrapperLogger) Warn(msg string, fields ...zapcore.Field) {
	for _, logger := range l.loggers {
		logger.Warn(msg, fields...)
	}
}

// Warnf implements Interface
func (l *zapWrapperLogger) Warnf(template string, args ...interface{}) {
	for _, sugaredLogger := range l.sugaredLoggers {
		sugaredLogger.Warnf(template, args...)
	}
}

func (l *zapWrapperLogger) Sync() error {
	var errs []error
	for _, logger := range l.loggers {
		if err := logger.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	for _, sugaredLogger := range l.sugaredLoggers {
		if err := sugaredLogger.Sync(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		reterr := errs[0]
		if len(errs) > 1 {
			for _, err := range errs[1:] {
				reterr = fmt.Errorf("%w %w", reterr, err)
			}
		}
		return reterr
	}
	return nil
}
