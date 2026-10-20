package core

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// mapSecrets is a Secrets store for the tests; the phone's Keystore is not available to Go.
type mapSecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func newMapSecrets() *mapSecrets { return &mapSecrets{values: map[string]string{}} }

func (m *mapSecrets) Get(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, found := m.values[key]
	if !found {
		return "", errors.New("not found")
	}
	return value, nil
}

func (m *mapSecrets) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
	return nil
}

func (m *mapSecrets) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, found := m.values[key]; !found {
		return errors.New("not found")
	}
	delete(m.values, key)
	return nil
}

func (m *mapSecrets) has(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, found := m.values[key]
	return found
}

// TestAccountGolden pins Account's shape and says a signed-out phone shows no account.
func TestAccountGolden(t *testing.T) {
	dir := t.TempDir()
	secrets := newMapSecrets()
	if err := secrets.Set(syncclient.RefreshTokenKey, "refresh-token"); err != nil {
		t.Fatal(err)
	}
	if err := syncclient.SaveConfigAt(filepath.Join(dir, "sync.json"), syncclient.Config{
		Server: "https://sync.example.com", Issuer: "https://issuer.example", ClientID: "client",
		Account: &syncclient.Account{Name: "Ada Lovelace", Email: "ada@example.com"},
	}); err != nil {
		t.Fatalf("write sync.json: %v", err)
	}
	phone, err := open(dir, secrets, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()

	raw, err := phone.Account()
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	checkGolden(t, "account.golden.json", raw)

	// SignOut forgets the refresh token; the account in sync.json is no longer shown.
	if err := phone.SignOut(); err != nil {
		t.Fatalf("SignOut: %v", err)
	}
	if raw, err := phone.Account(); err != nil || raw != "{}" {
		t.Errorf("Account after SignOut = %q, %v; want {}", raw, err)
	}

	// A phone that never signed in has no account either.
	emptyDir := t.TempDir()
	fresh, err := open(emptyDir, newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open fresh: %v", err)
	}
	defer func() { _ = fresh.Close() }()
	if raw, err := fresh.Account(); err != nil || raw != "{}" {
		t.Errorf("Account before sign-in = %q, %v; want {}", raw, err)
	}
}

// TestSignInStateGoldens pins the two JSON answers of SignIn.
func TestSignInStateGoldens(t *testing.T) {
	checkGolden(t, "signin-linked.golden.json", signInLinkedJSON)
	checkGolden(t, "signin-choose.golden.json", signInChooseJSON)
}
