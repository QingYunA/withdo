package syncclient_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// fakeOIDC is a test OIDC provider: discovery, a JWKS, an authorization endpoint that redirects
// with a code, and a token endpoint for the code and refresh grants. Access tokens are RS256 JWTs
// so the real OIDCAuthenticator can verify them.
type fakeOIDC struct {
	*httptest.Server
	clientID string
	key      *rsa.PrivateKey
	keyID    string

	mu            sync.Mutex
	codes         map[string]string // authorization code -> PKCE challenge
	subject       string
	name          string
	email         string
	expiresIn     int
	refreshRefuse bool
	// rotate accepts only the latest refresh token issued, as Zitadel does; onRefuse runs when a
	// rotated-out token is refused, before the answer.
	rotate    bool
	current   string
	onRefuse  func()
	refreshes int
	accesses  int
	challenge string
	method    string
	subjects  []string

	// In-app sign-in: the auth requests the authorize endpoint handed out, the account the fake
	// Zitadel accepts, the sessions it created, and what /v1/sign-in should answer instead.
	authRequests    map[string]*fakeAuthRequest
	requestSeq      int
	loginName       string
	password        string
	userID          string
	methods         []string
	sessions        map[string]*fakeSession
	deletedSessions []string
	signInStatus    int
	signInCalls     int
	callbackState   string
	callbackHost    string
}

func newFakeOIDC(t *testing.T, clientID string) *fakeOIDC {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeOIDC{
		clientID: clientID, key: key, keyID: "test-key",
		codes: map[string]string{}, subject: "user-1", expiresIn: 3600,
		name: "Ada Lovelace", email: "ada@example.com",
		authRequests: map[string]*fakeAuthRequest{}, sessions: map[string]*fakeSession{},
		loginName: "ada@example.com", password: "correct horse", userID: "user-1",
		methods: []string{"AUTHENTICATION_METHOD_TYPE_PASSWORD"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                                f.Server.URL,
			"jwks_uri":                              f.Server.URL + "/jwks",
			"authorization_endpoint":                f.Server.URL + "/authorize",
			"token_endpoint":                        f.Server.URL + "/token",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: &f.key.PublicKey, KeyID: f.keyID, Algorithm: string(jose.RS256), Use: "sig",
		}}})
	})
	mux.HandleFunc("GET /authorize", f.authorize)
	mux.HandleFunc("POST /token", f.token)
	mux.HandleFunc("POST /v1/sign-in", f.signIn)
	mux.HandleFunc("GET /v2/oidc/auth_requests/{id}", f.zitadelAuthRequest)
	mux.HandleFunc("POST /v2/sessions", f.zitadelCreateSession)
	mux.HandleFunc("GET /v2/sessions/{id}", f.zitadelSession)
	mux.HandleFunc("GET /v2/users/{id}/authentication_methods", f.zitadelAuthMethods)
	mux.HandleFunc("DELETE /v2/sessions/{id}", f.zitadelDeleteSession)
	mux.HandleFunc("POST /v2/oidc/auth_requests/{id}", f.zitadelCompleteAuthRequest)
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func (f *fakeOIDC) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("response_type") != "code" || q.Get("client_id") != f.clientID {
		http.Error(w, "bad authorize request", http.StatusBadRequest)
		return
	}
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" || q.Get("state") == "" {
		http.Error(w, "no PKCE or state", http.StatusBadRequest)
		return
	}
	if q.Get("redirect_uri") == "http://127.0.0.1/callback" {
		f.mu.Lock()
		f.requestSeq++
		id := fmt.Sprintf("V2_%d", f.requestSeq)
		f.authRequests[id] = &fakeAuthRequest{
			clientID: f.clientID, redirectURI: q.Get("redirect_uri"),
			state: q.Get("state"), challenge: q.Get("code_challenge"),
		}
		f.mu.Unlock()
		http.Redirect(w, r, f.Server.URL+"/ui/v2/login/login?authRequest="+url.QueryEscape(id),
			http.StatusFound)
		return
	}
	code := fmt.Sprintf("code-%d", time.Now().UnixNano())
	f.mu.Lock()
	f.codes[code] = q.Get("code_challenge")
	f.challenge, f.method = q.Get("code_challenge"), q.Get("code_challenge_method")
	f.mu.Unlock()
	redirect, err := url.Parse(q.Get("redirect_uri"))
	if err != nil {
		http.Error(w, "bad redirect_uri", http.StatusBadRequest)
		return
	}
	rq := redirect.Query()
	rq.Set("code", code)
	rq.Set("state", q.Get("state"))
	redirect.RawQuery = rq.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (f *fakeOIDC) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	if r.PostForm.Get("client_id") != f.clientID {
		writeOAuthError(w, "invalid_client")
		return
	}
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		code, verifier := r.PostForm.Get("code"), r.PostForm.Get("code_verifier")
		f.mu.Lock()
		challenge, ok := f.codes[code]
		delete(f.codes, code)
		f.mu.Unlock()
		if !ok {
			writeOAuthError(w, "invalid_grant")
			return
		}
		sum := sha256.Sum256([]byte(verifier))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
			writeOAuthError(w, "invalid_grant")
			return
		}
		f.respondToken(w, "")
	case "refresh_token":
		if r.PostForm.Get("refresh_token") == "" {
			writeOAuthError(w, "invalid_grant")
			return
		}
		f.mu.Lock()
		refused := f.refreshRefuse
		stale := f.rotate && r.PostForm.Get("refresh_token") != f.current
		onRefuse := f.onRefuse
		f.mu.Unlock()
		if stale && onRefuse != nil {
			onRefuse()
		}
		if refused || stale {
			writeOAuthError(w, "invalid_grant")
			return
		}
		f.respondToken(w, "refresh")
	default:
		writeOAuthError(w, "unsupported_grant_type")
	}
}

