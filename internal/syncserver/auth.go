package syncserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ErrUnauthenticated says a request carries no bearer token the authenticator knows.
var ErrUnauthenticated = errors.New("unauthenticated")

// Authenticator turns a request into the subject of its user, or ErrUnauthenticated. StaticTokens
// and OIDCAuthenticator implement it; Chain combines them.
type Authenticator interface {
	Subject(r *http.Request) (string, error)
}

// StaticTokens maps a bearer token to a user subject. It is for development and tests, never for
// a deployment that faces the network.
type StaticTokens map[string]string

// Subject implements Authenticator.
func (t StaticTokens) Subject(r *http.Request) (string, error) {
	token, err := bearerToken(r)
	if err != nil {
		return "", err
	}
	subject, ok := t[token]
	if !ok || subject == "" {
		return "", ErrUnauthenticated
	}
	return subject, nil
}

// bearerToken returns the token of an Authorization: Bearer header, or ErrUnauthenticated when
// there is none. The error intentionally never includes the header's value.
func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	const prefix = "bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", ErrUnauthenticated
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", ErrUnauthenticated
	}
	return token, nil
}

// LoadTokens reads a JSON object of token → subject, the file named by cmd/adhd-sync's -tokens.
func LoadTokens(path string) (StaticTokens, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tokens %s: %w", path, err)
	}
	var tokens map[string]string
	if err := json.Unmarshal(b, &tokens); err != nil {
		return nil, fmt.Errorf("decode tokens %s: %w", path, err)
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("tokens %s is empty", path)
	}
	return StaticTokens(tokens), nil
}
