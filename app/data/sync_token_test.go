package data

import (
	"context"
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

// TestTheAppKeysTheTokenToItsDatabase: each Real store names its own database, so the sign-in it
// stores cannot be replaced by a second profile.
func TestTheAppKeysTheTokenToItsDatabase(t *testing.T) {
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
}

// TestOneTokenSourceForTheSameSettings: the runner and the Settings pane share one token source,
// so the two never renew the token at once.
func TestOneTokenSourceForTheSameSettings(t *testing.T) {
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "adhd-todo.db"))
	if err != nil {
		t.Fatal(err)
	}
	real := NewReal(db, RealOptions{Secrets: newMemorySecrets()})
	t.Cleanup(func() { _ = real.Close() })
	cfg := syncclient.Config{Server: "https://todo.example", Auth: syncclient.AuthPocketBase}
	if real.accountSource(cfg) != real.accountSource(cfg) {
		t.Error("the same settings got two token sources")
	}
	other := cfg
	other.Server = "https://other.example"
	if real.accountSource(cfg) == real.accountSource(other) {
		t.Error("another server reused the old token source")
	}
}
