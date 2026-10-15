package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/ghpoll"
	"github.com/QingYunA/adhd-todo/internal/store"
)

// stubGH is a gh that answers from a function and records its calls. No test reaches GitHub.
type stubGH struct {
	mu    sync.Mutex
	calls []string
	reply func(args []string) (ghpoll.Output, error)
}

func (s *stubGH) Run(_ context.Context, args ...string) (ghpoll.Output, error) {
	s.mu.Lock()
	s.calls = append(s.calls, strings.Join(args, " "))
	s.mu.Unlock()
	return s.reply(args)
}

func (s *stubGH) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// Without gh, whatever test starts a backend.
func TestMain(m *testing.M) {
	ghRunner = &stubGH{reply: func([]string) (ghpoll.Output, error) { return ghpoll.Output{}, ghpoll.ErrNotInstalled }}
	os.Exit(m.Run())
}

func TestOpenBackendUsesTheEnvironmentDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ADHD_TODO_DB", filepath.Join(dir, "x", "my.db"))
	be, err := openBackend(false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = be.close() }()
	if be.dataFolder != filepath.Join(dir, "x") {
		t.Errorf("dataFolder = %q", be.dataFolder)
	}
	if _, ok := be.store.(*data.Real); !ok {
		t.Fatalf("store is %T, want *data.Real", be.store)
	}
	got := make(chan struct{}, 1)
	be.store.Subscribe(func() { got <- struct{}{} })
	be.start(nil)
	time.Sleep(100 * time.Millisecond) // let the watcher take its baseline
	if _, err := be.store.CreateTask(data.CreateTaskInput{Title: "hello"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(3 * time.Second):
		t.Error("no change event from the watcher")
	}
}

func TestOpenBackendFakes(t *testing.T) {
	for _, tc := range []struct {
		fake, empty bool
		wantTasks   bool
	}{{true, false, true}, {false, true, false}} {
		be, err := openBackend(tc.fake, tc.empty)
		if err != nil {
			t.Fatal(err)
		}
		ts, _ := be.store.ListTasks(data.Filter{})
		if (len(ts) > 0) != tc.wantTasks {
			t.Errorf("fake=%v empty=%v: %d tasks", tc.fake, tc.empty, len(ts))
		}
		stop := make(chan struct{})
		be.start(stop)
		close(stop)
	}
}

func TestOpenBackendFailsOnABadPath(t *testing.T) {
	t.Setenv("ADHD_TODO_DB", "/dev/null/nope/x.db")
	if _, err := openBackend(false, false); err == nil {
		t.Error("expected an error")
	}
}

const mergedPR = `{"state":"MERGED","mergedAt":"2026-10-14T09:02:13Z","closedAt":"2026-10-14T09:02:13Z","isDraft":false,"reviewDecision":"APPROVED","statusCheckRollup":[]}`

// The poller starts with the app, finishes the task of a merged pull request through the shared
// database, tells Settings who is signed in, and stops with the backend.
func TestRealBackendPollsPullRequestsFromLaunch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "my.db")
	t.Setenv("ADHD_TODO_DB", path)
	url := "https://github.com/acme/rocket/pull/7"
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	made, err := domain.New(db, domain.Options{}).CreateTask(context.Background(),
		domain.CreateTaskInput{Title: "Ship it", Source: domain.SourceAgent, Status: domain.StatusReview, PRURL: &url})
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}

	gh := &stubGH{reply: func(args []string) (ghpoll.Output, error) {
		if args[0] == "auth" {
			return ghpoll.Output{Stdout: []byte("github.com\n  ✓ Logged in to github.com account QingYunA (keyring)\n")}, nil
		}
		return ghpoll.Output{Stdout: []byte(mergedPR)}, nil
	}}
	defer func(old ghpoll.Runner) { ghRunner = old }(ghRunner)
	ghRunner = gh

	be, err := openBackend(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if be.github == nil {
		t.Fatal("the real backend has no GitHub poller")
	}
	be.start(nil)
	deadline := time.Now().Add(5 * time.Second)
	for {
		d, err := be.store.GetTask(int(made.ID))
		if err != nil {
			t.Fatal(err)
		}
		if d.Task.Status == data.Done && be.github.Status().State == data.GitHubReady {
			if st := be.github.Status(); st.Account != "QingYunA" || st.CheckedAt.IsZero() {
				t.Errorf("status = %+v", st)
			}
			if len(d.Reports) != 1 || d.Reports[0].Agent != data.GitHubAgent {
				t.Errorf("reports = %+v", d.Reports)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("not done in time: %+v, status %+v", d.Task, be.github.Status())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := be.close(); err != nil {
		t.Fatal(err)
	}
	// Stopped: asking again polls nothing.
	before := gh.count()
	be.github.CheckNow()
	time.Sleep(50 * time.Millisecond)
	if gh.count() != before {
		t.Error("the poller ran after the backend closed")
	}
}

func TestDemoBackendHasAGitHubStatus(t *testing.T) {
	be, err := openBackend(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if be.github == nil || be.github.Status().State != data.GitHubReady {
		t.Errorf("github = %v", be.github)
	}
}
