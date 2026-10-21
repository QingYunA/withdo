package syncclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

// fakeAuthRequest is one pending authorization request the fake issuer handed out.
type fakeAuthRequest struct {
	clientID    string
	redirectURI string
	state       string
	challenge   string
}

type fakeSession struct {
	id        string
	token     string
	loginName string
}

// signIn is the fake sync server's POST /v1/sign-in. It returns tokens for the right password and
// can leave one of them out for the client's checks.
func (f *fakeOIDC) signIn(w http.ResponseWriter, r *http.Request) {
	var req syncproto.SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.signInCalls++
	status := f.signInStatus
	missing := f.signInMissing
	login, password := f.loginName, f.password
	f.mu.Unlock()
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	if req.LoginName != login || req.Password != password {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"invalid_credentials"}`)
		return
	}
	if missing != "" {
		answer := map[string]any{
			"access_token": "access-token", "refresh_token": "refresh-token",
			"id_token": "id-token", "token_type": "Bearer", "expires_in": 3600,
		}
		delete(answer, missing)
		writeJSON(w, answer)
		return
	}
	f.respondToken(w, "")
}

// callback builds the loopback URL with a code bound to the auth request's PKCE challenge.
func (f *fakeOIDC) callback(auth *fakeAuthRequest) string {
	code := fmt.Sprintf("code-%d", time.Now().UnixNano())
	f.mu.Lock()
	f.codes[code] = auth.challenge
	f.mu.Unlock()
	redirect, err := url.Parse(auth.redirectURI)
	if err != nil {
		redirect = &url.URL{Scheme: "http", Host: "127.0.0.1", Path: "/callback"}
	}
	query := redirect.Query()
	query.Set("code", code)
	query.Set("state", auth.state)
	redirect.RawQuery = query.Encode()
	return redirect.String()
}

func (f *fakeOIDC) authorized(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer login-pat"
}

func (f *fakeOIDC) zitadelAuthRequest(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	auth, ok := f.authRequests[r.PathValue("id")]
	f.mu.Unlock()
	if !ok {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"code": 5, "message": "not found"})
		return
	}
	writeJSON(w, map[string]any{"authRequest": map[string]any{
		"id": r.PathValue("id"), "clientId": auth.clientID,
		"redirectUri": auth.redirectURI, "scope": []string{"openid", "profile"},
	}})
}

func (f *fakeOIDC) zitadelCreateSession(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var body struct {
		Checks struct {
			User struct {
				LoginName string `json:"loginName"`
			} `json:"user"`
			Password struct {
				Password string `json:"password"`
			} `json:"password"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"code": 3, "message": "bad request"})
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if body.Checks.User.LoginName != f.loginName {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"code": 5, "message": "not found"})
		return
	}
	if body.Checks.Password.Password != f.password {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"code": 3, "message": "password is invalid"})
		return
	}
	f.signInCalls++
	sess := &fakeSession{
		id: fmt.Sprintf("session-%d", f.signInCalls), token: fmt.Sprintf("token-%d", f.signInCalls),
		loginName: body.Checks.User.LoginName,
	}
	f.sessions[sess.id] = sess
	writeJSONStatus(w, http.StatusCreated, map[string]any{
		"sessionId": sess.id, "sessionToken": sess.token, "details": map[string]any{},
	})
}

func (f *fakeOIDC) zitadelSession(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	sess, ok := f.sessions[r.PathValue("id")]
	f.mu.Unlock()
	if !ok {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"code": 5, "message": "not found"})
		return
	}
	writeJSON(w, map[string]any{"session": map[string]any{
		"id": sess.id,
		"factors": map[string]any{
			"user":     map[string]string{"id": f.userID},
			"password": map[string]any{},
		},
	}})
}

func (f *fakeOIDC) zitadelAuthMethods(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	methods := append([]string(nil), f.methods...)
	f.mu.Unlock()
	writeJSON(w, map[string]any{"authMethodTypes": methods})
}

func (f *fakeOIDC) zitadelDeleteSession(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	delete(f.sessions, r.PathValue("id"))
	f.deletedSessions = append(f.deletedSessions, r.PathValue("id"))
	f.mu.Unlock()
	writeJSON(w, map[string]any{"details": map[string]any{}})
}

