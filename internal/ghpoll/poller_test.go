package ghpoll

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

var bg = context.Background()

const authOK = "github.com\n  ✓ Logged in to github.com account QingYunA (keyring)\n"

func prURL(n int) string { return fmt.Sprintf("https://github.com/acme/widgets/pull/%d", n) }

func sample(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// world is a database with a domain core, a fake gh that answers from replies (by pull request
// number), and a poller over them.
type world struct {
	t       *testing.T
	core    *domain.Core
	db      *sql.DB
	runner  *fakeRunner
	poller  *Poller
	mu      sync.Mutex
	replies map[string]Output // by "pr view <url>"
	auth    func() (Output, error)
	clock   time.Time
}

func newWorld(t *testing.T) *world {
	t.Helper()
	testutil.IsolateGit(t)
	db, _ := testutil.OpenDB(t)
	w := &world{t: t, db: db, replies: map[string]Output{}, clock: time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)}
	w.core = domain.New(db, domain.Options{Now: func() time.Time { return w.now() }})
	w.auth = func() (Output, error) { return Output{Stdout: []byte(authOK)}, nil }
	w.runner = &fakeRunner{reply: w.answer}
	w.poller = New(Config{Runner: w.runner, Store: w.core, Now: w.now, Interval: time.Hour})
	return w
}

func (w *world) now() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.clock
}

func (w *world) advance(d time.Duration) {
	w.mu.Lock()
	w.clock = w.clock.Add(d)
	w.mu.Unlock()
}

