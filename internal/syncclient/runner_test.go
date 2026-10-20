package syncclient_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// fakeClock is a Clock the test moves by hand. After records every duration it is asked for, so a
// test can wait until the runner has scheduled its next timer instead of sleeping.
type fakeClock struct {
	mu         sync.Mutex
	now        time.Time
	timers     []*fakeTimer
	registered chan time.Duration
}

type fakeTimer struct {
	at time.Time
	ch chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{
		now:        time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC),
		registered: make(chan time.Duration, 4096),
	}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	c.mu.Lock()
	c.timers = append(c.timers, &fakeTimer{at: c.now.Add(d), ch: ch})
	c.mu.Unlock()
	select {
	case c.registered <- d:
	default:
	}
	return ch
}

// Advance moves the clock and fires every timer that came due.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	target := c.now.Add(d)
	left := c.timers[:0]
	for _, timer := range c.timers {
		if timer.at.After(target) {
			left = append(left, timer)
			continue
		}
		timer.ch <- target
	}
	c.timers = left
	c.now = target
}

// waitTimer waits until the runner asked for a timer of exactly d.
func waitTimer(t *testing.T, clock *fakeClock, d time.Duration) {
	t.Helper()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case got := <-clock.registered:
			if got == d {
				return
			}
		case <-timeout:
			t.Fatalf("the runner never scheduled a %s timer", d)
		}
	}
}

// runnerServer is the tiny sync server the runner tests use: one workspace, empty pulls, id blocks
// and an SSE stream. Tests switch its answers to make a sync fail.
type runnerServer struct {
	*httptest.Server
	mu          sync.Mutex
	pullStatus  int
	pulls       int
	subs        map[chan struct{}]struct{}
	pullGate    chan struct{}
	gateEntered chan struct{}
	pullSignal  chan struct{}
	eventOpen   chan struct{}
}

func newRunnerServer(t *testing.T) *runnerServer {
	t.Helper()
	s := &runnerServer{
		pullStatus: http.StatusOK, subs: map[chan struct{}]struct{}{},
		pullSignal: make(chan struct{}, 1024), eventOpen: make(chan struct{}, 16),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/pull", s.handlePull)
	mux.HandleFunc("POST /v1/push", s.handlePush)
	mux.HandleFunc("POST /v1/ids", s.handleIDs)
	mux.HandleFunc("GET /v1/events", s.handleEvents)
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func (s *runnerServer) setPullStatus(status int) {
	s.mu.Lock()
	s.pullStatus = status
	s.mu.Unlock()
}

func (s *runnerServer) waitPulls(t *testing.T, want int) {
	t.Helper()
	timeout := time.After(10 * time.Second)
	for {
		s.mu.Lock()
		got := s.pulls
		s.mu.Unlock()
		if got >= want {
			return
		}
		select {
		case <-s.pullSignal:
		case <-timeout:
			t.Fatalf("pulls = %d, want %d", got, want)
		}
	}
}

func (s *runnerServer) pullCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pulls
}

// blockNextPull makes the next pull wait. It returns the channel that releases it and one that is
// closed once the request is waiting.
func (s *runnerServer) blockNextPull() (release chan struct{}, entered <-chan struct{}) {
	release = make(chan struct{})
	enteredCh := make(chan struct{})
	s.mu.Lock()
	s.pullGate, s.gateEntered = release, enteredCh
	s.mu.Unlock()
	return release, enteredCh
}

func (s *runnerServer) handlePull(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	status := s.pullStatus
	gate, entered := s.pullGate, s.gateEntered
	if gate != nil {
		s.pullGate, s.gateEntered = nil, nil
	}
	s.pulls++
	s.mu.Unlock()
	select {
	case s.pullSignal <- struct{}{}:
	default:
	}
	if gate != nil {
		close(entered)
		select {
		case <-gate:
		case <-r.Context().Done():
			return
		}
	}
	if status != http.StatusOK {
		http.Error(w, http.StatusText(status), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, `{"cookie":0,"last_mutation_id":0,"changes":[],"more":false}`)
}

func (s *runnerServer) handlePush(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, `{"last_mutation_id":0}`)
}

func (s *runnerServer) handleIDs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, `{"start":1000,"end":2000}`)
}

func (s *runnerServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no flush", http.StatusInternalServerError)
		return
	}
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	select {
	case s.eventOpen <- struct{}{}:
	default:
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			_, _ = fmt.Fprint(w, "event: poke\ndata: 1\n\n")
			flusher.Flush()
		}
	}
}

func (s *runnerServer) poke() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// waitEventOpen waits for one SSE connection.
func (s *runnerServer) waitEventOpen(t *testing.T) {
	t.Helper()
	select {
	case <-s.eventOpen:
	case <-time.After(10 * time.Second):
		t.Fatal("the runner never opened the SSE stream")
	}
}

