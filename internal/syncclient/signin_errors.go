package syncclient

import "errors"

// Errors a sign-in returns for the answers the client UI shows as text. Every other failure is an
// ordinary error with the request that failed.
var (
	// ErrInvalidCredentials says the login name or password was refused.
	ErrInvalidCredentials = errors.New("syncclient: invalid login name or password")
	// ErrAccountLocked says the account is locked or disabled.
	ErrAccountLocked = errors.New("syncclient: the account is locked or disabled")
	// ErrTooManyAttempts says the server rate-limited the sign-in.
	ErrTooManyAttempts = errors.New("syncclient: too many sign-in attempts")
	// ErrInvalidCode says the emailed sign-in code was wrong, used, expired, or the email malformed.
	ErrInvalidCode = errors.New("syncclient: invalid or expired sign-in code")
	// ErrSignInUnsupported says the sync server does not offer an in-app sign-in (its /v1/config
	// does not name "pocketbase").
	ErrSignInUnsupported = errors.New("syncclient: the sync server does not support in-app sign-in")
)
