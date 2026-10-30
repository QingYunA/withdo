package syncclient_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// fakePocketBase answers PocketBase's two auth routes for one account. Its tokens are unsigned
// JWTs with an exp claim, which is all the client reads.
type fakePocketBase struct {
	*httptest.Server
	mux         *http.ServeMux
	mu          sync.Mutex
	email       string
	password    string
	lifetime    time.Duration
	signInCalls int
	refreshes   int
	status      int // when not 0, every request answers this status
	issued      int
	valid       map[string]bool
	lastSignUp  string
}

func newFakePocketBase(t *testing.T) *fakePocketBase {
	t.Helper()
	f := &fakePocketBase{
		email: "ada@example.com", password: "correct horse", lifetime: 7 * 24 * time.Hour,
		valid: map[string]bool{},
	}
	mux := http.NewServeMux()
	f.mux = mux
	mux.HandleFunc("POST /api/collections/users/auth-with-password", f.authWithPassword)
	mux.HandleFunc("POST /api/collections/users/auth-refresh", f.authRefresh)
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakePocketBase) mint() string {
	f.issued++
	payload, _ := json.Marshal(map[string]any{
		"id": "rec1", "type": "auth", "n": f.issued, "exp": time.Now().Add(f.lifetime).Unix(),
	})
	token := "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	f.valid[token] = true
	return token
}

func (f *fakePocketBase) answer(w http.ResponseWriter, token string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":  token,
		"record": map[string]any{"id": "rec1", "email": f.email, "name": "Ada"},
	})
}

