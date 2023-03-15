package ntp

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.uber.org/zap"

	"github.com/beevik/ntp"
)

var l = log.L()

var (
	ErrNoServers              = errors.New("ntp: empty server list")
	ErrNTPQueriesUnsuccessful = errors.New("ntp: all query attempts unsuccessful")
	ErrUpdateSystemClock      = errors.New("ntp: updating system clock")
	ErrHWClockSync            = errors.New("ntp: syncing system clock with hardware clock")
)

func updateSystemClockError(err error) error {
	return fmt.Errorf("%w: %w", ErrUpdateSystemClock, err)
}

var (
	syscallSettimeofday func(tv *syscall.Timeval) error = syscall.Settimeofday
)

func SyncClock(ctx context.Context, servers []string) error {
	// validate servers
	if len(servers) == 0 {
		return ErrNoServers
	}

	// fire away an NTP query
	ch := make(chan *time.Time)
	defer close(ch)
	var t *time.Time
	for i := 0; i < 3; i++ {
		t = queryAttempt(ctx, servers, ch)
		if t != nil {
			break
		}
	}
	if t == nil {
		return ErrNTPQueriesUnsuccessful
	}
	l.Info("Time from NTP server", zap.Timep("time", t))

	// now set the system clock
	tv := syscall.Timeval{
		Sec:  t.Unix(), // might be int32 on 32 bit architectures
		Usec: t.UnixNano() / 1000 % 1000,
	}
	if err := syscallSettimeofday(&tv); err != nil {
		return updateSystemClockError(err)
	}

	return nil
}

func queryAttempt(ctx context.Context, servers []string, ch chan *time.Time) *time.Time {
	attemptCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	go queryTimeFromServers(attemptCtx, servers, ch)
	select {
	case t := <-ch:
		return t
	case <-attemptCtx.Done():
		return nil
	}
}

func queryTimeFromServers(ctx context.Context, servers []string, ch chan<- *time.Time) {
	for _, server := range servers {
		go queryTimeFromServer(ctx, server, ch)
	}
}

func queryTimeFromServer(ctx context.Context, server string, ch chan<- *time.Time) {
	defer func() {
		// this recovers from the problem that the channel might be closed
		// which is expected if this is not the first responding server
		if e := recover(); e != nil {
			l.Debug("panic in queryTimeFromServer", zap.Any("recover", e))
		}
	}()

	// timeout calculation
	deadline, _ := ctx.Deadline()
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return
	}

	// execute NTP query
	r, err := ntp.QueryWithOptions(server, ntp.QueryOptions{
		Timeout: timeout,
		Version: 4,
	})
	if err != nil {
		l.Warn("querying NTP server", zap.String("server", server), zap.Error(err))
		return
	}

	// write to channel
	t := time.Now().Add(r.ClockOffset)
	ch <- &t
}
