package stage

import (
	"context"
	"fmt"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/log/syslog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogSettings struct {
	Level          zapcore.Level   `json:"level,omitempty"`
	Development    bool            `json:"development,omitempty"`
	Format         string          `json:"format,omitempty"`
	SyslogServers  []string        `json:"syslog_servers,omitempty"`
	SyslogFacility syslog.Priority `json:"syslog_facility,omitempty"`
}

func InitializeGlobalLogger(ctx context.Context, settings *LogSettings) error {
	// initialize zap serial logger
	var logger log.Interface
	serialLogger, err := log.NewSerialConsole(settings.Level, settings.Format, settings.Development)
	if err != nil {
		return fmt.Errorf("failed to initialize serial logger: %w", err)
	}
	serialLogger.Debug("Initialized serial logger from command-line settings", zap.Bool("logDevelopment", settings.Development), zap.String("logLevel", settings.Level.String()), zap.String("logFormat", settings.Format))
	logger = log.NewZapWrappedLogger(serialLogger)

	// initialize zap syslog logger
	if len(settings.SyslogServers) > 0 {
		loggers := []*zap.Logger{serialLogger}
		for _, syslogServer := range settings.SyslogServers {
			syslogLogger, err := log.NewSyslog(ctx, settings.Level, settings.Development, settings.SyslogFacility, syslogServer, syslog.InternalLogger(serialLogger))
			if err != nil {
				return fmt.Errorf("failed to initialize syslog logger for '%s': %w", syslogServer, err)
			}
			serialLogger.Debug("Initialized syslog logger from command-line settings", zap.String("syslogServer", syslogServer), zap.String("syslogFacility", settings.SyslogFacility.String()))
			loggers = append(loggers, syslogLogger)
		}

		// now create a "tee" logger for both serial and syslog destinations
		logger = log.NewZapWrappedLogger(loggers...)
	}

	log.ReplaceGlobals(logger)
	return nil
}
