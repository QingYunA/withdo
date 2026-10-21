package cli

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// cliOIDC is a small OIDC provider for the CLI test: discovery and a token endpoint for refresh
// grants. `sync login` talks only to the sync server, so it never calls the issuer.
type cliOIDC struct {
	*httptest.Server
}

func newCLIOIDC(t *testing.T) *cliOIDC {
	t.Helper()
	f := &cliOIDC{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                f.Server.URL,
			"jwks_uri":                              f.Server.URL + "/jwks",
			"authorization_endpoint":                f.Server.URL + "/authorize",
			"token_endpoint":                        f.Server.URL + "/token",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("POST /token", f.token)
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// signInHandler is the fake sync server's POST /v1/sign-in: it answers with the tokens the CLI
// stores, without calling the issuer.
func (f *cliOIDC) signInHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req syncproto.SignInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.LoginName != "cli@example.com" || req.Password != "hunter2" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":"invalid_credentials"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "cli-access", "refresh_token": "cli-refresh",
			"id_token": cliIDToken(), "token_type": "Bearer", "expires_in": 3600,
		})
	}
}

func (f *cliOIDC) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.PostForm.Get("grant_type") != "refresh_token" || r.PostForm.Get("refresh_token") == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  "cli-access",
		"id_token":      cliIDToken(),
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "cli-refresh",
	})
}

