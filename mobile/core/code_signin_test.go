package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// codeFake is a sync server with PocketBase's code routes: request-otp hands out otp<N>,
// auth-with-otp accepts 123456 for the latest id, and POST /v1/devices fails so the sign-in ends
// right after the code was accepted.
type codeFake struct {
	*httptest.Server
	mux      *http.ServeMux
	mu       sync.Mutex
	requests int
	checks   int
	resets   int
	status   int
}

func newCodeFake(t *testing.T) *codeFake {
	t.Helper()
	f := &codeFake{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"auth":"pocketbase"}`))
	})
	mux.HandleFunc("POST /api/collections/users/request-otp", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.status != 0 {
			w.WriteHeader(f.status)
			return
		}
		f.requests++
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
		http.Error(w, "the server failed", http.StatusInternalServerError)
	})
	f.mux = mux
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *codeFake) counts() (requests, checks, resets int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests, f.checks, f.resets
}

// codePhone opens a phone core whose clock the test moves.
func codePhone(t *testing.T, f *codeFake) (*Core, func(time.Duration)) {
	t.Helper()
	var mu sync.Mutex
	now := demoNow
	phone, err := open(t.TempDir(), newMapSecrets(), func() time.Time { mu.Lock(); defer mu.Unlock(); return now })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	if err := phone.SyncSetServer(f.URL); err != nil {
		t.Fatalf("SyncSetServer: %v", err)
	}
	return phone, func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
}

func codeState(t *testing.T, phone *Core) codeStateJSON {
	t.Helper()
	raw, err := phone.CodeState()
	if err != nil {
		t.Fatal(err)
	}
	var out codeStateJSON
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("CodeState = %q: %v", raw, err)
	}
	return out
}

func TestRequestCodeStartsTheCodeStepAndResendWaits(t *testing.T) {
	f := newCodeFake(t)
	phone, advance := codePhone(t, f)
	if got := codeState(t, phone); got.Step != "" {
		t.Fatalf("before a request: %+v", got)
	}
	if err := phone.RequestCode(" ada@example.com "); err != nil {
		t.Fatalf("RequestCode: %v", err)
	}
	if got := codeState(t, phone); got.Step != "code" || got.Email != "ada@example.com" || got.ResendInSecs != 30 || got.Failure != "" {
		t.Fatalf("after a request: %+v", got)
	}
	advance(6 * time.Second)
	if got := codeState(t, phone); got.ResendInSecs != 24 {
		t.Fatalf("resend_in = %d, want 24", got.ResendInSecs)
	}
	if err := phone.ResendCode(); err == nil {
		t.Fatal("a resend after 6 s should be refused")
	}
	if requests, _, _ := f.counts(); requests != 1 {
		t.Fatalf("requests = %d; a refused resend must not ask the server", requests)
	}
	advance(25 * time.Second)
	if err := phone.ResendCode(); err != nil {
		t.Fatalf("ResendCode: %v", err)
	}
	if requests, _, _ := f.counts(); requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if got := codeState(t, phone); got.ResendInSecs != 30 {
		t.Fatalf("resend_in = %d after a resend, want 30", got.ResendInSecs)
	}
}

func TestWrongCodeThenRightCode(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	if err := phone.RequestCode("ada@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := phone.SignInWithCode("000000", "phone"); err == nil {
		t.Fatal("a wrong code should fail")
	}
	if got := codeState(t, phone); got.Step != "code" || got.Failure != "wrong_code" {
		t.Fatalf("after a wrong code: %+v", got)
	}
	// The right code is accepted; the sign-in then fails at the device call, but the token and the
	// account are stored and the code step is over.
	_, _ = phone.SignInWithCode("123456", "phone")
	if got := codeState(t, phone); got.Step != "" || got.Failure != "" {
		t.Fatalf("after the right code: %+v", got)
	}
	if raw, _ := phone.Account(); raw != `{"name":"ada","email":"ada@example.com"}` {
		t.Fatalf("Account = %s", raw)
	}
}

func TestAnOldCodeIsReplaced(t *testing.T) {
	f := newCodeFake(t)
	phone, advance := codePhone(t, f)
	if err := phone.RequestCode("ada@example.com"); err != nil {
		t.Fatal(err)
	}
	advance(10*time.Minute + time.Second)
	if _, err := phone.SignInWithCode("123456", "phone"); err == nil {
		t.Fatal("an old code should not sign in")
	}
	if requests, checks, _ := f.counts(); requests != 2 || checks != 0 {
		t.Fatalf("requests %d checks %d; the old code must not be tried", requests, checks)
	}
	if got := codeState(t, phone); got.Step != "code" || got.Failure != "expired" || got.ResendInSecs != 30 {
		t.Fatalf("after an old code: %+v", got)
	}
}

func TestCodeFailureKindsAndPasswordReset(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	f.mu.Lock()
	f.status = http.StatusTooManyRequests
	f.mu.Unlock()
	if err := phone.RequestCode("ada@example.com"); err == nil {
		t.Fatal("a rate-limited request should fail")
	}
	if got := codeState(t, phone); got.Step != "" || got.Failure != "too_many" {
		t.Fatalf("after 429: %+v", got)
	}
	if err := phone.RequestPasswordReset("ada@example.com"); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}
	if _, _, resets := f.counts(); resets != 1 {
		t.Fatalf("resets = %d", resets)
	}
}
