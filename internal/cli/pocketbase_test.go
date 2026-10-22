package cli

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
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
		_, _ = w.Write([]byte(`{"auth":"pocketbase"}`))
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

// TestCLILoginThroughPocketBase: `sync login` reads /v1/config, signs in with
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
	if cfg.Auth != syncclient.AuthPocketBase || cfg.Token != "" ||
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

// TestSyncLoginFlags: the flags `sync login` requires, and the flags it no longer has.
func TestSyncLoginFlags(t *testing.T) {
	h := newHarness(t, t.TempDir()+"/x.db")
	h.d.secrets = newCLISecrets()
	h.d.stdin = strings.NewReader("hunter2\n")
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"missing user", []string{"--server", "https://todo.example", "--password-stdin", "--name", "Mac"}, "--user is required"},
		{"missing password-stdin", []string{"--server", "https://todo.example", "--user", "u", "--name", "Mac"}, "--password-stdin is required"},
		{"removed --issuer", []string{"--server", "https://todo.example", "--issuer", "https://i.example",
			"--user", "u", "--password-stdin", "--name", "Mac"}, "flag provided but not defined"},
	} {
		code, _, errOut := runCLI(t, h, append([]string{"sync", "login"}, tt.args...)...)
		if code == 0 || !strings.Contains(errOut, tt.want) {
			t.Errorf("%s: exit %d, stderr %q, want %q", tt.name, code, errOut, tt.want)
		}
	}
}

// TestSyncLoginWrongPassword: a wrong password is one friendly line without the password, and
// writes no sync.json.
func TestSyncLoginWrongPassword(t *testing.T) {
	front := newPocketBaseFront(t)
	path := t.TempDir() + "/wrong.db"
	h := newHarness(t, path)
	h.d.secrets = newCLISecrets()
	const password = "do-not-print-me"
	h.d.stdin = strings.NewReader(password + "\n")
	code, out, errOut := runCLI(t, h, "sync", "login", "--server", front.URL,
		"--user", "cli@example.com", "--password-stdin", "--name", "Mac")
	if code == 0 {
		t.Fatal("a wrong password should fail")
	}
	if !strings.Contains(errOut, "login name or password is wrong") {
		t.Errorf("stderr = %q, want the friendly credentials error", errOut)
	}
	for _, text := range []string{out, errOut} {
		if strings.Contains(text, password) {
			t.Errorf("output contains the password: %q", text)
		}
	}
	oneLineError(t, errOut)
	if _, err := os.Stat(syncclient.ConfigPath(path)); err == nil {
		t.Error("a failed sign-in wrote sync.json")
	}
}

// cliSecrets is an in-memory syncclient.Secrets for the CLI test.
type cliSecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func newCLISecrets() *cliSecrets { return &cliSecrets{values: map[string]string{}} }

func (s *cliSecrets) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return "", syncclient.ErrSecretNotFound
	}
	return value, nil
}

func (s *cliSecrets) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *cliSecrets) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}
