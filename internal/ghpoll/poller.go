package ghpoll

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// Defaults of a Poller.
const (
	DefaultInterval    = 5 * time.Minute // between checks while the app runs
	DefaultMinGap      = time.Minute     // the least age of the last check for a window activation to start one
	DefaultConcurrency = 4               // gh calls at once
)

// Store is what the poller needs of internal/domain; *domain.Core satisfies it.
type Store interface {
	ListPRTasks(ctx context.Context) ([]domain.Task, error)
	RecordPR(ctx context.Context, taskID int64, u domain.PRUpdate) (bool, error)
}

// Config tunes a Poller. Zero values mean the defaults.
type Config struct {
	Runner      Runner
	Store       Store
	Now         func() time.Time
	Interval    time.Duration
	MinGap      time.Duration
	Concurrency int
	Log         func(format string, args ...any)
}

// Status is what Settings shows.
type Status struct {
	State     AuthState
	Account   string
	CheckedAt time.Time // when the last check ended; zero before the first
	Checking  bool
}

// Poller follows the pull requests of tasks: at Start, every Interval, and when asked. It never
// blocks its callers; everything it learns is written through the Store.
type Poller struct {
	cfg  Config
	kick chan struct{}

	mu      sync.Mutex
	status  Status
	subs    map[int]func()
	nextSub int
	cancel  context.CancelFunc
	done    chan struct{}
}

// New returns a Poller; call Start to run it.
func New(cfg Config) *Poller {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultInterval
	}
	if cfg.MinGap <= 0 {
		cfg.MinGap = DefaultMinGap
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = DefaultConcurrency
	}
	if cfg.Log == nil {
		cfg.Log = log.Printf
	}
	return &Poller{cfg: cfg, kick: make(chan struct{}, 1), subs: map[int]func(){}}
}

// Start begins polling in the background with a check right away. It does nothing when started.
func (p *Poller) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel, p.done = cancel, make(chan struct{})
	go p.loop(ctx, p.done)
}

// Stop ends polling and waits for a check in progress to give up.
func (p *Poller) Stop() {
	p.mu.Lock()
	cancel, done := p.cancel, p.done
	p.cancel = nil
	p.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func (p *Poller) loop(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(p.cfg.Interval)
	defer ticker.Stop()
	p.Poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-p.kick:
		}
		p.Poll(ctx)
	}
}

// CheckNow asks for a check at once, whatever the age of the last one.
func (p *Poller) CheckNow() {
	select {
	case p.kick <- struct{}{}:
	default: // one is already waiting
	}
}

// Activate is called when the window comes back to the front: it asks for a check when the last
// one is older than the minimum gap.
func (p *Poller) Activate() {
	st := p.Status()
	if st.Checking || !st.CheckedAt.IsZero() && p.cfg.Now().Sub(st.CheckedAt) < p.cfg.MinGap {
		return
	}
	p.CheckNow()
}

// Status is the state after the last check.
func (p *Poller) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

// Subscribe calls fn, on an arbitrary goroutine, whenever the status changes. The returned func
// cancels it.
func (p *Poller) Subscribe(fn func()) func() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nextSub++
	id := p.nextSub
	p.subs[id] = fn
	return func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		delete(p.subs, id)
	}
}

func (p *Poller) notify() {
	p.mu.Lock()
	fns := make([]func(), 0, len(p.subs))
	for _, fn := range p.subs {
		fns = append(fns, fn)
	}
	p.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// Poll runs one check: whether gh can be used, then every followed pull request. A check already
// running makes this call return at once. Cancelling ctx abandons the check, writing nothing more.
func (p *Poller) Poll(ctx context.Context) {
	p.mu.Lock()
	if p.status.Checking {
		p.mu.Unlock()
		return
	}
	p.status.Checking = true
	p.mu.Unlock()
	p.notify()

	auth := CheckAuth(ctx, p.cfg.Runner)
	if auth.State == AuthReady && ctx.Err() == nil {
		p.refresh(ctx)
	}

	p.mu.Lock()
	p.status.Checking = false
	if ctx.Err() == nil {
		p.status.State, p.status.Account, p.status.CheckedAt = auth.State, auth.Account, p.cfg.Now()
	}
	p.mu.Unlock()
	p.notify()
}

// refresh asks GitHub about each followed pull request, at most Concurrency at a time, then writes
// the outcomes one after another.
func (p *Poller) refresh(ctx context.Context) {
	tasks, err := p.cfg.Store.ListPRTasks(ctx)
	if err != nil {
		p.cfg.Log("ghpoll: list tasks: %v", err)
		return
	}
	var urls []string
	byURL := map[string][]domain.Task{}
	for _, t := range tasks {
		if t.PRURL == nil || t.PRNumber == nil { // not a github.com pull request URL
			continue
		}
		if _, seen := byURL[*t.PRURL]; !seen {
			urls = append(urls, *t.PRURL)
		}
		byURL[*t.PRURL] = append(byURL[*t.PRURL], t)
	}
	seen := p.fetch(ctx, urls)
	if ctx.Err() != nil {
		return
	}
	for _, url := range urls {
		snap, ok := seen[url]
		if !ok {
			continue
		}
		for _, t := range byURL[url] {
			plan := Decide(t.PullRequest(), snap, *t.PRNumber)
			_, err := p.cfg.Store.RecordPR(ctx, t.ID, domain.PRUpdate{
				URL: url, Snapshot: plan.Snapshot, Done: plan.Done, Reports: plan.Reports,
			})
			if err != nil {
				p.cfg.Log("ghpoll: record %s: %v", url, err)
			}
		}
	}
}

// fetch runs gh pr view for each url. A pull request gh cannot answer for is left out.
func (p *Poller) fetch(ctx context.Context, urls []string) map[string]domain.PRSnapshot {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = map[string]domain.PRSnapshot{}
		sem  = make(chan struct{}, p.cfg.Concurrency)
	)
	for _, url := range urls {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			out, err := p.cfg.Runner.Run(ctx, "pr", "view", url, "--json", PRViewFields)
			if err == nil {
				var snap domain.PRSnapshot
				if snap, err = ParsePRView(out.Stdout); err == nil {
					mu.Lock()
					seen[url] = snap
					mu.Unlock()
					return
				}
			}
			if ctx.Err() == nil {
				p.cfg.Log("ghpoll: %s: %v", url, err)
			}
		}()
	}
	wg.Wait()
	return seen
}
