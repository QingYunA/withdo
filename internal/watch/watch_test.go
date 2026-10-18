package watch_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/testutil"
	"github.com/QingYunA/adhd-todo/internal/watch"
)

const fast = 10 * time.Millisecond

type recorder struct {
	mu        sync.Mutex
	changes   int
	reminders []domain.Reminder
	errs      []error
}

func (r *recorder) onChange() { r.mu.Lock(); r.changes++; r.mu.Unlock() }
func (r *recorder) onReminders(d []domain.Reminder) {
	r.mu.Lock()
	r.reminders = append(r.reminders, d...)
	r.mu.Unlock()
}
func (r *recorder) onError(err error) { r.mu.Lock(); r.errs = append(r.errs, err); r.mu.Unlock() }
func (r *recorder) snapshot() (int, []domain.Reminder, []error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.changes, append([]domain.Reminder(nil), r.reminders...), append([]error(nil), r.errs...)
}

// start runs a watcher over its own connection pool and returns a second,
// independent connection to the same file, standing in for another process.
func start(t *testing.T, mutate func(*watch.Config)) (*recorder, *domain.Core, *domain.Core, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "w.db")
	watcherDB, _ := testutil.OpenDBAt(t, path)
	writerDB, _ := testutil.OpenDBAt(t, path)
	watched := domain.New(watcherDB, domain.Options{})
	writer := domain.New(writerDB, domain.Options{})
	rec := &recorder{}
	cfg := watch.Config{DB: watcherDB, Core: watched, Interval: fast,
		OnChange: rec.onChange, OnReminders: rec.onReminders, OnError: rec.onError}
	if mutate != nil {
		mutate(&cfg)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- watch.Run(ctx, cfg) }()
	stop := func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run returned %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Run did not stop after cancel")
		}
	}
	t.Cleanup(cancel)
	return rec, watched, writer, stop
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestChangesFromAnotherConnectionAreReported(t *testing.T) {
	rec, _, writer, stop := start(t, nil)
	defer stop()
	time.Sleep(5 * fast)
	if n, _, _ := rec.snapshot(); n != 0 {
		t.Fatalf("no change expected before any write, got %d", n)
	}
	if _, err := writer.CreateTask(context.Background(), domain.CreateTaskInput{Title: "from elsewhere"}); err != nil {
		t.Fatal(err)
	}
	eventually(t, "OnChange after a write", func() bool { n, _, _ := rec.snapshot(); return n >= 1 })

	before, _, _ := rec.snapshot()
	time.Sleep(5 * fast)
	if after, _, _ := rec.snapshot(); after != before {
		t.Errorf("no write happened but OnChange fired %d more times", after-before)
	}
}

// Open takes the baseline before it returns, so a change committed right after it is told even
// when Run has not polled yet.
func TestAChangeRightAfterOpenIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.db")
	watcherDB, _ := testutil.OpenDBAt(t, path)
	writerDB, _ := testutil.OpenDBAt(t, path)
	rec := &recorder{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := watch.Open(ctx, watch.Config{DB: watcherDB, Interval: fast, OnChange: rec.onChange})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.New(writerDB, domain.Options{}).CreateTask(ctx, domain.CreateTaskInput{Title: "at once"}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); w.Run(ctx) }()
	eventually(t, "OnChange for the write made before Run", func() bool { n, _, _ := rec.snapshot(); return n >= 1 })
	cancel()
	<-done
}

