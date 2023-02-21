package log

import (
	"sync"

	"go.uber.org/zap"
)

var logger = zap.Must(zap.NewProduction())
var loggerLock sync.RWMutex

type ReinitializeLoggerFunc func()

var registeredLoggers map[string]ReinitializeLoggerFunc
var registeredLoggersLock sync.RWMutex

func L() *zap.Logger {
	loggerLock.RLock()
	defer loggerLock.RUnlock()
	return logger
}

func RegisterLogger(name string, f ReinitializeLoggerFunc) {
	registeredLoggersLock.Lock()
	defer registeredLoggersLock.Unlock()
	registeredLoggers[name] = f
}
