package data

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// codeServer is a sync server with a PocketBase's three code routes. request-otp hands out
// otp<N>; auth-with-otp accepts only 123456 for the latest id.
type codeServer struct {
	*httptest.Server
	mux        *http.ServeMux
	mu         sync.Mutex
	requests   int
	checks     int
	resets     int
	emails     []string
	status     int // when not 0, request-otp answers it
	devicesHit chan struct{}
	release    chan struct{}
}

func newCodeServer(t *testing.T) *codeServer {
	t.Helper()
	f := &codeServer{devicesHit: make(chan struct{}, 1), release: make(chan struct{})}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer":"","client_id":"","scopes":[],"auth":"pocketbase"}`))
	})
	mux.HandleFunc("POST /api/collections/users/request-otp", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.status != 0 {
			w.WriteHeader(f.status)
			return
		}
		f.requests++
		f.emails = append(f.emails, body.Email)
		_, _ = fmt.Fprintf(w, `{"otpId":"otp%d"}`, f.requests)
	})
	mux.HandleFunc("POST /api/collections/users/auth-with-otp", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OTPID    string `json:"otpId"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.checks++
		if body.OTPID != fmt.Sprintf("otp%d", f.requests) || body.Password != "123456" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		payload, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix()})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig",
			"record": map[string]any{"email": "ada@example.com", "name": "ada"},
		})
	})
	mux.HandleFunc("POST /api/collections/users/request-password-reset", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		f.resets++
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		f.devicesHit <- struct{}{}
		<-f.release
		http.Error(w, "the server failed", http.StatusInternalServerError)
	})
	f.mux = mux
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *codeServer) counts() (requests, checks, resets int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests, f.checks, f.resets
}

// codeApp is a Real pointed at the server, with a clock the test moves.
func codeApp(t *testing.T, f *codeServer) (*Real, func(time.Duration)) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	now := time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)
	r := NewReal(db, RealOptions{
		SyncConfigPath: syncclient.ConfigPath(path), Secrets: appSecrets{},
		Now: func() time.Time { mu.Lock(); defer mu.Unlock(); return now },
	})
	t.Cleanup(func() { _ = r.Close() })
	if err := r.SyncSetServer(f.URL); err != nil {
		t.Fatal(err)
	}
	return r, func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
}

// waitSnap polls the snapshot until ok holds.
func waitSnap(t *testing.T, r *Real, what string, ok func(SyncSnapshot) bool) SyncSnapshot {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		if snap := r.SyncSnapshot(); ok(snap) {
			return snap
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func atCodeStep(s SyncSnapshot) bool {
	return s.Phase == SyncSignedOut && s.SignInStep == SignInStepCode
}

func TestRequestCodeMovesToTheCodeStepAndResendWaits(t *testing.T) {
	f := newCodeServer(t)
	r, advance := codeApp(t, f)

	r.SyncRequestCode(" ada@example.com ")
	snap := waitSnap(t, r, "the code step", atCodeStep)
	if snap.SignInEmail != "ada@example.com" || !snap.ResendAt.Equal(time.Date(2026, 10, 22, 9, 0, 30, 0, time.UTC)) {
		t.Fatalf("email %q, resend at %v", snap.SignInEmail, snap.ResendAt)
	}
	if snap.SignInFailure != SignInFailureNone || snap.SignInStage != SignInStageNone {
		t.Fatalf("failure %q stage %q", snap.SignInFailure, snap.SignInStage)
	}

	r.SyncResendCode() // too early
	advance(29 * time.Second)
	r.SyncResendCode()
	time.Sleep(50 * time.Millisecond)
	if requests, _, _ := f.counts(); requests != 1 {
		t.Fatalf("requests = %d; a resend before 30 s must do nothing", requests)
	}
	advance(2 * time.Second)
	r.SyncResendCode()
	waitSnap(t, r, "the resend", func(s SyncSnapshot) bool {
		return atCodeStep(s) && s.ResendAt.After(snap.ResendAt)
	})
	if requests, _, _ := f.counts(); requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestWrongCodeKeepsTheCodeStep(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncRequestCode("ada@example.com")
	waitSnap(t, r, "the code step", atCodeStep)

	r.SyncSignInWithCode("000000")
	snap := waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureWrongCode || snap.SignInStep != SignInStepCode || snap.Phase != SyncSignedOut {
		t.Fatalf("failure %q step %q phase %v", snap.SignInFailure, snap.SignInStep, snap.Phase)
	}
}

func TestAnOldCodeIsReplacedAtOnce(t *testing.T) {
	f := newCodeServer(t)
	r, advance := codeApp(t, f)
	r.SyncRequestCode("ada@example.com")
	waitSnap(t, r, "the code step", atCodeStep)

	advance(10*time.Minute + time.Second)
	r.SyncSignInWithCode("123456")
	snap := waitSnap(t, r, "the new code", func(s SyncSnapshot) bool { return s.SignInFailure == SignInFailureCodeExpired })
	if snap.SignInStep != SignInStepCode || !snap.ResendAt.After(time.Date(2026, 10, 22, 9, 10, 0, 0, time.UTC)) {
		t.Fatalf("step %q, resend at %v", snap.SignInStep, snap.ResendAt)
	}
	if requests, checks, _ := f.counts(); requests != 2 || checks != 0 {
		t.Fatalf("requests %d checks %d; the old code must not be tried", requests, checks)
	}
}

func TestRightCodeLinksLikeAPasswordSignIn(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncRequestCode("ada@example.com")
	waitSnap(t, r, "the code step", atCodeStep)

	r.SyncSignInWithCode("123456")
	if snap := r.SyncSnapshot(); snap.Phase != SyncSigningIn || snap.SignInStage != SignInStageChecking {
		t.Fatalf("while checking: phase %v stage %q", snap.Phase, snap.SignInStage)
	}
	waitSignal(t, f.devicesHit, "the device registration")
	snap := r.SyncSnapshot()
	if snap.Phase != SyncSigningIn || snap.SignInStage != SignInStageDownloading {
		t.Fatalf("after the code: phase %v stage %q", snap.Phase, snap.SignInStage)
	}
	if snap.AccountEmail != "ada@example.com" {
		t.Fatalf("account = %q", snap.AccountEmail)
	}
	close(f.release)
	waitSnap(t, r, "the sign-in to end", func(s SyncSnapshot) bool { return s.Phase != SyncSigningIn })
}

func TestPasswordResetAndStepChanges(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncSetSignInStep(SignInStepPassword)
	if snap := r.SyncSnapshot(); snap.SignInStep != SignInStepPassword {
		t.Fatalf("step = %q", snap.SignInStep)
	}
	r.SyncRequestPasswordReset("ada@example.com")
	snap := waitSnap(t, r, "the reset step", func(s SyncSnapshot) bool { return s.SignInStep == SignInStepResetSent })
	if snap.SignInEmail != "ada@example.com" || snap.Phase != SyncSignedOut {
		t.Fatalf("email %q phase %v", snap.SignInEmail, snap.Phase)
	}
	if _, _, resets := f.counts(); resets != 1 {
		t.Fatalf("resets = %d", resets)
	}
	r.SyncSetSignInStep(SignInStepEmail)
	if snap := r.SyncSnapshot(); snap.SignInStep != SignInStepEmail || snap.SignInEmail != "ada@example.com" {
		t.Fatalf("step %q email %q; going back keeps the address", snap.SignInStep, snap.SignInEmail)
	}
}

func TestCodeRequestFailureKinds(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	f.mu.Lock()
	f.status = http.StatusTooManyRequests
	f.mu.Unlock()
	r.SyncRequestCode("ada@example.com")
	snap := waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureTooMany || snap.SignInStep != SignInStepEmail {
		t.Fatalf("failure %q step %q", snap.SignInFailure, snap.SignInStep)
	}
}

func TestCheckServerFlagsAServerWithoutCodes(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncCheckServer()
	waitSnap(t, r, "the probe", func(s SyncSnapshot) bool { r.mu.Lock(); defer r.mu.Unlock(); return r.sync.code.probed == s.Server })
	if r.SyncSnapshot().PasswordOnly {
		t.Fatal("a PocketBase server was taken for a password-only one")
	}
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"issuer":"https://id.example.com","client_id":"x","scopes":["openid"]}`))
	}))
	t.Cleanup(plain.Close)
	if err := r.SyncSetServer(plain.URL); err != nil {
		t.Fatal(err)
	}
	if r.SyncSnapshot().PasswordOnly {
		t.Fatal("PasswordOnly held for a server that was not asked yet")
	}
	r.SyncCheckServer()
	waitSnap(t, r, "the second probe", func(s SyncSnapshot) bool { return s.PasswordOnly })
}