func TestDueReminderIsClaimedOnceEvenWithTwoWatchers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.db")
	dbA, _ := testutil.OpenDBAt(t, path)
	dbB, _ := testutil.OpenDBAt(t, path)
	a, b := domain.New(dbA, domain.Options{}), domain.New(dbB, domain.Options{})
	if _, err := a.CreateTask(context.Background(), domain.CreateTaskInput{
		Title: "call mom", RemindAt: ptr(domain.FormatTime(time.Now().Add(-time.Minute)))}); err != nil {
		t.Fatal(err)
	}
	recA, recB := &recorder{}, &recorder{}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, c := range []struct {
		cfg watch.Config
	}{
		{watch.Config{DB: dbA, Core: a, Interval: fast, OnReminders: recA.onReminders}},
		{watch.Config{DB: dbB, Core: b, Interval: fast, OnReminders: recB.onReminders}},
	} {
		wg.Add(1)
		go func() { defer wg.Done(); _ = watch.Run(ctx, c.cfg) }()
	}
	eventually(t, "a reminder", func() bool {
		_, ra, _ := recA.snapshot()
		_, rb, _ := recB.snapshot()
		return len(ra)+len(rb) >= 1
	})
	time.Sleep(10 * fast)
	cancel()
	wg.Wait()
	_, ra, _ := recA.snapshot()
	_, rb, _ := recB.snapshot()
	if len(ra)+len(rb) != 1 {
		t.Fatalf("reminder delivered %d times, want exactly once", len(ra)+len(rb))
	}
}

func TestReminderThatBecomesDueLater(t *testing.T) {
	var mu sync.Mutex
	clock := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { mu.Lock(); defer mu.Unlock(); return clock }
	rec, watched, _, stop := start(t, func(c *watch.Config) { c.Now = now })
	defer stop()
	if _, err := watched.CreateTask(context.Background(), domain.CreateTaskInput{
		Title: "later", RemindAt: ptr("2026-06-01T13:00:00.000Z")}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * fast)
	if _, got, _ := rec.snapshot(); len(got) != 0 {
		t.Fatalf("reminder fired early: %+v", got)
	}
	mu.Lock()
	clock = clock.Add(2 * time.Hour)
	mu.Unlock()
	eventually(t, "the reminder", func() bool { _, got, _ := rec.snapshot(); return len(got) == 1 })
	if _, got, _ := rec.snapshot(); got[0].Title != "later" {
		t.Errorf("got %+v", got[0])
	}
	// Claiming writes reminded_at, which counts as a change for the watcher.
	eventually(t, "OnChange for the claim", func() bool { n, _, _ := rec.snapshot(); return n >= 1 })
}

func TestCallbacksAreOptional(t *testing.T) {
	_, _, writer, stop := start(t, func(c *watch.Config) {
		c.OnChange, c.OnReminders, c.OnError, c.Interval = nil, nil, nil, 0
	})
	if _, err := writer.CreateTask(context.Background(), domain.CreateTaskInput{Title: "x"}); err != nil {
		t.Fatal(err)
	}
	stop()
}

func TestRunRequiresDB(t *testing.T) {
	if err := watch.Run(context.Background(), watch.Config{}); err == nil {
		t.Error("expected an error without a database")
	}
}

func TestRunFailsOnClosedDatabase(t *testing.T) {
	db, _ := testutil.OpenDB(t)
	_ = db.Close()
	if err := watch.Run(context.Background(), watch.Config{DB: db}); err == nil {
		t.Error("expected an error for a closed database")
	}
}

func TestPollErrorsAreReportedNotFatal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.db")
	db, _ := testutil.OpenDBAt(t, path)
	rec := &recorder{}
	// A second pool the watcher polls through, closed under it to force errors.
	broken, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	brokenCore := domain.New(broken, domain.Options{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = watch.Run(ctx, watch.Config{DB: db, Core: brokenCore, Interval: fast,
			OnReminders: rec.onReminders, OnError: rec.onError})
	}()
	time.Sleep(3 * fast)
	_ = broken.Close()
	eventually(t, "an error report", func() bool { _, _, errs := rec.snapshot(); return len(errs) > 0 })
	cancel()
	<-done
}

func TestErrorsDuringShutdownAreNotReported(t *testing.T) {
	rec, _, _, stop := start(t, nil)
	time.Sleep(3 * fast)
	stop()
	if _, _, errs := rec.snapshot(); len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errors.Join(errs...))
	}
}

func ptr[T any](v T) *T { return &v }

func TestRunReturnsNilWhenCancelledBeforeStart(t *testing.T) {
	db, _ := testutil.OpenDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := watch.Run(ctx, watch.Config{DB: db}); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
}
