package data

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

const foreignBody = `{"error":"device does not belong to this user","code":"device_not_in_account"}`

// accountServer is a sync server and PocketBase in one: it signs in as whoever the test says, owns
// the devices the test lists, and counts the sync requests it got.
type accountServer struct {
	*httptest.Server
	mu       sync.Mutex
	email    string
	devices  []string
	created  int
	syncHits atomic.Int64
	pullCode int
	linked   bool
	oauthID  string
}

func newAccountServer(t *testing.T) *accountServer {
	t.Helper()
	f := &accountServer{email: "ada@example.com"}
	mux := http.NewServeMux()
	json200 := func(w http.ResponseWriter, body string) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) { json200(w, `{"auth":"pocketbase"}`) })
	mux.HandleFunc("POST /api/collections/users/request-otp", func(w http.ResponseWriter, _ *http.Request) {
		json200(w, `{"otpId":"otp1"}`)
	})
	mux.HandleFunc("POST /api/collections/users/auth-with-otp", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		email := f.email
		f.mu.Unlock()
		payload, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix(), "id": "rec1"})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig",
			"record": map[string]any{"id": "rec1", "email": email, "name": "n"},
		})
	})
	mux.HandleFunc("GET /v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		list := []map[string]any{}
		for _, id := range f.devices {
			list = append(list, map[string]any{"id": id, "name": id, "last_seen_at": nil})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"devices": list})
	})
	mux.HandleFunc("POST /v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		f.created++
		id := fmt.Sprintf("new-device-%d", f.created)
		f.devices = append(f.devices, id)
		f.mu.Unlock()
		json200(w, fmt.Sprintf(`{"device_id":%q,"workspace_id":9,"empty":true,"seeding":true}`, id))
	})
	mux.HandleFunc("POST /v1/devices/seeded", func(w http.ResponseWriter, _ *http.Request) { json200(w, `{"seeded":true}`) })
	mux.HandleFunc("POST /v1/ids", func(w http.ResponseWriter, _ *http.Request) { json200(w, `{"start":1000,"end":2000}`) })
	mux.HandleFunc("POST /v1/push", func(w http.ResponseWriter, r *http.Request) {
		f.syncHits.Add(1)
		var req struct {
			Mutations []struct {
				ID int64 `json:"id"`
			} `json:"mutations"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var last int64
		for _, m := range req.Mutations {
			last = max(last, m.ID)
		}
		json200(w, fmt.Sprintf(`{"last_mutation_id":%d}`, last))
	})
	mux.HandleFunc("GET /v1/pull", func(w http.ResponseWriter, _ *http.Request) {
		f.syncHits.Add(1)
		f.mu.Lock()
		code := f.pullCode
		f.mu.Unlock()
		if code != 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_, _ = fmt.Fprint(w, foreignBody)
			return
		}
		json200(w, `{"cookie":0,"last_mutation_id":0,"changes":[],"more":false}`)
	})
	mux.HandleFunc("GET /v1/events", func(w http.ResponseWriter, r *http.Request) {
		f.syncHits.Add(1)
		f.mu.Lock()
		code := f.pullCode
		f.mu.Unlock()
		if code != 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_, _ = fmt.Fprint(w, foreignBody)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, ": connected\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// macApp is a Real whose database is linked to device "mac-device" with one task, signed out, with
// sync.json pointing at the server.
func macApp(t *testing.T, f *accountServer) *Real {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"device_id": "mac-device", "workspace_id": "7", "enabled": "1", "cookie": "0",
	} {
		if _, err := db.ExecContext(ctx, "INSERT INTO sync_state (key, value) VALUES (?, ?)", key, value); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO sync_id_blocks (entity, next, end) VALUES ('tasks', 1000, 2000)"); err != nil {
		t.Fatal(err)
	}
	r := NewReal(db, RealOptions{
		SyncConfigPath: syncclient.ConfigPath(path), Secrets: appSecrets{},
		Now: func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC) },
	})
	t.Cleanup(func() { _ = r.Close() })
	if err := r.SyncSetServer(f.URL); err != nil {
		t.Fatal(err)
	}
	if _, err := r.core.CreateTask(ctx, domain.CreateTaskInput{Title: "Mac task"}); err != nil {
		t.Fatal(err)
	}
	return r
}

// rememberAccountForTest stores the email the Mac was linked as, as a sync that worked would.
func rememberAccountForTest(t *testing.T, r *Real, email string) {
	t.Helper()
	if _, err := r.db.Exec(`INSERT INTO sync_state (key, value) VALUES ('account_email', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, email); err != nil {
		t.Fatal(err)
	}
}

func rememberedForTest(r *Real) string {
	var email string
	_ = r.db.QueryRow("SELECT value FROM sync_state WHERE key = 'account_email'").Scan(&email)
	return email
}

func (r *Real) signInWithCodeForTest(t *testing.T, email string) {
	t.Helper()
	r.SyncRequestCode(email)
	waitSnap(t, r, "the code step", atCodeStep)
	r.SyncSignInWithCode("123456")
}

// A sign-in as another account than the one this Mac is linked to stops at the sheet: the old
// email is named, nothing syncs, and the Mac's task is still waiting to upload.
func TestSignInOntoAnotherAccountsDeviceAsksFirst(t *testing.T) {
	f := newAccountServer(t)
	f.email = "bee@example.com"
	r := macApp(t, f)
	rememberAccountForTest(t, r, "ada@example.com")
	pendingBefore, _ := r.core.SyncPending(context.Background())

	r.signInWithCodeForTest(t, "bee@example.com")
	snap := waitSnap(t, r, "the sheet", func(s SyncSnapshot) bool { return s.Phase == SyncOtherAccountSheet })
	if snap.OtherAccount == nil || snap.OtherAccount.PreviousEmail != "ada@example.com" ||
		snap.OtherAccount.AccountEmail != "bee@example.com" {
		t.Fatalf("other account = %+v", snap.OtherAccount)
	}
	if n := f.syncHits.Load(); n != 0 {
		t.Fatalf("%d sync requests were made before the choice", n)
	}
	if pending, _ := r.core.SyncPending(context.Background()); pending != pendingBefore {
		t.Fatalf("pending %d -> %d: something synced", pendingBefore, pending)
	}

	// Sign in with the old account instead: signed out, the email card with ada's address.
	r.SyncUseOldAccount()
	snap = r.SyncSnapshot()
	if snap.Phase != SyncSignedOut || snap.SignInStep != SignInStepEmail || snap.SignInEmail != "ada@example.com" ||
		snap.OtherAccount != nil {
		t.Fatalf("after choosing the old account: phase %v step %q email %q other %+v",
			snap.Phase, snap.SignInStep, snap.SignInEmail, snap.OtherAccount)
	}
	if id, found, _ := r.core.SyncState(context.Background(), "device_id"); !found || id != "mac-device" {
		t.Fatalf("device id = %q, want the old link kept", id)
	}
}

// Moving this Mac to the new account forgets the old link, keeps the task and continues as a new
// device, which the new (empty) account is seeded from.
func TestMoveThisMacToTheNewAccount(t *testing.T) {
	f := newAccountServer(t)
	f.email = "bee@example.com"
	r := macApp(t, f)
	rememberAccountForTest(t, r, "ada@example.com")
	r.signInWithCodeForTest(t, "bee@example.com")
	waitSnap(t, r, "the sheet", func(s SyncSnapshot) bool { return s.Phase == SyncOtherAccountSheet })

	r.SyncMoveToAccount()
	waitSnap(t, r, "the move", func(s SyncSnapshot) bool {
		return s.Linked && s.Phase != SyncSigningIn && s.Phase != SyncOtherAccountSheet && s.ThisDeviceID == "new-device-1"
	})
	if id, _, _ := r.core.SyncState(context.Background(), "device_id"); id != "new-device-1" {
		t.Fatalf("device id = %q, want the new account's device", id)
	}
	if got, _ := r.core.SyncPending(context.Background()); got < 0 {
		t.Fatalf("pending = %d", got)
	}
	var titles int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE title = 'Mac task'").Scan(&titles); err != nil || titles != 1 {
		t.Fatalf("Mac task rows = %d, %v; want kept", titles, err)
	}
	if got := rememberedForTest(r); got != "bee@example.com" {
		t.Fatalf("remembered account = %q, want the new one", got)
	}
}

// The runner's 403 for a foreign device is a pane of its own, not "no response", and does not
// retry.
func TestWrongAccountPaneAndNoRetry(t *testing.T) {
	f := newAccountServer(t)
	f.pullCode = http.StatusForbidden
	path := linkedAppDB(t, nil, true, false)
	if err := syncclient.SaveConfig(path, syncclient.Config{Server: f.URL, Token: "token"}); err != nil {
		t.Fatal(err)
	}
	r, err := OpenReal(context.Background(), path, RealOptions{
		DataFolder: filepath.Dir(path), Interval: time.Hour, Secrets: appSecrets{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	r.Start()
	snap := waitSnap(t, r, "the wrong account pane", func(s SyncSnapshot) bool { return s.Phase == SyncWrongAccount })
	if snap.Failure != SyncFailureWrongAccount {
		t.Fatalf("snapshot = %+v", snap)
	}
	hits := f.syncHits.Load()
	time.Sleep(300 * time.Millisecond)
	if got := f.syncHits.Load(); got != hits {
		t.Fatalf("requests %d -> %d: the runner kept trying", hits, got)
	}
	r.SyncSignOut()
	if snap := r.SyncSnapshot(); snap.Phase != SyncSignedOut {
		t.Fatalf("after Sign out: phase %v", snap.Phase)
	}
	if got := rememberedForTest(r); got != "" {
		t.Fatalf("signing out of the wrong account remembered %q as the linked one", got)
	}
}

// Only the network and the server being away are "no answer"; every other error is not.
func TestSyncFailureKinds(t *testing.T) {
	for message, want := range map[string]SyncFailure{
		"sync server: boom (status 502)":                              SyncFailureNoAnswer,
		"sync server: internal server error (status 500)":             SyncFailureNoAnswer,
		"GET /v1/pull: dial tcp: lookup x: no such host":              SyncFailureNoAnswer,
		"sync: this device is linked to another account (status 403)": SyncFailureWrongAccount,
		"sync server: device is not seeding (status 403)":             SyncFailureUnknown,
		"sync server: not found (status 404)":                         SyncFailureUnknown,
		"sync server: bad request (status 400)":                       SyncFailureUnknown,
		"sync server: request too large (status 413)":                 SyncFailureUnknown,
		"sync server: unauthorized (status 401)":                      SyncFailureAuth,
		"a row did not fit":                                           SyncFailureUnknown,
		"sync server: 5001 entries do not fit (status 422)":           SyncFailureUnknown,
	} {
		if got := syncFailureKind(message); got != want {
			t.Errorf("syncFailureKind(%q) = %v, want %v", message, got, want)
		}
	}
}

// googleLinkApp is an app signed in by code, with the fake server's Google routes and an
// _externalAuths list that says whether the account has Google.
func googleLinkApp(t *testing.T, linked *bool) (*Real, func(code, errText string)) {
	t.Helper()
	f := newCodeServer(t)
	redirect := addGoogle(t, f, true)
	f.mux.HandleFunc("GET /api/collections/_externalAuths/records", func(w http.ResponseWriter, _ *http.Request) {
		items := []map[string]string{}
		if *linked {
			items = append(items, map[string]string{"provider": "google"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	})
	r, _ := codeApp(t, f)
	// Sign in by code first so the app has a token (the device link fails against this server).
	r.SyncRequestCode("ada@example.com")
	waitSnap(t, r, "the code step", atCodeStep)
	close(f.release)
	r.SyncSignInWithCode("123456")
	waitSnap(t, r, "the sign-in", func(s SyncSnapshot) bool { return s.AccountEmail == "ada@example.com" && s.Phase != SyncSigningIn })
	return r, redirect
}

// The Google row: the server's answer decides Link or Linked, and a link that works flips it.
func TestGoogleLinkRow(t *testing.T) {
	linked := false
	r, redirect := googleLinkApp(t, &linked)
	r.SyncRefreshGoogleLink()
	waitSnap(t, r, "the row", func(s SyncSnapshot) bool { return s.GoogleLink == GoogleLinkAvailable })

	opened := make(chan string, 1)
	r.SyncLinkGoogle(func(u string) { opened <- u })
	waitString(t, opened, "the browser")
	if s := r.SyncSnapshot(); s.GoogleLink != GoogleLinkWaiting {
		t.Fatalf("state = %v, want waiting", s.GoogleLink)
	}
	redirect("authcode", "")
	waitSnap(t, r, "linked", func(s SyncSnapshot) bool { return s.GoogleLink == GoogleLinkLinked })
}

// A server whose account already has Google shows Linked at once.
func TestGoogleLinkRowShowsLinked(t *testing.T) {
	linked := true
	r, _ := googleLinkApp(t, &linked)
	r.SyncRefreshGoogleLink()
	waitSnap(t, r, "the row", func(s SyncSnapshot) bool { return s.GoogleLink == GoogleLinkLinked })
}

// Cancel drops the wait and the row goes back to Link, with no failure.
func TestGoogleLinkCancel(t *testing.T) {
	linked := false
	r, _ := googleLinkApp(t, &linked)
	r.SyncRefreshGoogleLink()
	waitSnap(t, r, "the row", func(s SyncSnapshot) bool { return s.GoogleLink == GoogleLinkAvailable })
	opened := make(chan string, 1)
	r.SyncLinkGoogle(func(u string) { opened <- u })
	waitString(t, opened, "the browser")
	r.SyncCancelGoogleLink()
	if s := r.SyncSnapshot(); s.GoogleLink != GoogleLinkAvailable || s.GoogleLinkFailure != GoogleLinkFailureNone {
		t.Fatalf("after cancel: %v %q", s.GoogleLink, s.GoogleLinkFailure)
	}
}

// A Google account of another record is the "already used" failure and the row stays Link.
func TestGoogleLinkAlreadyUsed(t *testing.T) {
	f := newCodeServer(t)
	redirect := addGoogle(t, f, true)
	f.foreignID = "someone-else"
	f.mux.HandleFunc("GET /api/collections/_externalAuths/records", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]string{}})
	})
	r, _ := codeApp(t, f)
	r.SyncRequestCode("ada@example.com")
	waitSnap(t, r, "the code step", atCodeStep)
	close(f.release)
	r.SyncSignInWithCode("123456")
	waitSnap(t, r, "the sign-in", func(s SyncSnapshot) bool { return s.AccountEmail == "ada@example.com" && s.Phase != SyncSigningIn })
	r.SyncRefreshGoogleLink()
	waitSnap(t, r, "the row", func(s SyncSnapshot) bool { return s.GoogleLink == GoogleLinkAvailable })
	opened := make(chan string, 1)
	r.SyncLinkGoogle(func(u string) { opened <- u })
	waitString(t, opened, "the browser")
	redirect("authcode", "")
	snap := waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.GoogleLinkFailure != GoogleLinkFailureNone })
	if snap.GoogleLinkFailure != GoogleLinkFailureUsed || snap.GoogleLink != GoogleLinkAvailable {
		t.Fatalf("state %v failure %q, want Link with the already-used failure", snap.GoogleLink, snap.GoogleLinkFailure)
	}
}
