package syncclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

// Default durations of the background runner.
const (
	DefaultDebounce   = time.Second
	DefaultInterval   = 5 * time.Minute
	DefaultMinBackoff = 2 * time.Second
	DefaultMaxBackoff = 5 * time.Minute
)

// State is what a Runner is doing.
type State string

// The states of a Runner.
const (
	StateIdle      State = "idle"
	StateSyncing   State = "syncing"
	StateOffline   State = "offline"
	StateError     State = "error"
	StateSeeding   State = "seeding"
	StateSignedOut State = "signed out"
	// StateWrongAccount is a signed-in account that does not own this database's device. The runner
	// stops, like signed out: nothing it could retry would change the answer.
	StateWrongAccount State = "wrong account"
)

// Clock is the time source a Runner uses for its debounce, timer and backoff. Tests inject a fake
// so they never wait for the wall clock.
type Clock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}

// RealClock is the wall clock.
var RealClock Clock = systemClock{}

type systemClock struct{}

func (systemClock) Now() time.Time                         { return time.Now() }
func (systemClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Status is one snapshot of a Runner.
type Status struct {
	State      State
	LastSynced time.Time
	LastError  string
	Pending    int
	// RetryAt is when a failed runner will try again; zero when it is not waiting to retry.
	RetryAt time.Time
}

// RunnerOptions tune a Runner. The zero value uses RealClock and the documented durations.
type RunnerOptions struct {
	Clock Clock
	// HTTP overrides the Client's transport for this runner, for tests and for an app that wants
	// its own timeouts. Nil keeps the Client's.
	HTTP       *http.Client
	OnStatus   func(Status)
	Debounce   time.Duration
	Interval   time.Duration
	MinBackoff time.Duration
	MaxBackoff time.Duration
}

// Runner keeps one linked database in sync in the background. Start it once the app is running,
// call Nudge when the local database changed, and Close it before the database.
type Runner struct {
	client   *Client
	clock    Clock
	onStatus func(Status)
	debounce time.Duration
	interval time.Duration
	minWait  time.Duration
	maxWait  time.Duration

	nudges chan struct{}
	pokes  chan struct{}

	mu     sync.Mutex
	status Status
	life   *runnerLife
	closed bool
}

// runnerLife is one Start: its context and the channel closed when both of its goroutines stop.
type runnerLife struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// NewRunner makes a runner for client. Starting it does nothing until Start is called.
func NewRunner(client *Client, options ...RunnerOptions) *Runner {
	var opts RunnerOptions
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.Clock == nil {
		opts.Clock = RealClock
	}
	if opts.Debounce <= 0 {
		opts.Debounce = DefaultDebounce
	}
	if opts.Interval <= 0 {
		opts.Interval = DefaultInterval
	}
	if opts.MinBackoff <= 0 {
		opts.MinBackoff = DefaultMinBackoff
	}
	if opts.MaxBackoff < opts.MinBackoff {
		opts.MaxBackoff = DefaultMaxBackoff
	}
	if opts.HTTP != nil && client != nil {
		clone := *client
		clone.http = opts.HTTP
		client = &clone
	}
	return &Runner{
		client: client, clock: opts.Clock, onStatus: opts.OnStatus,
		debounce: opts.Debounce, interval: opts.Interval,
		minWait: opts.MinBackoff, maxWait: opts.MaxBackoff,
		nudges: make(chan struct{}, 1), pokes: make(chan struct{}, 1),
		status: Status{State: StateIdle},
	}
}

// Start launches the background loop and its SSE connection. Starting an already running runner is
// a no-op; after a 401 stopped it, Start starts it again. It returns an error only for a nil client
// or a runner that Close has closed.
func (r *Runner) Start(ctx context.Context) error {
	if r == nil || r.client == nil {
		return errors.New("syncclient: cannot start a runner without a client")
	}
	for {
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			return errors.New("syncclient: the runner is closed")
		}
		if life := r.life; life != nil {
			if r.status.State != StateSignedOut && r.status.State != StateWrongAccount {
				r.mu.Unlock()
				return nil
			}
			r.mu.Unlock()
			life.cancel()
			<-life.done
			continue
		}
		runCtx, cancel := context.WithCancel(ctx)
		life := &runnerLife{cancel: cancel, done: make(chan struct{})}
		r.life = life
		r.mu.Unlock()

		go func() {
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); r.runLoop(runCtx) }()
			go func() { defer wg.Done(); r.watchEvents(runCtx) }()
			wg.Wait()
			close(life.done)
			r.mu.Lock()
			if r.life == life {
				r.life = nil
			}
			r.mu.Unlock()
		}()
		return nil
	}
}

