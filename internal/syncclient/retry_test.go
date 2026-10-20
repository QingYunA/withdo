package syncclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"
)

// TestRetryableIsOnlyForRequestsThatMayNotHaveArrived is A3-fix 3's rule: a name, a dial, a refused,
// reset or aborted connection may be tried again; a timeout and an HTTP answer may not.
func TestRetryableIsOnlyForRequestsThatMayNotHaveArrived(t *testing.T) {
	for _, err := range []error{
		&net.DNSError{Err: "no such host", Name: "sync.example.com"},
		&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
		fmt.Errorf("POST /v1/devices: %w", syscall.ECONNRESET),
		&net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNABORTED},
	} {
		if !retryable(err) {
			t.Errorf("retryable(%v) = false, want true", err)
		}
	}
	for _, err := range []error{
		nil,
		errors.New("the server said no"),
		context.Canceled,
		context.DeadlineExceeded,
		&net.OpError{Op: "read", Net: "tcp", Err: timeoutError{}},
	} {
		if retryable(err) {
			t.Errorf("retryable(%v) = true, want false", err)
		}
	}
}

// timeoutError is a plain net.Error timeout, which may already have reached the server.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// TestRetryTransientTriesFourTimes is A3-fix 3: one, two and three seconds apart, four attempts in
// all, and no retry for anything else.
func TestRetryTransientTriesFourTimes(t *testing.T) {
	old := retrySleep
	defer func() { retrySleep = old }()
	var delays []time.Duration
	retrySleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}

	attempts := 0
	err := retryTransient(context.Background(), func() error {
		attempts++
		if attempts < 4 {
			return syscall.ECONNRESET
		}
		return nil
	})
	if err != nil || attempts != 4 {
		t.Fatalf("err = %v after %d attempts, want nil after 4", err, attempts)
	}
	if len(delays) != 3 || delays[0] != time.Second || delays[1] != 2*time.Second || delays[2] != 3*time.Second {
		t.Errorf("delays = %v, want 1s, 2s and 3s", delays)
	}

	attempts = 0
	want := errors.New("the server said no")
	delays = nil
	if got := retryTransient(context.Background(), func() error { attempts++; return want }); !errors.Is(got, want) {
		t.Errorf("err = %v, want %v", got, want)
	}
	if attempts != 1 || len(delays) != 0 {
		t.Errorf("a non-network error was tried %d times with delays %v", attempts, delays)
	}

	attempts = 0
	err = retryTransient(context.Background(), func() error { attempts++; return syscall.ECONNREFUSED })
	if !errors.Is(err, syscall.ECONNREFUSED) || attempts != 4 {
		t.Errorf("always transient: err = %v after %d attempts, want the error after 4", err, attempts)
	}
}

// TestRetryTransientStopsWithTheContext: a cancelled wait ends the retries.
func TestRetryTransientStopsWithTheContext(t *testing.T) {
	old := retrySleep
	defer func() { retrySleep = old }()
	retrySleep = func(context.Context, time.Duration) error { return context.Canceled }
	attempts := 0
	err := retryTransient(context.Background(), func() error {
		attempts++
		return syscall.ECONNRESET
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if attempts != 1 {
		t.Errorf("fn ran %d times, want 1", attempts)
	}
}