func writeOAuthError(w http.ResponseWriter, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

func (f *fakeOIDC) respondToken(w http.ResponseWriter, grant string) {
	f.mu.Lock()
	f.accesses++
	n := f.accesses
	if grant == "refresh" {
		f.refreshes++
	}
	subject := f.subject
	name := f.name
	email := f.email
	expiresIn := f.expiresIn
	f.subjects = append(f.subjects, subject)
	f.current = fmt.Sprintf("refresh-%d", n)
	f.mu.Unlock()
	exp := time.Now().Add(time.Duration(expiresIn) * time.Second).Unix()
	access, err := f.sign(map[string]any{
		"iss": f.Server.URL, "sub": subject, "aud": f.clientID,
		"iat": time.Now().Unix(), "exp": exp,
	})
	if err != nil {
		http.Error(w, "signing failed", http.StatusInternalServerError)
		return
	}
	idToken, err := f.sign(map[string]any{
		"iss": f.Server.URL, "sub": subject, "aud": f.clientID,
		"iat": time.Now().Unix(), "exp": exp, "name": name, "email": email,
	})
	if err != nil {
		http.Error(w, "signing failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"access_token": access, "id_token": idToken, "token_type": "Bearer", "expires_in": expiresIn,
		"refresh_token": fmt.Sprintf("refresh-%d", n),
	})
}

func (f *fakeOIDC) sign(claims map[string]any) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: f.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", f.keyID))
	if err != nil {
		return "", err
	}
	object, err := signer.Sign(payload)
	if err != nil {
		return "", err
	}
	return object.CompactSerialize()
}

func (f *fakeOIDC) setExpiresIn(seconds int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.expiresIn = seconds
}

func (f *fakeOIDC) refuseRefresh() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshRefuse = true
}

func (f *fakeOIDC) stats() (refreshes, accesses int, method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshes, f.accesses, f.method
}

func (f *fakeOIDC) signedInSubjects() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.subjects...)
}

// fakeSecrets is an in-memory syncclient.Secrets.
type fakeSecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func newFakeSecrets() *fakeSecrets { return &fakeSecrets{values: map[string]string{}} }

func (s *fakeSecrets) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return "", syncclient.ErrSecretNotFound
	}
	return value, nil
}

