package syncclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/paths"
)

// TokenSource returns the bearer token for one request. The Runner and Client ask it for every
// request, so a source can renew an expired token in between.
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

// Secrets is the keychain a source keeps its token in. app/ implements it over
// mygo.Secrets; tests use a map. Get returns ErrSecretNotFound when the key is absent.
type Secrets interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// RefreshTokenKey is the prefix of the Secrets keys that held the refresh token of the sign-in
// earlier versions used, which is gone. Signing out still deletes what an older version left there (see
// DeleteRefreshTokenAt), and the PocketBase token is keyed per database with the same suffix.
const RefreshTokenKey = "sync/refresh-token"

// legacyDBPath is the database the old fixed key belonged to: the platform's default database,
// which the app and the plain adhd-core used before the key became per database. ADHD_TODO_DB is
// ignored here on purpose: a database it points at never owned that token. Tests replace it.
var legacyDBPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return paths.DBPath(func(string) string { return "" }, runtime.GOOS, home)
}

// ownsLegacyToken reports whether dbPath is the database the old fixed key belonged to. Only it
// may take or delete that token; a test profile or another data folder signing in must not.
func ownsLegacyToken(dbPath string) bool {
	owner := legacyDBPath()
	if owner == "" || strings.TrimSpace(dbPath) == "" {
		return false
	}
	return cleanAbs(dbPath) == cleanAbs(owner)
}

// cleanAbs is p as one spelling: absolute, with symlinks resolved when the file exists (a temp
// folder on macOS is /var and /private/var).
func cleanAbs(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return filepath.Clean(p)
}

// RefreshTokenKeyFor is the Secrets key suffix of dbPath's token: the fixed prefix and the first 16
// hex digits of the SHA-256 of the database's absolute path. One database, one token.
func RefreshTokenKeyFor(dbPath string) string {
	path := strings.TrimSpace(dbPath)
	if path == "" {
		return RefreshTokenKey
	}
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	sum := sha256.Sum256([]byte(path))
	return RefreshTokenKey + "/" + hex.EncodeToString(sum[:8])
}

// DeleteRefreshTokenAt removes the refresh token an older version stored for dbPath. For the
// database that owned the old fixed key it removes that key too; any other database leaves it alone.
func DeleteRefreshTokenAt(secrets Secrets, dbPath string) error {
	if secrets == nil {
		return nil
	}
	if err := deleteSecret(secrets, RefreshTokenKeyFor(dbPath)); err != nil {
		return err
	}
	if ownsLegacyToken(dbPath) {
		if err := deleteSecret(secrets, RefreshTokenKey); err != nil {
			return err
		}
	}
	return nil
}

// getSecret reads one key; a missing key is ErrSecretNotFound.
func getSecret(secrets Secrets, key string) (string, error) {
	value, err := secrets.Get(key)
	if errors.Is(err, ErrSecretNotFound) {
		return "", ErrSecretNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// deleteSecret deletes one key; a missing key is not an error.
func deleteSecret(secrets Secrets, key string) error {
	if err := secrets.Delete(key); err != nil && !errors.Is(err, ErrSecretNotFound) {
		return err
	}
	return nil
}
