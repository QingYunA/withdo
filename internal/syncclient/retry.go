package syncclient

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"
)

// retryAttempts is how many times a request that may not have reached the server is tried: the
// first attempt and three retries, one, two and three seconds later. Six seconds cover a phone
// coming back to the front after the browser, when Android gives a background app its network back.
const retryAttempts = 4

// retrySleep waits between attempts. Tests replace it so they do not spend three seconds asleep.
var retrySleep = func(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryTransient runs fn, retrying only errors where the request may not have reached the server:
// a name that did not resolve, a dial that failed, a connection refused, reset or aborted. A
// timeout is not retried, because the server may have handled the request already. fn must be safe
// to run again when its request did not arrive.
func retryTransient(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 0; attempt < retryAttempts; attempt++ {
		if attempt > 0 {
			if waitErr := retrySleep(ctx, time.Duration(attempt)*time.Second); waitErr != nil {
				return waitErr
			}
		}
		err = fn()
		if err == nil || !retryable(err) {
			return err
		}
	}
	return err
}

// retryable says whether err is one of the failures retryTransient retries. A context error is not
// retryable: the caller asked to stop.
func retryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return true
	}
	var op *net.OpError
	if errors.As(err, &op) && op.Op == "dial" {
		return true
	}
	for _, target := range []error{
		syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
