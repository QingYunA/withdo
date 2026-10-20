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
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// accountFixture is the sync acceptance setup: a desktop that seeded the account with one task, and
// a fake OIDC issuer the phone signs in through.
type accountFixture struct {
	h             *syncHarness
	oidc          *fakeOIDC
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
	return &accountFixture{h: h, oidc: newFakeOIDC(t), desktop: desktop, desktopClient: client}
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
	if err := phone.SyncConfigure(f.h.http.URL, f.oidc.URL, "mobile-client", ""); err != nil {
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

	state, err := phone.SignIn(browserOpener{}, "phone")
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
	if state, err := phone.SignIn(browserOpener{}, "phone"); err != nil || state != signInLinkedJSON {
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

	state, err := phone.SignIn(browserOpener{}, "phone")
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

// browserOpener is BrowserOpener for the fake issuer: it follows the authorization page's redirect
// to the core's loopback callback, as a Custom Tab would.
type browserOpener struct{}

func (browserOpener) Open(raw string) error {
	response, err := http.Get(raw)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the sign-in page answered %s", response.Status)
	}
	return nil
}

// fakeOIDC is a minimal OIDC provider for the tests: discovery, an authorization endpoint that
// redirects with a code, and a token endpoint for the code and refresh grants. The access token is
// the sync server's static test token, so the real client authenticates with it.
type fakeOIDC struct{ *httptest.Server }

func newFakeOIDC(t *testing.T) *fakeOIDC {
	t.Helper()
	f := &fakeOIDC{}
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
		redirect, err := url.Parse(query.Get("redirect_uri"))
		if err != nil || redirect.String() == "" {
			http.Error(w, "no redirect_uri", http.StatusBadRequest)
			return
		}
		callback := redirect.Query()
		callback.Set("code", "mobile-code")
		callback.Set("state", query.Get("state"))
		redirect.RawQuery = callback.Encode()
		http.Redirect(w, r, redirect.String(), http.StatusFound)
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
