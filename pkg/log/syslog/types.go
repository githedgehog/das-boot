package syslog

import (
	"fmt"
	"strings"
)

// Priority maps to the syslog priority levels
type Priority int

const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.

	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

const (
	// Facility.

	// From /usr/include/sys/syslog.h.
	// These are the same up to LOG_FTP on Linux, BSD, and OS X.

	LOG_KERN Priority = iota << 3
	LOG_USER
	LOG_MAIL
	LOG_DAEMON
	LOG_AUTH
	LOG_SYSLOG
	LOG_LPR
	LOG_NEWS
	LOG_UUCP
	LOG_CRON
	LOG_AUTHPRIV
	LOG_FTP
	_ // unused
	_ // unused
	_ // unused
	_ // unused
	LOG_LOCAL0
	LOG_LOCAL1
	LOG_LOCAL2
	LOG_LOCAL3
	LOG_LOCAL4
	LOG_LOCAL5
	LOG_LOCAL6
	LOG_LOCAL7
)

var (
	facilityMap = map[string]Priority{
		"KERN":     LOG_KERN,
		"USER":     LOG_USER,
		"MAIL":     LOG_MAIL,
		"DAEMON":   LOG_DAEMON,
		"AUTH":     LOG_AUTH,
		"SYSLOG":   LOG_SYSLOG,
		"LPR":      LOG_LPR,
		"NEWS":     LOG_NEWS,
		"UUCP":     LOG_UUCP,
		"CRON":     LOG_CRON,
		"AUTHPRIV": LOG_AUTHPRIV,
		"FTP":      LOG_FTP,
		"LOCAL0":   LOG_LOCAL0,
		"LOCAL1":   LOG_LOCAL1,
		"LOCAL2":   LOG_LOCAL2,
		"LOCAL3":   LOG_LOCAL3,
		"LOCAL4":   LOG_LOCAL4,
		"LOCAL5":   LOG_LOCAL5,
		"LOCAL6":   LOG_LOCAL6,
		"LOCAL7":   LOG_LOCAL7,
	}
)

// FacilityPriority converts a facility string into
// an appropriate priority level or returns an error
func FacilityPriority(facility string) (Priority, error) {
	facility = strings.ToUpper(facility)
	if prio, ok := facilityMap[facility]; ok {
		return prio, nil
	}
	return 0, fmt.Errorf("invalid syslog facility: %s", facility)
}

func (l Priority) String() string {
	switch l {
	// priorities (NOTE: EMERG missing because of overlapping KERN)
	case LOG_ALERT:
		return "ALERT"
	case LOG_CRIT:
		return "CRIT"
	case LOG_ERR:
		return "ERR"
	case LOG_WARNING:
		return "WARNING"
	case LOG_NOTICE:
		return "NOTICE"
	case LOG_INFO:
		return "INFO"
	case LOG_DEBUG:
		return "DEBUG"
	// facilities
	case LOG_KERN:
		return "KERN"
	case LOG_USER:
		return "USER"
	case LOG_MAIL:
		return "MAIL"
	case LOG_DAEMON:
		return "DAEMON"
	case LOG_AUTH:
		return "AUTH"
	case LOG_SYSLOG:
		return "SYSLOG"
	case LOG_LPR:
		return "LPR"
	case LOG_NEWS:
		return "NEWS"
	case LOG_UUCP:
		return "UUCP"
	case LOG_CRON:
		return "CRON"
	case LOG_AUTHPRIV:
		return "AUTHPRIV"
	case LOG_FTP:
		return "FTP"
	case LOG_LOCAL0:
		return "LOCAL0"
	case LOG_LOCAL1:
		return "LOCAL1"
	case LOG_LOCAL2:
		return "LOCAL2"
	case LOG_LOCAL3:
		return "LOCAL3"
	case LOG_LOCAL4:
		return "LOCAL4"
	case LOG_LOCAL5:
		return "LOCAL5"
	case LOG_LOCAL6:
		return "LOCAL6"
	case LOG_LOCAL7:
		return "LOCAL7"
	default:
		return "UNKNOWN"
	}
}

// Set sets the level for the flag.Value interface.
func (l *Priority) Set(s string) error {
	f, err := FacilityPriority(s)
	if err != nil {
		return err
	}
	*l = f
	return nil
}

// Get gets the level for the flag.Getter interface.
func (l *Priority) Get() interface{} {
	return *l
}
