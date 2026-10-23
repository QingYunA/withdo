package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// accountFixture is the sync acceptance setup: a desktop that seeded the account with one task, and
// a fake PocketBase the phone signs in through.
type accountFixture struct {
	h  *syncHarness
	pb *fakePocketBase
	// sync is the phone's sync server: the harness's server with the fake PocketBase in front.
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
	pb := newFakePocketBase()
	server := httptest.NewServer(pb.wrap(h.server.Handler()))
	t.Cleanup(server.Close)
	return &accountFixture{h: h, pb: pb, sync: server, desktop: desktop, desktopClient: client}
}

// phone opens an unlinked phone core configured for the fixture's server.
func (f *accountFixture) phone(t *testing.T) (*Core, *mapSecrets) {
	t.Helper()
	secrets := newMapSecrets()
	phone, err := open(t.TempDir(), secrets, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	if err := phone.SyncSetServer(f.sync.URL); err != nil {
		t.Fatalf("SyncSetServer: %v", err)
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
	if !syncclient.HasPocketBaseToken(secretAdapter{secrets}, phone.dbPath) {
		t.Error("SignIn did not keep the token; the user would sign in again")
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

// TestSignInSendsTheAccountPasswordAndDeviceName: the account and password reach
// PocketBase, and the device name reaches the server's device list.
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
	if login, password := f.pb.lastSignIn(); login != "cyan@example.com" || password != "secret" {
		t.Errorf("sign-in got account %q, password %q", login, password)
	}
	assertThisPhone(t, phone, "phone-under-test")
}

// TestSignInStatusStages: SyncStatus says checking while the password is checked, then downloading while
// Link registers the phone.
func TestSignInStatusStages(t *testing.T) {
	h := newSyncHarness(t)
	pb := newFakePocketBase()
	signInHit := make(chan struct{}, 1)
	releaseSignIn := make(chan struct{})
	devicesHit := make(chan struct{}, 1)
	releaseDevices := make(chan struct{})
	base := pb.wrap(h.server.Handler())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/collections/users/auth-with-password" {
			signInHit <- struct{}{}
			<-releaseSignIn
		}
		if r.Method == http.MethodPost && r.URL.Path == "/v1/devices" {
			devicesHit <- struct{}{}
			<-releaseDevices
		}
		base.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	phone, err := open(t.TempDir(), newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	if err := phone.SyncSetServer(server.URL); err != nil {
		t.Fatalf("SyncSetServer: %v", err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := phone.SignIn("ada@example.com", "secret", "phone")
		result <- err
	}()

	waitForStep(t, signInHit, "the sign-in request")
	if stage := syncStage(phone); stage != "checking" {
		t.Fatalf("while the server checks: sign_in_stage = %q", stage)
	}
	close(releaseSignIn)
	waitForStep(t, devicesHit, "the device registration")
	if stage := syncStage(phone); stage != "downloading" {
		t.Fatalf("while the device links: sign_in_stage = %q", stage)
	}
	close(releaseDevices)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("SignIn: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("SignIn never finished")
	}
	if stage := syncStage(phone); stage != "" {
		t.Fatalf("a finished sign-in kept sign_in_stage %q", stage)
	}
}

// syncStage reads sign_in_stage from SyncStatus.
func syncStage(phone *Core) string {
	var status struct {
		SignInStage string `json:"sign_in_stage"`
	}
	_ = json.Unmarshal([]byte(phone.SyncStatus()), &status)
	return status.SignInStage
}

// waitForStep waits for one step of a gated fake server.
func waitForStep(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
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
	if syncclient.HasPocketBaseToken(secretAdapter{secrets}, phone.dbPath) {
		t.Error("a refused sign-in stored a token")
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

// fakePocketBase is the sync server's front for the tests: GET /v1/config names PocketBase and
// PocketBase's password route answers with a token the sync server accepts. It answers 400 for any
// password but "secret".
type fakePocketBase struct {
	mu       sync.Mutex
	login    string
	password string
	// token and email are who the next password sign-in answers as; zero means the first account.
	token string
	email string
}

// signInAs makes the next sign-ins answer as another account.
func (f *fakePocketBase) signInAs(token, email string) {
	f.mu.Lock()
	f.token, f.email = token, email
	f.mu.Unlock()
}

func newFakePocketBase() *fakePocketBase { return &fakePocketBase{} }

func (f *fakePocketBase) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/config":
			writeTestJSON(w, map[string]string{"auth": "pocketbase"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/collections/users/auth-with-password":
			var req struct {
				Identity string `json:"identity"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			f.mu.Lock()
			f.login, f.password = req.Identity, req.Password
			f.mu.Unlock()
			if req.Password != "secret" {
				w.WriteHeader(http.StatusBadRequest)
				writeTestJSON(w, map[string]string{"message": "Failed to authenticate."})
				return
			}
			f.mu.Lock()
			token, email := f.token, f.email
			f.mu.Unlock()
			if token == "" {
				token, email = syncTestToken, "ada@example.com"
			}
			writeTestJSON(w, map[string]any{
				"token":  token,
				"record": map[string]string{"email": email, "name": "Ada Lovelace"},
			})
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// lastSignIn is the account and password the last password sign-in carried.
func (f *fakePocketBase) lastSignIn() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.login, f.password
}

func writeTestJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
