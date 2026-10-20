package data

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/watch"
)

// RealOptions tune a Real store. The zero value uses the wall clock and the default poll interval,
// and finds the agents on this Mac.
type RealOptions struct {
	Now      func() time.Time
	Interval time.Duration
	// Agents says which agents are installed and where; nil looks at this Mac.
	Agents *agentsetup.Env
	// DataFolder is where the pictures of tasks are kept, in images/. Empty keeps none.
	DataFolder string
	// ImageGrace is how long the files of a picture stay after its last task let go of it, so
	// that Undo can bring the picture back; zero is two minutes.
	ImageGrace time.Duration
	// SyncConfigPath is the sync.json to read; empty means next to the database in OpenReal and
	// no sync in NewReal.
	SyncConfigPath string
	// Secrets is where an OIDC sync keeps its refresh token; nil means the app has no keychain.
	Secrets syncclient.Secrets
	// SyncOpen opens a URL in the browser for the OIDC sign-in; nil means the app cannot sign in.
	SyncOpen func(string) error
}

// Real is the Store backed by the shared SQLite database. Writes go through internal/domain, and
// changes made by other processes, such as the MCP server, arrive through internal/watch.
type Real struct {
	core *domain.Core
	db   *sql.DB
	opts RealOptions

	mu       sync.Mutex
	dbPathMu sync.Mutex
	// tokens is the one OIDC source of this database: the issuer rotates refresh tokens, so two
	// sources refreshing at once (the runner and the Settings pane) would sign each other out.
	tokensMu  sync.Mutex
	tokens    *syncclient.OIDCSource
	tokensKey string
	subs      map[int]func()
	remSubs   map[int]func(Reminder)
	evSubs    map[int]func(Event)
	nextSub   int
	// eventMark is how far the reports were told; only the watcher's goroutine touches it, after
	// Start.
	eventMark eventMark
	cancel    context.CancelFunc
	done      chan struct{}
	syncPath  string
	dbPath    string
	runner    *syncclient.Runner
	sync      syncTask
	// syncNotify asks the shell to redraw after a sync status change; set by the app.
	syncNotify func()

	// images writes the pictures of tasks; imgMu keeps a save and its link from meeting a
	// collection in between. sweep is the pending collection after a detach or a delete.
	images imageFiles
	imgMu  sync.Mutex
	sweep  *time.Timer

	// demo and tempDir are set by OpenDemo: the clock the seed moves, and the folder Close removes.
	demo    *demoClock
	tempDir string
}

var _ Store = (*Real)(nil)

// OpenReal opens (and migrates) the database at path. Call Start once the subscriptions are in
// place, and Close on exit.
func OpenReal(ctx context.Context, path string, opts RealOptions) (*Real, error) {
	db, err := store.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	r := NewReal(db, opts)
	r.dbPath = path
	if opts.SyncConfigPath == "" {
		r.syncPath = syncclient.ConfigPath(path)
	}
	return r, nil
}

// NewReal wraps an open database; Close closes it.
func NewReal(db *sql.DB, opts RealOptions) *Real {
	if opts.Agents == nil {
		home, _ := os.UserHomeDir()
		opts.Agents = agentsetup.New(home)
	}
	return &Real{
		core: domain.New(db, domain.Options{Now: opts.Now}), db: db, opts: opts,
		subs: map[int]func(){}, remSubs: map[int]func(Reminder){}, evSubs: map[int]func(Event){},
		images: imageFiles{dir: imagesDir(opts.DataFolder)}, syncPath: opts.SyncConfigPath,
	}
}

// Core is the domain layer the store writes through, for the built-in agent's tools.
func (r *Real) Core() *domain.Core { return r.core }