func (f *fakePocketBase) authWithPassword(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.signInCalls++
	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}
	var body struct{ Identity, Password string }
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Identity != f.email || body.Password != f.password {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"status":400,"message":"Failed to authenticate."}`)
		return
	}
	f.answer(w, f.mint())
}

func (f *fakePocketBase) authRefresh(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshes++
	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}
	if !f.valid[r.Header.Get("Authorization")] {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.answer(w, f.mint())
}

func (f *fakePocketBase) setLifetime(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lifetime = d
}

func (f *fakePocketBase) setStatus(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = status
}

func (f *fakePocketBase) counts() (signIns, refreshes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.signInCalls, f.refreshes
}

func TestPocketBaseSignInStoresTheTokenAndTheAccount(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	secrets := newFakeSecrets()
	source := syncclient.NewPocketBaseSource(pb.URL, secrets)

	if _, err := source.Token(ctx); !errors.Is(err, syncclient.ErrSignedOut) {
		t.Fatalf("Token before sign-in = %v, want ErrSignedOut", err)
	}
	if err := source.SignIn(ctx, pb.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	token, err := source.Token(ctx)
	if err != nil || token == "" {
		t.Fatalf("Token = %q, %v", token, err)
	}
	if signIns, refreshes := pb.counts(); signIns != 1 || refreshes != 0 {
		t.Fatalf("calls = %d sign-ins, %d refreshes; a fresh token needs no refresh", signIns, refreshes)
	}
	if account, ok := source.Account(); !ok || account.Email != "ada@example.com" || account.Name != "Ada" {
		t.Fatalf("Account = %+v, %v", account, ok)
	}

	// A second source over the same secrets (the app restarting) finds the token without a sign-in.
	again := syncclient.NewPocketBaseSource(pb.URL, secrets)
	if got, err := again.Token(ctx); err != nil || got != token {
		t.Fatalf("Token after restart = %q, %v; want the stored token", got, err)
	}
}

func TestPocketBaseSignInFailures(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name     string
		status   int
		password string
		want     error
	}{
		{"wrong password", 0, "nope", syncclient.ErrInvalidCredentials},
		{"rate limited", http.StatusTooManyRequests, "correct horse", syncclient.ErrTooManyAttempts},
		{"not a PocketBase server", http.StatusNotFound, "correct horse", syncclient.ErrSignInUnsupported},
		{"locked", http.StatusForbidden, "correct horse", syncclient.ErrAccountLocked},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pb := newFakePocketBase(t)
			pb.setStatus(tt.status)
			secrets := newFakeSecrets()
			source := syncclient.NewPocketBaseSource(pb.URL, secrets)
			err := source.SignIn(ctx, pb.URL, "ada@example.com", tt.password)
			if !errors.Is(err, tt.want) {
				t.Fatalf("SignIn error = %v, want %v", err, tt.want)
			}
			if len(secrets.values) != 0 {
				t.Fatalf("a failed sign-in stored %v", secrets.values)
			}
			if strings.Contains(err.Error(), "nope") {
				t.Fatalf("the error repeats the password: %v", err)
			}
		})
	}
}

func TestPocketBaseRefreshesATokenWithLessThanADayLeft(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	pb.setLifetime(2 * time.Hour)
	source := syncclient.NewPocketBaseSource(pb.URL, newFakeSecrets())
	if err := source.SignIn(ctx, pb.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatal(err)
	}
	first, _ := source.Token(ctx) // already inside the window: refreshed at once
	if _, refreshes := pb.counts(); refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", refreshes)
	}
	pb.setLifetime(7 * 24 * time.Hour)
	second, err := source.Token(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("the token was not replaced")
	}
	third, _ := source.Token(ctx)
	if third != second {
		t.Fatal("a token with a week left was refreshed again")
	}
}

func TestPocketBaseRefreshRefusedSignsOut(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	pb.setLifetime(time.Hour)
	secrets := newFakeSecrets()
	source := syncclient.NewPocketBaseSource(pb.URL, secrets)
	if err := source.SignIn(ctx, pb.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatal(err)
	}
	pb.setStatus(http.StatusUnauthorized) // the account was removed or the password changed
	if _, err := source.Token(ctx); !errors.Is(err, syncclient.ErrSignedOut) {
		t.Fatalf("Token = %v, want ErrSignedOut", err)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("the refused token stayed in the store: %v", secrets.values)
	}
}

func TestPocketBaseKeepsAValidTokenWhenRenewalFails(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	pb.setLifetime(2 * time.Hour)
	source := syncclient.NewPocketBaseSource(pb.URL, newFakeSecrets())
	if err := source.SignIn(ctx, pb.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatal(err)
	}
	pb.setStatus(http.StatusInternalServerError)
	if token, err := source.Token(ctx); err != nil || token == "" {
		t.Fatalf("a valid token was dropped on a server error: %q, %v", token, err)
	}

	// An expired one cannot be kept.
	pb.setStatus(0)
	pb.setLifetime(-time.Minute)
	expired := syncclient.NewPocketBaseSource(pb.URL, newFakeSecrets())
	if err := expired.SignIn(ctx, pb.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatal(err)
	}
	pb.setStatus(http.StatusInternalServerError)
	if _, err := expired.Token(ctx); err == nil || errors.Is(err, syncclient.ErrSignedOut) {
		t.Fatalf("an expired token with the server down: err = %v, want a plain error", err)
	}
}

func TestPocketBaseLogoutAndPerDatabaseKeys(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	secrets := newFakeSecrets()
	a := syncclient.NewPocketBaseSource(pb.URL, secrets)
	a.DBPath = "/data/one/adhd-todo.db"
	b := syncclient.NewPocketBaseSource(pb.URL, secrets)
	b.DBPath = "/data/two/adhd-todo.db"
	for _, s := range []*syncclient.PocketBaseSource{a, b} {
		if err := s.SignIn(ctx, pb.URL, "ada@example.com", "correct horse"); err != nil {
			t.Fatal(err)
		}
	}
	if len(secrets.values) != 2 {
		t.Fatalf("two databases stored %d tokens, want 2", len(secrets.values))
	}
	if !syncclient.HasPocketBaseToken(secrets, "/data/one/adhd-todo.db") {
		t.Fatal("HasPocketBaseToken is false after sign-in")
	}
	if err := a.Logout(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Token(ctx); !errors.Is(err, syncclient.ErrSignedOut) {
		t.Fatalf("Token after logout = %v", err)
	}
	if _, ok := a.Account(); ok {
		t.Fatal("the account survived logout")
	}
	if _, err := b.Token(ctx); err != nil {
		t.Fatalf("the other database was signed out too: %v", err)
	}
	if err := syncclient.DeletePocketBaseToken(secrets, "/data/two/adhd-todo.db"); err != nil {
		t.Fatal(err)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("tokens left: %v", secrets.values)
	}
}

func TestConfigPicksTheSourceFromTheServer(t *testing.T) {
	var cfg syncclient.Config
	cfg.Server = "https://todo.example"
	if err := cfg.ApplyPublic(syncproto.ConfigResponse{Auth: "pocketbase"}); err != nil {
		t.Fatal(err)
	}
	if cfg.Auth != syncclient.AuthPocketBase {
		t.Fatalf("cfg = %+v; the server names PocketBase", cfg)
	}
	if _, ok := syncclient.NewAccountSource(cfg, newFakeSecrets(), "").(*syncclient.PocketBaseSource); !ok {
		t.Fatal("auth pocketbase did not give a PocketBase source")
	}
	if err := cfg.ApplyPublic(syncproto.ConfigResponse{}); !errors.Is(err, syncclient.ErrSignInUnsupported) {
		t.Fatalf("a server with no sign-in: err = %v", err)
	}
}

// TestAnOlderSyncJSONWithAnIssuerLoadsSignedOut: a sync.json written by an earlier version
// (an issuer, a client id, a signed-in account, no token) loads without error as a
// PocketBase config with no account, so the pane shows the sign-in card.
func TestAnOlderSyncJSONWithAnIssuerLoadsSignedOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), syncclient.ConfigFile)
	older := `{"server":"https://todo.example/","issuer":"https://auth.example","client_id":"adhd","scopes":["aud"],` +
		`"account":{"name":"Ada","email":"ada@example.com"}}`
	if err := os.WriteFile(path, []byte(older), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, found, err := syncclient.LoadConfigAt(path)
	if err != nil || !found {
		t.Fatalf("LoadConfigAt = %+v, %v, %v", cfg, found, err)
	}
	if cfg.Server != "https://todo.example" || cfg.Auth != syncclient.AuthPocketBase || cfg.Account != nil || cfg.Token != "" {
		t.Fatalf("cfg = %+v; want a signed-out PocketBase config", cfg)
	}
	source := syncclient.NewAccountSource(cfg, newFakeSecrets(), filepath.Join(t.TempDir(), "adhd-todo.db"))
	if _, err := source.Token(context.Background()); !errors.Is(err, syncclient.ErrSignedOut) {
		t.Fatalf("Token = %v, want ErrSignedOut", err)
	}
	if _, ok := source.Account(); ok {
		t.Fatal("a signed-out source has an account")
	}
	// Saving it back drops the old keys.
	if err := syncclient.SaveConfigAt(path, cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "issuer") {
		t.Fatalf("the saved file keeps the issuer: %s", raw)
	}
}

// fakeCodeServer answers the routes of the sign-up: users create (an email that is taken or
// malformed, or a short password, answers PocketBase's 400 with the field's code), request-otp hands
// out an id, auth-with-otp and the reset route accept only the code 123456 for that id. A non-zero
// status makes request-otp, auth-with-otp and the reset route answer it.
func fakeCodeServer(t *testing.T, f *fakePocketBase, status int) {
	t.Helper()
	fieldError := func(w http.ResponseWriter, field, code string) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `{"status":400,"message":"Failed to create record.","data":{%q:{"code":%q,"message":"x"}}}`, field, code)
	}
	f.mux.HandleFunc("POST /api/collections/users/records", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email, Password, PasswordConfirm, Lang string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.lastSignUp = body.Email + "|" + body.Password + "|" + body.PasswordConfirm + "|" + body.Lang
		f.mu.Unlock()
		switch {
		case body.Email == f.email:
			fieldError(w, "email", "validation_not_unique")
		case !strings.Contains(body.Email, "@"):
			fieldError(w, "email", "validation_is_email")
		case len(body.Password) < 8:
			fieldError(w, "password", "validation_min_text_constraint")
		default:
			_, _ = fmt.Fprint(w, `{"id":"rec2","email":"`+body.Email+`","verified":false}`)
		}
	})
	f.mux.HandleFunc("POST /api/collections/users/request-otp", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		if !strings.Contains(body.Email, "@") {
			fieldError(w, "email", "validation_is_email")
			return
		}
		_, _ = fmt.Fprint(w, `{"otpId":"otp1"}`)
	})
	f.mux.HandleFunc("POST /api/collections/users/auth-with-otp", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OTPID    string `json:"otpId"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		if body.OTPID != "otp1" || body.Password != "123456" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.answer(w, f.mint())
	})
	f.mux.HandleFunc("POST /api/withdo/reset-password", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ OTPID, Code, Password, PasswordConfirm string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		if len(body.Password) < 8 {
			fieldError(w, "password", "validation_min_text_constraint")
			return
		}
		if body.OTPID != "otp1" || body.Code != "123456" || body.Password != body.PasswordConfirm {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"status":400,"message":"Invalid or expired OTP.","data":{}}`)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.answer(w, f.mint())
	})
}

func TestPocketBaseSignUpAndVerify(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	fakeCodeServer(t, pb, 0)
	secrets := newFakeSecrets()
	source := syncclient.NewPocketBaseSource(pb.URL, secrets)
	var _ syncclient.CodeSignIn = source

	otpID, err := source.SignUp(ctx, pb.URL, " grace@example.com ", "a long password", "zh-Hans")
	if err != nil || otpID != "otp1" {
		t.Fatalf("SignUp = %q, %v", otpID, err)
	}
	pb.mu.Lock()
	sent := pb.lastSignUp
	pb.mu.Unlock()
	if sent != "grace@example.com|a long password|a long password|zh-Hans" {
		t.Fatalf("the sign-up body = %q", sent)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("a sign-up stored %v before the code", secrets.values)
	}
	if err := source.SignInWithCode(ctx, pb.URL, otpID, "000000"); !errors.Is(err, syncclient.ErrInvalidCode) {
		t.Fatalf("wrong code = %v, want ErrInvalidCode", err)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("a wrong code stored %v", secrets.values)
	}
	if err := source.SignInWithCode(ctx, pb.URL, otpID, " 123456 "); err != nil {
		t.Fatalf("SignInWithCode: %v", err)
	}
	if token, err := source.Token(ctx); err != nil || token == "" {
		t.Fatalf("Token = %q, %v", token, err)
	}
	if account, ok := source.Account(); !ok || account.Email != "ada@example.com" {
		t.Fatalf("Account = %+v, %v", account, ok)
	}
}

func TestPocketBaseSignUpErrors(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	fakeCodeServer(t, pb, 0)
	source := syncclient.NewPocketBaseSource(pb.URL, newFakeSecrets())
	for _, tt := range []struct {
		email, password string
		want            error
	}{
		{"ada@example.com", "a long password", syncclient.ErrEmailTaken},
		{"nope", "a long password", syncclient.ErrInvalidEmail},
		{"grace@example.com", "short", syncclient.ErrWeakPassword},
	} {
		if _, err := source.SignUp(ctx, pb.URL, tt.email, tt.password, "en"); !errors.Is(err, tt.want) {
			t.Errorf("SignUp(%q, %q) = %v, want %v", tt.email, tt.password, err, tt.want)
		}
	}
}

func TestPocketBaseRequestCodeMalformedEmail(t *testing.T) {
	pb := newFakePocketBase(t)
	fakeCodeServer(t, pb, 0)
	source := syncclient.NewPocketBaseSource(pb.URL, newFakeSecrets())
	if _, err := source.RequestCode(context.Background(), pb.URL, "nope"); !errors.Is(err, syncclient.ErrInvalidEmail) {
		t.Fatalf("bad email = %v, want ErrInvalidEmail (and not ErrInvalidCode)", err)
	}
}

func TestPocketBaseUnverifiedSignIn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"status":403,"message":"The email is not verified.","data":{"email":{"code":"email_not_verified","message":"x"}}}`)
	}))
	t.Cleanup(srv.Close)
	source := syncclient.NewPocketBaseSource(srv.URL, newFakeSecrets())
	err := source.SignIn(context.Background(), srv.URL, "ada@example.com", "correct horse")
	if !errors.Is(err, syncclient.ErrEmailNotVerified) || errors.Is(err, syncclient.ErrAccountLocked) {
		t.Fatalf("unverified sign-in = %v, want ErrEmailNotVerified", err)
	}
}

