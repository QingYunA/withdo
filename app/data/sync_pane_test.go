package data

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// configIssuer is a sync server and the public half of its OIDC issuer: /v1/config, discovery, an
// authorize endpoint that starts a pending auth request, and a /v1/sign-in that refuses every
// password and records what it got.
type configIssuer struct {
	*httptest.Server
	mu       sync.Mutex
	config   int
	login    string
	password string
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
	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, f.Server.URL+"/ui/v2/login/login?authRequest=V2_1", http.StatusFound)
	})
	mux.HandleFunc("POST /v1/sign-in", func(w http.ResponseWriter, r *http.Request) {
		var req syncproto.SignInRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.mu.Lock()
		f.login, f.password = req.LoginName, req.Password
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_credentials"})
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
// /v1/config, then uses the issuer and client it names.
func TestTheAppSignsInWithTheServerConfig(t *testing.T) {
	ctx := context.Background()
	issuer := newConfigIssuer(t)
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	r := NewReal(db, RealOptions{
		SyncConfigPath: syncclient.ConfigPath(path), Secrets: appSecrets{},
		Now: func() time.Time { return time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC) },
	})
	t.Cleanup(func() { _ = r.Close() })
	if err := r.SyncSetServer(issuer.URL); err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 16)
	r.SetSyncNotify(func() { changed <- struct{}{} })
	r.SyncSignIn("ada@example.com", "secret")

	deadline := time.After(10 * time.Second)
	for {
		if snap := r.SyncSnapshot(); snap.Phase == SyncSignedOut && snap.SignInFailure != "" {
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
}

// TestSignInFailureKinds is the classification the pane words: the five answers of /v1/sign-in
// and a network failure, without matching the error's text.
func TestSignInFailureKinds(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want SignInFailure
	}{
		{"credentials", syncclient.ErrInvalidCredentials, SignInFailureCredentials},
		{"too many", syncclient.ErrTooManyAttempts, SignInFailureTooMany},
		{"second factor", syncclient.ErrSecondFactor, SignInFailureSecondFactor},
		{"locked", syncclient.ErrAccountLocked, SignInFailureLocked},
		{"unsupported", syncclient.ErrSignInUnsupported, SignInFailureUnsupported},
		{"network", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, SignInFailureNetwork},
		{"other", errors.New("a row did not fit"), SignInFailureOther},
		{"no failure", nil, SignInFailureNone},
	}
	for _, tc := range cases {
		if got := signInFailureKind(tc.err); got != tc.want {
			t.Errorf("%s: signInFailureKind(%v) = %q, want %q", tc.name, tc.err, got, tc.want)
		}
	}
}

// TestSyncSignInSendsTheAccountAndPassword checks what the pane hands the sign-in source and that
// the password reaches neither the snapshot nor sync.json.
func TestSyncSignInSendsTheAccountAndPassword(t *testing.T) {
	issuer := newConfigIssuer(t)
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	r := NewReal(db, RealOptions{
		SyncConfigPath: syncclient.ConfigPath(path), Secrets: appSecrets{},
		Now: func() time.Time { return time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC) },
	})
	t.Cleanup(func() { _ = r.Close() })
	if err := r.SyncSetServer(issuer.URL); err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 16)
	r.SetSyncNotify(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	r.SyncSignIn("cyan@example.com", "hunter2")

	deadline := time.After(10 * time.Second)
	for {
		if snap := r.SyncSnapshot(); snap.Phase == SyncSignedOut && snap.SignInFailure == SignInFailureCredentials {
			if strings.Contains(snap.LastError, "hunter2") {
				t.Errorf("the snapshot carries the password: %q", snap.LastError)
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
	login, password := issuer.login, issuer.password
	issuer.mu.Unlock()
	if login != "cyan@example.com" || password != "hunter2" {
		t.Errorf("/v1/sign-in got account %q, password %q", login, password)
	}
	raw, err := os.ReadFile(syncclient.ConfigPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "hunter2") {
		t.Errorf("sync.json carries the password: %s", raw)
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