// Start begins watching the database. Reminders that came due while the app was closed are
// delivered right away, so subscribe first. Start is a no-op when already started.
func (r *Real) Start() {
	r.mu.Lock()
	if r.cancel != nil {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel, r.done = cancel, make(chan struct{})
	r.mu.Unlock()
	// The watcher's baseline comes before the mark, so a report filed once Start returns, or in
	// between, changes the database after the baseline and is not lost.
	w, _ := watch.Open(ctx, watch.Config{
		DB: r.db, Core: r.core, Interval: r.opts.Interval, Now: r.opts.Now,
		OnChange: r.changed, OnReminders: r.deliver,
	})
	r.markEvents()
	go func() {
		defer close(r.done)
		if w != nil {
			w.Run(ctx)
		}
	}()
	r.startSync(ctx)
}

// startSync starts the background sync runner when sync.json is next to the database and the
// database is linked. It is a no-op otherwise; a runner a 401 stopped is started again.
func (r *Real) startSync(ctx context.Context) {
	r.mu.Lock()
	existing := r.runner
	if existing != nil {
		r.mu.Unlock()
		// A runner a 401 stopped can start again once the user signed in; its client reads the
		// refresh token the sign-in just stored.
		if existing.Status().State == syncclient.StateSignedOut {
			r.mu.Lock()
			r.sync.status = syncclient.Status{State: syncclient.StateSyncing}
			r.mu.Unlock()
			_ = existing.Start(ctx)
		}
		return
	}
	if r.syncPath == "" {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	cfg, found, err := syncclient.LoadConfigAt(r.syncPath)
	if err != nil || !found {
		return
	}
	device, linked, err := r.core.SyncState(ctx, "device_id")
	if err != nil || !linked || device == "" {
		return
	}
	var tokens syncclient.TokenSource
	switch {
	case cfg.Issuer != "":
		if r.opts.Secrets == nil {
			return
		}
		tokens = r.oidcSource(cfg.Issuer, cfg.ClientID, cfg.Scopes)
	case cfg.Token != "":
		tokens = syncclient.StaticToken(cfg.Token)
	default:
		return
	}
	client, err := syncclient.New(r.core, syncclient.Options{BaseURL: cfg.Server, TokenSource: tokens})
	if err != nil {
		return
	}
	runner := syncclient.NewRunner(client, syncclient.RunnerOptions{OnStatus: r.syncStatusChanged})
	r.mu.Lock()
	if r.runner == nil && r.cancel != nil {
		r.runner = runner
	} else {
		runner = nil
	}
	r.mu.Unlock()
	if runner == nil {
		return
	}
	if err := runner.Start(ctx); err != nil {
		r.mu.Lock()
		if r.runner == runner {
			r.runner = nil
		}
		r.mu.Unlock()
	}
}

// oidcSource is this database's token source, keyed to it so a second profile or the CLI cannot
// replace this Mac's refresh token. The same settings get the same source, so the runner and the
// pane share one refresh.
func (r *Real) oidcSource(issuer, clientID string, scopes []string) *syncclient.OIDCSource {
	key := issuer + "\n" + clientID + "\n" + strings.Join(scopes, " ")
	r.tokensMu.Lock()
	defer r.tokensMu.Unlock()
	if r.tokens != nil && r.tokensKey == key {
		return r.tokens
	}
	source := syncclient.NewOIDCSource(issuer, clientID, scopes, r.opts.Secrets)
	source.DBPath = r.databasePath()
	r.tokens, r.tokensKey = source, key
	return source
}

// databasePath is this database's file path, for keying the refresh token. OpenReal knows it; a
// database opened by NewReal is asked once through PRAGMA database_list.
func (r *Real) databasePath() string {
	r.dbPathMu.Lock()
	defer r.dbPathMu.Unlock()
	if r.dbPath != "" {
		return r.dbPath
	}
	if r.db == nil {
		return ""
	}
	rows, err := r.db.Query("PRAGMA database_list")
	if err != nil {
		return ""
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var seq int
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return ""
		}
		if name == "main" {
			r.dbPath = file
			return file
		}
	}
	return ""
}

// syncRunner is the running background runner, for the tests.
func (r *Real) syncRunner() *syncclient.Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runner
}

// nudgeSync asks the runner to upload a local change. Changes made by the sync itself (the cookie,
// the applying flag) reach the watcher too, so only a pending outbox is worth a nudge.
func (r *Real) nudgeSync() {
	runner := r.syncRunner()
	if runner == nil {
		return
	}
	if pending, err := r.core.SyncPending(r.ctx()); err == nil && pending > 0 {
		runner.Nudge()
	}
}

// Close stops the watcher and closes the database.
func (r *Real) Close() error {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel = nil
	runner := r.runner
	r.runner = nil
	if r.sweep != nil {
		r.sweep.Stop()
	}
	r.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	if runner != nil {
		_ = runner.Close()
	}
	err := r.db.Close()
	if r.tempDir != "" {
		_ = os.RemoveAll(r.tempDir)
	}
	return err
}

func (r *Real) ctx() context.Context { return context.Background() }

// Now is the store's clock.
func (r *Real) Now() time.Time {
	if r.opts.Now != nil {
		return r.opts.Now()
	}
	return time.Now()
}

func (r *Real) CreateTask(in CreateTaskInput) (Task, error) {
	t, err := r.core.CreateTask(r.ctx(), domain.CreateTaskInput{
		Title: in.Title, Source: domain.Source(in.Source), ProjectID: int64Ptr(in.ProjectID),
		Status: domain.Status(in.Status), Priority: domain.Priority(in.Priority), DueAt: formatTimePtr(in.DueAt),
		ParentID: int64Ptr(in.ParentID), CategoryID: nonZero(in.Category),
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

// changed runs from the watcher after the database changed: new events first, then the redraw.
func (r *Real) changed() {
	r.deliverEvents()
	r.nudgeSync()
	r.notify()
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
