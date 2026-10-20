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
		t.Errorf("key for no database = %q, want the legacy key", got)
	}
}

// TestTheOldRefreshTokenMovesToTheDatabaseKey is A3-fix 8's migration: an app that signed in before
// the key changed keeps its login, and the old fixed key cannot be read by another database.
func TestTheOldRefreshTokenMovesToTheDatabaseKey(t *testing.T) {
	secrets := newFakeSecrets()
	first := filepath.Join(t.TempDir(), "adhd-todo.db")
	second := filepath.Join(t.TempDir(), "test-profile.db")
	t.Cleanup(syncclient.SetLegacyDBPath(first)) // first is the default database
	if err := secrets.Set(syncclient.RefreshTokenKey, "old-token"); err != nil {
		t.Fatal(err)
	}

	token, err := syncclient.LoadRefreshToken(secrets, first)
	if err != nil || token != "old-token" {
		t.Fatalf("LoadRefreshToken = %q, %v; want the old token", token, err)
	}
	if _, err := secrets.Get(syncclient.RefreshTokenKey); !errors.Is(err, syncclient.ErrSecretNotFound) {
		t.Errorf("the legacy key still holds a token: %v", err)
	}
	if moved, err := secrets.Get(syncclient.RefreshTokenKeyFor(first)); err != nil || moved != "old-token" {
		t.Errorf("the moved token = %q, %v", moved, err)
	}

	// A second database does not inherit the first one's sign-in.
	if _, err := syncclient.LoadRefreshToken(secrets, second); !errors.Is(err, syncclient.ErrSecretNotFound) {
		t.Errorf("the second database got a token: %v", err)
	}
	// Reading again is stable.
	if token, err := syncclient.LoadRefreshToken(secrets, first); err != nil || token != "old-token" {
		t.Errorf("second LoadRefreshToken = %q, %v", token, err)
	}
}

// TestSavingAndDeletingOneDatabaseKeepsAnother is A3-fix 8: two databases keep their own tokens,
// and signing out of one does not sign out the other.
func TestSavingAndDeletingOneDatabaseKeepsAnother(t *testing.T) {
	secrets := newFakeSecrets()
	first := filepath.Join(t.TempDir(), "adhd-todo.db")
	second := filepath.Join(t.TempDir(), "test-profile.db")
	if err := syncclient.SaveRefreshToken(secrets, first, "first-token"); err != nil {
		t.Fatal(err)
	}
	if err := syncclient.SaveRefreshToken(secrets, second, "second-token"); err != nil {
		t.Fatal(err)
	}
	if err := syncclient.DeleteRefreshTokenAt(secrets, first); err != nil {
		t.Fatal(err)
	}
	if _, err := syncclient.LoadRefreshToken(secrets, first); !errors.Is(err, syncclient.ErrSecretNotFound) {
		t.Errorf("the first database still has a token: %v", err)
	}
	if token, err := syncclient.LoadRefreshToken(secrets, second); err != nil || token != "second-token" {
		t.Errorf("the second database's token = %q, %v", token, err)
	}
}

// TestAnotherDatabaseNeverTakesTheOldToken is A3-fix 8: a test profile or ADHD_TODO_DB database
// that reads or signs out before the app has moved its token neither takes nor deletes it.
func TestAnotherDatabaseNeverTakesTheOldToken(t *testing.T) {
	secrets := newFakeSecrets()
	app := filepath.Join(t.TempDir(), "adhd-todo.db")
	other := filepath.Join(t.TempDir(), "test-profile.db")
	t.Cleanup(syncclient.SetLegacyDBPath(app))
	if err := secrets.Set(syncclient.RefreshTokenKey, "app-token"); err != nil {
		t.Fatal(err)
	}

	if _, err := syncclient.LoadRefreshToken(secrets, other); !errors.Is(err, syncclient.ErrSecretNotFound) {
		t.Fatalf("the other database got a token: %v", err)
	}
	if err := syncclient.DeleteRefreshTokenAt(secrets, other); err != nil {
		t.Fatal(err)
	}
	if token, err := syncclient.LoadRefreshToken(secrets, app); err != nil || token != "app-token" {
		t.Errorf("the app's token after the other database read and signed out = %q, %v", token, err)
	}
}
