package syncserver_test

import (
	"bytes"
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

// fakeZitadel implements the six Session API calls the sync server makes. It also answers the
// auth request the client's authorize call would have started.
type fakeZitadel struct {
	*httptest.Server
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
	created      int
	deleted      []string
	next         int
}

type fakeAuthRequest struct {
	ID          string
	ClientID    string
	RedirectURI string
	State       string
	Scope       []string
}

type fakeSession struct {
	ID        string
	Token     string
	LoginName string
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
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/oidc/auth_requests/{id}", f.getAuthRequest)
	mux.HandleFunc("POST /v2/sessions", f.createSession)
	mux.HandleFunc("GET /v2/sessions/{id}", f.getSession)
	mux.HandleFunc("GET /v2/users/{id}/authentication_methods", f.authMethods)
	mux.HandleFunc("DELETE /v2/sessions/{id}", f.deleteSession)
	mux.HandleFunc("POST /v2/oidc/auth_requests/{id}", f.completeAuthRequest)
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
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
	redirect, err := url.Parse(ar.RedirectURI)
	if err != nil {
		writeZitadelError(w, http.StatusInternalServerError, 13, "bad redirect")
		return
	}
	query := redirect.Query()
	query.Set("code", "code-for-"+sess.ID)
	query.Set("state", ar.State)
	redirect.RawQuery = query.Encode()
	writeZitadelJSON(w, http.StatusOK, map[string]any{
		"callbackUrl": redirect.String(), "details": map[string]any{},
	})
}

func newSignInHarness(t *testing.T, z *fakeZitadel) *harness {
	t.Helper()
	h := newHarness(t)
	if err := h.server.EnablePasswordSignIn(z.URL, z.clientID, z.pat); err != nil {
		t.Fatalf("EnablePasswordSignIn: %v", err)
	}
	h.server.SetPublicConfig(syncserver.PublicConfig{Issuer: z.URL, ClientID: z.clientID})
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
	return syncproto.SignInRequest{
		AuthRequest: "V2_1", LoginName: "ada@example.com", Password: "correct horse",
	}
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

// TestSignIn covers the answers of POST /v1/sign-in: success, the credential errors, another
// client's auth request, a second factor, and invalid bodies.
func TestSignIn(t *testing.T) {
	t.Run("success returns the callback", func(t *testing.T) {
		h, z := newSignInCase(t, nil)
		w := postSignIn(t, h, "203.0.113.1:1111", "", signInBody())
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

	t.Run("another client's auth request is refused without a session", func(t *testing.T) {
		h, z := newSignInCase(t, func(z *fakeZitadel) {
			z.authRequests["V2_1"] = fakeAuthRequest{
				ID: "V2_1", ClientID: "another-app", RedirectURI: "http://127.0.0.1/callback", State: "x",
			}
		})
		requireSignInError(t, postSignIn(t, h, "203.0.113.5:1111", "", signInBody()),
			http.StatusBadRequest, "bad_request")
		z.mu.Lock()
		defer z.mu.Unlock()
		if z.created != 0 {
			t.Fatalf("created %d sessions for another client", z.created)
		}
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
			{"blank login name", syncproto.SignInRequest{AuthRequest: "V2_1", LoginName: "   ", Password: "p"}},
			{"blank password", syncproto.SignInRequest{AuthRequest: "V2_1", LoginName: "a", Password: "   "}},
			{"long login name", syncproto.SignInRequest{AuthRequest: "V2_1", LoginName: long, Password: "p"}},
			{"long password", syncproto.SignInRequest{AuthRequest: "V2_1", LoginName: "a", Password: long}},
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
