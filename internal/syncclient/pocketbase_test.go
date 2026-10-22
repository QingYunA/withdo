package syncclient_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	mu          sync.Mutex
	email       string
	password    string
	lifetime    time.Duration
	signInCalls int
	refreshes   int
	status      int // when not 0, every request answers this status
	issued      int
	valid       map[string]bool
}

func newFakePocketBase(t *testing.T) *fakePocketBase {
	t.Helper()
	f := &fakePocketBase{
		email: "ada@example.com", password: "correct horse", lifetime: 7 * 24 * time.Hour,
		valid: map[string]bool{},
	}
	mux := http.NewServeMux()
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
	cfg.Issuer, cfg.ClientID = "https://old.example", "client"
	if err := cfg.ApplyPublic(syncproto.ConfigResponse{Auth: "pocketbase"}); err != nil {
		t.Fatal(err)
	}
	if cfg.Auth != syncclient.AuthPocketBase || cfg.Issuer != "" || cfg.ClientID != "" {
		t.Fatalf("cfg = %+v; PocketBase should replace the issuer", cfg)
	}
	if _, ok := syncclient.NewAccountSource(cfg, newFakeSecrets(), "").(*syncclient.PocketBaseSource); !ok {
		t.Fatal("auth pocketbase did not give a PocketBase source")
	}
	// A server that names an issuer and no auth is the old Zitadel setup.
	if err := cfg.ApplyPublic(syncproto.ConfigResponse{Issuer: "https://issuer.example/", ClientID: "client"}); err != nil {
		t.Fatal(err)
	}
	if cfg.Auth != "" || cfg.Issuer != "https://issuer.example" {
		t.Fatalf("cfg = %+v; the issuer should replace PocketBase", cfg)
	}
	if _, ok := syncclient.NewAccountSource(cfg, newFakeSecrets(), "").(*syncclient.OIDCSource); !ok {
		t.Fatal("an issuer did not give an OIDC source")
	}
	if err := cfg.ApplyPublic(syncproto.ConfigResponse{}); !errors.Is(err, syncclient.ErrSignInUnsupported) {
		t.Fatalf("a server with no sign-in: err = %v", err)
	}
}
