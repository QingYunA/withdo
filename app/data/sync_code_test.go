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

// codeServer is a sync server with the PocketBase routes of the sign-up. users create refuses the
// email taken@example.com (as PocketBase's unique check does) and makes any other; request-otp hands
// out otp<N>; auth-with-otp and the reset route accept only 123456 for the latest id;
// auth-with-password says unverified@example.com is not verified.
type codeServer struct {
	*httptest.Server
	mux      *http.ServeMux
	mu       sync.Mutex
	requests int
	checks   int
	resets   int
	emails   []string
	signUps  []string // email|password|lang of every sign-up
	status   int      // when not 0, request-otp answers it
	// foreignID makes auth-with-oauth2 answer another record, as PocketBase does for a Google
	// identity another account owns.
	foreignID  string
	devicesHit chan struct{}
	release    chan struct{}
	// holdCheck, when set before the code is sent, keeps auth-with-otp waiting until it is closed,
	// so a test can see the checking stage before the sign-in moves on.
	holdCheck chan struct{}
}

func newCodeServer(t *testing.T) *codeServer {
	t.Helper()
	f := &codeServer{devicesHit: make(chan struct{}, 1), release: make(chan struct{})}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"auth":"pocketbase"}`))
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
		if f.holdCheck != nil {
			<-f.holdCheck
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.checks++
		if body.OTPID != fmt.Sprintf("otp%d", f.requests) || body.Password != "123456" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.writeToken(w)
	})
	mux.HandleFunc("POST /api/collections/users/records", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email, Password, PasswordConfirm, Lang string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.signUps = append(f.signUps, body.Email+"|"+body.Password+"|"+body.Lang)
		if body.Email == "taken@example.com" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":400,"data":{"email":{"code":"validation_not_unique","message":"x"}}}`))
			return
		}
		_, _ = fmt.Fprintf(w, `{"id":"rec2","email":%q,"verified":false}`, body.Email)
	})
	mux.HandleFunc("POST /api/collections/users/auth-with-password", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Identity, Password string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Identity == "unverified@example.com" && body.Password == "a long password" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"status":403,"data":{"email":{"code":"email_not_verified","message":"x"}}}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	})
	mux.HandleFunc("POST /api/withdo/reset-password", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ OTPID, Code, Password, PasswordConfirm string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.resets++
		if len(body.Password) < 8 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":400,"data":{"password":{"code":"validation_min_text_constraint","message":"x"}}}`))
			return
		}
		if body.OTPID != fmt.Sprintf("otp%d", f.requests) || body.Code != "123456" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":400,"message":"Invalid or expired OTP.","data":{}}`))
			return
		}
		f.writeToken(w)
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

// writeToken answers a sign-in with a token and the record of ada@example.com.
func (f *codeServer) writeToken(w http.ResponseWriter) {
	payload, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix(), "id": "rec1"})
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":  "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig",
		"record": map[string]any{"id": "rec1", "email": "ada@example.com", "name": "ada"},
	})
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

func atVerifyStep(s SyncSnapshot) bool {
	return s.Phase == SyncSignedOut && s.SignInStep == SignInStepVerify
}

func atResetCodeStep(s SyncSnapshot) bool {
	return s.Phase == SyncSignedOut && s.SignInStep == SignInStepResetCode
}

func TestSignUpMovesToVerifyAndResendWaits(t *testing.T) {
	f := newCodeServer(t)
	r, advance := codeApp(t, f)

	r.SyncSignUp(" ada@example.com ", "a long password", "zh-Hans")
	snap := waitSnap(t, r, "the verify step", atVerifyStep)
	if snap.SignInEmail != "ada@example.com" || !snap.ResendAt.Equal(time.Date(2026, 10, 22, 9, 0, 30, 0, time.UTC)) {
		t.Fatalf("email %q, resend at %v", snap.SignInEmail, snap.ResendAt)
	}
	if snap.SignInFailure != SignInFailureNone || snap.SignInStage != SignInStageNone {
		t.Fatalf("failure %q stage %q", snap.SignInFailure, snap.SignInStage)
	}
	f.mu.Lock()
	signUps := append([]string(nil), f.signUps...)
	f.mu.Unlock()
	if len(signUps) != 1 || signUps[0] != "ada@example.com|a long password|zh-Hans" {
		t.Fatalf("sign-ups %v", signUps)
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
		return atVerifyStep(s) && s.ResendAt.After(snap.ResendAt)
	})
	if requests, _, _ := f.counts(); requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	f.mu.Lock()
	n := len(f.signUps)
	f.mu.Unlock()
	if n != 1 {
		t.Fatalf("a resend signed up again (%d sign-ups)", n)
	}
}

func TestSignUpWithATakenEmailStaysOnSignUp(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncSetSignInStep(SignInStepSignUp)
	r.SyncSignUp("taken@example.com", "a long password", "en")
	snap := waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureEmailTaken || snap.SignInStep != SignInStepSignUp || snap.Phase != SyncSignedOut ||
		snap.SignInEmail != "taken@example.com" {
		t.Fatalf("failure %q step %q phase %v email %q", snap.SignInFailure, snap.SignInStep, snap.Phase, snap.SignInEmail)
	}
	if requests, _, _ := f.counts(); requests != 0 {
		t.Fatalf("a code was requested for a taken email (%d)", requests)
	}
}

