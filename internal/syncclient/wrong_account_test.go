package syncclient_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

const foreignBody = `{"error":"device does not belong to this user","code":"device_not_in_account"}`

// A 403 for a device of another account is ErrDeviceNotInAccount, with or without the code (a
// server older than the code says the same words); any other 403 or 4xx is not.
func TestForeignDeviceStatusMapping(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"with the code", 403, foreignBody, true},
		{"older server, same text", 403, `{"error":"device does not belong to this user"}`, true},
		{"another 403", 403, `{"error":"device is not seeding"}`, false},
		{"a 404", 404, `{"error":"nope"}`, false},
		{"a 400", 400, `{"error":"bad"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newRunnerServer(t)
			server.setPullFailure(tc.status, tc.body)
			err := newRunnerClient(t, linkedCore(t), server).Sync(testCtx)
			if err == nil {
				t.Fatal("sync succeeded")
			}
			if got := errors.Is(err, syncclient.ErrDeviceNotInAccount); got != tc.want {
				t.Fatalf("errors.Is(ErrDeviceNotInAccount) = %v for %q, want %v", got, err, tc.want)
			}
		})
	}
}

// The runner stops in "wrong account" at the first 403 of that kind and never tries again, however
// long it waits.
func TestRunnerStopsOnForeignDevice(t *testing.T) {
	server := newRunnerServer(t)
	server.setPullFailure(403, foreignBody)
	clock := newFakeClock()
	_, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
	status := waitState(t, statuses, syncclient.StateWrongAccount)
	if !strings.Contains(status.LastError, "linked to another account") {
		t.Fatalf("last error = %q, want the wrong account text", status.LastError)
	}
	server.waitPulls(t, 1)
	for i := 0; i < 5; i++ {
		clock.Advance(10 * time.Minute)
	}
	time.Sleep(100 * time.Millisecond)
	if got := server.pullCount(); got != 1 {
		t.Fatalf("pulls = %d, want 1: the runner retried a refusal that cannot change", got)
	}
}

// Signed in as another account, a linked database finds out before it syncs; moving it to the new
// account keeps its rows and the new account gets them.
func TestCheckDeviceAccountAndUnlink(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	newTask(t, a.core, taskInput("kept task"))
	a.link(t, "A")
	if err := a.client.CheckDeviceAccount(testCtx); err != nil {
		t.Fatalf("owner check: %v", err)
	}
	// A change made before the sign-in, still waiting to upload.
	newTask(t, a.core, taskInput("unsynced task"))

	other, err := syncclient.New(a.core, syncclient.Options{BaseURL: h.http.URL, Token: otherToken})
	if err != nil {
		t.Fatal(err)
	}
	if err := other.CheckDeviceAccount(testCtx); !errors.Is(err, syncclient.ErrDeviceNotInAccount) {
		t.Fatalf("check as another account = %v, want ErrDeviceNotInAccount", err)
	}
	if err := other.Sync(testCtx); !errors.Is(err, syncclient.ErrDeviceNotInAccount) {
		t.Fatalf("sync as another account = %v, want ErrDeviceNotInAccount", err)
	}

	if err := other.Unlink(testCtx); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if _, err := other.DeviceID(testCtx); !errors.Is(err, syncclient.ErrNotLinked) {
		t.Fatalf("device id after unlink = %v, want ErrNotLinked", err)
	}
	if got := count(t, a.db, "SELECT COUNT(*) FROM tasks WHERE title IN ('kept task','unsynced task')"); got != 2 {
		t.Fatalf("%d tasks after unlink, want both kept", got)
	}
	if err := other.Link(testCtx, "A moved"); err != nil {
		t.Fatalf("link to the new account: %v", err)
	}
	if err := other.CheckDeviceAccount(testCtx); err != nil {
		t.Fatalf("check after the move: %v", err)
	}
	workspace, err := other.DeviceWorkspaceID(testCtx)
	if err != nil || workspace == 0 {
		t.Fatalf("workspace of the new device = %d, %v", workspace, err)
	}
	var onServer int
	if err := h.admin.QueryRow(testCtx,
		"SELECT COUNT(*) FROM rows WHERE workspace_id = $1 AND entity = $2 AND NOT deleted",
		workspace, syncproto.EntityTasks).Scan(&onServer); err != nil {
		t.Fatal(err)
	}
	if onServer != 2 {
		t.Fatalf("the new account has %d tasks, want both of this database's", onServer)
	}
	devices, err := other.Devices(testCtx)
	if err != nil || len(devices) != 1 {
		t.Fatalf("new account devices = %+v, %v", devices, err)
	}
	if left, err := other.Pending(testCtx); err != nil || left != 0 {
		t.Fatalf("pending after the move = %d, %v", left, err)
	}
}
