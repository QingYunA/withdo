// Package core is the Go core of the Android app, bound to Kotlin with gomobile.
//
// It wraps app/data (the page models the desktop draws), internal/domain and internal/syncclient.
// Every read returns JSON; every write takes the JSON the Kotlin forms send. The JSON is documented
// on each method and pinned by the golden files under testdata.
package core

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/watch"
)

// Secrets is the phone's key/value store, backed by the Android Keystore. Get returns an error
// whose text is "not found" when the key is absent.
type Secrets interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// Listener receives the core's calls. Both methods run on Go goroutines, never on the UI thread.
type Listener interface {
	// OnChange says the database changed, including changes a sync made.
	OnChange()
	// OnStatus carries the same JSON SyncStatus returns when the sync state changes.
	OnStatus(statusJSON string)
}

// Core is one opened database and its sync. Create it with Open and close it with Close.
type Core struct {
	mu       sync.Mutex
	db       *sql.DB
	real     *data.Real
	dom      *domain.Core
	dataDir  string
	dbPath   string
	syncPath string
	secrets  Secrets
	now      func() time.Time

	cancel context.CancelFunc
	done   chan struct{}

	listener Listener
	runner   *syncclient.Runner
	// tokens is the phone's one OIDC source: the issuer rotates refresh tokens, so two sources
	// refreshing at once (the runner and a Sync now) would sign each other out.
	tokensMu    sync.Mutex
	tokens      *syncclient.OIDCSource
	tokensKey   string
	lastStatus  syncclient.Status
	failedSince time.Time
	// links counts SignIn, LinkMerge and LinkReplace in flight; StartSync waits them out.
	links  int
	closed bool
	// signInFn replaces syncclient's in-app sign-in in tests, so the arguments SignIn sends and
	// the device it links can be checked without the real server.
	signInFn func(ctx context.Context, source *syncclient.OIDCSource, server, loginName, password string) error
}

// Open opens dataDir/adhd-todo.db, migrating it, and starts watching it for changes.
func Open(dataDir string, secrets Secrets) (*Core, error) {
	return open(dataDir, secrets, time.Now)
}

// open is Open with an injectable clock, for the tests that pin timestamps.
func open(dataDir string, secrets Secrets, now func() time.Time) (*Core, error) {
	if dataDir == "" {
		return nil, errors.New("a data folder is required")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, errors.New("could not create the data folder")
	}
	dbPath := filepath.Join(dataDir, "adhd-todo.db")
	db, err := store.Open(context.Background(), dbPath)
	if err != nil {
		return nil, errors.New("could not open the database")
	}
	real := data.NewReal(db, data.RealOptions{
		Now: now, Agents: &agentsetup.Env{}, DataFolder: dataDir,
	})
	c := &Core{
		db: db, real: real, dom: real.Core(), dataDir: dataDir, dbPath: dbPath,
		syncPath: syncclient.ConfigPath(dbPath), secrets: secrets, now: now,
		lastStatus: syncclient.Status{State: syncclient.StateIdle},
	}
	ctx, cancel := context.WithCancel(context.Background())
	w, err := watch.Open(ctx, watch.Config{DB: db, Core: c.dom, OnChange: c.dbChanged})
	if err != nil {
		cancel()
		_ = db.Close()
		return nil, errors.New("could not watch the database")
	}
	c.cancel, c.done = cancel, make(chan struct{})
	done := c.done
	go func() {
		defer close(done)
		if w != nil {
			w.Run(ctx)
		}
	}()
	return c, nil
}

// dbChanged is the watcher's signal: the page changed, and a local change may need uploading.
func (c *Core) dbChanged() {
	c.notifyChange()
	c.nudgeSync()
}

// notifyChange tells the listener, outside the core's lock.
func (c *Core) notifyChange() {
	c.mu.Lock()
	l := c.listener
	c.mu.Unlock()
	if l != nil {
		l.OnChange()
	}
}

// nudgeSync asks the runner to upload a change. Changes the sync itself makes (the cookie, the
// applying flag) reach the watcher too, so only a pending outbox is worth a nudge.
func (c *Core) nudgeSync() {
	c.mu.Lock()
	runner := c.runner
	c.mu.Unlock()
	if runner == nil {
		return
	}
	if pending, err := c.dom.SyncPending(context.Background()); err == nil && pending > 0 {
		runner.Nudge()
	}
}

// SetListener replaces the listener; nil removes it.
func (c *Core) SetListener(l Listener) {
	c.mu.Lock()
	c.listener = l
	c.mu.Unlock()
}

// Close stops the watcher and the sync, then closes the database. It is safe to call twice.
func (c *Core) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	cancel, done, runner := c.cancel, c.done, c.runner
	c.cancel, c.done, c.runner, c.listener = nil, nil, nil, nil
	c.mu.Unlock()
	if runner != nil {
		_ = runner.Close()
	}
	if cancel != nil {
		cancel()
		<-done
	}
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// nowTime is the clock the pages are built with.
func (c *Core) nowTime() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}
