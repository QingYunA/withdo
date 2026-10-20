package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// accountFixture is the sync acceptance setup: a desktop that seeded the account with one task, and
// a fake OIDC issuer the phone signs in through.
type accountFixture struct {
	h    *syncHarness
	oidc *fakeOIDC
	// sync is the phone's sync server: the harness's server with the fake /v1/sign-in in front.
	sync          *httptest.Server
	desktop       *domain.Core
	desktopClient *syncclient.Client
}

func newAccountFixture(t *testing.T) *accountFixture {
	t.Helper()
	h := newSyncHarness(t)
	desktopDB, _ := testutil.OpenDB(t)
	desktop := domain.New(desktopDB, domain.Options{Now: func() time.Time { return demoNow }})
	if _, err := desktop.CreateTask(context.Background(), domain.CreateTaskInput{Title: "Desktop task"}); err != nil {
		t.Fatalf("create the desktop task: %v", err)
	}
	client, err := syncclient.New(desktop, syncclient.Options{BaseURL: h.http.URL, Token: syncTestToken})
	if err != nil {
		t.Fatalf("desktop sync client: %v", err)
	}
	if err := client.Link(context.Background(), "desktop"); err != nil {
		t.Fatalf("link the desktop: %v", err)
	}
	oidc := newFakeOIDC(t)
	server := httptest.NewServer(oidc.withSignIn(h.server.Handler()))
	t.Cleanup(server.Close)
	return &accountFixture{h: h, oidc: oidc, sync: server, desktop: desktop, desktopClient: client}
}

