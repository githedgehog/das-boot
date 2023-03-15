package ntp

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

var (
	ErrNoRTCFound = errors.New("rtc: no device found")
	ErrOpenRTC    = errors.New("rtc: open RTC")
)

func openRTCError(err error) error {
	return fmt.Errorf("%w: %w", ErrOpenRTC, err)
}

var (
	unixIoctlGetRTCTime func(fd int) (*unix.RTCTime, error)     = unix.IoctlGetRTCTime
	unixIoctlSetRTCTime func(fd int, value *unix.RTCTime) error = unix.IoctlSetRTCTime
	osOpen              func(name string) (*os.File, error)     = os.Open
)

type RTC struct {
	f *os.File
}

func OpenRTC() (*RTC, error) {
	devs := []string{
		"/dev/rtc",
		"/dev/rtc0",
		"/dev/misc/rtc",
		"/dev/misc/rtc0",
	}

	for _, dev := range devs {
		f, err := osOpen(dev)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, openRTCError(err)
			}
			continue
		}
		return &RTC{f: f}, nil
	}

	return nil, ErrNoRTCFound
}

func (r *RTC) Close() error {
	return r.f.Close()
}

// Read reads the time from the RTC
func (r *RTC) Read() (*time.Time, error) {
	rt, err := unixIoctlGetRTCTime(int(r.f.Fd()))
	if err != nil {
		return nil, err
	}

	ret := time.Date(
		int(rt.Year)+1900,
		time.Month(rt.Mon+1),
		int(rt.Mday),
		int(rt.Hour),
		int(rt.Min),
		int(rt.Sec),
		0,
		time.UTC,
	)
	return &ret, nil
}

// Set updates the RTC with the given time
func (r *RTC) Set(t *time.Time) error {
	return unixIoctlSetRTCTime(int(r.f.Fd()), timeToRTCTime(t))
}

func timeToRTCTime(t *time.Time) *unix.RTCTime {
	return &unix.RTCTime{
		Sec:   int32(t.Second()),
		Min:   int32(t.Minute()),
		Hour:  int32(t.Hour()),
		Mday:  int32(t.Day()),
		Mon:   int32(t.Month() - 1),
		Year:  int32(t.Year() - 1900),
		Wday:  int32(0),
		Yday:  int32(0),
		Isdst: int32(0),
	}
}