// linkedCore is a database the runner may sync: enabled with a device id, no local changes.
func linkedCore(t *testing.T) *domain.Core {
	t.Helper()
	db, _ := testutil.OpenDB(t)
	for key, value := range map[string]string{
		"device_id": "runner-device", "workspace_id": "7", "enabled": "1", "cookie": "0",
	} {
		if _, err := db.Exec("INSERT INTO sync_state (key, value) VALUES (?, ?)", key, value); err != nil {
			t.Fatalf("prepare %s: %v", key, err)
		}
	}
	return domain.New(db, domain.Options{})
}

func newRunnerClient(t *testing.T, core *domain.Core, server *runnerServer) *syncclient.Client {
	t.Helper()
	client, err := syncclient.New(core, syncclient.Options{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func startRunner(t *testing.T, client *syncclient.Client, clock *fakeClock) (*syncclient.Runner, chan syncclient.Status) {
	t.Helper()
	statuses := make(chan syncclient.Status, 256)
	runner := syncclient.NewRunner(client, syncclient.RunnerOptions{
		Clock: clock, Interval: 24 * time.Hour,
		OnStatus: func(status syncclient.Status) {
			select {
			case statuses <- status:
			default:
			}
		},
	})
	t.Cleanup(func() { _ = runner.Close() })
	if err := runner.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	return runner, statuses
}

func waitState(t *testing.T, statuses <-chan syncclient.Status, want syncclient.State) syncclient.Status {
	t.Helper()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-statuses:
			if status.State == want {
				return status
			}
		case <-timeout:
			t.Fatalf("the runner never reached %q", want)
		}
	}
}

// TestS3Core1Runner is acceptance criterion 1: debounce, one sync at a time, backoff to 5
// minutes and reset, 401 signing out, a poke, and the status callback.
func TestS3Core1Runner(t *testing.T) {
	t.Run("nudges debounce into one sync", func(t *testing.T) {
		server := newRunnerServer(t)
		clock := newFakeClock()
		runner, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
		waitState(t, statuses, syncclient.StateIdle)
		server.waitPulls(t, 1)

		runner.Nudge()
		waitTimer(t, clock, time.Second)
		runner.Nudge()
		waitTimer(t, clock, time.Second)
		runner.Nudge()
		waitTimer(t, clock, time.Second)
		clock.Advance(time.Second)
		server.waitPulls(t, 2)
		waitState(t, statuses, syncclient.StateIdle)

		clock.Advance(3 * time.Second)
		if got := server.pullCount(); got != 2 {
			t.Fatalf("pulls = %d after the debounce, want 2", got)
		}
	})

	t.Run("a nudge during a sync makes one more", func(t *testing.T) {
		server := newRunnerServer(t)
		clock := newFakeClock()
		runner, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
		waitState(t, statuses, syncclient.StateIdle)
		server.waitPulls(t, 1)

		release, entered := server.blockNextPull()
		runner.Nudge()
		waitTimer(t, clock, time.Second)
		clock.Advance(time.Second)
		select {
		case <-entered:
		case <-time.After(10 * time.Second):
			t.Fatal("the debounced sync never reached the server")
		}
		runner.Nudge()
		runner.Nudge()
		close(release)
		waitState(t, statuses, syncclient.StateIdle)
		waitTimer(t, clock, time.Second)
		clock.Advance(time.Second)
		server.waitPulls(t, 3)
		waitState(t, statuses, syncclient.StateIdle)

		clock.Advance(10 * time.Second)
		if got := server.pullCount(); got != 3 {
			t.Fatalf("pulls = %d, want exactly 3", got)
		}
	})

	t.Run("backoff doubles to five minutes and resets", func(t *testing.T) {
		server := newRunnerServer(t)
		server.setPullStatus(http.StatusInternalServerError)
		clock := newFakeClock()
		runner, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
		waitState(t, statuses, syncclient.StateOffline)
		server.waitPulls(t, 1)

		retries := []time.Duration{
			2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second,
			32 * time.Second, 64 * time.Second, 128 * time.Second, 256 * time.Second,
			5 * time.Minute,
		}
		for _, wait := range retries {
			waitTimer(t, clock, wait)
			before := server.pullCount()
			clock.Advance(wait)
			server.waitPulls(t, before+1)
		}
		// The ninth failure waits the capped five minutes again.
		waitTimer(t, clock, 5*time.Minute)

		server.setPullStatus(http.StatusOK)
		clock.Advance(5 * time.Minute)
		waitState(t, statuses, syncclient.StateIdle)

		server.setPullStatus(http.StatusInternalServerError)
		before := server.pullCount()
		runner.Nudge()
		waitTimer(t, clock, time.Second)
		clock.Advance(time.Second)
		server.waitPulls(t, before+1)
		waitState(t, statuses, syncclient.StateOffline)
		waitTimer(t, clock, 2*time.Second) // a success reset the backoff
		wantRetry := clock.Now().Add(2 * time.Second)
		select {
		case status := <-statuses:
			if !status.RetryAt.Equal(wantRetry) {
				t.Errorf("RetryAt = %v, want %v", status.RetryAt, wantRetry)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the retry time never reached the status callback")
		}
	})

	t.Run("401 stops in signed out and Start runs again", func(t *testing.T) {
		server := newRunnerServer(t)
		server.setPullStatus(http.StatusUnauthorized)
		clock := newFakeClock()
		runner, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
		waitState(t, statuses, syncclient.StateSignedOut)
		server.waitPulls(t, 1)

		server.setPullStatus(http.StatusOK)
		if err := runner.Start(context.Background()); err != nil {
			t.Fatalf("start after signed out: %v", err)
		}
		waitState(t, statuses, syncclient.StateIdle)
		if got := server.pullCount(); got < 2 {
			t.Fatalf("pulls after signing in again = %d, want at least 2", got)
		}
	})

	t.Run("a poke syncs at once", func(t *testing.T) {
		server := newRunnerServer(t)
		clock := newFakeClock()
		_, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
		waitState(t, statuses, syncclient.StateIdle)
		server.waitEventOpen(t)
		server.waitPulls(t, 1)

		server.poke()
		server.waitPulls(t, 2)
		waitState(t, statuses, syncclient.StateIdle)
		// The fake clock was never advanced, so only the poke can have caused this sync.
	})

	t.Run("SyncNow syncs without the debounce", func(t *testing.T) {
		server := newRunnerServer(t)
		clock := newFakeClock()
		runner, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
		waitState(t, statuses, syncclient.StateIdle)
		server.waitPulls(t, 1)

		runner.SyncNow()
		server.waitPulls(t, 2)
		waitState(t, statuses, syncclient.StateIdle)
		// The fake clock was never advanced, so no debounce or timer can have caused this sync.
	})

	t.Run("status changes reach OnStatus", func(t *testing.T) {
		server := newRunnerServer(t)
		clock := newFakeClock()
		_, statuses := startRunner(t, newRunnerClient(t, linkedCore(t), server), clock)
		first := waitState(t, statuses, syncclient.StateSyncing)
		if first.State != syncclient.StateSyncing {
			t.Fatalf("first status = %+v", first)
		}
		idle := waitState(t, statuses, syncclient.StateIdle)
		if idle.LastSynced.IsZero() {
			t.Fatalf("idle status has no LastSynced: %+v", idle)
		}
	})
}

// waitEventConnection waits until an SSE request for device arrived at the wrapped server.
func waitEventConnection(t *testing.T, events <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-events:
	case <-time.After(15 * time.Second):
		t.Fatalf("%s never opened the SSE stream", what)
	}
}

// TestS3Core2EndToEndPoke is acceptance criterion 2: a task made on A reaches B through the SSE
// poke while B's five-minute timer never fires.
func TestS3Core2EndToEndPoke(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	a.link(t, "A")
	b := h.newDevice(t, nil)
	b.link(t, "B")
	bDevice := str(t, b.db, "SELECT value FROM sync_state WHERE key = 'device_id'")

	eventConns := make(chan struct{}, 16)
	wrapped := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/events" && r.URL.Query().Get("device_id") == bDevice {
			select {
			case eventConns <- struct{}{}:
			default:
			}
		}
		h.server.Handler().ServeHTTP(w, r)
	}))
	t.Cleanup(wrapped.Close)

	bClient, err := syncclient.New(b.core, syncclient.Options{BaseURL: wrapped.URL, Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	aClient, err := syncclient.New(a.core, syncclient.Options{BaseURL: h.http.URL, Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	bClock, aClock := newFakeClock(), newFakeClock()
	bStatus := make(chan syncclient.Status, 64)
	aStatus := make(chan syncclient.Status, 64)
	bRunner := syncclient.NewRunner(bClient, syncclient.RunnerOptions{
		Clock: bClock, Interval: 24 * time.Hour,
		OnStatus: func(s syncclient.Status) { bStatus <- s },
	})
	aRunner := syncclient.NewRunner(aClient, syncclient.RunnerOptions{
		Clock: aClock, Interval: 24 * time.Hour,
		OnStatus: func(s syncclient.Status) { aStatus <- s },
	})
	t.Cleanup(func() { _ = bRunner.Close() })
	t.Cleanup(func() { _ = aRunner.Close() })

	if err := bRunner.Start(testCtx); err != nil {
		t.Fatal(err)
	}
	waitEventConnection(t, eventConns, "B")
	waitState(t, bStatus, syncclient.StateIdle)

	if err := aRunner.Start(testCtx); err != nil {
		t.Fatal(err)
	}
	waitState(t, aStatus, syncclient.StateIdle)

	task := newTask(t, a.core, domain.CreateTaskInput{Title: "through the poke"})
	aRunner.Nudge()
	waitTimer(t, aClock, time.Second)
	aClock.Advance(time.Second)
	waitState(t, aStatus, syncclient.StateIdle)

	// B's fake clock never moves: only its open SSE connection can bring the task over.
	waitState(t, bStatus, syncclient.StateIdle)
	got := must[*domain.Task](t)(b.core.GetTask(testCtx, task.ID))
	if got == nil || got.Title != "through the poke" {
		t.Fatalf("B's task = %+v, want through the poke", got)
	}
}
