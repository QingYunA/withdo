package data

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/watch"
)

// RealOptions tune a Real store. The zero value uses the wall clock and the default poll interval.
type RealOptions struct {
	Now      func() time.Time
	Interval time.Duration
}

// Real is the Store backed by the shared SQLite database. Writes go through internal/domain, and
// changes made by other processes, such as the MCP server, arrive through internal/watch.
type Real struct {
	core *domain.Core
	db   *sql.DB
	opts RealOptions

	mu      sync.Mutex
	subs    map[int]func()
	remSubs map[int]func(Reminder)
	nextSub int
	cancel  context.CancelFunc
	done    chan struct{}
}

var _ Store = (*Real)(nil)

// OpenReal opens (and migrates) the database at path. Call Start once the subscriptions are in
// place, and Close on exit.
func OpenReal(ctx context.Context, path string, opts RealOptions) (*Real, error) {
	db, err := store.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	return NewReal(db, opts), nil
}

// NewReal wraps an open database; Close closes it.
func NewReal(db *sql.DB, opts RealOptions) *Real {
	return &Real{
		core: domain.New(db, domain.Options{Now: opts.Now}), db: db, opts: opts,
		subs: map[int]func(){}, remSubs: map[int]func(Reminder){},
	}
}

// Core is the domain layer the store writes through, for the built-in agent's tools.
func (r *Real) Core() *domain.Core { return r.core }

// Start begins watching the database. Reminders that came due while the app was closed are
// delivered right away, so subscribe first. Start is a no-op when already started.
func (r *Real) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel, r.done = cancel, make(chan struct{})
	go func() {
		defer close(r.done)
		_ = watch.Run(ctx, watch.Config{
			DB: r.db, Core: r.core, Interval: r.opts.Interval, Now: r.opts.Now,
			OnChange: r.notify, OnReminders: r.deliver,
		})
	}()
}

// Close stops the watcher and closes the database.
func (r *Real) Close() error {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	return r.db.Close()
}

func (r *Real) ctx() context.Context { return context.Background() }

func (r *Real) ListProjects() ([]Project, error) {
	ps, err := r.core.ListProjects(r.ctx())
	if err != nil {
		return nil, err
	}
	var out []Project
	for _, p := range ps {
		if p.ArchivedAt == nil {
			out = append(out, fromProject(p))
		}
	}
	slices.SortStableFunc(out, func(a, b Project) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		}
		return 0
	})
	return out, nil
}

func (r *Real) ListMilestones(projectID int) ([]Milestone, error) {
	ms, err := r.core.ListMilestones(r.ctx(), int64(projectID))
	if err != nil {
		return nil, err
	}
	out := make([]Milestone, len(ms))
	for i, m := range ms {
		out[i] = fromMilestone(m)
	}
	return out, nil
}

func (r *Real) ListTasks(f Filter) ([]Task, error) {
	ts, err := r.core.ListTasks(r.ctx(), domain.TaskFilter{
		ProjectID: int64Ptr(f.ProjectID), Inbox: f.Inbox, Statuses: toStatuses(f.Status),
		MilestoneID: int64Ptr(f.MilestoneID), Limit: math.MaxInt32,
	})
	if err != nil {
		return nil, err
	}
	hidden, err := r.archivedIDs()
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, t := range ts {
		inArchived := t.ProjectID != nil && hidden[int(*t.ProjectID)]
		if !inArchived && (f.Source == "" || Source(t.Source) == f.Source) {
			out = append(out, fromTask(t))
		}
	}
	return out, nil
}

func (r *Real) GetTask(id int) (TaskDetail, error) {
	t, err := r.core.GetTask(r.ctx(), int64(id))
	if err != nil {
		return TaskDetail{}, err
	}
	if t == nil {
		return TaskDetail{}, fmt.Errorf("task %d not found", id)
	}
	rs, err := r.core.ListReports(r.ctx(), int64(id))
	if err != nil {
		return TaskDetail{}, err
	}
	d := TaskDetail{Task: fromTask(*t)}
	for i := len(rs) - 1; i >= 0; i-- { // newest first
		d.Reports = append(d.Reports, fromReport(rs[i]))
	}
	return d, nil
}

func (r *Real) CreateTask(in CreateTaskInput) (Task, error) {
	t, err := r.core.CreateTask(r.ctx(), domain.CreateTaskInput{
		Title: in.Title, Source: domain.Source(in.Source), ProjectID: int64Ptr(in.ProjectID),
		Status: domain.Status(in.Status), Priority: domain.Priority(in.Priority),
	})
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}

func (r *Real) UpdateTask(id int, p Patch) (Task, error) {
	t, err := r.core.UpdateTask(r.ctx(), int64(id), toPatch(p))
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}

// KeepClosedPR implements Store.
func (r *Real) KeepClosedPR(id int) error {
	return r.core.DismissClosedPR(r.ctx(), int64(id))
}

func (r *Real) Overview() (Overview, error) {
	o, err := r.core.GetOverview(r.ctx(), domain.OverviewOptions{})
	if err != nil {
		return Overview{}, err
	}
	return fromOverview(o), nil
}

// Subscribe implements Store.
func (r *Real) Subscribe(fn func()) func() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextSub++
	id := r.nextSub
	r.subs[id] = fn
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.subs, id)
	}
}

// SubscribeReminders implements Store.
func (r *Real) SubscribeReminders(fn func(Reminder)) func() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextSub++
	id := r.nextSub
	r.remSubs[id] = fn
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.remSubs, id)
	}
}

func (r *Real) notify() {
	r.mu.Lock()
	fns := make([]func(), 0, len(r.subs))
	for _, fn := range r.subs {
		fns = append(fns, fn)
	}
	r.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (r *Real) deliver(rs []domain.Reminder) {
	r.mu.Lock()
	fns := make([]func(Reminder), 0, len(r.remSubs))
	for _, fn := range r.remSubs {
		fns = append(fns, fn)
	}
	r.mu.Unlock()
	for _, d := range rs {
		rem := Reminder{TaskID: int(d.TaskID), Title: d.Title, ProjectName: str(d.ProjectName)}
		for _, fn := range fns {
			fn(rem)
		}
	}
}
