package data

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// configIssuer is the public half of an OIDC issuer, enough for a sign-in to reach the browser.
type configIssuer struct {
	*httptest.Server
	mu     sync.Mutex
	config int
	opened string
}

func newConfigIssuer(t *testing.T) *configIssuer {
	t.Helper()
	f := &configIssuer{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		f.config++
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": f.Server.URL, "client_id": "adhd-todo", "scopes": []string{"openid", "offline_access"},
		})
	})
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                f.Server.URL,
			"jwks_uri":                              f.Server.URL + "/jwks",
			"authorization_endpoint":                f.Server.URL + "/authorize",
			"token_endpoint":                        f.Server.URL + "/token",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// TestTheAppSignsInWithTheServerConfig is the client half: Sign in and sync first reads
// /v1/config, then uses the issuer and client it names. The fake browser stops after the
// authorization URL, so no token is needed.
func TestTheAppSignsInWithTheServerConfig(t *testing.T) {
	ctx := context.Background()
	issuer := newConfigIssuer(t)
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var opened string
	r := NewReal(db, RealOptions{
		SyncConfigPath: syncclient.ConfigPath(path), Secrets: appSecrets{},
		Now: func() time.Time { return time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC) },
		SyncOpen: func(url string) error {
			mu.Lock()
			opened = url
			mu.Unlock()
			return errors.New("the browser is not here")
		},
	})
	t.Cleanup(func() { _ = r.Close() })
	if err := r.SyncSetServer(issuer.URL); err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 16)
	r.SetSyncNotify(func() { changed <- struct{}{} })
	r.SyncSignIn()

	deadline := time.After(10 * time.Second)
	for {
		if snap := r.SyncSnapshot(); snap.Phase == SyncFailed {
			if snap.LastError == "" {
				t.Fatal("the failure has no reason")
			}
			break
		}
		select {
		case <-changed:
		case <-deadline:
			t.Fatal("the sign-in never failed")
		}
	}
	issuer.mu.Lock()
	hits := issuer.config
	issuer.mu.Unlock()
	if hits != 1 {
		t.Errorf("GET /v1/config called %d times, want 1", hits)
	}
	mu.Lock()
	openedURL := opened
	mu.Unlock()
	if !strings.Contains(openedURL, "/authorize") {
		t.Errorf("browser opened %q, want the issuer's authorization URL", openedURL)
	}
}

// TestMergeRewritesTaskIDsInTheThreeFiles is the file half of Merge: every task id key and every
// task id value moves through the map LinkMerge returns, and a number that is not a task id stays.
func TestMergeRewritesTaskIDsInTheThreeFiles(t *testing.T) {
	dir := t.TempDir()
	seen := filepath.Join(dir, syncSeenFile)
	dismissed := filepath.Join(dir, syncDismissed)
	recent := filepath.Join(dir, syncRecentCache)
	if err := os.WriteFile(seen, []byte(`{"45":"2026-10-14T10:00:00Z","7":"2026-10-13T10:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dismissed, []byte(`{"45":"2026-10-14T10:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// The cache is keyed by URL today. taskId and numeric keys show that a future task id in it
	// moves too, while days is left alone.
	if err := os.WriteFile(recent, []byte(`{"items":{"en https://example.com/pr/1":{"updatedAt":"2026-10-14T10:00:00Z"}},"tasks":{"45":{"sentence":"x"}},"taskId":45,"days":7}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := rewriteSyncTaskIDs(dir, map[int64]int64{45: 2045, 7: 9}); err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]struct {
		path string
		want []string
		not  []string
	}{
		"seen":      {seen, []string{`"2045"`, `"9"`}, []string{`"45"`, `"7"`}},
		"dismissed": {dismissed, []string{`"2045"`}, []string{`"45"`}},
		"recent":    {recent, []string{`"2045"`, `"taskId":2045`, `"days":7`}, []string{`"45"`, `"taskId":45`}},
	} {
		raw, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, want := range check.want {
			if !strings.Contains(string(raw), want) {
				t.Errorf("%s = %s, want %s", name, raw, want)
			}
		}
		for _, not := range check.not {
			if strings.Contains(string(raw), not) {
				t.Errorf("%s = %s, must not contain %s", name, raw, not)
			}
		}
	}
	tmp, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tmp) != 0 {
		t.Errorf("temporary files left: %v", tmp)
	}
	// A missing file is not an error: a Mac may never have written one.
	if err := rewriteSyncTaskIDs(t.TempDir(), map[int64]int64{1: 2}); err != nil {
		t.Errorf("missing files: %v", err)
	}
}
