package log

import "go.uber.org/zap/zapcore"

// Interface wraps the most important calls to a zap logger.
// It provides the standard structured logging functions, but also
// the formatted message functions which are very useful once in a
// while. However, they are cumbersome to access through the
// SugaredLogger. This also allows to unit test log statments very
// well with a go mock.
type Interface interface {
	Debug(msg string, fields ...zapcore.Field)
	Debugf(template string, args ...interface{})
	Info(msg string, fields ...zapcore.Field)
	Infof(template string, args ...interface{})
	Warn(msg string, fields ...zapcore.Field)
	Warnf(template string, args ...interface{})
	Error(msg string, fields ...zapcore.Field)
	Errorf(template string, args ...interface{})
	DPanic(msg string, fields ...zapcore.Field)
	DPanicf(template string, args ...interface{})
	Panic(msg string, fields ...zapcore.Field)
	Panicf(template string, args ...interface{})
	Fatal(msg string, fields ...zapcore.Field)
	Fatalf(template string, args ...interface{})
	Sync() error
}