func TestPocketBaseResetPassword(t *testing.T) {
	ctx := context.Background()
	pb := newFakePocketBase(t)
	fakeCodeServer(t, pb, 0)
	secrets := newFakeSecrets()
	source := syncclient.NewPocketBaseSource(pb.URL, secrets)

	if err := source.ResetPassword(ctx, pb.URL, "otp1", "000000", "a long password"); !errors.Is(err, syncclient.ErrInvalidCode) {
		t.Fatalf("wrong code = %v, want ErrInvalidCode", err)
	}
	if err := source.ResetPassword(ctx, pb.URL, "ghost", "123456", "a long password"); !errors.Is(err, syncclient.ErrInvalidCode) {
		t.Fatalf("unknown otp = %v, want ErrInvalidCode", err)
	}
	if err := source.ResetPassword(ctx, pb.URL, "otp1", "123456", "short"); !errors.Is(err, syncclient.ErrWeakPassword) {
		t.Fatalf("short password = %v, want ErrWeakPassword", err)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("a refused reset stored %v", secrets.values)
	}
	if err := source.ResetPassword(ctx, pb.URL, "otp1", "123456", "a long password"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if token, err := source.Token(ctx); err != nil || token == "" {
		t.Fatalf("Token = %q, %v", token, err)
	}
}

func TestPocketBaseCodeSignInFailures(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		status int
		want   error
	}{
		{http.StatusTooManyRequests, syncclient.ErrTooManyAttempts},
		{http.StatusNotFound, syncclient.ErrSignInUnsupported},
		{http.StatusForbidden, syncclient.ErrAccountLocked},
	} {
		pb := newFakePocketBase(t)
		fakeCodeServer(t, pb, tt.status)
		source := syncclient.NewPocketBaseSource(pb.URL, newFakeSecrets())
		if _, err := source.RequestCode(ctx, pb.URL, "ada@example.com"); !errors.Is(err, tt.want) {
			t.Errorf("RequestCode %d = %v, want %v", tt.status, err, tt.want)
		}
		if err := source.SignInWithCode(ctx, pb.URL, "otp1", "123456"); !errors.Is(err, tt.want) {
			t.Errorf("SignInWithCode %d = %v, want %v", tt.status, err, tt.want)
		}
		if err := source.ResetPassword(ctx, pb.URL, "otp1", "123456", "a long password"); !errors.Is(err, tt.want) {
			t.Errorf("ResetPassword %d = %v, want %v", tt.status, err, tt.want)
		}
	}
}