// phone opens an unlinked phone core configured for the fixture's server and issuer.
func (f *accountFixture) phone(t *testing.T) (*Core, *mapSecrets) {
	t.Helper()
	secrets := newMapSecrets()
	phone, err := open(t.TempDir(), secrets, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	if err := phone.SyncConfigure(f.sync.URL, f.oidc.URL, "mobile-client", ""); err != nil {
		t.Fatalf("SyncConfigure: %v", err)
	}
	return phone, secrets
}

// TestSignInChooseThenLinkMerge is acceptance 4's merge path: a phone with a task signs in to an
// account that has one, gets "choose", LinkMerge keeps both, and Devices marks this phone.
func TestSignInChooseThenLinkMerge(t *testing.T) {
	f := newAccountFixture(t)
	phone, secrets := f.phone(t)
	createPhoneTask(t, phone, "Phone task")

	state, err := phone.SignIn("ada@example.com", "secret", "phone")
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	assertChoose(t, state, 1, 0, 1)
	if !secrets.has(syncclient.RefreshTokenKeyFor(phone.dbPath)) {
		t.Error("SignIn did not keep the refresh token; the user would sign in again")
	}
	var account struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	rawAccount, err := phone.Account()
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if err := json.Unmarshal([]byte(rawAccount), &account); err != nil {
		t.Fatalf("decode Account: %v (%s)", err, rawAccount)
	}
	if account.Name != "Ada Lovelace" || account.Email != "ada@example.com" {
		t.Errorf("Account = %+v", account)
	}
	if titles := phoneTitles(t, phone); !contains(titles, "Phone task") {
		t.Fatalf("the phone lost its own task before choosing: %v", titles)
	}

	if err := phone.LinkMerge("phone"); err != nil {
		t.Fatalf("LinkMerge: %v", err)
	}
	titles := phoneTitles(t, phone)
	for _, want := range []string{"Phone task", "Desktop task"} {
		if !contains(titles, want) {
			t.Errorf("phone after merge has %v, want %q", titles, want)
		}
	}

	// Signing in again on a linked phone is a no-op that says so.
	if state, err := phone.SignIn("ada@example.com", "secret", "phone"); err != nil || state != signInLinkedJSON {
		t.Errorf("SignIn when linked = %s, %v; want %s", state, err, signInLinkedJSON)
	}

	if err := f.desktopClient.Sync(context.Background()); err != nil {
		t.Fatalf("desktop sync: %v", err)
	}
	desktopTitles := desktopTaskTitles(t, f.desktop)
	for _, want := range []string{"Phone task", "Desktop task"} {
		if !contains(desktopTitles, want) {
			t.Errorf("desktop after merge has %v, want %q", desktopTitles, want)
		}
	}

	assertThisPhone(t, phone, "phone")
}

// TestSignInChooseThenLinkReplace is acceptance 4's replace path: the phone's own task goes away,
// the account's stays, and a backup of the old database is left behind.
func TestSignInChooseThenLinkReplace(t *testing.T) {
	f := newAccountFixture(t)
	phone, _ := f.phone(t)
	createPhoneTask(t, phone, "Phone only task")

	state, err := phone.SignIn("ada@example.com", "secret", "phone")
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	assertChoose(t, state, 1, 0, 1)

	backup, err := phone.LinkReplace("phone")
	if err != nil {
		t.Fatalf("LinkReplace: %v", err)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Errorf("backup file: %v", err)
	}
	titles := phoneTitles(t, phone)
	if contains(titles, "Phone only task") {
		t.Errorf("phone kept its own task after replace: %v", titles)
	}
	if !contains(titles, "Desktop task") {
		t.Errorf("phone after replace has %v, want the account's task", titles)
	}

	if err := f.desktopClient.Sync(context.Background()); err != nil {
		t.Fatalf("desktop sync: %v", err)
	}
	if desktopTitles := desktopTaskTitles(t, f.desktop); contains(desktopTitles, "Phone only task") {
		t.Errorf("the replaced phone's task reached the desktop: %v", desktopTitles)
	}

	assertThisPhone(t, phone, "phone")
}

// TestSignInSendsTheAccountPasswordAndDeviceName: the account and password reach the sync
// server's /v1/sign-in, and the device name reaches the server's device list.
func TestSignInSendsTheAccountPasswordAndDeviceName(t *testing.T) {
	f := newAccountFixture(t)
	phone, _ := f.phone(t)
	state, err := phone.SignIn("cyan@example.com", "secret", "phone-under-test")
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if state != signInLinkedJSON {
		t.Errorf("SignIn = %s, want %s", state, signInLinkedJSON)
	}
	if login, password := f.oidc.lastSignIn(); login != "cyan@example.com" || password != "secret" {
		t.Errorf("/v1/sign-in got account %q, password %q", login, password)
	}
	assertThisPhone(t, phone, "phone-under-test")
}

// TestSignInWithAWrongPasswordKeepsNothing: a refused password is the sentence the screen shows,
// and the phone stays signed out and unlinked.
func TestSignInWithAWrongPasswordKeepsNothing(t *testing.T) {
	f := newAccountFixture(t)
	phone, secrets := f.phone(t)
	_, err := phone.SignIn("cyan@example.com", "wrong", "phone")
	if err == nil || err.Error() != "wrong account or password" {
		t.Fatalf("SignIn with a wrong password = %v, want the wrong-password sentence", err)
	}
	if secrets.has(syncclient.RefreshTokenKeyFor(phone.dbPath)) {
		t.Error("a refused sign-in stored a refresh token")
	}
	if _, linked, err := phone.dom.SyncState(context.Background(), "device_id"); err != nil || linked {
		t.Errorf("a refused sign-in linked the phone (%v)", err)
	}
}

// assertChoose checks SignIn's choose answer and the counts the both-have-tasks sheet shows.
func assertChoose(t *testing.T, state string, tasks, projects, devices int) {
	t.Helper()
	var answer signInChooseJSON
	if err := json.Unmarshal([]byte(state), &answer); err != nil {
		t.Fatalf("decode the choose answer: %v (%s)", err, state)
	}
	if answer.State != "choose" || answer.LocalTasks != tasks ||
		answer.LocalProjects != projects || answer.AccountDevices != devices {
		t.Errorf("choose = %+v, want state choose with %d tasks, %d projects and %d devices",
			answer, tasks, projects, devices)
	}
}

// assertThisPhone checks that Devices marks exactly this phone and names it.
func assertThisPhone(t *testing.T, phone *Core, name string) {
	t.Helper()
	raw, err := phone.Devices()
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	var devices []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		This bool   `json:"this"`
	}
	if err := json.Unmarshal([]byte(raw), &devices); err != nil {
		t.Fatalf("decode Devices: %v (%s)", err, raw)
	}
	var this int
	for _, device := range devices {
		if device.This {
			this++
			if device.Name != name {
				t.Errorf("this device = %+v, want name %q", device, name)
			}
		}
	}
	if this != 1 {
		t.Errorf("Devices marks %d devices as this phone, want 1: %s", this, raw)
	}
}