func (s *fakeSecrets) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *fakeSecrets) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}

// followBrowser returns an opener that behaves like a browser: it follows the authorization URL to
// the loopback redirect and requires the sign-in page.
func followBrowser(t *testing.T) func(url string) error {
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
		if !strings.Contains(string(body), "close this tab") {
			return fmt.Errorf("sign-in page = %q", body)
		}
		return nil
	}
}

// TestS4a2Login is acceptance criterion 2: a fake browser follows the authorization URL to the
// loopback redirect, the refresh token lands in the fake Secrets, a redirect with the wrong state
// does not end the sign-in, and a timeout fails without storing anything.
func TestS4a2Login(t *testing.T) {
	const clientID = "adhd-desktop"
	ctx := context.Background()

	t.Run("browser sign-in uses PKCE and stores the refresh token", func(t *testing.T) {
		issuer := newFakeOIDC(t, clientID)
		secrets := newFakeSecrets()
		source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
		var authURL string
		if err := source.Login(ctx, func(url string) error {
			authURL = url
			return followBrowser(t)(url)
		}); err != nil {
			t.Fatalf("Login: %v", err)
		}
		if !strings.Contains(authURL, "code_challenge_method=S256") {
			t.Errorf("authorization URL lacks S256: %s", authURL)
		}
		refresh, err := secrets.Get(syncclient.RefreshTokenKey)
		if err != nil || refresh == "" {
			t.Fatalf("refresh token = %q, %v", refresh, err)
		}
		if _, _, method := issuer.stats(); method != "S256" {
			t.Errorf("issuer saw code_challenge_method %q, want S256", method)
		}
		token, err := source.Token(ctx)
		if err != nil || token == "" {
			t.Fatalf("Token = %q, %v", token, err)
		}
	})

	t.Run("a redirect with the wrong state is refused and the sign-in goes on", func(t *testing.T) {
		issuer := newFakeOIDC(t, clientID)
		secrets := newFakeSecrets()
		source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
		forged := 0
		err := source.Login(ctx, func(authURL string) error {
			u, err := url.Parse(authURL)
			if err != nil {
				return err
			}
			// A page in the browser probes the loopback port first.
			resp, err := http.Get(u.Query().Get("redirect_uri") + "?code=forged&state=wrong")
			if err != nil {
				return err
			}
			forged = resp.StatusCode
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return followBrowser(t)(authURL)
		})
		if err != nil {
			t.Fatalf("Login after a forged redirect: %v, want the real sign-in to finish", err)
		}
		if forged != http.StatusBadRequest {
			t.Errorf("forged redirect answered %d, want 400", forged)
		}
		if refresh, err := secrets.Get(syncclient.RefreshTokenKey); err != nil || refresh == "" {
			t.Fatalf("refresh token = %q, %v", refresh, err)
		}
	})

	t.Run("a DoneURL sends the browser back to the app, then the exchange finishes", func(t *testing.T) {
		issuer := newFakeOIDC(t, clientID)
		secrets := newFakeSecrets()
		source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
		source.DoneURL = "xyz.sriai.adhdtodo://signed-in"
		var location string
		err := source.Login(ctx, func(authURL string) error {
			// A browser hands the app link to the app: stop at it instead of following it.
			client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, _ []*http.Request) error {
				if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
					return http.ErrUseLastResponse
				}
				return nil
			}}
			resp, err := client.Get(authURL)
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusFound {
				return fmt.Errorf("loopback answered %d, want 302", resp.StatusCode)
			}
			location = resp.Header.Get("Location")
			return nil
		})
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		if location != source.DoneURL {
			t.Errorf("the browser was sent to %q, want %q", location, source.DoneURL)
		}
		if refresh, err := secrets.Get(syncclient.RefreshTokenKey); err != nil || refresh == "" {
			t.Fatalf("refresh token = %q, %v", refresh, err)
		}
	})

	t.Run("timeout fails and stores nothing", func(t *testing.T) {
		issuer := newFakeOIDC(t, clientID)
		secrets := newFakeSecrets()
		source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
		source.LoginTimeout = 100 * time.Millisecond
		if err := source.Login(ctx, func(string) error { return nil }); err == nil ||
			!strings.Contains(err.Error(), "timed out") {
			t.Fatalf("Login error = %v, want a timeout", err)
		}
		if _, err := secrets.Get(syncclient.RefreshTokenKey); err == nil {
			t.Fatal("a timed-out sign-in stored a refresh token")
		}
	})
}