// cliIDToken has an unsigned payload, which is all the account display reads.
func cliIDToken() string {
	claims, _ := json.Marshal(map[string]any{
		"sub": "cli-user", "name": "CLI User", "email": "cli@example.com",
	})
	return "e30." + base64.RawURLEncoding.EncodeToString(claims) + "."
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

// TestS4a5CLILoginLogout is acceptance criterion 5: `sync login` signs in with a password from
// stdin and links, and `sync logout` deletes the stored refresh token.
func TestS4a5CLILoginLogout(t *testing.T) {
	syncServer := newCLISyncServer(t, true)
	issuer := newCLIOIDC(t)
	syncServer.signIn = issuer.signInHandler()
	path := t.TempDir() + "/login.db"
	h := newHarness(t, path)
	secrets := newCLISecrets()
	h.d.secrets = secrets
	h.d.stdin = strings.NewReader("hunter2\r\n")

	code, out, errOut := runCLI(t, h, "sync", "login",
		"--server", syncServer.URL, "--issuer", issuer.URL, "--client-id", "cli-client",
		"--user", "cli@example.com", "--password-stdin", "--name", "Mac")
	if code != 0 {
		t.Fatalf("login exit %d: %s", code, errOut)
	}
	for _, want := range []string{"signed in", "linked device cli-device"} {
		if !strings.Contains(out, want) {
			t.Errorf("login output lacks %q:\n%s", want, out)
		}
	}
	cfg, found, err := syncclient.LoadConfig(path)
	if err != nil || !found {
		t.Fatalf("sync.json: found = %v, %v", found, err)
	}
	if cfg.Issuer != issuer.URL || cfg.ClientID != "cli-client" || cfg.Token != "" {
		t.Errorf("sync.json = %+v, want issuer, client id and no token", cfg)
	}
	if refresh, err := secrets.Get(syncclient.RefreshTokenKeyFor(path)); err != nil || refresh != "cli-refresh" {
		t.Fatalf("stored refresh token = %q, %v", refresh, err)
	}

	code, out, errOut = runCLI(t, h, "sync", "now")
	if code != 0 {
		t.Fatalf("now exit %d: %s", code, errOut)
	}
	if !strings.Contains(out, "pushed:") || !strings.Contains(out, "pulled:") {
		t.Errorf("now output = %q", out)
	}

	code, out, errOut = runCLI(t, h, "sync", "logout")
	if code != 0 {
		t.Fatalf("logout exit %d: %s", code, errOut)
	}
	if strings.Count(strings.TrimSpace(out), "\n") != 0 || !strings.Contains(out, "signed out") {
		t.Errorf("logout output = %q, want one signed out line", out)
	}
	if _, err := secrets.Get(syncclient.RefreshTokenKeyFor(path)); err == nil {
		t.Fatal("logout kept the refresh token")
	}
}

// TestS3UIa2LoginWritesAccount is S3-UI-a acceptance criterion 2 in the CLI: a password sign-in
// writes the ID token's name and email into sync.json's account object.
func TestS3UIa2LoginWritesAccount(t *testing.T) {
	syncServer := newCLISyncServer(t, true)
	issuer := newCLIOIDC(t)
	syncServer.signIn = issuer.signInHandler()
	path := t.TempDir() + "/account.db"
	h := newHarness(t, path)
	h.d.secrets = newCLISecrets()
	h.d.stdin = strings.NewReader("hunter2\n")

	code, _, errOut := runCLI(t, h, "sync", "login",
		"--server", syncServer.URL, "--issuer", issuer.URL, "--client-id", "cli-client",
		"--user", "cli@example.com", "--password-stdin", "--name", "Mac")
	if code != 0 {
		t.Fatalf("login exit %d: %s", code, errOut)
	}
	cfg, found, err := syncclient.LoadConfig(path)
	if err != nil || !found {
		t.Fatalf("sync.json: found = %v, %v", found, err)
	}
	if cfg.Account == nil {
		t.Fatal("sync.json has no account")
	}
	if cfg.Account.Name != "CLI User" || cfg.Account.Email != "cli@example.com" {
		t.Fatalf("sync.json account = %+v, want CLI User <cli@example.com>", *cfg.Account)
	}
}

// TestSyncLoginScopes: the extra --scope flags are kept in sync.json for later refreshes.
func TestSyncLoginScopes(t *testing.T) {
	syncServer := newCLISyncServer(t, true)
	issuer := newCLIOIDC(t)
	syncServer.signIn = issuer.signInHandler()
	path := t.TempDir() + "/scopes.db"
	h := newHarness(t, path)
	h.d.secrets = newCLISecrets()
	h.d.stdin = strings.NewReader("hunter2\n")

	const audience = "urn:zitadel:iam:org:project:id:42:aud"
	code, _, errOut := runCLI(t, h, "sync", "login", "--server", syncServer.URL, "--issuer", issuer.URL,
		"--client-id", "cli-client", "--user", "cli@example.com", "--password-stdin", "--name", "Mac",
		"--scope", audience, "--scope", "profile")
	if code != 0 {
		t.Fatalf("login exit %d: %s", code, errOut)
	}
	cfg, found, err := syncclient.LoadConfig(path)
	if err != nil || !found {
		t.Fatalf("sync.json: found = %v, %v", found, err)
	}
	if !slices.Equal(cfg.Scopes, []string{audience, "profile"}) {
		t.Errorf("sync.json scopes = %q", cfg.Scopes)
	}
}

// TestSyncLoginPasswordStdin checks the new required flags, a successful sign-in that writes
// sync.json, and that a wrong password never reaches stdout or stderr.
func TestSyncLoginPasswordStdin(t *testing.T) {
	newLogin := func(t *testing.T) (*cliOIDC, *cliSyncServer, *harness, string) {
		t.Helper()
		syncServer := newCLISyncServer(t, true)
		issuer := newCLIOIDC(t)
		syncServer.signIn = issuer.signInHandler()
		path := t.TempDir() + "/stdin.db"
		h := newHarness(t, path)
		h.d.secrets = newCLISecrets()
		return issuer, syncServer, h, path
	}

	t.Run("success writes sync.json", func(t *testing.T) {
		issuer, syncServer, h, path := newLogin(t)
		h.d.stdin = strings.NewReader("hunter2\n")
		code, _, errOut := runCLI(t, h, "sync", "login", "--server", syncServer.URL,
			"--issuer", issuer.URL, "--client-id", "cli-client",
			"--user", "cli@example.com", "--password-stdin", "--name", "Mac")
		if code != 0 {
			t.Fatalf("login exit %d: %s", code, errOut)
		}
		if _, err := os.Stat(syncclient.ConfigPath(path)); err != nil {
			t.Fatalf("sync.json: %v", err)
		}
	})

	t.Run("missing user and password-stdin", func(t *testing.T) {
		issuer, syncServer, h, _ := newLogin(t)
		for _, tt := range []struct {
			name string
			args []string
			want string
		}{
			{"missing user", []string{"--server", syncServer.URL, "--issuer", issuer.URL,
				"--client-id", "cli-client", "--password-stdin", "--name", "Mac"}, "--user is required"},
			{"missing password-stdin", []string{"--server", syncServer.URL, "--issuer", issuer.URL,
				"--client-id", "cli-client", "--user", "cli@example.com", "--name", "Mac"},
				"--password-stdin is required"},
		} {
			code, _, errOut := runCLI(t, h, append([]string{"sync", "login"}, tt.args...)...)
			if code == 0 || !strings.Contains(errOut, tt.want) {
				t.Errorf("%s: exit %d, stderr %q, want %q", tt.name, code, errOut, tt.want)
			}
		}
	})

	t.Run("wrong password is one line without the password", func(t *testing.T) {
		issuer, syncServer, h, path := newLogin(t)
		const password = "do-not-print-me"
		h.d.stdin = strings.NewReader(password + "\n")
		code, out, errOut := runCLI(t, h, "sync", "login", "--server", syncServer.URL,
			"--issuer", issuer.URL, "--client-id", "cli-client",
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
	})
}
