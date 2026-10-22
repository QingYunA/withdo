package syncserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// oidcClockSkew is how much clock difference an OIDC issuer may have from this server. The
// verifier thinks it is this much earlier, so a token that just expired is still accepted; the
// library already allows five minutes for a not-before claim.
const oidcClockSkew = 60 * time.Second

// OIDCAuthenticator verifies OIDC access tokens (JWTs) from one issuer for one audience and uses
// the token's subject as the user. Provider discovery and the signing keys are fetched once and
// refreshed by go-oidc; an invalid or expired token is ErrUnauthenticated, never an internal error
// that would tell a caller why it failed.
type OIDCAuthenticator struct {
	verifier *oidc.IDTokenVerifier
}

// NewOIDCAuthenticator discovers issuer's endpoints and keys. The audience is the client or API
// the tokens are issued for; giving the wrong one rejects every token.
func NewOIDCAuthenticator(ctx context.Context, issuer, audience string) (*OIDCAuthenticator, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	audience = strings.TrimSpace(audience)
	if issuer == "" {
		return nil, errors.New("syncserver: OIDC issuer is required")
	}
	if audience == "" {
		return nil, errors.New("syncserver: OIDC audience is required")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC issuer %s: %w", issuer, err)
	}
	return &OIDCAuthenticator{verifier: provider.Verifier(&oidc.Config{
		ClientID: audience,
		// Verifying against a slightly earlier clock accepts a token that expired within the
		// skew; not-before has the library's own five minute leeway.
		Now: func() time.Time { return time.Now().Add(-oidcClockSkew) },
	})}, nil
}

// Subject implements Authenticator.
func (a *OIDCAuthenticator) Subject(r *http.Request) (string, error) {
	if a == nil || a.verifier == nil {
		return "", ErrUnauthenticated
	}
	token, err := bearerToken(r)
	if err != nil {
		return "", err
	}
	idToken, err := a.verifier.Verify(r.Context(), token)
	if err != nil || idToken.Subject == "" {
		return "", ErrUnauthenticated
	}
	return idToken.Subject, nil
}

// Logged wraps an authenticator to log, once per subject and process, that it accepted a request.
// The log tells which sign-in systems are still in use, such as Zitadel before it is removed. The
// subject is an account id, not a secret.
func Logged(name string, auth Authenticator) Authenticator {
	return &loggedAuth{name: name, auth: auth}
}

type loggedAuth struct {
	name string
	auth Authenticator
	seen sync.Map
}

// Subject implements Authenticator.
func (l *loggedAuth) Subject(r *http.Request) (string, error) {
	subject, err := l.auth.Subject(r)
	if err == nil {
		if _, seen := l.seen.LoadOrStore(subject, struct{}{}); !seen {
			log.Printf("sync server: %s accepted subject %s", l.name, subject)
		}
	}
	return subject, err
}

// Chain tries its authenticators in order and returns the first subject one accepts. A static
// token is tried before OIDC, so development tokens keep working when both are configured.
type Chain []Authenticator

// Subject implements Authenticator.
func (c Chain) Subject(r *http.Request) (string, error) {
	for _, auth := range c {
		if auth == nil {
			continue
		}
		subject, err := auth.Subject(r)
		if err == nil {
			return subject, nil
		}
		if !errors.Is(err, ErrUnauthenticated) {
			return "", err
		}
	}
	return "", ErrUnauthenticated
}
