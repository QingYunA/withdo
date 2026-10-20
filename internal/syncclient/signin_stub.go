// TEMPORARY: replaced by the real SignIn when dsh/inapp-signin is merged.
package syncclient

import (
	"context"
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("syncclient: wrong account or password")
	ErrAccountLocked      = errors.New("syncclient: the account is locked or deactivated")
	ErrSecondFactor       = errors.New("syncclient: the account uses a second factor, which in-app sign-in does not support")
	ErrTooManyAttempts    = errors.New("syncclient: too many sign-in attempts; wait a few minutes")
	ErrSignInUnsupported  = errors.New("syncclient: the server does not support signing in from the app")
)

// SignIn signs in with an account and password through the sync server's /v1/sign-in.
func (s *OIDCSource) SignIn(ctx context.Context, server, loginName, password string) error {
	return ErrSignInUnsupported
}