func (w *world) answer(args []string) (Output, error) {
	if args[0] == "auth" {
		return w.auth()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if out, ok := w.replies["pr view "+args[2]]; ok {
		return out, nil
	}
	return Output{}, &ExitError{Code: 1, Stderr: []byte("GraphQL: Could not resolve to a PullRequest")}
}

func (w *world) reply(n int, sampleFile string) {
	w.mu.Lock()
	w.replies["pr view "+prURL(n)] = Output{Stdout: sample(w.t, sampleFile)}
	w.mu.Unlock()
}

func (w *world) task(n int, status domain.Status) *domain.Task {
	w.t.Helper()
	task, err := w.core.CreateTask(bg, domain.CreateTaskInput{Title: fmt.Sprintf("task %d", n), Source: domain.SourceAgent, Status: status, PRURL: ptr(prURL(n))})
	if err != nil {
		w.t.Fatal(err)
	}
	return task
}

func (w *world) get(id int64) *domain.Task {
	w.t.Helper()
	task, err := w.core.GetTask(bg, id)
	if err != nil || task == nil {
		w.t.Fatalf("get %d: %v", id, err)
	}
	return task
}

func (w *world) reports(id int64) []domain.Report {
	w.t.Helper()
	rs, err := w.core.ListReports(bg, id)
	if err != nil {
		w.t.Fatal(err)
	}
	return rs
}

func ptr[T any](v T) *T { return &v }

func summaries(rs []domain.Report) []string {
	out := []string{}
	for _, r := range rs {
		out = append(out, r.Agent+": "+r.Summary)
	}
	return out
}

func TestPollStoresStateAndFilesReports(t *testing.T) {
	w := newWorld(t)
	merged, failing, changes, plain := w.task(1, domain.StatusReview), w.task(2, domain.StatusReview), w.task(3, domain.StatusDoing), w.task(4, domain.StatusDoing)
	w.reply(1, "merged.json")
	w.reply(2, "open-failing.json")
	w.reply(3, "changes-requested.json")
	w.reply(4, "open-passing-approved.json")

	w.poller.Poll(bg)

	if got := w.get(merged.ID); got.Status != domain.StatusDone || got.PullRequest().State != domain.PRMerged || got.PRCheckedAt == nil {
		t.Errorf("merged: %+v", got)
	}
	if got := summaries(w.reports(merged.ID)); len(got) != 1 || got[0] != "github: PR #1 merged" {
		t.Errorf("merged reports = %q", got)
	}
	if got := w.get(failing.ID); got.Status != domain.StatusReview || got.PullRequest().Checks != domain.PRChecksFailing {
		t.Errorf("failing: %+v", got)
	}
	if got := summaries(w.reports(failing.ID)); len(got) != 1 || got[0] != "github: Checks are failing on PR #2" {
		t.Errorf("failing reports = %q", got)
	}
	if got := summaries(w.reports(changes.ID)); len(got) != 1 || got[0] != "github: Changes were requested on PR #3" {
		t.Errorf("changes reports = %q", got)
	}
	if got := w.reports(plain.ID); len(got) != 0 {
		t.Errorf("a quiet pull request got reports: %q", summaries(got))
	}
	if got := w.get(plain.ID); got.PullRequest().Review != domain.PRReviewApproved {
		t.Errorf("plain: %+v", got)
	}
}

func TestPollAgainWithNothingNewFilesNothing(t *testing.T) {
	w := newWorld(t)
	task := w.task(2, domain.StatusReview)
	w.reply(2, "open-failing.json")
	w.poller.Poll(bg)
	w.advance(5 * time.Minute)
	w.poller.Poll(bg)

	if got := w.reports(task.ID); len(got) != 1 {
		t.Errorf("reports = %q, want just the first", summaries(got))
	}
	if got := w.get(task.ID); got.PRCheckedAt == nil || *got.PRCheckedAt != domain.FormatTime(w.now()) {
		t.Errorf("pr_checked_at = %v, want %s", got.PRCheckedAt, domain.FormatTime(w.now()))
	}
}

func TestPollFollowsOnlyOpenTasksWithAPullRequestOnGitHub(t *testing.T) {
	w := newWorld(t)
	w.task(1, domain.StatusDone)
	w.task(2, domain.StatusCancelled)
	if _, err := w.core.CreateTask(bg, domain.CreateTaskInput{Title: "no pr"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.core.CreateTask(bg, domain.CreateTaskInput{Title: "elsewhere", PRURL: ptr("https://gitlab.com/a/b/-/merge_requests/3")}); err != nil {
		t.Fatal(err)
	}
	w.poller.Poll(bg)
	if n := w.runner.callsTo("pr view"); n != 0 {
		t.Errorf("%d pr view calls, want none", n)
	}
}

func TestPollAsksOncePerPullRequest(t *testing.T) {
	w := newWorld(t)
	a := w.task(7, domain.StatusReview)
	b := w.task(7, domain.StatusDoing)
	w.reply(7, "open-failing.json")
	w.poller.Poll(bg)
	if n := w.runner.callsTo("pr view"); n != 1 {
		t.Errorf("%d pr view calls, want 1", n)
	}
	for _, id := range []int64{a.ID, b.ID} {
		if got := w.get(id); got.PullRequest().Checks != domain.PRChecksFailing {
			t.Errorf("task %d: %+v", id, got)
		}
	}
}

func TestPollAsksForTheFieldsItReads(t *testing.T) {
	w := newWorld(t)
	w.task(5, domain.StatusReview)
	w.reply(5, "open-pending.json")
	w.poller.Poll(bg)
	want := "pr view " + prURL(5) + " --json " + PRViewFields
	if got := strings.Join(w.runner.calls, "\n"); !strings.Contains(got, want) {
		t.Errorf("calls = %q, want %q", w.runner.calls, want)
	}
}

func TestOnePullRequestFailingDoesNotStopTheOthers(t *testing.T) {
	w := newWorld(t)
	gone, fine := w.task(8, domain.StatusReview), w.task(9, domain.StatusReview)
	w.reply(9, "merged.json") // pull request 8 has no reply: gh fails for it
	w.poller.Poll(bg)
	if got := w.get(gone.ID); got.PRState != nil || got.PRCheckedAt != nil {
		t.Errorf("an unanswered pull request was changed: %+v", got)
	}
	if got := w.get(fine.ID); got.Status != domain.StatusDone {
		t.Errorf("the other one: %+v", got)
	}
	if st := w.poller.Status(); st.State != AuthReady || st.CheckedAt.IsZero() {
		t.Errorf("status = %+v", st)
	}
}

func TestPollReadsStaleAnswersSafely(t *testing.T) {
	// The user finishes the task while gh is answering: the answer is dropped.
	w := newWorld(t)
	task := w.task(2, domain.StatusReview)
	w.reply(2, "open-failing.json")
	w.runner.reply = func(args []string) (Output, error) {
		if args[0] == "pr" {
			done := domain.StatusDone
			if _, err := w.core.UpdateTask(bg, task.ID, domain.TaskPatch{Status: &done}); err != nil {
				t.Error(err)
			}
		}
		return w.answer(args)
	}
	w.poller.Poll(bg)
	if got := w.reports(task.ID); len(got) != 0 {
		t.Errorf("reports = %q", summaries(got))
	}
}

func TestPollWithoutGHOrLogin(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want AuthState
	}{
		"missing":    {ErrNotInstalled, AuthMissing},
		"signed out": {&ExitError{Code: 1, Stderr: []byte("You are not logged into any GitHub hosts.")}, AuthSignedOut},
		"offline":    {&ExitError{Code: 1, Stderr: []byte("error connecting to api.github.com")}, AuthOffline},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			task := w.task(1, domain.StatusReview)
			w.reply(1, "merged.json")
			w.auth = func() (Output, error) { return Output{}, tc.err }
			w.poller.Poll(bg)
			if st := w.poller.Status(); st.State != tc.want || st.Account != "" {
				t.Errorf("status = %+v, want state %v", st, tc.want)
			}
			if n := w.runner.callsTo("pr view"); n != 0 {
				t.Errorf("%d pr view calls without a login", n)
			}
			if got := w.get(task.ID); got.Status != domain.StatusReview {
				t.Errorf("task changed: %+v", got)
			}
		})
	}
}

