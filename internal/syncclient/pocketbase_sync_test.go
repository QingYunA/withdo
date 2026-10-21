package syncclient_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// TestSyncThroughRealPocketBase runs the whole path with the real PocketBase: an imported user signs
// in with PocketBaseSource, a device links and pushes, and a second device of the same account
// (a new source over its own secrets) pulls the task. A wrong password and a signed-out source are
// refused, and the chain still takes the development token.
func TestSyncThroughRealPocketBase(t *testing.T) {
	h := newHarness(t)
	app, err := pbauth.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pbauth.Close(app) })
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	// The harness server only knows static tokens; serve a second one over the same database
	// with the PocketBase authenticator, on PocketBase's own listener.
	pbServer, err := syncserver.Open(testCtx, h.databaseURL, syncserver.Chain{pbauth.NewAuthenticator(app), testTokens})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pbServer.Close)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(testCtx)
	done := make(chan error, 1)
	go func() { done <- pbauth.Serve(ctx, app, ln, pbServer.Handler()) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("PocketBase did not stop")
		}
	})
	base := "http://" + ln.Addr().String()

	device := func(secrets syncclient.Secrets) (*domain.Core, *syncclient.Client, *syncclient.PocketBaseSource) {
		db, _ := testutil.OpenDB(t)
		core := domain.New(db, domain.Options{})
		source := syncclient.NewPocketBaseSource(base, secrets)
		client, err := syncclient.New(core, syncclient.Options{BaseURL: base, TokenSource: source})
		if err != nil {
			t.Fatal(err)
		}
		return core, client, source
	}

	coreA, clientA, sourceA := device(newFakeSecrets())
	if err := sourceA.SignIn(testCtx, base, "ada@example.com", "wrong password"); !errors.Is(err, syncclient.ErrInvalidCredentials) {
		t.Fatalf("wrong password: err = %v", err)
	}
	if err := clientA.Link(testCtx, "A"); !syncclient.IsSignedOut(err) {
		t.Fatalf("link before sign-in: err = %v, want signed out", err)
	}
	if err := sourceA.SignIn(testCtx, base, "ada@example.com", "correct horse battery"); err != nil {
		t.Fatalf("sign in: %v", err)
	}
	if err := clientA.Link(testCtx, "A"); err != nil {
		t.Fatalf("link: %v", err)
	}
	project := newProject(t, coreA, "alpha", nil)
	task := newTask(t, coreA, domain.CreateTaskInput{Title: "through PocketBase", ProjectID: &project.ID})
	if err := clientA.Sync(testCtx); err != nil {
		t.Fatalf("sync A: %v", err)
	}

	coreB, clientB, sourceB := device(newFakeSecrets())
	if err := sourceB.SignIn(testCtx, base, "ada@example.com", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if err := clientB.Link(testCtx, "B"); err != nil {
		t.Fatalf("link B: %v", err)
	}
	if err := clientB.Sync(testCtx); err != nil {
		t.Fatalf("sync B: %v", err)
	}
	got := must[*domain.Task](t)(coreB.GetTask(testCtx, task.ID))
	if got == nil || got.Title != "through PocketBase" {
		t.Fatalf("B's task = %+v", got)
	}

	if err := sourceA.Logout(); err != nil {
		t.Fatal(err)
	}
	if err := clientA.Sync(testCtx); !syncclient.IsSignedOut(err) {
		t.Fatalf("sync after logout: err = %v, want signed out", err)
	}
}
