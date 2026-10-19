package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// cliOIDC is a small OIDC provider for the CLI test: discovery, an authorize redirect and a token
// endpoint. The CLI's fake sync server does not check the token, so an opaque one is enough.
type cliOIDC struct {
	*httptest.Server
	mu     sync.Mutex
	codes  map[string]bool
	tokens int
}

func newCLIOIDC(t *testing.T) *cliOIDC {
	t.Helper()
	f := &cliOIDC{codes: map[string]bool{}}
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
		code := fmt.Sprintf("cli-code-%d", time.Now().UnixNano())
		f.mu.Lock()
		f.codes[code] = true
		f.mu.Unlock()
		redirect, err := url.Parse(q.Get("redirect_uri"))
		if err != nil {
			http.Error(w, "bad redirect", http.StatusBadRequest)
			return
		}
		rq := redirect.Query()
		rq.Set("code", code)
		rq.Set("state", q.Get("state"))
		redirect.RawQuery = rq.Encode()
		http.Redirect(w, r, redirect.String(), http.StatusFound)
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		ok := r.PostForm.Get("grant_type") == "refresh_token" && r.PostForm.Get("refresh_token") != ""
		if r.PostForm.Get("grant_type") == "authorization_code" {
			ok = f.codes[r.PostForm.Get("code")]
			delete(f.codes, r.PostForm.Get("code"))
		}
		f.tokens++
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "cli-access", "token_type": "Bearer", "expires_in": 3600,
			"refresh_token": "cli-refresh",
		})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
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

// followCLIBrowser follows the authorization URL like the system browser would.
func followCLIBrowser(t *testing.T) func(string) error {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	return func(authURL string) error {
		resp, err := client.Get(authURL)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("sign-in page returned %d: %s", resp.StatusCode, body)
		}
		return nil
	}
}

// TestS4a5CLILoginLogout is acceptance criterion 5: `sync login` signs in through the fake issuer
// and links, and `sync logout` deletes the stored refresh token.
func TestS4a5CLILoginLogout(t *testing.T) {
	syncServer := newCLISyncServer(t, true)
	issuer := newCLIOIDC(t)
	path := t.TempDir() + "/login.db"
	h := newHarness(t, path)
	secrets := newCLISecrets()
	h.d.secrets = secrets
	h.d.openBrowser = followCLIBrowser(t)

	code, out, errOut := runCLI(t, h, "sync", "login",
		"--server", syncServer.URL, "--issuer", issuer.URL, "--client-id", "cli-client", "--name", "Mac")
	if code != 0 {
		t.Fatalf("login exit %d: %s", code, errOut)
	}
	for _, want := range []string{
		"open this URL to sign in:", "signed in", "linked device cli-device",
	} {
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
	if refresh, err := secrets.Get(syncclient.RefreshTokenKey); err != nil || refresh != "cli-refresh" {
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
	if _, err := secrets.Get(syncclient.RefreshTokenKey); err == nil {
		t.Fatal("logout kept the refresh token")
	}
}