// TestS4a3Refresh is acceptance criterion 3: an access token about to expire is refreshed; a
// refused refresh deletes the stored token, returns ErrSignedOut, and signs a Runner out.
func TestS4a3Refresh(t *testing.T) {
	const clientID = "adhd-desktop"
	ctx := context.Background()

	t.Run("an access token about to expire is refreshed", func(t *testing.T) {
		issuer := newFakeOIDC(t, clientID)
		issuer.setExpiresIn(30)
		secrets := newFakeSecrets()
		source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
		if err := source.Login(ctx, followBrowser(t)); err != nil {
			t.Fatalf("Login: %v", err)
		}
		refreshes, accesses, _ := issuer.stats()
		if refreshes != 0 || accesses != 1 {
			t.Fatalf("after login refreshes = %d, accesses = %d", refreshes, accesses)
		}
		token, err := source.Token(ctx)
		if err != nil || token == "" {
			t.Fatalf("Token = %q, %v", token, err)
		}
		refreshes, accesses, _ = issuer.stats()
		if refreshes != 1 || accesses != 2 {
			t.Fatalf("refreshes = %d, accesses = %d, want one refresh", refreshes, accesses)
		}
	})

	t.Run("a refused refresh deletes the token and signs the runner out", func(t *testing.T) {
		issuer := newFakeOIDC(t, clientID)
		issuer.refuseRefresh()
		secrets := newFakeSecrets()
		if err := secrets.Set(syncclient.RefreshTokenKey, "refresh-old"); err != nil {
			t.Fatal(err)
		}
		source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
		if _, err := source.Token(ctx); !errors.Is(err, syncclient.ErrSignedOut) {
			t.Fatalf("Token error = %v, want ErrSignedOut", err)
		}
		if _, err := secrets.Get(syncclient.RefreshTokenKey); err == nil {
			t.Fatal("a refused refresh kept the stored token")
		}

		clock := newFakeClock()
		client, err := syncclient.New(linkedCore(t), syncclient.Options{
			BaseURL: newRunnerServer(t).URL, TokenSource: source,
		})
		if err != nil {
			t.Fatal(err)
		}
		runner, statuses := startRunner(t, client, clock)
		waitState(t, statuses, syncclient.StateSignedOut)
		if got := runner.Status().State; got != syncclient.StateSignedOut {
			t.Fatalf("runner state = %q, want signed out", got)
		}
	})
}