// Nudge says the local database changed. The runner syncs after the debounce; nudges that arrive
// while one is already waiting collapse into that one.
func (r *Runner) Nudge() {
	if r == nil {
		return
	}
	select {
	case r.nudges <- struct{}{}:
	default:
	}
}

// SyncNow syncs at once, without the debounce a nudge waits for. It is what Settings' Sync now
// asks for.
func (r *Runner) SyncNow() {
	if r == nil {
		return
	}
	select {
	case r.pokes <- struct{}{}:
	default:
	}
}

// Status is the current state. It is safe to call at any time.
func (r *Runner) Status() Status {
	if r == nil {
		return Status{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// Close stops the loops and waits for them. The runner cannot be started again.
func (r *Runner) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	r.closed = true
	life := r.life
	r.life = nil
	r.mu.Unlock()
	if life != nil {
		life.cancel()
		<-life.done
	}
	return nil
}

// runLoop is the only place a sync starts, so two syncs never overlap.
func (r *Runner) runLoop(ctx context.Context) {
	var debounceCh <-chan time.Time
	var retryCh <-chan time.Time
	retryWait := r.minWait
	periodic := r.clock.After(r.interval)

	run := func() bool {
		ok, stop := r.syncOnce(ctx)
		if stop {
			return true
		}
		if ok {
			retryWait = r.minWait
			retryCh = nil
			debounceCh = nil
			periodic = r.clock.After(r.interval)
			return false
		}
		retryCh = r.clock.After(retryWait)
		r.setRetryAt(r.clock.Now().Add(retryWait))
		retryWait = nextWait(retryWait, r.maxWait)
		return false
	}

	if run() {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.nudges:
			r.refreshPending(ctx)
			if retryCh == nil {
				debounceCh = r.clock.After(r.debounce)
			}
		case <-r.pokes:
			// A poke is the server saying it changed: sync at once.
			debounceCh = nil
			retryCh = nil
			if run() {
				return
			}
		case <-debounceCh:
			debounceCh = nil
			if retryCh != nil {
				continue
			}
			if run() {
				return
			}
		case <-retryCh:
			retryCh = nil
			if run() {
				return
			}
		case <-periodic:
			periodic = r.clock.After(r.interval)
			if retryCh != nil {
				continue
			}
			if run() {
				return
			}
		}
	}
}

// syncOnce runs one Sync and updates the status. ok is false when it failed; stop is true when the
// runner must not try again, which a 401 means.
func (r *Runner) syncOnce(ctx context.Context) (ok, stop bool) {
	current, started := r.setStatusIfLive(ctx, func(current Status) Status {
		return Status{
			State: StateSyncing, LastSynced: current.LastSynced,
			LastError: current.LastError, Pending: current.Pending,
		}
	})
	if !started {
		return false, true
	}
	err := r.client.Sync(ctx)
	if ctx.Err() != nil {
		return false, true
	}
	pending := current.Pending
	if n, pendingErr := r.client.Pending(ctx); pendingErr == nil {
		pending = n
	}
	next := Status{LastSynced: current.LastSynced, Pending: pending}
	switch {
	case err == nil:
		next.State = StateIdle
		next.LastSynced = r.clock.Now()
		r.client.RecordResult(ctx, nil)
	case IsUnauthorized(err) || IsSignedOut(err):
		next.State = StateSignedOut
		next.LastError = err.Error()
		r.client.RecordResult(ctx, err)
	case errors.Is(err, ErrDeviceNotInAccount):
		next.State = StateWrongAccount
		next.LastError = err.Error()
		r.client.RecordResult(ctx, err)
	case errors.Is(err, ErrSeeding):
		next.State = StateSeeding
		next.LastError = err.Error()
		r.client.RecordResult(ctx, err)
	case offline(err):
		next.State = StateOffline
		next.LastError = err.Error()
		r.client.RecordResult(ctx, err)
	default:
		next.State = StateError
		next.LastError = err.Error()
		r.client.RecordResult(ctx, err)
	}
	if next.State == StateSignedOut || next.State == StateWrongAccount {
		r.signOut(next)
		return false, true
	}
	r.setStatus(next)
	return err == nil, false
}

// refreshPending updates the Pending count without syncing.
func (r *Runner) refreshPending(ctx context.Context) {
	n, err := r.client.Pending(ctx)
	if err != nil {
		return
	}
	r.mu.Lock()
	next := r.status
	next.Pending = n
	changed := next != r.status
	r.status = next
	callback := r.onStatus
	r.mu.Unlock()
	if changed && callback != nil {
		callback(next)
	}
}

// setRetryAt says when the next retry is due, for a pane that shows it.
func (r *Runner) setRetryAt(at time.Time) {
	r.mu.Lock()
	next := r.status
	next.RetryAt = at
	changed := next != r.status
	r.status = next
	callback := r.onStatus
	r.mu.Unlock()
	if changed && callback != nil {
		callback(next)
	}
}

// setStatus stores next and tells OnStatus when anything changed. Callbacks run after the lock is
// released so they may call Status.
func (r *Runner) setStatus(next Status) {
	r.mu.Lock()
	changed := next != r.status
	r.status = next
	callback := r.onStatus
	r.mu.Unlock()
	if changed && callback != nil {
		callback(next)
	}
}

// setStatusIfLive stores build(current) and returns current. It stores nothing and returns false
// once ctx is done, checked under the lock signOut holds, so a stopping run never overwrites the
// signed out that signOut stored.
func (r *Runner) setStatusIfLive(ctx context.Context, build func(current Status) Status) (Status, bool) {
	r.mu.Lock()
	current := r.status
	if ctx.Err() != nil {
		r.mu.Unlock()
		return current, false
	}
	next := build(current)
	changed := next != current
	r.status = next
	callback := r.onStatus
	r.mu.Unlock()
	if changed && callback != nil {
		callback(next)
	}
	return current, true
}

// signOut stores next and stops the current run under one lock, used when the token is refused.
// Doing both at once is what lets setStatusIfLive see the stop before it writes.
func (r *Runner) signOut(next Status) {
	r.mu.Lock()
	changed := next != r.status
	r.status = next
	callback := r.onStatus
	if r.life != nil {
		r.life.cancel()
	}
	r.mu.Unlock()
	if changed && callback != nil {
		callback(next)
	}
}

// watchEvents keeps one SSE connection open, reconnecting with a growing delay. A 401 means the
// token is gone: the runner stops in signed out.
func (r *Runner) watchEvents(ctx context.Context) {
	wait := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		deviceID, err := r.client.DeviceID(ctx)
		if err != nil {
			r.setStatusIfLive(ctx, func(current Status) Status {
				return Status{
					State: StateError, LastSynced: current.LastSynced,
					LastError: err.Error(), Pending: current.Pending,
				}
			})
			return
		}
		err = r.client.events(ctx, deviceID, func() {
			select {
			case r.pokes <- struct{}{}:
			default:
			}
		})
		if ctx.Err() != nil {
			return
		}
		if IsUnauthorized(err) || IsSignedOut(err) {
			current := r.Status()
			r.signOut(Status{
				State: StateSignedOut, LastSynced: current.LastSynced,
				LastError: err.Error(), Pending: current.Pending,
			})
			return
		}
		if errors.Is(err, ErrDeviceNotInAccount) {
			current := r.Status()
			r.signOut(Status{
				State: StateWrongAccount, LastSynced: current.LastSynced,
				LastError: err.Error(), Pending: current.Pending,
			})
			return
		}
		if err == nil || errors.Is(err, io.EOF) {
			wait = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-r.clock.After(wait):
		}
		wait = nextWait(wait, r.maxWait)
	}
}

// offline reports whether err is the network or the server being away, as opposed to a refusal.
func offline(err error) bool {
	var se *statusError
	if errors.As(err, &se) {
		return se.status >= 500
	}
	return !errors.Is(err, ErrSeeding) && !errors.Is(err, context.Canceled)
}

// nextWait doubles wait, stopping at max.
func nextWait(wait, max time.Duration) time.Duration {
	if wait <= 0 {
		return DefaultMinBackoff
	}
	if wait >= max {
		return max
	}
	if wait > max/2 {
		return max
	}
	return wait * 2
}
