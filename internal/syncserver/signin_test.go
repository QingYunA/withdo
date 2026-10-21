package syncserver_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

// fakeZitadel implements the Session API calls and the OIDC endpoints the sync server drives: an
// authorize endpoint that hands out an auth request, the token endpoint that redeems its code, and
// the session calls that check the password.
type fakeZitadel struct {
	*httptest.Server
	mux *http.ServeMux
	// internal is the address of an on-network server the sync server can be pointed at; it must
	// receive the issuer's host.
	internal *httptest.Server

	mu       sync.Mutex
	clientID string
	pat      string

	loginName string
	password  string
	userID    string
	methods   []string
	locked    bool

	authRequests map[string]fakeAuthRequest
	sessions     map[string]fakeSession
	codes        map[string]fakeCode
	created      int
	deleted      []string
	next         int

	authorizeFailure bool
	authorizeCalls   int
	tokenCalls       int
	callbackState    string
	callbackHost     string
	publicHits       int
	internalHits     int
}

type fakeAuthRequest struct {
	ID          string
	ClientID    string
	RedirectURI string
	State       string
	Challenge   string
	Scope       []string
}

type fakeSession struct {
	ID        string
	Token     string
	LoginName string
}

type fakeCode struct {
	challenge   string
	redirectURI string
	clientID    string
}