// phoneTitles returns every task title the phone's search finds.
func phoneTitles(t *testing.T, phone *Core) []string {
	t.Helper()
	raw, err := phone.Search("task")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	var tasks []data.Task
	if err := json.Unmarshal([]byte(raw), &tasks); err != nil {
		t.Fatalf("decode Search: %v (%s)", err, raw)
	}
	titles := make([]string, len(tasks))
	for i, task := range tasks {
		titles[i] = task.Title
	}
	return titles
}

// desktopTaskTitles returns every task title the desktop's domain lists.
func desktopTaskTitles(t *testing.T, desktop *domain.Core) []string {
	t.Helper()
	tasks, err := desktop.ListTasks(context.Background(), domain.TaskFilter{Limit: 1000})
	if err != nil {
		t.Fatalf("desktop ListTasks: %v", err)
	}
	titles := make([]string, len(tasks))
	for i, task := range tasks {
		titles[i] = task.Title
	}
	return titles
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// fakeOIDC is a minimal OIDC provider for the tests: discovery, an authorization endpoint that
// starts a pending auth request the way Zitadel's login v2 does, the sync server's /v1/sign-in
// that completes it for the password "secret", and a token endpoint for the code and refresh
// grants. The access token is the sync server's static test token, so the real client
// authenticates with it.
type fakeOIDC struct {
	*httptest.Server
	mu       sync.Mutex
	pending  map[string]string // auth request id → the client's state
	login    string
	password string
}

func newFakeOIDC(t *testing.T) *fakeOIDC {
	t.Helper()
	f := &fakeOIDC{pending: map[string]string{}}
	mux := http.NewServeMux()
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(w, map[string]any{
			"issuer":                                f.URL,
			"authorization_endpoint":                f.URL + "/authorize",
			"token_endpoint":                        f.URL + "/token",
			"jwks_uri":                              f.URL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("redirect_uri") != "http://127.0.0.1/callback" || query.Get("code_challenge") == "" {
			http.Error(w, "unexpected authorize request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		id := fmt.Sprintf("V2_%d", len(f.pending)+1)
		f.pending[id] = query.Get("state")
		f.mu.Unlock()
		http.Redirect(w, r, f.URL+"/ui/v2/login/login?authRequest="+id, http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(w, map[string]any{
			"access_token":  syncTestToken,
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "mobile-refresh",
			"id_token":      fakeIDToken(),
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(w, map[string]any{"keys": []any{}})
	})
	return f
}

// withSignIn puts the fake POST /v1/sign-in in front of the sync server's handler. It completes
// the pending auth request when the password is "secret" and answers 401 otherwise.
func (f *fakeOIDC) withSignIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sign-in" {
			next.ServeHTTP(w, r)
			return
		}
		var req syncproto.SignInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.login, f.password = req.LoginName, req.Password
		state, ok := f.pending[req.AuthRequest]
		delete(f.pending, req.AuthRequest)
		f.mu.Unlock()
		if !ok || req.Password != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			writeTestJSON(w, map[string]string{"error": "invalid_credentials"})
			return
		}
		callback := url.Values{"code": {"mobile-code"}, "state": {state}}
		writeTestJSON(w, syncproto.SignInResponse{CallbackURL: "http://127.0.0.1/callback?" + callback.Encode()})
	})
}

// lastSignIn is the account and password the last /v1/sign-in carried.
func (f *fakeOIDC) lastSignIn() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.login, f.password
}

func writeTestJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

// fakeIDToken has an unsigned payload, which is all the account display reads.
func fakeIDToken() string {
	payload, _ := json.Marshal(map[string]string{
		"name": "Ada Lovelace", "email": "ada@example.com", "sub": "mobile-core-subject",
	})
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
