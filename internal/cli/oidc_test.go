package cli

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// cliOIDC is a small OIDC provider for the CLI test: discovery, an authorize endpoint that hands
// out an auth request, a sign-in endpoint that completes it, and a token endpoint. The CLI's fake
// sync server does not check the token, so an opaque one is enough.
type cliOIDC struct {
	*httptest.Server
	mu       sync.Mutex
	codes    map[string]string // code -> PKCE challenge
	requests map[string]cliAuthRequest
	seq      int
	tokens   int
	authURL  string
}

type cliAuthRequest struct {
	state, challenge, redirectURI string
}

func newCLIOIDC(t *testing.T) *cliOIDC {
	t.Helper()
	f := &cliOIDC{codes: map[string]string{}, requests: map[string]cliAuthRequest{}}
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
	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("redirect_uri") != "http://127.0.0.1/callback" ||
			q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
			http.Error(w, "bad authorize request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.seq++
		id := fmt.Sprintf("V2_%d", f.seq)
		f.requests[id] = cliAuthRequest{
			state: q.Get("state"), challenge: q.Get("code_challenge"),
			redirectURI: q.Get("redirect_uri"),
		}
		f.authURL = r.URL.String()
		f.mu.Unlock()
		http.Redirect(w, r, f.Server.URL+"/ui/v2/login/login?authRequest="+url.QueryEscape(id),
			http.StatusFound)
	})
	mux.HandleFunc("POST /token", f.token)
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// signInHandler is the fake sync server's POST /v1/sign-in.
func (f *cliOIDC) signInHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req syncproto.SignInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		auth, ok := f.requests[req.AuthRequest]
		f.mu.Unlock()
		if !ok || req.LoginName != "cli@example.com" || req.Password != "hunter2" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":"invalid_credentials"}`)
			return
		}
		code := fmt.Sprintf("cli-code-%d", time.Now().UnixNano())
		f.mu.Lock()
		f.codes[code] = auth.challenge
		f.mu.Unlock()
		callback, err := url.Parse(auth.redirectURI)
		if err != nil {
			http.Error(w, "bad redirect", http.StatusInternalServerError)
			return
		}
		query := callback.Query()
		query.Set("code", code)
		query.Set("state", auth.state)
		callback.RawQuery = query.Encode()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"callback_url": callback.String()})
	}
}

func (f *cliOIDC) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	f.mu.Lock()
	ok := false
	if r.PostForm.Get("grant_type") == "refresh_token" {
		ok = r.PostForm.Get("refresh_token") != ""
	} else if r.PostForm.Get("grant_type") == "authorization_code" {
		code, challenge := r.PostForm.Get("code"), f.codes[r.PostForm.Get("code")]
		delete(f.codes, code)
		ok = challenge != "" && pkceFor(r.PostForm.Get("code_verifier")) == challenge
	}
	f.tokens++
	f.mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	claims, _ := json.Marshal(map[string]any{
		"sub": "cli-user", "name": "CLI User", "email": "cli@example.com",
	})
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  "cli-access",
		"id_token":      "e30." + base64.RawURLEncoding.EncodeToString(claims) + ".",
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "cli-refresh",
	})
}

// pkceFor mirrors the client's S256 challenge without importing an unexported helper.
func pkceFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *cliOIDC) lastAuthURL() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.authURL
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

// TestSyncLoginScopes: extra --scope flags reach the authorization request on top of
// "openid offline_access", and sync.json keeps them for later sign-ins and refreshes.
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
	u, err := url.Parse(issuer.lastAuthURL())
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(u.Query().Get("scope"))
	for _, want := range []string{"openid", "offline_access", audience, "profile"} {
		if !slices.Contains(got, want) {
			t.Errorf("authorization scope %q lacks %q", got, want)
		}
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
