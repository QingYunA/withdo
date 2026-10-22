package syncserver_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

func TestMoveSubjectKeepsTheOldSubjectAsAlias(t *testing.T) {
	h := newHarnessWithAuth(t, syncserver.StaticTokens{"old": "zitadel-user", "new": "pb:new"})
	ctx := context.Background()
	dev := h.device(t, "old", "laptop")
	if err := h.server.MoveSubject(ctx, "zitadel-user", "pb:new", true); err != nil {
		t.Fatalf("move: %v", err)
	}
	// Both identities reach the one workspace, and the device id made under the old one works under both.
	for _, token := range []string{"old", "new"} {
		if got := devices(t, h, token); len(got) != 1 || got[0].Name != "laptop" {
			t.Fatalf("devices with %s = %+v", token, got)
		}
		if code, _ := h.pull(t, token, dev.DeviceID, 0, 10); code != http.StatusOK {
			t.Fatalf("pull with %s = %d", token, code)
		}
	}
	infos, err := h.server.ListSubjects(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var got []string
	for _, i := range infos {
		got = append(got, fmt.Sprintf("%s alias=%t devices=%d", i.Subject, i.Alias, i.Devices))
	}
	want := "pb:new alias=false devices=1;zitadel-user alias=true devices=1"
	if strings.Join(got, ";") != want {
		t.Fatalf("subjects = %v, want %s", got, want)
	}
	// Dropping the alias gives the old subject a new empty user again.
	if err := h.server.DropAlias(ctx, "zitadel-user"); err != nil {
		t.Fatalf("drop alias: %v", err)
	}
	if got := devices(t, h, "old"); len(got) != 0 {
		t.Fatalf("after drop-alias the old subject sees %d devices", len(got))
	}
	if err := h.server.DropAlias(ctx, "nobody"); err == nil {
		t.Fatal("dropping an unknown alias should fail")
	}
}

func TestMoveSubjectRefusesAnAliasTarget(t *testing.T) {
	h := newHarnessWithAuth(t, syncserver.StaticTokens{"a": "subject-a", "b": "subject-b"})
	ctx := context.Background()
	h.device(t, "a", "a-laptop")
	h.device(t, "b", "b-laptop")
	if err := h.server.MoveSubject(ctx, "subject-a", "pb:a", true); err != nil {
		t.Fatalf("move: %v", err)
	}
	// subject-a is now an alias; moving another user onto it would make it both.
	err := h.server.MoveSubject(ctx, "subject-b", "subject-a", true)
	if err == nil || !strings.Contains(err.Error(), "already an alias") {
		t.Fatalf("err = %v, want the alias refusal", err)
	}
	if err := h.server.AddAlias(ctx, "subject-a", "subject-b"); err == nil {
		t.Fatal("adding an existing alias should fail")
	}
}

func TestNewSubjectStillGetsANewUser(t *testing.T) {
	h := newHarnessWithAuth(t, syncserver.StaticTokens{"a": "subject-a", "c": "subject-c"})
	h.device(t, "a", "a-laptop")
	if err := h.server.MoveSubject(context.Background(), "subject-a", "pb:a", true); err != nil {
		t.Fatalf("move: %v", err)
	}
	if got := devices(t, h, "c"); len(got) != 0 {
		t.Fatalf("a brand-new subject sees %d devices", len(got))
	}
	infos, err := h.server.ListSubjects(context.Background())
	if err != nil || len(infos) != 3 {
		t.Fatalf("subjects = %+v, err %v; want pb:a, its alias and subject-c", infos, err)
	}
}

func TestAddAliasBackfillsAMovedAccount(t *testing.T) {
	h := newHarnessWithAuth(t, syncserver.StaticTokens{"old": "zitadel-user", "new": "pb:new"})
	ctx := context.Background()
	h.device(t, "old", "laptop")
	if err := h.server.MoveSubject(ctx, "zitadel-user", "pb:new", false); err != nil {
		t.Fatalf("move: %v", err)
	}
	// The old device's next request made an empty user for the old subject.
	if got := devices(t, h, "old"); len(got) != 0 {
		t.Fatalf("the old subject sees %d devices", len(got))
	}
	if err := h.server.AddAlias(ctx, "zitadel-user", "pb:new"); err != nil {
		t.Fatalf("add alias: %v", err)
	}
	if got := devices(t, h, "old"); len(got) != 1 {
		t.Fatalf("after add-alias the old subject sees %d devices", len(got))
	}
	if err := h.server.AddAlias(ctx, "zitadel-user", "nobody"); err == nil {
		t.Fatal("aliasing an unknown target should fail")
	}
}
