package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapWrapperLogger struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
}

var _ Interface = &zapWrapperLogger{}

func NewZapWrappedLogger(logger *zap.Logger) Interface {
	return &zapWrapperLogger{
		logger:        logger,
		sugaredLogger: logger.Sugar(),
	}
}

// DPanic implements Interface
func (l *zapWrapperLogger) DPanic(msg string, fields ...zapcore.Field) {
	l.logger.DPanic(msg, fields...)
}

// DPanicf implements Interface
func (l *zapWrapperLogger) DPanicf(template string, args ...interface{}) {
	l.sugaredLogger.DPanicf(template, args...)
}

// Debug implements Interface
func (l *zapWrapperLogger) Debug(msg string, fields ...zapcore.Field) {
	l.logger.Debug(msg, fields...)
}

// Debugf implements Interface
func (l *zapWrapperLogger) Debugf(template string, args ...interface{}) {
	l.sugaredLogger.Debugf(template, args...)
}

// Error implements Interface
func (l *zapWrapperLogger) Error(msg string, fields ...zapcore.Field) {
	l.logger.Error(msg, fields...)
}

// Errorf implements Interface
func (l *zapWrapperLogger) Errorf(template string, args ...interface{}) {
	l.sugaredLogger.Errorf(template, args...)
}

// Fatal implements Interface
func (l *zapWrapperLogger) Fatal(msg string, fields ...zapcore.Field) {
	l.logger.Fatal(msg, fields...)
}

// Fatalf implements Interface
func (l *zapWrapperLogger) Fatalf(template string, args ...interface{}) {
	l.sugaredLogger.Fatalf(template, args...)
}

// Info implements Interface
func (l *zapWrapperLogger) Info(msg string, fields ...zapcore.Field) {
	l.logger.Info(msg, fields...)
}

// Infof implements Interface
func (l *zapWrapperLogger) Infof(template string, args ...interface{}) {
	l.sugaredLogger.Infof(template, args...)
}

// Panic implements Interface
func (l *zapWrapperLogger) Panic(msg string, fields ...zapcore.Field) {
	l.logger.Panic(msg, fields...)
}

// Panicf implements Interface
func (l *zapWrapperLogger) Panicf(template string, args ...interface{}) {
	l.sugaredLogger.Panicf(template, args...)
}

// Warn implements Interface
func (l *zapWrapperLogger) Warn(msg string, fields ...zapcore.Field) {
	l.logger.Warn(msg, fields...)
}

// Warnf implements Interface
func (l *zapWrapperLogger) Warnf(template string, args ...interface{}) {
	l.sugaredLogger.Warnf(template, args...)
}
