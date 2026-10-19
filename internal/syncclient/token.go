package syncclient

import (
	"context"
	"errors"
)

// TokenSource returns the bearer token for one request. The Runner and Client ask it for every
// request, so an OIDC source can refresh an expired access token in between.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// StaticToken is a fixed bearer token, the development setup of S3.
type StaticToken string

// Token implements TokenSource.
func (t StaticToken) Token(context.Context) (string, error) { return string(t), nil }

// ErrSignedOut says there is no usable sign-in: no refresh token, or the issuer refused it. The
// Runner treats it like a 401 and stops in the signed out state until it is started again.
var ErrSignedOut = errors.New("sync: signed out; sign in again")

// IsSignedOut reports whether err means the user must sign in again: a refused refresh, or the
// Runner already knowing there is no token. It is treated like a 401.
func IsSignedOut(err error) bool { return errors.Is(err, ErrSignedOut) }

// ErrSecretNotFound says the Secrets store has no value under a key.
var ErrSecretNotFound = errors.New("sync: secret not found")

// Secrets is the keychain an OIDC source keeps its refresh token in. app/ implements it over
// mygo.Secrets; tests use a map. Get returns ErrSecretNotFound when the key is absent.
type Secrets interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// RefreshTokenKey is the Secrets key of the OIDC refresh token. The access token stays in memory.
const RefreshTokenKey = "sync/refresh-token"

// DeleteRefreshToken removes the stored refresh token. A missing key is not an error; a nil store
// has nothing to delete.
func DeleteRefreshToken(secrets Secrets) error {
	if secrets == nil {
		return nil
	}
	if err := secrets.Delete(RefreshTokenKey); err != nil && !errors.Is(err, ErrSecretNotFound) {
		return err
	}
	return nil
}
