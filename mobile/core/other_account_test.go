package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// otherAccountAnswer decodes SignIn's other_account answer.
func otherAccountAnswer(t *testing.T, state string) signInOtherAccountJSON {
	t.Helper()
	var answer signInOtherAccountJSON
	if err := json.Unmarshal([]byte(state), &answer); err != nil {
		t.Fatalf("decode the answer: %v (%s)", err, state)
	}
	return answer
}

// A phone linked to account A that signs in as account B is told so before anything syncs: the
// answer names both emails, the phone stays linked to A and keeps its unsynced task.
func TestSignInOntoAnotherAccountsDevice(t *testing.T) {
	f := newAccountFixture(t)
	phone, _ := f.phone(t)
	if state, err := phone.SignIn("ada@example.com", "secret", "phone"); err != nil || state != signInLinkedJSON {
		t.Fatalf("first SignIn = %s, %v", state, err)
	}
	createPhoneTask(t, phone, "Unsynced phone task")
	device, _, _ := phone.dom.SyncState(context.Background(), "device_id")
	if err := phone.SignOut(); err != nil {
		t.Fatal(err)
	}

	f.pb.signInAs(syncOtherToken, "bee@example.com")
	state, err := phone.SignIn("bee@example.com", "secret", "phone")
	if err != nil {
		t.Fatalf("SignIn as the other account: %v", err)
	}
	answer := otherAccountAnswer(t, state)
	if answer.State != "other_account" || answer.PreviousEmail != "ada@example.com" || answer.AccountEmail != "bee@example.com" {
		t.Fatalf("answer = %+v", answer)
	}
	if got, _, _ := phone.dom.SyncState(context.Background(), "device_id"); got != device {
		t.Errorf("device id changed to %q; the phone must stay linked until the user chooses", got)
	}
	if pending, _ := phone.dom.SyncPending(context.Background()); pending == 0 {
		t.Error("the unsynced task was uploaded to the other account")
	}

	// The other choice: sign out, and the old account signs in without the sheet.
	if err := phone.SignOut(); err != nil {
		t.Fatal(err)
	}
	f.pb.signInAs("", "")
	if state, err := phone.SignIn("ada@example.com", "secret", "phone"); err != nil || state != signInLinkedJSON {
		t.Fatalf("SignIn as the old account = %s, %v; want linked", state, err)
	}
}

// Move this phone to the new account: the link to the old account is forgotten, the phone's tasks
// (the unsynced one included) go to the new account, and the phone is a device of it.
func TestMoveToAccountKeepsTheTasks(t *testing.T) {
	f := newAccountFixture(t)
	phone, _ := f.phone(t)
	if state, err := phone.SignIn("ada@example.com", "secret", "phone"); err != nil || state != signInLinkedJSON {
		t.Fatalf("first SignIn = %s, %v", state, err)
	}
	createPhoneTask(t, phone, "Unsynced phone task")
	if err := phone.SignOut(); err != nil {
		t.Fatal(err)
	}
	f.pb.signInAs(syncOtherToken, "bee@example.com")
	state, err := phone.SignIn("bee@example.com", "secret", "phone")
	if err != nil || otherAccountAnswer(t, state).State != "other_account" {
		t.Fatalf("SignIn = %s, %v", state, err)
	}

	moved, err := phone.MoveToAccount("phone moved")
	if err != nil || moved != signInLinkedJSON {
		t.Fatalf("MoveToAccount = %s, %v; want linked (the new account is empty)", moved, err)
	}
	assertThisPhone(t, phone, "phone moved")
	if titles := phoneTitles(t, phone); !contains(titles, "Unsynced phone task") {
		t.Fatalf("the phone lost its task: %v", titles)
	}
	if err := phone.SyncNow(); err != nil {
		t.Fatalf("sync after the move: %v", err)
	}
	if pending, _ := phone.dom.SyncPending(context.Background()); pending != 0 {
		t.Errorf("%d changes still waiting after the move", pending)
	}
	// The first account never saw the task.
	if err := f.desktopClient.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if contains(desktopTaskTitles(t, f.desktop), "Unsynced phone task") {
		t.Error("the old account got the task that was made after sign-out")
	}
}

// A 403 for a foreign device and any other 4xx are never worded as "no response"; only a real
// network failure is.
func TestSyncErrorWordsByKind(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   string
	}{
		{403, `{"error":"device does not belong to this user","code":"device_not_in_account"}`, "this phone is linked to another account"},
		{403, `{"error":"device is not seeding"}`, "the server refused the request"},
		{404, `{"error":"nope"}`, "the server refused the request"},
		{400, `{"error":"bad"}`, "the server refused the request"},
		{500, `{"error":"boom"}`, "the server had a problem; try again later"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		client, err := syncclient.New(nil2core(t), syncclient.Options{BaseURL: server.URL, Token: "t", HTTP: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Devices(context.Background())
		server.Close()
		if err == nil {
			t.Fatalf("status %d: no error", tc.status)
		}
		if got := syncError(err).Error(); got != tc.want {
			t.Errorf("status %d: syncError = %q, want %q", tc.status, got, tc.want)
		}
	}
	if got := syncError(errors.New("plain")).Error(); got != "something went wrong; try again" {
		t.Errorf("syncError(plain) = %q", got)
	}
}

// nil2core is a domain core on a scratch database, for a client that never reads it.
func nil2core(t *testing.T) *domain.Core {
	t.Helper()
	db, _ := testutil.OpenDB(t)
	return domain.New(db, domain.Options{})
}