func newFakeZitadel(t *testing.T, clientID string) *fakeZitadel {
	t.Helper()
	f := &fakeZitadel{
		clientID: clientID, pat: "login-pat",
		loginName: "ada@example.com", password: "correct horse", userID: "user-1",
		methods: []string{"AUTHENTICATION_METHOD_TYPE_PASSWORD"},
		authRequests: map[string]fakeAuthRequest{
			"V2_1": {
				ID: "V2_1", ClientID: clientID, RedirectURI: "http://127.0.0.1/callback",
				State: "state-1", Scope: []string{"openid", "profile"},
			},
		},
		sessions: map[string]fakeSession{},
		codes:    map[string]fakeCode{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth/v2/authorize", f.authorize)
	mux.HandleFunc("POST /oauth/v2/token", f.token)
	mux.HandleFunc("GET /v2/oidc/auth_requests/{id}", f.getAuthRequest)
	mux.HandleFunc("POST /v2/sessions", f.createSession)
	mux.HandleFunc("GET /v2/sessions/{id}", f.getSession)
	mux.HandleFunc("GET /v2/users/{id}/authentication_methods", f.authMethods)
	mux.HandleFunc("DELETE /v2/sessions/{id}", f.deleteSession)
	mux.HandleFunc("POST /v2/oidc/auth_requests/{id}", f.completeAuthRequest)
	f.mux = mux
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.publicHits++
		f.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(f.Close)
	return f
}

// startInternal serves the same fake on another address and refuses a request whose Host is not
// the issuer's, which is what Zitadel's Traefik network address needs.
func (f *fakeZitadel) startInternal(t *testing.T) *httptest.Server {
	t.Helper()
	issuer, err := url.Parse(f.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.internalHits++
		f.mu.Unlock()
		if r.Host != issuer.Host {
			http.Error(w, "wrong host: "+r.Host, http.StatusBadRequest)
			return
		}
		f.mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func (f *fakeZitadel) authorize(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.authorizeCalls++
	fail := f.authorizeFailure
	f.mu.Unlock()
	if fail {
		writeZitadelError(w, http.StatusInternalServerError, 13, "authorize failed")
		return
	}
	query := r.URL.Query()
	if query.Get("client_id") != f.clientID ||
		query.Get("redirect_uri") != "http://127.0.0.1/callback" ||
		query.Get("response_type") != "code" ||
		query.Get("code_challenge_method") != "S256" ||
		query.Get("code_challenge") == "" || query.Get("state") == "" ||
		!strings.Contains(query.Get("scope"), "openid") ||
		!strings.Contains(query.Get("scope"), "offline_access") {
		http.Error(w, "bad authorize request", http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.next++
	id := fmt.Sprintf("V2_%d", f.next)
	f.authRequests[id] = fakeAuthRequest{
		ID: id, ClientID: query.Get("client_id"), RedirectURI: query.Get("redirect_uri"),
		State: query.Get("state"), Challenge: query.Get("code_challenge"),
		Scope: strings.Fields(query.Get("scope")),
	}
	f.mu.Unlock()
	// Zitadel builds the login URL from its public issuer, whichever address answered.
	http.Redirect(w, r, "https://auth.example/ui/v2/login/login?authRequest="+url.QueryEscape(id),
		http.StatusFound)
}

func (f *fakeZitadel) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	form := r.PostForm
	f.mu.Lock()
	f.tokenCalls++
	code, ok := f.codes[form.Get("code")]
	if ok {
		delete(f.codes, form.Get("code"))
	}
	f.mu.Unlock()
	if !ok || form.Get("grant_type") != "authorization_code" ||
		form.Get("redirect_uri") != "http://127.0.0.1/callback" ||
		form.Get("client_id") != f.clientID ||
		pkceFor(form.Get("code_verifier")) != code.challenge {
		writeZitadelError(w, http.StatusBadRequest, 3, "invalid code")
		return
	}
	writeZitadelJSON(w, http.StatusOK, map[string]any{
		"access_token": "access-token", "refresh_token": "refresh-token",
		"id_token": "id-token", "token_type": "Bearer", "expires_in": 3600,
	})
}

func pkceFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *fakeZitadel) authorized(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer "+f.pat
}

func writeZitadelError(w http.ResponseWriter, status, code int, message string, details ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]any{"code": code, "message": message}
	if len(details) > 0 {
		body["details"] = details
	}
	_ = json.NewEncoder(w).Encode(body)
}

func writeZitadelJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (f *fakeZitadel) getAuthRequest(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		writeZitadelError(w, http.StatusUnauthorized, 16, "unauthenticated")
		return
	}
	f.mu.Lock()
	ar, ok := f.authRequests[r.PathValue("id")]
	f.mu.Unlock()
	if !ok {
		writeZitadelError(w, http.StatusNotFound, 5, "auth request not found")
		return
	}
	writeZitadelJSON(w, http.StatusOK, map[string]any{"authRequest": map[string]any{
		"id": ar.ID, "clientId": ar.ClientID, "redirectUri": ar.RedirectURI, "scope": ar.Scope,
	}})
}

func (f *fakeZitadel) createSession(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		writeZitadelError(w, http.StatusUnauthorized, 16, "unauthenticated")
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
		writeZitadelError(w, http.StatusBadRequest, 3, "bad request")
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if body.Checks.User.LoginName != f.loginName {
		writeZitadelError(w, http.StatusNotFound, 5, "user not found")
		return
	}
	if f.locked {
		writeZitadelError(w, http.StatusBadRequest, 9, "Errors.User.NotActive (SESSION-Gj4ko)")
		return
	}
	if body.Checks.Password.Password != f.password {
		writeZitadelError(w, http.StatusBadRequest, 3, "password is invalid",
			map[string]string{"@type": "type.googleapis.com/zitadel.v1.CredentialsCheckError"})
		return
	}
	f.next++
	id := fmt.Sprintf("session-%d", f.next)
	sess := fakeSession{ID: id, Token: "session-token-" + id, LoginName: f.loginName}
	f.sessions[id] = sess
	f.created++
	writeZitadelJSON(w, http.StatusCreated, map[string]any{
		"sessionId": sess.ID, "sessionToken": sess.Token, "details": map[string]any{},
	})
}

func (f *fakeZitadel) getSession(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		writeZitadelError(w, http.StatusUnauthorized, 16, "unauthenticated")
		return
	}
	f.mu.Lock()
	sess, ok := f.sessions[r.PathValue("id")]
	f.mu.Unlock()
	if !ok {
		writeZitadelError(w, http.StatusNotFound, 5, "session not found")
		return
	}
	writeZitadelJSON(w, http.StatusOK, map[string]any{"session": map[string]any{
		"id": sess.ID,
		"factors": map[string]any{
			"user":     map[string]string{"id": f.userID},
			"password": map[string]any{},
		},
	}})
}

func (f *fakeZitadel) authMethods(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		writeZitadelError(w, http.StatusUnauthorized, 16, "unauthenticated")
		return
	}
	f.mu.Lock()
	methods := append([]string(nil), f.methods...)
	f.mu.Unlock()
	writeZitadelJSON(w, http.StatusOK, map[string]any{"authMethodTypes": methods})
}

