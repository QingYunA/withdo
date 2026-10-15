package ghpoll

import (
	"context"
	"errors"
	"testing"
)

func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name string
		out  Output
		err  error
		want Auth
	}{
		{"signed in, new gh", Output{Stdout: []byte("github.com\n  ✓ Logged in to github.com account QingYunA (keyring)\n  - Active account: true\n  - Git operations protocol: ssh\n")},
			nil, Auth{State: AuthReady, Account: "QingYunA"}},
		{"signed in, older gh on stderr", Output{Stderr: []byte("github.com\n  ✓ Logged in to github.com as octocat (oauth_token)\n")},
			nil, Auth{State: AuthReady, Account: "octocat"}},
		{"signed in, unrecognised text", Output{Stdout: []byte("all good")}, nil, Auth{State: AuthReady}},
		{"gh is not installed", Output{}, ErrNotInstalled, Auth{State: AuthMissing}},
		{"not logged in", Output{}, &ExitError{Code: 1, Stderr: []byte("You are not logged into any GitHub hosts. To log in, run: gh auth login")}, Auth{State: AuthSignedOut}},
		{"token rejected", Output{}, &ExitError{Code: 1, Stdout: []byte("github.com\n  X Failed to log in to github.com account QingYunA (keyring)\n  - The token in keyring is invalid.")}, Auth{State: AuthSignedOut}},
		{"offline", Output{}, &ExitError{Code: 1, Stderr: []byte("error connecting to api.github.com\ncheck your internet connection or https://githubstatus.com")}, Auth{State: AuthOffline}},
		{"dns failure", Output{}, &ExitError{Code: 1, Stderr: []byte("dial tcp: lookup api.github.com: no such host")}, Auth{State: AuthOffline}},
		{"timed out", Output{}, deadlineErr(), Auth{State: AuthOffline}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeRunner{reply: func(args []string) (Output, error) { return tt.out, tt.err }}
			if got := CheckAuth(context.Background(), r); got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if len(r.calls) != 1 || r.calls[0] != "auth status --hostname github.com" {
				t.Errorf("calls = %q", r.calls)
			}
		})
	}
}

func deadlineErr() error { return errors.Join(errors.New("run gh"), context.DeadlineExceeded) }