func TestStatusAfterAPoll(t *testing.T) {
	w := newWorld(t)
	if st := w.poller.Status(); st.State != AuthUnknown || !st.CheckedAt.IsZero() {
		t.Errorf("before: %+v", st)
	}
	w.poller.Poll(bg)
	st := w.poller.Status()
	if st.State != AuthReady || st.Account != "QingYunA" || !st.CheckedAt.Equal(w.now()) || st.Checking {
		t.Errorf("after: %+v", st)
	}
}

func TestPollRunsAtMostFourCallsAtOnce(t *testing.T) {
	w := newWorld(t)
	const tasks = 10
	for n := 1; n <= tasks; n++ {
		w.task(n, domain.StatusReview)
		w.reply(n, "open-passing-approved.json")
	}
	var running, peak atomic.Int32
	reached := make(chan struct{})
	var once sync.Once
	w.runner.reply = func(args []string) (Output, error) {
		if args[0] != "pr" {
			return w.answer(args)
		}
		now := running.Add(1)
		defer running.Add(-1)
		for {
			p := peak.Load()
			if now <= p || peak.CompareAndSwap(p, now) {
				break
			}
		}
		if now == 4 {
			once.Do(func() { close(reached) })
		}
		// Hold the first calls until four are in flight, so a lower limit would deadlock here
		// and a higher one would show in peak.
		select {
		case <-reached:
		case <-time.After(5 * time.Second):
			t.Error("four calls never ran together")
		}
		time.Sleep(20 * time.Millisecond)
		return w.answer(args)
	}
	w.poller.Poll(bg)
	if got := peak.Load(); got != 4 {
		t.Errorf("peak concurrency = %d, want 4", got)
	}
	if n := w.runner.callsTo("pr view"); n != tasks {
		t.Errorf("%d pr view calls, want %d", n, tasks)
	}
}

