//go:build arm && linux

package ntp

import (
	"syscall"
	"time"
)

func TimevalFromTime(t *time.Time) *syscall.Timeval {
	return &syscall.Timeval{
		Sec:  int32(t.Unix()),
		Usec: int32(t.UnixNano() / 1000 % 1000),
	}
}
