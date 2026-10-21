package pbauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

const testPassword = "correct horse battery"

func openApp(t *testing.T) core.App {
	t.Helper()
	app, err := pbauth.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = pbauth.Close(app) })
	return app
}

// serve runs PocketBase with a /v1 handler that answers the caller's subject, and returns its URL.
func serve(t *testing.T, app core.App, auth syncserver.Authenticator) string {
	t.Helper()
	handler := http.NewServeMux()
	handler.HandleFunc("GET /v1/whoami", func(w http.ResponseWriter, r *http.Request) {
		subject, err := auth.Subject(r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, subject)
	})
	handler.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") })
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pbauth.Serve(ctx, app, ln, handler) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("serve: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("serve did not stop")
		}
	})
	return "http://" + ln.Addr().String()
}

func post(t *testing.T, url string, body any) (int, []byte) {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func get(t *testing.T, url, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
}

func signIn(t *testing.T, base, collection, email, password string) (int, string) {
	t.Helper()
	status, raw := post(t, base+"/api/collections/"+collection+"/auth-with-password",
		map[string]string{"identity": email, "password": password})
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(raw, &out)
	return status, out.Token
}

func TestAuthenticatorAcceptsUsersAndRejectsTheRest(t *testing.T) {
	app := openApp(t)
	subject, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(subject, "pb:") || len(subject) < 5 {
		t.Fatalf("subject = %q", subject)
	}
	if err := pbauth.UpsertSuperuser(app, "root@example.com", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))

	status, token := signIn(t, base, "users", "ada@example.com", testPassword)
	if status != http.StatusOK || token == "" {
		t.Fatalf("sign in = %d", status)
	}
	if status, body := get(t, base+"/v1/whoami", token); status != http.StatusOK || body != subject {
		t.Fatalf("whoami = %d %q, want %q", status, body, subject)
	}
	if status, _ := get(t, base+"/v1/whoami", token+"x"); status != http.StatusUnauthorized {
		t.Fatalf("tampered token = %d", status)
	}
	if status, _ := get(t, base+"/v1/whoami", ""); status != http.StatusUnauthorized {
		t.Fatalf("no token = %d", status)
	}
	if status, body := get(t, base+"/healthz", ""); status != http.StatusOK || body != "ok" {
		t.Fatalf("healthz = %d %q", status, body)
	}

	// A superuser is not a sync user.
	status, super := signIn(t, base, "_superusers", "root@example.com", testPassword)
	if status != http.StatusOK || super == "" {
		t.Fatalf("superuser sign in = %d", status)
	}
	if status, _ := get(t, base+"/v1/whoami", super); status != http.StatusUnauthorized {
		t.Fatalf("superuser token on /v1 = %d, want 401", status)
	}
}

func TestSignInFailuresAndClosedSignUp(t *testing.T) {
	app := openApp(t)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))

	if status, _ := signIn(t, base, "users", "ada@example.com", "wrong password"); status != http.StatusBadRequest {
		t.Fatalf("wrong password = %d, want 400", status)
	}
	status, _ := post(t, base+"/api/collections/users/records", map[string]string{
		"email": "eve@example.com", "password": testPassword, "passwordConfirm": testPassword,
	})
	if status != http.StatusForbidden && status != http.StatusNotFound {
		t.Fatalf("sign-up = %d, want it closed", status)
	}
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err == nil {
		t.Fatal("importing the same email twice should fail")
	}
}

func TestMigrationSetsUpUsersCollection(t *testing.T) {
	app := openApp(t)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	if users.CreateRule != nil || users.ListRule != nil || users.UpdateRule != nil {
		t.Error("the users API rules should be closed")
	}
	if !users.PasswordAuth.Enabled || users.OTP.Enabled || users.OAuth2.Enabled {
		t.Error("only password sign-in should be on")
	}
	if got := app.Settings().Meta.AppName; got != "ADHD Todo" {
		t.Errorf("app name = %q", got)
	}
}

func TestChainOrder(t *testing.T) {
	app := openApp(t)
	subject, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, nil)
	_, token := signIn(t, base, "users", "ada@example.com", testPassword)

	chain := syncserver.Chain{
		pbauth.NewAuthenticator(app),
		syncserver.StaticTokens{"dev-token": "dev-subject"},
	}
	for token, want := range map[string]string{token: subject, "dev-token": "dev-subject"} {
		req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		got, err := chain.Subject(req)
		if err != nil || got != want {
			t.Errorf("chain subject = %q, %v; want %q", got, err, want)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.Header.Set("Authorization", "Bearer nope")
	if _, err := chain.Subject(req); !errors.Is(err, syncserver.ErrUnauthenticated) {
		t.Errorf("unknown token error = %v", err)
	}
}
