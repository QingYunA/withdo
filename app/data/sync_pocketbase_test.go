package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// TestTheAppSignsInThroughPocketBase: when /v1/config says "auth":"pocketbase" the pane posts the
// email and password to PocketBase's route, with no issuer, and a refused password shows as a
// credentials failure in the same place the Zitadel sign-in used.
func TestTheAppSignsInThroughPocketBase(t *testing.T) {
	ctx := context.Background()
	var mu sync.Mutex
	var identity string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer":"","client_id":"","scopes":[],"auth":"pocketbase"}`))
	})
	mux.HandleFunc("POST /api/collections/users/auth-with-password", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Identity string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		identity = body.Identity
		mu.Unlock()
		w.WriteHeader(http.StatusBadRequest)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	r := NewReal(db, RealOptions{
		SyncConfigPath: syncclient.ConfigPath(path), Secrets: appSecrets{},
		Now: func() time.Time { return time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC) },
	})
	t.Cleanup(func() { _ = r.Close() })
	if err := r.SyncSetServer(server.URL); err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 16)
	r.SetSyncNotify(func() { changed <- struct{}{} })
	r.SyncSignIn("ada@example.com", "secret")

	deadline := time.After(10 * time.Second)
	for {
		if snap := r.SyncSnapshot(); snap.Phase == SyncSignedOut && snap.SignInFailure != "" {
			if snap.SignInFailure != SignInFailureCredentials {
				t.Fatalf("failure = %q, want credentials", snap.SignInFailure)
			}
			if strings.Contains(snap.LastError, "secret") {
				t.Fatalf("the password reached the snapshot: %q", snap.LastError)
			}
			break
		}
		select {
		case <-changed:
		case <-deadline:
			t.Fatal("the sign-in never failed")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if identity != "ada@example.com" {
		t.Fatalf("PocketBase got identity %q", identity)
	}
}
