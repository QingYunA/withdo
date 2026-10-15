package ghpoll

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

// AuthState says whether gh can be used.
type AuthState int

const (
	AuthUnknown   AuthState = iota // not checked yet
	AuthReady                      // gh is installed and signed in to github.com
	AuthMissing                    // there is no gh: brew install gh
	AuthSignedOut                  // gh has no usable login: gh auth login
	AuthOffline                    // GitHub could not be reached; try again later
)

// Auth is the answer of gh auth status.
type Auth struct {
	State   AuthState
	Account string // the signed-in account, when gh says it
}

// accountPattern finds the account in "Logged in to github.com account NAME (keyring)" and in the
// older "Logged in to github.com as NAME (oauth_token)".
var accountPattern = regexp.MustCompile(`Logged in to github\.com (?:account|as) (\S+)`)

// offlineMarkers are parts of gh's messages when the network, not the login, is the problem.
var offlineMarkers = []string{"error connecting", "dial tcp", "no such host", "timeout", "timed out",
	"network is unreachable", "connection refused", "i/o timeout"}

// CheckAuth runs gh auth status for github.com, the host pull request URLs name. Exit code 0 means
// signed in; any other exit means signed out, unless gh's words say the network failed.
func CheckAuth(ctx context.Context, r Runner) Auth {
	out, err := r.Run(ctx, "auth", "status", "--hostname", "github.com")
	var exit *ExitError
	switch {
	case err == nil:
		// Newer gh prints to stdout, older to stderr.
		if m := accountPattern.FindSubmatch(append(out.Stdout, out.Stderr...)); m != nil {
			return Auth{State: AuthReady, Account: string(m[1])}
		}
		return Auth{State: AuthReady}
	case errors.Is(err, ErrNotInstalled):
		return Auth{State: AuthMissing}
	case errors.As(err, &exit):
		text := strings.ToLower(exit.Output())
		for _, marker := range offlineMarkers {
			if strings.Contains(text, marker) {
				return Auth{State: AuthOffline}
			}
		}
		return Auth{State: AuthSignedOut}
	}
	return Auth{State: AuthOffline} // a timeout or a failure to start
}