func TestSigningInToAnUnverifiedAccountShowsVerify(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncSignIn("unverified@example.com", "a long password")
	snap := waitSnap(t, r, "the verify step", atVerifyStep)
	if snap.SignInEmail != "unverified@example.com" || snap.SignInFailure != SignInFailureNone {
		t.Fatalf("email %q failure %q", snap.SignInEmail, snap.SignInFailure)
	}
	if requests, _, _ := f.counts(); requests != 1 {
		t.Fatalf("code requests = %d, want the one mailed after the refused sign-in", requests)
	}
	// A wrong password is only a failure on the sign-in tab.
	r.SyncSetSignInStep(SignInStepSignIn)
	r.SyncSignIn("unverified@example.com", "wrong password")
	snap = waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureCredentials || snap.SignInStep != SignInStepSignIn {
		t.Fatalf("failure %q step %q", snap.SignInFailure, snap.SignInStep)
	}
}

func TestWrongCodeKeepsTheVerifyStep(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncSignUp("ada@example.com", "a long password", "en")
	waitSnap(t, r, "the verify step", atVerifyStep)

	r.SyncSignInWithCode("000000")
	snap := waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureWrongCode || snap.SignInStep != SignInStepVerify || snap.Phase != SyncSignedOut {
		t.Fatalf("failure %q step %q phase %v", snap.SignInFailure, snap.SignInStep, snap.Phase)
	}
}

func TestAnOldCodeIsReplacedAtOnce(t *testing.T) {
	f := newCodeServer(t)
	r, advance := codeApp(t, f)
	r.SyncSignUp("ada@example.com", "a long password", "en")
	waitSnap(t, r, "the verify step", atVerifyStep)

	advance(10*time.Minute + time.Second)
	r.SyncSignInWithCode("123456")
	snap := waitSnap(t, r, "the new code", func(s SyncSnapshot) bool { return s.SignInFailure == SignInFailureCodeExpired })
	if snap.SignInStep != SignInStepVerify || !snap.ResendAt.After(time.Date(2026, 10, 22, 9, 10, 0, 0, time.UTC)) {
		t.Fatalf("step %q, resend at %v", snap.SignInStep, snap.ResendAt)
	}
	if requests, checks, _ := f.counts(); requests != 2 || checks != 0 {
		t.Fatalf("requests %d checks %d; the old code must not be tried", requests, checks)
	}
}

func TestRightCodeLinksLikeAPasswordSignIn(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncSignUp("ada@example.com", "a long password", "en")
	waitSnap(t, r, "the verify step", atVerifyStep)

	f.holdCheck = make(chan struct{})
	r.SyncSignInWithCode("123456")
	if snap := r.SyncSnapshot(); snap.Phase != SyncSigningIn || snap.SignInStage != SignInStageChecking {
		t.Fatalf("while checking: phase %v stage %q", snap.Phase, snap.SignInStage)
	}
	close(f.holdCheck)
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

func TestForgotPasswordResetsInTheApp(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	r.SyncSetSignInStep(SignInStepResetEmail)
	if snap := r.SyncSnapshot(); snap.SignInStep != SignInStepResetEmail {
		t.Fatalf("step = %q", snap.SignInStep)
	}
	r.SyncRequestResetCode(" ada@example.com ")
	snap := waitSnap(t, r, "the reset code step", atResetCodeStep)
	if snap.SignInEmail != "ada@example.com" || snap.Phase != SyncSignedOut {
		t.Fatalf("email %q phase %v", snap.SignInEmail, snap.Phase)
	}
	if requests, _, _ := f.counts(); requests != 1 {
		t.Fatalf("requests = %d", requests)
	}
	f.mu.Lock()
	n := len(f.signUps)
	f.mu.Unlock()
	if n != 0 {
		t.Fatal("a reset made an account")
	}

	// A wrong code stays on the step; a short password is its own failure; a right pair signs in.
	r.SyncResetPassword("000000", "a new password")
	snap = waitSnap(t, r, "the wrong code", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureWrongCode || snap.SignInStep != SignInStepResetCode {
		t.Fatalf("failure %q step %q", snap.SignInFailure, snap.SignInStep)
	}
	r.SyncResetPassword("123456", "short")
	waitSnap(t, r, "the weak password", func(s SyncSnapshot) bool { return s.SignInFailure == SignInFailureWeakPassword })
	close(f.release)
	r.SyncResetPassword("123456", "a new password")
	waitSnap(t, r, "the sign-in", func(s SyncSnapshot) bool { return s.AccountEmail == "ada@example.com" && s.Phase != SyncSigningIn })
	if _, _, resets := f.counts(); resets != 3 {
		t.Fatalf("reset calls = %d, want 3", resets)
	}

	// Going back keeps the address.
	r.SyncSetSignInStep(SignInStepSignIn)
	if snap := r.SyncSnapshot(); snap.SignInStep != SignInStepSignIn {
		t.Fatalf("step %q", snap.SignInStep)
	}
}

func TestCodeRequestFailureKinds(t *testing.T) {
	f := newCodeServer(t)
	r, _ := codeApp(t, f)
	f.mu.Lock()
	f.status = http.StatusTooManyRequests
	f.mu.Unlock()
	r.SyncSignUp("ada@example.com", "a long password", "en")
	snap := waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureTooMany || snap.SignInStep != SignInStepSignIn {
		t.Fatalf("failure %q step %q", snap.SignInFailure, snap.SignInStep)
	}
}