// TestS4a4EndToEnd is acceptance criterion 4: two devices sign in as the same subject against the
// S2 server with OIDCAuthenticator and sync a task between them.
func TestS4a4EndToEnd(t *testing.T) {
	databaseURL := os.Getenv("ADHD_SYNC_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ADHD_SYNC_TEST_DATABASE_URL is not set")
	}
	const clientID = "adhd-desktop"
	ctx := context.Background()
	issuer := newFakeOIDC(t, clientID)
	auth, err := syncserver.NewOIDCAuthenticator(ctx, issuer.URL, clientID)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator: %v", err)
	}
	server, err := syncserver.Open(ctx, isolatedDatabase(t, databaseURL), auth)
	if err != nil {
		t.Fatalf("open sync server: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(func() {
		httpServer.Close()
		server.Close()
	})

	signIn := func() *syncclient.OIDCSource {
		secrets := newFakeSecrets()
		source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
		if err := source.Login(ctx, followBrowser(t)); err != nil {
			t.Fatalf("Login: %v", err)
		}
		return source
	}

	dbA, _ := testutil.OpenDB(t)
	coreA := domain.New(dbA, domain.Options{})
	task, err := coreA.CreateTask(ctx, domain.CreateTaskInput{Title: "made on A"})
	if err != nil {
		t.Fatal(err)
	}
	clientA, err := syncclient.New(coreA, syncclient.Options{BaseURL: httpServer.URL, TokenSource: signIn()})
	if err != nil {
		t.Fatal(err)
	}
	if err := clientA.Link(ctx, "device A"); err != nil {
		t.Fatalf("link A: %v", err)
	}

	dbB, _ := testutil.OpenDB(t)
	coreB := domain.New(dbB, domain.Options{})
	clientB, err := syncclient.New(coreB, syncclient.Options{BaseURL: httpServer.URL, TokenSource: signIn()})
	if err != nil {
		t.Fatal(err)
	}
	if err := clientB.Link(ctx, "device B"); err != nil {
		t.Fatalf("link B: %v", err)
	}

	got, err := coreB.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("B has no task %d: %v", task.ID, err)
	}
	if got.Title != "made on A" {
		t.Fatalf("B task title = %q", got.Title)
	}
	workspaceA, err := clientA.DeviceWorkspaceID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	workspaceB, err := clientB.DeviceWorkspaceID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if workspaceA != workspaceB {
		t.Fatalf("devices are in workspaces %d and %d, want the same", workspaceA, workspaceB)
	}
	subjects := issuer.signedInSubjects()
	if len(subjects) != 2 || subjects[0] != subjects[1] {
		t.Fatalf("signed-in subjects = %v, want the same subject twice", subjects)
	}
}

// isolatedDatabase returns databaseURL pointing at a new schema this test drops afterwards.
func isolatedDatabase(t *testing.T, databaseURL string) string {
	t.Helper()
	schema := fmt.Sprintf("s4a_e2e_%d", time.Now().UnixNano())
	admin, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to Postgres: %v", err)
	}
	if _, err := admin.Exec(context.Background(), "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_ = admin.Close(context.Background())
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(),
			"DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		_ = admin.Close(context.Background())
	})
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

// TestARotatedRefreshTokenIsReadAgain: the issuer rotates refresh tokens, and another source (the
// app's runner beside its Settings pane, or adhd-core beside the app) refreshed first and stored the
// next one. The refused source reads the stored token again instead of signing out.
func TestARotatedRefreshTokenIsReadAgain(t *testing.T) {
	ctx := context.Background()
	issuer := newFakeOIDC(t, "adhd-desktop")
	secrets := newFakeSecrets()
	first := syncclient.NewOIDCSource(issuer.URL, "adhd-desktop", nil, secrets)
	if err := first.Login(ctx, followBrowser(t)); err != nil {
		t.Fatalf("Login: %v", err)
	}
	issued, err := secrets.Get(syncclient.RefreshTokenKey)
	if err != nil {
		t.Fatal(err)
	}
	issuer.mu.Lock()
	issuer.rotate = true
	issuer.mu.Unlock()

	// The other source refreshes with the issued token, which rotates it and stores the next one.
	other := syncclient.NewOIDCSource(issuer.URL, "adhd-desktop", nil, secrets)
	if _, err := other.Token(ctx); err != nil {
		t.Fatalf("the other source's refresh: %v", err)
	}
	// This source raced it: it read the issued token before the other stored the next one.
	if err := secrets.Set(syncclient.RefreshTokenKey, issued); err != nil {
		t.Fatal(err)
	}
	next := issuer.current
	issuer.mu.Lock()
	issuer.onRefuse = func() { _ = secrets.Set(syncclient.RefreshTokenKey, next) }
	issuer.mu.Unlock()

	racer := syncclient.NewOIDCSource(issuer.URL, "adhd-desktop", nil, secrets)
	if token, err := racer.Token(ctx); err != nil || token == "" {
		t.Fatalf("Token after a rotation elsewhere = %q, %v; want a fresh token", token, err)
	}
	if stored, err := secrets.Get(syncclient.RefreshTokenKey); err != nil || stored == "" {
		t.Fatalf("the refresh token was deleted: %q, %v", stored, err)
	}
}
