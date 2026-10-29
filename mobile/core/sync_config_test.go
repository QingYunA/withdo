package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// TestInitialStatus pins the initial status shape of a phone that has not synced.
func TestInitialStatus(t *testing.T) {
	c, err := open(t.TempDir(), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = c.Close() }()

	var status statusJSON
	if err := json.Unmarshal([]byte(c.SyncStatus()), &status); err != nil {
		t.Fatalf("decode SyncStatus: %v", err)
	}
	if status.State != string(syncclient.StateIdle) || status.Pending != 0 || status.LastSynced != "" {
		t.Errorf("initial status = %+v", status)
	}
}

// TestAnOlderSyncJSONSignsInAgain: a phone that signed in with an earlier version has a
// sync.json with an issuer and an account and a refresh token in its secure store. It loads
// without error, shows no account, and the next sign-in reads the server's config and goes on
// through PocketBase.
func TestAnOlderSyncJSONSignsInAgain(t *testing.T) {
	f := newCodeFake(t)
	dir := t.TempDir()
	older := `{"server":"` + f.URL + `","issuer":"https://auth.example","client_id":"phone",` +
		`"account":{"name":"Ada Lovelace","email":"ada@example.com"}}`
	if err := os.WriteFile(filepath.Join(dir, "sync.json"), []byte(older), 0o600); err != nil {
		t.Fatal(err)
	}
	secrets := newMapSecrets()
	if err := secrets.Set(syncclient.RefreshTokenKeyFor(filepath.Join(dir, "adhd-todo.db")), "old-refresh"); err != nil {
		t.Fatal(err)
	}
	phone, err := open(dir, secrets, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()

	if raw, err := phone.Account(); err != nil || raw != "{}" {
		t.Fatalf("Account = %q, %v; want {} (signed out)", raw, err)
	}
	var status statusJSON
	if err := json.Unmarshal([]byte(phone.SyncStatus()), &status); err != nil {
		t.Fatalf("decode SyncStatus: %v", err)
	}
	if status.SignedIn {
		t.Errorf("status = %+v; the old sign-in cannot be used", status)
	}
	if err := phone.SignOut(); err != nil {
		t.Fatalf("SignOut: %v", err)
	}
	if secrets.has(syncclient.RefreshTokenKeyFor(phone.dbPath)) {
		t.Error("SignOut left the old refresh token in the secure store")
	}
	if err := phone.RequestResetCode("ada@example.com"); err != nil {
		t.Fatalf("RequestResetCode on an older config: %v", err)
	}
}
