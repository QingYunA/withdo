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
)

// configServer is a sync server with PocketBase's front: /v1/config naming PocketBase, and a
// password route that refuses every password and records what it got.
type configServer struct {
	*httptest.Server
	mu       sync.Mutex
	config   int
	login    string
	password string
}

func newConfigServer(t *testing.T) *configServer {
	t.Helper()
	f := &configServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		f.config++
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"auth":"pocketbase"}`))
	})
	mux.HandleFunc("POST /api/collections/users/auth-with-password", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Identity, Password string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.mu.Lock()
		f.login, f.password = req.Identity, req.Password
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Failed to authenticate."}`))
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// TestTheAppSignsInWithTheServerConfig is the client half: Sign in and sync first reads
// /v1/config, then signs in through the PocketBase it names.
func TestTheAppSignsInWithTheServerConfig(t *testing.T) {
	ctx := context.Background()
	server := newConfigServer(t)
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
	if err := r.SyncSetServer(server.URL); err != nil {
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
	server.mu.Lock()
	hits := server.config
	server.mu.Unlock()
	if hits != 1 {
		t.Errorf("GET /v1/config called %d times, want 1", hits)
	}
}

// TestSyncSignInStages: the snapshot says the server is checking the account, then that the tasks
// are downloading while Link registers the device. The device call is allowed to fail after that.
func TestSyncSignInStages(t *testing.T) {
	signInHit := make(chan struct{}, 1)
	releaseSignIn := make(chan struct{})
	devicesHit := make(chan struct{}, 1)
	releaseDevices := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"auth":"pocketbase"}`))
	})
	mux.HandleFunc("POST /api/collections/users/auth-with-password", func(w http.ResponseWriter, _ *http.Request) {
		signInHit <- struct{}{}
		<-releaseSignIn
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"e30.eyJleHAiOjQxMDI0NDQ4MDB9.sig","record":{"email":"cyan@example.com","name":"Cyan"}}`))
	})
	mux.HandleFunc("POST /v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		devicesHit <- struct{}{}
		<-releaseDevices
		http.Error(w, "the server failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

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
	if err := r.SyncSetServer(server.URL); err != nil {
		t.Fatal(err)
	}
	r.SyncSignIn("cyan@example.com", "hunter2")

	if snap := r.SyncSnapshot(); snap.Phase != SyncSigningIn || snap.SignInStage != SignInStageChecking {
		t.Fatalf("before the password check: phase %v, stage %q", snap.Phase, snap.SignInStage)
	}
	waitSignal(t, signInHit, "the sign-in request")
	close(releaseSignIn)
	waitSignal(t, devicesHit, "the device registration")
	if snap := r.SyncSnapshot(); snap.Phase != SyncSigningIn || snap.SignInStage != SignInStageDownloading {
		t.Fatalf("after the password check: phase %v, stage %q", snap.Phase, snap.SignInStage)
	}
	close(releaseDevices)

	deadline := time.After(10 * time.Second)
	for r.SyncSnapshot().Phase == SyncSigningIn {
		select {
		case <-deadline:
			t.Fatal("the failed device registration never ended the sign-in")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if snap := r.SyncSnapshot(); snap.SignInStage != SignInStageNone {
		t.Fatalf("a finished sign-in kept stage %q", snap.SignInStage)
	}
}

// waitSignal waits for one step of a gated fake server.
func waitSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// TestSignInFailureKinds is the classification the pane words: the answers of a sign-in
// and a network failure, without matching the error's text.
func TestSignInFailureKinds(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want SignInFailure
	}{
		{"credentials", syncclient.ErrInvalidCredentials, SignInFailureCredentials},
		{"too many", syncclient.ErrTooManyAttempts, SignInFailureTooMany},
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
	server := newConfigServer(t)
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
	if err := r.SyncSetServer(server.URL); err != nil {
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
	server.mu.Lock()
	login, password := server.login, server.password
	server.mu.Unlock()
	if login != "cyan@example.com" || password != "hunter2" {
		t.Errorf("sign-in got account %q, password %q", login, password)
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
