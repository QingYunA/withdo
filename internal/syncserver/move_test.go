package syncserver_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

func devices(t *testing.T, h *harness, token string) []syncproto.Device {
	t.Helper()
	var out syncproto.DevicesResponse
	requireStatus(t, "GET /v1/devices", h.get(t, "/v1/devices", token, &out), http.StatusOK)
	return out.Devices
}

func TestMoveSubjectCarriesTheWorkspace(t *testing.T) {
	h := newHarnessWithAuth(t, syncserver.StaticTokens{
		"old": "zitadel-user", "new": "pb:new",
	})
	h.device(t, "old", "laptop")
	// The new identity signed in before the move: its empty workspace must not block it.
	if got := devices(t, h, "new"); len(got) != 0 {
		t.Fatalf("new user has %d devices before the move", len(got))
	}
	if err := h.server.MoveSubject(context.Background(), "zitadel-user", "pb:new"); err != nil {
		t.Fatalf("move: %v", err)
	}
	if got := devices(t, h, "new"); len(got) != 1 || got[0].Name != "laptop" {
		t.Fatalf("devices after the move = %+v", got)
	}
	if got := devices(t, h, "old"); len(got) != 0 {
		t.Fatalf("the old subject still sees %d devices", len(got))
	}
}

func TestMoveSubjectRefusesWhatItShould(t *testing.T) {
	h := newHarnessWithAuth(t, syncserver.StaticTokens{"a": "subject-a", "b": "subject-b"})
	h.device(t, "a", "a-laptop")
	h.device(t, "b", "b-laptop")
	ctx := context.Background()

	err := h.server.MoveSubject(ctx, "subject-a", "subject-b")
	if err == nil || !strings.Contains(err.Error(), "already has a workspace") {
		t.Fatalf("target with data: err = %v", err)
	}
	if err := h.server.MoveSubject(ctx, "nobody", "subject-new"); err == nil {
		t.Fatal("moving an unknown subject should fail")
	}
	if err := h.server.MoveSubject(ctx, "subject-a", "subject-a"); err == nil {
		t.Fatal("moving a subject onto itself should fail")
	}
	// Nothing moved.
	if got := devices(t, h, "a"); len(got) != 1 {
		t.Fatalf("a has %d devices after refused moves", len(got))
	}
	if got := devices(t, h, "b"); len(got) != 1 {
		t.Fatalf("b has %d devices after refused moves", len(got))
	}
}
