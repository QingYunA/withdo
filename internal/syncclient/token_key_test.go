package syncclient_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// TestRefreshTokenKeyIsPerDatabase is A3-fix 8: the key names one database, not every database.
func TestRefreshTokenKeyIsPerDatabase(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "one", "adhd-todo.db")
	second := filepath.Join(dir, "two", "adhd-todo.db")
	firstKey := syncclient.RefreshTokenKeyFor(first)
	secondKey := syncclient.RefreshTokenKeyFor(second)
	if firstKey == secondKey {
		t.Fatalf("both databases use the key %q", firstKey)
	}
	const prefix = "sync/refresh-token/"
	if !strings.HasPrefix(firstKey, prefix) || len(firstKey) != len(prefix)+16 {
		t.Errorf("key = %q, want %q plus 16 hex digits", firstKey, prefix)
	}
	// A relative path and its absolute form are the same database.
	absolute, err := filepath.Abs(first)
	if err != nil {
		t.Fatal(err)
	}
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(working, absolute)
	if err != nil {
		t.Fatal(err)
	}
	if got := syncclient.RefreshTokenKeyFor(relative); got != firstKey {
		t.Errorf("key for the relative path = %q, want %q", got, firstKey)
	}
	if got := syncclient.RefreshTokenKeyFor(""); got != syncclient.RefreshTokenKey {
		t.Errorf("key for no database = %q, want the prefix", got)
	}
}

// TestDeleteRefreshTokenAtCleansOldSignIns: signing out removes the refresh token an older version
// left for this database, and the old fixed key only for the default database.
func TestDeleteRefreshTokenAtCleansOldSignIns(t *testing.T) {
	secrets := newFakeSecrets()
	app := filepath.Join(t.TempDir(), "adhd-todo.db")
	other := filepath.Join(t.TempDir(), "test-profile.db")
	t.Cleanup(syncclient.SetLegacyDBPath(app))
	for key, value := range map[string]string{
		syncclient.RefreshTokenKey:           "fixed",
		syncclient.RefreshTokenKeyFor(app):   "app",
		syncclient.RefreshTokenKeyFor(other): "other",
	} {
		if err := secrets.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}

	if err := syncclient.DeleteRefreshTokenAt(secrets, other); err != nil {
		t.Fatal(err)
	}
	if _, err := secrets.Get(syncclient.RefreshTokenKeyFor(other)); !errors.Is(err, syncclient.ErrSecretNotFound) {
		t.Errorf("the other database keeps its token: %v", err)
	}
	if got, err := secrets.Get(syncclient.RefreshTokenKey); err != nil || got != "fixed" {
		t.Errorf("the other database's sign-out touched the fixed key: %q, %v", got, err)
	}

	if err := syncclient.DeleteRefreshTokenAt(secrets, app); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{syncclient.RefreshTokenKey, syncclient.RefreshTokenKeyFor(app)} {
		if _, err := secrets.Get(key); !errors.Is(err, syncclient.ErrSecretNotFound) {
			t.Errorf("%s survived the default database's sign-out: %v", key, err)
		}
	}
}
