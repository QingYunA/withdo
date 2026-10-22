package syncclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// AuthPocketBase is Config.Auth for a server that signs in through its PocketBase.
const AuthPocketBase = "pocketbase"

// AccountSource is a TokenSource that signs in with an account and a password and remembers the
// sign-in in Secrets. OIDCSource (Zitadel) and PocketBaseSource implement it, so the app, the
// phone core and adhd-core pick one by the server's config and treat both alike.
type AccountSource interface {
	TokenSource
	// SignIn signs in against the sync server and stores what Token needs later.
	SignIn(ctx context.Context, server, loginName, password string) error
	// Logout forgets the sign-in.
	Logout() error
	// Account says who is signed in, once a sign-in or a renewal has told.
	Account() (Account, bool)
}

var (
	_ AccountSource = (*OIDCSource)(nil)
	_ AccountSource = (*PocketBaseSource)(nil)
)

// SignsInWithAccount reports whether the config names an account sign-in (PocketBase, or an OIDC
// issuer) rather than a development token.
func (c Config) SignsInWithAccount() bool { return c.Auth == AuthPocketBase || c.Issuer != "" }

// SourceKey identifies the account settings a source was built from, so a caller keeps one source
// for them and replaces it when they change.
func (c Config) SourceKey() string {
	return c.Auth + "\n" + c.Server + "\n" + c.Issuer + "\n" + c.ClientID + "\n" + strings.Join(c.Scopes, " ")
}

// ApplyPublic copies the sign-in a server's GET /v1/config names into the config: PocketBase when
// the server says so, else its OIDC issuer. A server that offers neither is
// ErrSignInUnsupported. The fields of the other system are cleared.
func (c *Config) ApplyPublic(public syncproto.ConfigResponse) error {
	switch {
	case public.Auth == AuthPocketBase:
		c.Auth, c.Issuer, c.ClientID, c.Scopes = AuthPocketBase, "", "", nil
	case public.Issuer != "" && public.ClientID != "":
		c.Auth = ""
		c.Issuer = strings.TrimRight(strings.TrimSpace(public.Issuer), "/")
		c.ClientID = strings.TrimSpace(public.ClientID)
		c.Scopes = public.Scopes
	default:
		return fmt.Errorf("the server does not offer sign-in: %w", ErrSignInUnsupported)
	}
	return nil
}

// NewAccountSource returns the source for cfg: PocketBase when cfg.Auth says so, else OIDC. The
// token is keyed to dbPath so another profile cannot replace this database's sign-in.
func NewAccountSource(cfg Config, secrets Secrets, dbPath string) AccountSource {
	if cfg.Auth == AuthPocketBase {
		source := NewPocketBaseSource(cfg.Server, secrets)
		source.DBPath = dbPath
		return source
	}
	source := NewOIDCSource(cfg.Issuer, cfg.ClientID, cfg.Scopes, secrets)
	source.DBPath = dbPath
	return source
}

// CodeSignIn is the emailed-code sign-in of a PocketBase server (M13 P2): sign-up and sign-in are
// one path, an email and then the 6-digit code mailed to it. PocketBaseSource implements it; a
// caller holding an AccountSource type-asserts for it.
type CodeSignIn interface {
	// RequestCode mails a code to email (creating the account if there is none) and returns the
	// id SignInWithCode needs.
	RequestCode(ctx context.Context, server, email string) (otpID string, err error)
	// SignInWithCode checks the code and stores the token. ErrInvalidCode when it is wrong or
	// expired.
	SignInWithCode(ctx context.Context, server, otpID, code string) error
	// RequestPasswordReset mails a password reset link.
	RequestPasswordReset(ctx context.Context, server, email string) error
}

var _ CodeSignIn = (*PocketBaseSource)(nil)
