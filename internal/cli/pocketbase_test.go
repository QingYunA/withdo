package cli

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// newPocketBaseFront puts a /v1/config that names PocketBase and PocketBase's auth route in front
// of the fake sync server, so `sync login` finds out how to sign in from the server alone.
func newPocketBaseFront(t *testing.T) *httptest.Server {
	t.Helper()
	syncServer := newCLISyncServer(t, true)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer":"","client_id":"","scopes":[],"auth":"pocketbase"}`))
	})
	mux.HandleFunc("POST /api/collections/users/auth-with-password", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Identity, Password string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Identity != "cli@example.com" || body.Password != "hunter2" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		claims, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix()})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "e30." + base64.RawURLEncoding.EncodeToString(claims) + ".sig",
			"record": map[string]any{"email": "cli@example.com", "name": "CLI User"},
		})
	})
	mux.Handle("/", syncServer.Config.Handler)
	front := httptest.NewServer(mux)
	t.Cleanup(front.Close)
	return front
}

// TestCLILoginThroughPocketBase: without --issuer, `sync login` reads /v1/config, signs in with
// PocketBase, keeps the token in the keychain and not in sync.json, syncs with it, and `sync
// logout` removes it.
func TestCLILoginThroughPocketBase(t *testing.T) {
	front := newPocketBaseFront(t)
	path := t.TempDir() + "/pb.db"
	h := newHarness(t, path)
	secrets := newCLISecrets()
	h.d.secrets = secrets
	h.d.stdin = strings.NewReader("hunter2\n")

	code, out, errOut := runCLI(t, h, "sync", "login", "--server", front.URL,
		"--user", "cli@example.com", "--password-stdin", "--name", "Mac")
	if code != 0 {
		t.Fatalf("login exit %d: %s", code, errOut)
	}
	if !strings.Contains(out, "linked device cli-device") {
		t.Errorf("login output = %q", out)
	}
	cfg, found, err := syncclient.LoadConfig(path)
	if err != nil || !found {
		t.Fatalf("sync.json: found = %v, %v", found, err)
	}
	if cfg.Auth != syncclient.AuthPocketBase || cfg.Issuer != "" || cfg.Token != "" ||
		cfg.Account == nil || cfg.Account.Email != "cli@example.com" {
		t.Fatalf("sync.json = %+v", cfg)
	}
	if !syncclient.HasPocketBaseToken(secrets, path) {
		t.Fatal("the token is not in the keychain")
	}
	if code, out, errOut = runCLI(t, h, "sync", "now"); code != 0 || !strings.Contains(out, "pushed:") {
		t.Fatalf("now exit %d: %s %s", code, out, errOut)
	}
	if code, _, errOut = runCLI(t, h, "sync", "logout"); code != 0 {
		t.Fatalf("logout exit %d: %s", code, errOut)
	}
	if syncclient.HasPocketBaseToken(secrets, path) {
		t.Fatal("logout kept the token")
	}
}

func TestCLILoginIssuerNeedsClientID(t *testing.T) {
	h := newHarness(t, t.TempDir()+"/x.db")
	h.d.secrets = newCLISecrets()
	h.d.stdin = strings.NewReader("hunter2\n")
	code, _, errOut := runCLI(t, h, "sync", "login", "--server", "https://todo.example", "--issuer", "https://i.example",
		"--user", "u", "--password-stdin", "--name", "Mac")
	if code == 0 || !strings.Contains(errOut, "--issuer and --client-id go together") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
}
