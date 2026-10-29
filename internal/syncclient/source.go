package syncclient

import (
	"context"
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// AuthPocketBase is Config.Auth for a server that signs in through its PocketBase.
const AuthPocketBase = "pocketbase"

// AccountSource is a TokenSource that signs in with an account and a password and remembers the
// sign-in in Secrets. PocketBaseSource implements it, so the app, the phone core and withdo
// treat it through one interface.
type AccountSource interface {
	TokenSource
	// SignIn signs in against the sync server and stores what Token needs later.
	SignIn(ctx context.Context, server, loginName, password string) error
	// Logout forgets the sign-in.
	Logout() error
	// Account says who is signed in, once a sign-in or a renewal has told.
	Account() (Account, bool)
}

var _ AccountSource = (*PocketBaseSource)(nil)

// SignsInWithAccount reports whether the config names an account sign-in (PocketBase) rather than
// a development token.
func (c Config) SignsInWithAccount() bool { return c.Auth == AuthPocketBase }

// SourceKey identifies the account settings a source was built from, so a caller keeps one source
// for them and replaces it when they change.
func (c Config) SourceKey() string { return c.Auth + "\n" + c.Server }

// ApplyPublic copies the sign-in a server's GET /v1/config names into the config: PocketBase when
// the server says so. A server that does not is ErrSignInUnsupported.
func (c *Config) ApplyPublic(public syncproto.ConfigResponse) error {
	if public.Auth != AuthPocketBase {
		return fmt.Errorf("the server does not offer sign-in: %w", ErrSignInUnsupported)
	}
	c.Auth = AuthPocketBase
	return nil
}

// NewAccountSource returns the PocketBase source for cfg. The token is keyed to dbPath so another
// profile cannot replace this database's sign-in.
func NewAccountSource(cfg Config, secrets Secrets, dbPath string) AccountSource {
	source := NewPocketBaseSource(cfg.Server, secrets)
	source.DBPath = dbPath
	return source
}

// CodeSignIn is the sign-up, verification and password reset of a PocketBase server (v42): an
// account is made with an email and a password, its email is verified with a 6-digit code, and a
// forgotten password is replaced with a code. PocketBaseSource implements it; a caller holding an
// AccountSource type-asserts for it.
type CodeSignIn interface {
	// SignUp makes an unverified account, mails its first code (lang is "en" or "zh-Hans", the
	// language of the mails) and returns the id of that code.
	SignUp(ctx context.Context, server, email, password, lang string) (otpID string, err error)
	// RequestCode mails a code to email and returns the id SignInWithCode and ResetPassword need.
	RequestCode(ctx context.Context, server, email string) (otpID string, err error)
	// SignInWithCode checks the code, verifies the email and stores the token. ErrInvalidCode when it
	// is wrong or expired.
	SignInWithCode(ctx context.Context, server, otpID, code string) error
	// ResetPassword sets a new password with the code and signs in.
	ResetPassword(ctx context.Context, server, otpID, code, password string) error
}

var _ CodeSignIn = (*PocketBaseSource)(nil)
