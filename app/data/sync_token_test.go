package data

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/paths"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// memorySecrets is a syncclient.Secrets for the app's token-key test.
type memorySecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func newMemorySecrets() *memorySecrets { return &memorySecrets{values: map[string]string{}} }

func (s *memorySecrets) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found := s.values[key]
	if !found {
		return "", syncclient.ErrSecretNotFound
	}
	return value, nil
}

func (s *memorySecrets) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *memorySecrets) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}

// TestTheAppKeysTheRefreshTokenToItsDatabase is A3-fix 8 in the app: the Real store asks for its
// own database's token, and an existing sign-in under the old fixed key is moved, not lost. The
// app's database is the default one (under HOME), the only one that owned the old key.
func TestTheAppKeysTheRefreshTokenToItsDatabase(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	first := paths.DBPath(func(string) string { return "" }, runtime.GOOS, home)
	if err := os.MkdirAll(filepath.Dir(first), 0o755); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(t.TempDir(), "test-profile.db")
	db, err := store.Open(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	real := NewReal(db, RealOptions{SyncConfigPath: syncclient.ConfigPath(first), Secrets: newMemorySecrets()})
	t.Cleanup(func() { _ = real.Close() })
	otherDB, err := store.Open(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	other := NewReal(otherDB, RealOptions{SyncConfigPath: syncclient.ConfigPath(second), Secrets: real.opts.Secrets})
	t.Cleanup(func() { _ = other.Close() })

	resolved, err := filepath.EvalSymlinks(first)
	if err != nil {
		t.Fatal(err)
	}
	if got := real.databasePath(); got != first && got != resolved {
		t.Fatalf("databasePath = %q, want %q or %q", got, first, resolved)
	}
	if real.databasePath() == other.databasePath() {
		t.Fatalf("two databases share the path %q", real.databasePath())
	}

	secrets := real.opts.Secrets
	if err := secrets.Set(syncclient.RefreshTokenKey, "old-token"); err != nil {
		t.Fatal(err)
	}
	token, err := syncclient.LoadRefreshToken(secrets, real.databasePath())
	if err != nil || token != "old-token" {
		t.Fatalf("LoadRefreshToken = %q, %v; want the old sign-in", token, err)
	}
	if _, err := secrets.Get(syncclient.RefreshTokenKeyFor(real.databasePath())); err != nil {
		t.Errorf("the app's database key has no token: %v", err)
	}
	if _, err := syncclient.LoadRefreshToken(secrets, other.databasePath()); !errors.Is(err, syncclient.ErrSecretNotFound) {
		t.Errorf("the second database inherited the sign-in: %v", err)
	}
}

// TestOneTokenSourceForTheSameSettings: the runner and the Settings pane share one OIDC source,
// because the issuer rotates refresh tokens and two sources refreshing at once sign each other out.
func TestOneTokenSourceForTheSameSettings(t *testing.T) {
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "adhd-todo.db"))
	if err != nil {
		t.Fatal(err)
	}
	real := NewReal(db, RealOptions{Secrets: newMemorySecrets()})
	t.Cleanup(func() { _ = real.Close() })
	scopes := []string{"aud"}
	if real.oidcSource("https://issuer.example", "client", scopes) != real.oidcSource("https://issuer.example", "client", scopes) {
		t.Error("the same settings got two token sources")
	}
	if real.oidcSource("https://issuer.example", "client", scopes) == real.oidcSource("https://other.example", "client", scopes) {
		t.Error("another issuer reused the old token source")
	}
}
