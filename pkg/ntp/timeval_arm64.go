//go:build arm64 && linux

package ntp

import (
	"syscall"
	"time"
)

func TimevalFromTime(t *time.Time) *syscall.Timeval {
	return &syscall.Timeval{
		Sec:  t.Unix(),
		Usec: t.UnixNano() / 1000 % 1000,
	}
}
