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

// codeFake is a sync server with the PocketBase routes of the sign-up: users create refuses
// taken@example.com, request-otp hands out otp<N>, auth-with-otp and the reset route accept 123456
// for the latest id, auth-with-password says unverified@example.com is not verified, and POST
// /v1/devices fails so the sign-in ends right after the code was accepted.
type codeFake struct {
	*httptest.Server
	mux      *http.ServeMux
	mu       sync.Mutex
	requests int
	checks   int
	resets   int
	signUps  []string
	status   int
	// foreignID makes auth-with-oauth2 answer another record, as PocketBase does for a Google
	// identity another account owns.
	foreignID string
	// linked is what the _externalAuths list answers.
	linked bool
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
		payload, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix(), "id": "rec1"})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig",
			"record": map[string]any{"email": "ada@example.com", "name": "ada"},
		})
	})
	mux.HandleFunc("POST /api/collections/users/records", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email, Password, Lang string }
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
		var body struct{ OTPID, Code, Password string }
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
		payload, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix(), "id": "rec1"})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig",
			"record": map[string]any{"email": "ada@example.com", "name": "ada"},
		})
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

func TestSignUpStartsTheVerifyStepAndResendWaits(t *testing.T) {
	f := newCodeFake(t)
	phone, advance := codePhone(t, f)
	if got := codeState(t, phone); got.Step != "" {
		t.Fatalf("before a request: %+v", got)
	}
	if err := phone.SignUp(" ada@example.com ", "a long password", "zh-Hans"); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
	f.mu.Lock()
	signUps := append([]string(nil), f.signUps...)
	f.mu.Unlock()
	if len(signUps) != 1 || signUps[0] != "ada@example.com|a long password|zh-Hans" {
		t.Fatalf("sign-ups %v", signUps)
	}
	if got := codeState(t, phone); got.Step != "verify" || got.Email != "ada@example.com" || got.ResendInSecs != 30 || got.Failure != "" {
		t.Fatalf("after a sign-up: %+v", got)
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
	if got := codeState(t, phone); got.Step != "verify" || got.ResendInSecs != 30 {
		t.Fatalf("after a resend: %+v", got)
	}
	f.mu.Lock()
	n := len(f.signUps)
	f.mu.Unlock()
	if n != 1 {
		t.Fatalf("a resend signed up again (%d)", n)
	}
}

func TestSignUpFailureKinds(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	if err := phone.SignUp("taken@example.com", "a long password", "en"); err == nil {
		t.Fatal("a taken email should fail")
	}
	if got := codeState(t, phone); got.Step != "" || got.Failure != "email_taken" || got.Email != "taken@example.com" {
		t.Fatalf("after a taken email: %+v", got)
	}
	if requests, _, _ := f.counts(); requests != 0 {
		t.Fatalf("a code was requested for a taken email")
	}
}

func TestWrongCodeThenRightCode(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	if err := phone.SignUp("ada@example.com", "a long password", "en"); err != nil {
		t.Fatal(err)
	}
	if _, err := phone.SignInWithCode("000000", "phone"); err == nil {
		t.Fatal("a wrong code should fail")
	}
	if got := codeState(t, phone); got.Step != "verify" || got.Failure != "wrong_code" {
		t.Fatalf("after a wrong code: %+v", got)
	}
	// The right code is accepted; the sign-in then fails at the device call, but the token and the
	// account are stored and the step is over.
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
	if err := phone.SignUp("ada@example.com", "a long password", "en"); err != nil {
		t.Fatal(err)
	}
	advance(10*time.Minute + time.Second)
	if _, err := phone.SignInWithCode("123456", "phone"); err == nil {
		t.Fatal("an old code should not sign in")
	}
	if requests, checks, _ := f.counts(); requests != 2 || checks != 0 {
		t.Fatalf("requests %d checks %d; the old code must not be tried", requests, checks)
	}
	if got := codeState(t, phone); got.Step != "verify" || got.Failure != "expired" || got.ResendInSecs != 30 {
		t.Fatalf("after an old code: %+v", got)
	}
}

func TestSigningInToAnUnverifiedAccountAnswersVerify(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	state, err := phone.SignIn("unverified@example.com", "a long password", "phone")
	if err != nil || state != signInVerifyJSON {
		t.Fatalf("SignIn = %s, %v; want %s", state, err, signInVerifyJSON)
	}
	if got := codeState(t, phone); got.Step != "verify" || got.Email != "unverified@example.com" || got.Failure != "" {
		t.Fatalf("after the refused sign-in: %+v", got)
	}
	if requests, _, _ := f.counts(); requests != 1 {
		t.Fatalf("requests = %d, want the one code mailed", requests)
	}
	if _, err := phone.SignIn("unverified@example.com", "wrong password", "phone"); err == nil {
		t.Fatal("a wrong password should fail")
	}
	if got := codeState(t, phone); got.Failure != "wrong_password" {
		t.Fatalf("after a wrong password: %+v", got)
	}
}

func TestForgotPasswordResetsInTheApp(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	if err := phone.RequestResetCode("ada@example.com"); err != nil {
		t.Fatalf("RequestResetCode: %v", err)
	}
	if got := codeState(t, phone); got.Step != "reset_code" || got.Email != "ada@example.com" {
		t.Fatalf("after the request: %+v", got)
	}
	if _, err := phone.ResetPassword("000000", "a new password", "phone"); err == nil {
		t.Fatal("a wrong code should fail")
	}
	if got := codeState(t, phone); got.Step != "reset_code" || got.Failure != "wrong_code" {
		t.Fatalf("after a wrong code: %+v", got)
	}
	if _, err := phone.ResetPassword("123456", "short", "phone"); err == nil {
		t.Fatal("a short password should fail")
	}
	if got := codeState(t, phone); got.Failure != "weak_password" {
		t.Fatalf("after a short password: %+v", got)
	}
	_, _ = phone.ResetPassword("123456", "a new password", "phone") // the device call then fails
	if got := codeState(t, phone); got.Step != "" || got.Failure != "" {
		t.Fatalf("after the reset: %+v", got)
	}
	if _, _, resets := f.counts(); resets != 3 {
		t.Fatalf("reset calls = %d, want 3", resets)
	}
	if raw, _ := phone.Account(); raw != `{"name":"ada","email":"ada@example.com"}` {
		t.Fatalf("Account = %s", raw)
	}
}

func TestCodeFailureKinds(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	f.mu.Lock()
	f.status = http.StatusTooManyRequests
	f.mu.Unlock()
	if err := phone.RequestResetCode("ada@example.com"); err == nil {
		t.Fatal("a rate-limited request should fail")
	}
	if got := codeState(t, phone); got.Step != "" || got.Failure != "too_many" {
		t.Fatalf("after 429: %+v", got)
	}
}