func TestPollStopsWhenItsContextEnds(t *testing.T) {
	w := newWorld(t)
	task := w.task(1, domain.StatusReview)
	w.reply(1, "merged.json")
	ctx, cancel := context.WithCancel(bg)
	w.runner.reply = func(args []string) (Output, error) {
		if args[0] == "pr" {
			cancel()
			return Output{}, context.Canceled
		}
		return w.answer(args)
	}
	w.poller.Poll(ctx)
	if got := w.get(task.ID); got.PRState != nil {
		t.Errorf("task changed after cancel: %+v", got)
	}
}

func TestRunPollsAtLaunchAndEveryInterval(t *testing.T) {
	w := newWorld(t)
	w.poller = New(Config{Runner: w.runner, Store: w.core, Now: w.now, Interval: 30 * time.Millisecond})
	w.task(1, domain.StatusReview)
	w.reply(1, "open-pending.json")
	w.poller.Start()
	defer w.poller.Stop()
	waitFor(t, func() bool { return w.runner.callsTo("auth status") >= 3 }, "three polls")
}

func TestStopWaitsForTheLoopAndIsRepeatable(t *testing.T) {
	w := newWorld(t)
	w.poller.Start()
	w.poller.Start() // a second Start does nothing
	waitFor(t, func() bool { return w.runner.callsTo("auth status") == 1 }, "the launch poll")
	w.poller.Stop()
	w.poller.Stop()
	before := w.runner.callsTo("auth status")
	w.poller.Activate()
	w.poller.CheckNow()
	time.Sleep(50 * time.Millisecond)
	if after := w.runner.callsTo("auth status"); after != before {
		t.Errorf("polled after Stop: %d then %d", before, after)
	}
}

func TestActivateChecksOnlyWhenTheLastCheckIsOlderThanAMinute(t *testing.T) {
	w := newWorld(t)
	w.poller.Start()
	defer w.poller.Stop()
	waitFor(t, func() bool { return w.runner.callsTo("auth status") == 1 }, "the launch poll")

	w.advance(30 * time.Second)
	w.poller.Activate()
	time.Sleep(60 * time.Millisecond)
	if n := w.runner.callsTo("auth status"); n != 1 {
		t.Fatalf("activated 30 s after a check: %d polls, want 1", n)
	}
	w.advance(31 * time.Second)
	w.poller.Activate()
	waitFor(t, func() bool { return w.runner.callsTo("auth status") == 2 }, "a poll after a minute")
}

func TestCheckNowAlwaysChecks(t *testing.T) {
	w := newWorld(t)
	w.poller.Start()
	defer w.poller.Stop()
	waitFor(t, func() bool { return w.runner.callsTo("auth status") == 1 }, "the launch poll")
	w.poller.CheckNow()
	waitFor(t, func() bool { return w.runner.callsTo("auth status") == 2 }, "a poll on request")
}

func TestSubscribersHearWhenACheckStartsAndEnds(t *testing.T) {
	w := newWorld(t)
	var seen []bool
	var mu sync.Mutex
	cancel := w.poller.Subscribe(func() {
		mu.Lock()
		seen = append(seen, w.poller.Status().Checking)
		mu.Unlock()
	})
	w.poller.Poll(bg)
	cancel()
	w.poller.Poll(bg)
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 || !seen[0] || seen[1] {
		t.Errorf("notifications (checking?) = %v, want [true false]", seen)
	}
}

func TestAPollInProgressIsNotStartedTwice(t *testing.T) {
	w := newWorld(t)
	w.task(1, domain.StatusReview)
	w.reply(1, "open-pending.json")
	release := make(chan struct{})
	entered := make(chan struct{}, 2)
	w.runner.reply = func(args []string) (Output, error) {
		if args[0] == "pr" {
			entered <- struct{}{}
			<-release
		}
		return w.answer(args)
	}
	done := make(chan struct{})
	go func() { w.poller.Poll(bg); close(done) }()
	<-entered
	if !w.poller.Status().Checking {
		t.Error("Checking should be true during a poll")
	}
	w.poller.Poll(bg) // returns at once
	close(release)
	<-done
	if n := w.runner.callsTo("auth status"); n != 1 {
		t.Errorf("%d auth checks, want 1", n)
	}
}

func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