func (f *fakeOIDC) zitadelCompleteAuthRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session struct {
			SessionID    string `json:"sessionId"`
			SessionToken string `json:"sessionToken"`
		} `json:"session"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"code": 3, "message": "bad request"})
		return
	}
	f.mu.Lock()
	auth, ok := f.authRequests[r.PathValue("id")]
	sess, sessionOK := f.sessions[body.Session.SessionID]
	f.mu.Unlock()
	if !ok || !sessionOK || sess.token != body.Session.SessionToken {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"code": 3, "message": "invalid session"})
		return
	}
	writeJSON(w, map[string]any{"callbackUrl": f.callback(auth), "details": map[string]any{}})
}

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func setSignInStatus(f *fakeOIDC, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.signInStatus = status
}

func requireNoToken(t *testing.T, secrets *fakeSecrets) {
	t.Helper()
	if value, err := secrets.Get(syncclient.RefreshTokenKey); err == nil {
		t.Fatalf("a failed sign-in stored %q", value)
	}
}

// TestSignInPassword covers a successful in-app sign-in: the refresh token is stored and the
// account comes from the ID token.
func TestSignInPassword(t *testing.T) {
	const clientID = "adhd-desktop"
	issuer := newFakeOIDC(t, clientID)
	secrets := newFakeSecrets()
	source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
	if err := source.SignIn(context.Background(), issuer.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	refresh, err := secrets.Get(syncclient.RefreshTokenKey)
	if err != nil || refresh == "" {
		t.Fatalf("refresh token = %q, %v", refresh, err)
	}
	account, ok := source.Account()
	if !ok || account.Name != "Ada Lovelace" {
		t.Fatalf("Account = %+v, %v", account, ok)
	}
	token, err := source.Token(context.Background())
	if err != nil || token == "" {
		t.Fatalf("Token = %q, %v", token, err)
	}
}

// TestSignInOneRequest: the whole sign-in is one POST to the sync server, with no discovery or
// issuer request.
func TestSignInOneRequest(t *testing.T) {
	const clientID = "adhd-desktop"
	issuer := newFakeOIDC(t, clientID)
	secrets := newFakeSecrets()
	source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
	if err := source.SignIn(context.Background(), issuer.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	issuer.mu.Lock()
	hits, calls := issuer.hits, issuer.signInCalls
	issuer.mu.Unlock()
	if hits != 1 || calls != 1 {
		t.Fatalf("requests = %d, sign-in posts = %d, want one of each", hits, calls)
	}
}

// TestSignInMissingToken stores nothing when the answer lacks an access or refresh token.
func TestSignInMissingToken(t *testing.T) {
	const clientID = "adhd-desktop"
	for _, missing := range []string{"access_token", "refresh_token"} {
		t.Run(missing, func(t *testing.T) {
			issuer := newFakeOIDC(t, clientID)
			issuer.mu.Lock()
			issuer.signInMissing = missing
			issuer.mu.Unlock()
			secrets := newFakeSecrets()
			source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
			if err := source.SignIn(context.Background(), issuer.URL, "ada@example.com", "correct horse"); err == nil {
				t.Fatalf("SignIn accepted an answer without %s", missing)
			}
			requireNoToken(t, secrets)
		})
	}
}

// TestSignInStatuses maps the server's answers to the five exported errors and stores nothing.
func TestSignInStatuses(t *testing.T) {
	const clientID = "adhd-desktop"
	tests := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, syncclient.ErrInvalidCredentials},
		{http.StatusLocked, syncclient.ErrAccountLocked},
		{http.StatusForbidden, syncclient.ErrSecondFactor},
		{http.StatusTooManyRequests, syncclient.ErrTooManyAttempts},
		{http.StatusNotFound, syncclient.ErrSignInUnsupported},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status %d", tt.status), func(t *testing.T) {
			issuer := newFakeOIDC(t, clientID)
			setSignInStatus(issuer, tt.status)
			secrets := newFakeSecrets()
			source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
			err := source.SignIn(context.Background(), issuer.URL, "ada@example.com", "correct horse")
			if !errors.Is(err, tt.want) {
				t.Fatalf("SignIn error = %v, want %v", err, tt.want)
			}
			requireNoToken(t, secrets)
		})
	}
}

// TestSignInEndToEnd signs in through the real sync server and the fake Zitadel, then uses the
// access token against the server's OIDC authenticator.
func TestSignInEndToEnd(t *testing.T) {
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
	pool, err := pgxpool.New(ctx, isolatedDatabase(t, databaseURL))
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	server, err := syncserver.NewServer(ctx, pool, auth)
	if err != nil {
		pool.Close()
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(server.Close)
	if err := server.EnablePasswordSignIn(syncserver.PasswordSignInOptions{
		Issuer: issuer.URL, ClientID: clientID, PAT: "login-pat",
	}); err != nil {
		t.Fatalf("EnablePasswordSignIn: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	secrets := newFakeSecrets()
	source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
	if err := source.SignIn(ctx, httpServer.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	token, err := source.Token(ctx)
	if err != nil || token == "" {
		t.Fatalf("Token = %q, %v", token, err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/pull", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	subject, err := auth.Subject(request)
	if err != nil || subject != "user-1" {
		t.Fatalf("OIDCAuthenticator.Subject = %q, %v", subject, err)
	}

	body := strings.NewReader(`{"name":"phone"}`)
	request, err = http.NewRequestWithContext(ctx, http.MethodPost, httpServer.URL+"/v1/devices", body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /v1/devices: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /v1/devices = %d %s", resp.StatusCode, raw)
	}
}
