package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"syscall"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// timeoutError is a net.Error that is not a DNS or dial error, such as an HTTP client timeout.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// TestSyncErrorMapsNetworkFailures is A3-fix 2: a failure below HTTP becomes the one sentence the
// Sync screen shows, while a non-network failure keeps the generic sentence.
func TestSyncErrorMapsNetworkFailures(t *testing.T) {
	const unreachable = "could not reach the server; check the connection"
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"dns", &net.DNSError{Err: "no such host", Name: "sync.example.com"}, unreachable},
		{"dial refused", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, unreachable},
		{"wrapped reset", fmt.Errorf("GET /v1/pull: %w", syscall.ECONNRESET), unreachable},
		{"aborted", &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNABORTED}, unreachable},
		{"timeout", &net.OpError{Op: "read", Net: "tcp", Err: timeoutError{}}, unreachable},
		{"context deadline", context.DeadlineExceeded, unreachable},
		{"cancelled", context.Canceled, "the sync was cancelled"},
		{"ordinary", errors.New("a row did not fit"), "could not sync; check the connection and try again"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := syncError(tc.err); got == nil || got.Error() != tc.want {
				t.Errorf("syncError(%v) = %v, want %q", tc.err, got, tc.want)
			}
		})
	}
}

// TestSyncErrorLogsTheRawCause is A3-fix 2's other half: the cause reaches logcat, and a token in
// the request is never part of the error net/http returns.
func TestSyncErrorLogsTheRawCause(t *testing.T) {
	var logged bytes.Buffer
	oldWriter, oldFlags := log.Writer(), log.Flags()
	log.SetOutput(&logged)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	}()

	raw := fmt.Errorf("GET https://sync.example.com/v1/pull: %w", syscall.ECONNRESET)
	_ = syncError(raw)
	text := logged.String()
	if !strings.Contains(text, "connection reset by peer") {
		t.Errorf("the raw error was not logged: %q", text)
	}
	if strings.Contains(text, "Bearer") {
		t.Errorf("the log carries a token: %q", text)
	}
}

// TestSyncErrorMapsSignInFailures is the five answers /v1/sign-in gives the app: each sentinel
// becomes the English sentence Kotlin looks up in strings.xml.
func TestSyncErrorMapsSignInFailures(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{syncclient.ErrInvalidCredentials, "wrong account or password"},
		{syncclient.ErrTooManyAttempts, "too many tries; wait a few minutes"},
		{syncclient.ErrSecondFactor, "this account uses two-step verification, which in-app sign-in does not support yet"},
		{syncclient.ErrAccountLocked, "this account is deactivated; ask the admin"},
		{syncclient.ErrSignInUnsupported, "this server does not support signing in from the app"},
	}
	for _, tc := range cases {
		if got := syncError(tc.err); got == nil || got.Error() != tc.want {
			t.Errorf("syncError(%v) = %v, want %q", tc.err, got, tc.want)
		}
	}
}

// TestLinkErrorKeepsTheNetworkSentence: Link's mapping falls through to syncError for the network.
func TestLinkErrorKeepsTheNetworkSentence(t *testing.T) {
	err := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	if got := linkError(err); got == nil || got.Error() != "could not reach the server; check the connection" {
		t.Errorf("linkError = %v", got)
	}
}