func (f *fakeZitadel) deleteSession(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		writeZitadelError(w, http.StatusUnauthorized, 16, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	f.mu.Lock()
	delete(f.sessions, id)
	f.deleted = append(f.deleted, id)
	f.mu.Unlock()
	writeZitadelJSON(w, http.StatusOK, map[string]any{"details": map[string]any{}})
}

func (f *fakeZitadel) completeAuthRequest(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		writeZitadelError(w, http.StatusUnauthorized, 16, "unauthenticated")
		return
	}
	var body struct {
		Session struct {
			SessionID    string `json:"sessionId"`
			SessionToken string `json:"sessionToken"`
		} `json:"session"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeZitadelError(w, http.StatusBadRequest, 3, "bad request")
		return
	}
	f.mu.Lock()
	ar, ok := f.authRequests[r.PathValue("id")]
	sess, sessionOK := f.sessions[body.Session.SessionID]
	f.mu.Unlock()
	if !ok || !sessionOK || sess.Token != body.Session.SessionToken {
		writeZitadelError(w, http.StatusBadRequest, 3, "session is invalid")
		return
	}
	code := fmt.Sprintf("code-for-%s", sess.ID)
	f.mu.Lock()
	f.codes[code] = fakeCode{
		challenge: ar.Challenge, redirectURI: ar.RedirectURI, clientID: ar.ClientID,
	}
	state, host := ar.State, ""
	if f.callbackState != "" {
		state = f.callbackState
	}
	if f.callbackHost != "" {
		host = f.callbackHost
	}
	f.mu.Unlock()
	redirect, err := url.Parse(ar.RedirectURI)
	if err != nil {
		writeZitadelError(w, http.StatusInternalServerError, 13, "bad redirect")
		return
	}
	if host != "" {
		redirect.Host = host
	}
	query := redirect.Query()
	query.Set("code", code)
	query.Set("state", state)
	redirect.RawQuery = query.Encode()
	writeZitadelJSON(w, http.StatusOK, map[string]any{
		"callbackUrl": redirect.String(), "details": map[string]any{},
	})
}

func newSignInHarness(t *testing.T, z *fakeZitadel) *harness {
	t.Helper()
	h := newHarness(t)
	internal := ""
	if z.internal != nil {
		internal = z.internal.URL
	}
	if err := h.server.EnablePasswordSignIn(syncserver.PasswordSignInOptions{
		Issuer: z.URL, ClientID: z.clientID, PAT: z.pat, InternalURL: internal,
		Scopes: []string{"urn:example:audience"},
	}); err != nil {
		t.Fatalf("EnablePasswordSignIn: %v", err)
	}
	h.server.SetPublicConfig(syncserver.PublicConfig{
		Issuer: z.URL, ClientID: z.clientID, Scopes: []string{"urn:example:audience"},
	})
	return h
}

func newSignInCase(t *testing.T, mutate func(*fakeZitadel)) (*harness, *fakeZitadel) {
	t.Helper()
	z := newFakeZitadel(t, "adhd-desktop")
	if mutate != nil {
		mutate(z)
	}
	return newSignInHarness(t, z), z
}

func signInBody() syncproto.SignInRequest {
	return syncproto.SignInRequest{LoginName: "ada@example.com", Password: "correct horse"}
}

func signInLegacyBody() syncproto.SignInRequest {
	body := signInBody()
	body.AuthRequest = "V2_1"
	return body
}

func postSignInRaw(h *harness, remoteAddr, forwardedFor string, raw []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/sign-in", bytes.NewReader(raw))
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	w := httptest.NewRecorder()
	h.server.Handler().ServeHTTP(w, req)
	return w
}

func postSignIn(t *testing.T, h *harness, remoteAddr, forwardedFor string, body syncproto.SignInRequest) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return postSignInRaw(h, remoteAddr, forwardedFor, raw)
}

func requireSignInError(t *testing.T, w *httptest.ResponseRecorder, status int, want string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, body %s, want %d", w.Code, w.Body.String(), status)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	if body.Error != want {
		t.Fatalf("error = %q, want %q", body.Error, want)
	}
}

// TestSignIn covers the answers of POST /v1/sign-in: the token answer, the credential errors, the
// second factor, an authorize failure, a mismatched callback, and invalid bodies.
func TestSignIn(t *testing.T) {
	t.Run("success returns tokens without caching", func(t *testing.T) {
		h, z := newSignInCase(t, nil)
		w := postSignIn(t, h, "203.0.113.1:1111", "", signInBody())
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
		}
		var out syncproto.SignInTokenResponse
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.AccessToken != "access-token" || out.RefreshToken != "refresh-token" ||
			out.IDToken != "id-token" || out.TokenType != "Bearer" || out.ExpiresIn != 3600 {
			t.Fatalf("token answer = %+v", out)
		}
		if got := w.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control = %q, want no-store", got)
		}
		z.mu.Lock()
		defer z.mu.Unlock()
		if z.created != 1 || len(z.deleted) != 0 {
			t.Fatalf("sessions created = %d, deleted = %v", z.created, z.deleted)
		}
		if z.authorizeCalls != 1 || z.tokenCalls != 1 {
			t.Fatalf("authorize calls = %d, token calls = %d", z.authorizeCalls, z.tokenCalls)
		}
	})

	t.Run("wrong password is invalid credentials", func(t *testing.T) {
		h, _ := newSignInCase(t, nil)
		body := signInBody()
		body.Password = "wrong"
		requireSignInError(t, postSignIn(t, h, "203.0.113.2:1111", "", body),
			http.StatusUnauthorized, "invalid_credentials")
	})

	t.Run("unknown user is also invalid credentials", func(t *testing.T) {
		h, _ := newSignInCase(t, nil)
		body := signInBody()
		body.LoginName = "nobody@example.com"
		requireSignInError(t, postSignIn(t, h, "203.0.113.3:1111", "", body),
			http.StatusUnauthorized, "invalid_credentials")
	})

	t.Run("a locked account is 423", func(t *testing.T) {
		h, _ := newSignInCase(t, func(z *fakeZitadel) { z.locked = true })
		requireSignInError(t, postSignIn(t, h, "203.0.113.4:1111", "", signInBody()),
			http.StatusLocked, "locked")
	})

	t.Run("a second factor is refused and the session is deleted", func(t *testing.T) {
		h, z := newSignInCase(t, func(z *fakeZitadel) {
			z.methods = []string{
				"AUTHENTICATION_METHOD_TYPE_PASSWORD",
				"AUTHENTICATION_METHOD_TYPE_TOTP",
			}
		})
		requireSignInError(t, postSignIn(t, h, "203.0.113.6:1111", "", signInBody()),
			http.StatusForbidden, "second_factor")
		z.mu.Lock()
		defer z.mu.Unlock()
		if len(z.deleted) != 1 {
			t.Fatalf("deleted sessions = %v, want one", z.deleted)
		}
	})

	t.Run("a failed authorize deletes the session", func(t *testing.T) {
		h, z := newSignInCase(t, func(z *fakeZitadel) { z.authorizeFailure = true })
		requireSignInError(t, postSignIn(t, h, "203.0.113.10:1111", "", signInBody()),
			http.StatusBadGateway, "upstream")
		z.mu.Lock()
		defer z.mu.Unlock()
		if z.created != 1 || len(z.deleted) != 1 {
			t.Fatalf("sessions created = %d, deleted = %v", z.created, z.deleted)
		}
		if z.tokenCalls != 0 {
			t.Fatalf("token calls = %d, want none", z.tokenCalls)
		}
	})

	t.Run("a callback state mismatch is 502", func(t *testing.T) {
		h, z := newSignInCase(t, func(z *fakeZitadel) { z.callbackState = "wrong" })
		requireSignInError(t, postSignIn(t, h, "203.0.113.11:1111", "", signInBody()),
			http.StatusBadGateway, "upstream")
		z.mu.Lock()
		defer z.mu.Unlock()
		if z.tokenCalls != 0 {
			t.Fatalf("token calls = %d, want none", z.tokenCalls)
		}
	})

	t.Run("a callback on another host is 502", func(t *testing.T) {
		h, z := newSignInCase(t, func(z *fakeZitadel) { z.callbackHost = "evil.example" })
		requireSignInError(t, postSignIn(t, h, "203.0.113.12:1111", "", signInBody()),
			http.StatusBadGateway, "upstream")
		z.mu.Lock()
		defer z.mu.Unlock()
		if z.tokenCalls != 0 {
			t.Fatalf("token calls = %d, want none", z.tokenCalls)
		}
	})

	t.Run("invalid requests are 400", func(t *testing.T) {
		h, z := newSignInCase(t, nil)
		long := strings.Repeat("x", 201)
		cases := []struct {
			name string
			body syncproto.SignInRequest
		}{
			{"bad auth request prefix", syncproto.SignInRequest{AuthRequest: "V1_1", LoginName: "a", Password: "p"}},
			{"empty auth request number", syncproto.SignInRequest{AuthRequest: "V2_", LoginName: "a", Password: "p"}},
			{"non-numeric auth request", syncproto.SignInRequest{AuthRequest: "V2_abc", LoginName: "a", Password: "p"}},
			{"blank login name", syncproto.SignInRequest{LoginName: "   ", Password: "p"}},
			{"blank password", syncproto.SignInRequest{LoginName: "a", Password: "   "}},
			{"long login name", syncproto.SignInRequest{LoginName: long, Password: "p"}},
			{"long password", syncproto.SignInRequest{LoginName: "a", Password: long}},
		}
		for _, tt := range cases {
			requireSignInError(t, postSignIn(t, h, "203.0.113.7:1111", "", tt.body),
				http.StatusBadRequest, "bad_request")
		}
		requireSignInError(t,
			postSignInRaw(h, "203.0.113.7:1111", "", bytes.Repeat([]byte("x"), 5<<10)),
			http.StatusBadRequest, "bad_request")
		z.mu.Lock()
		defer z.mu.Unlock()
		if z.created != 0 {
			t.Fatalf("invalid requests created %d sessions", z.created)
		}
	})
}

// TestSignInLegacyShape is the 0.6.1 answer: an auth_request still gets only the callback URL.
func TestSignInLegacyShape(t *testing.T) {
	h, z := newSignInCase(t, nil)
	w := postSignIn(t, h, "203.0.113.20:1111", "", signInLegacyBody())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var out syncproto.SignInResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	callback, err := url.Parse(out.CallbackURL)
	if err != nil {
		t.Fatal(err)
	}
	if callback.Host != "127.0.0.1" || callback.Path != "/callback" ||
		callback.Query().Get("state") != "state-1" || callback.Query().Get("code") == "" {
		t.Fatalf("callback = %s", out.CallbackURL)
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	if z.created != 1 || len(z.deleted) != 0 {
		t.Fatalf("sessions created = %d, deleted = %v", z.created, z.deleted)
	}
	if z.tokenCalls != 0 {
		t.Fatalf("token calls = %d, want none for 0.6.1", z.tokenCalls)
	}
}

// TestSignInInternalURL points every Zitadel call at an internal address and checks the requests
// still ask for the issuer's host.
func TestSignInInternalURL(t *testing.T) {
	h, z := newSignInCase(t, func(z *fakeZitadel) {
		z.internal = z.startInternal(t)
	})
	w := postSignIn(t, h, "203.0.113.21:1111", "", signInBody())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	if z.internalHits == 0 {
		t.Fatal("no request reached the internal address")
	}
	if z.publicHits != 0 {
		t.Fatalf("the public address got %d requests", z.publicHits)
	}
}
