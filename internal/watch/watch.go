// Package watch notices changes made to the database by other processes (the
// MCP server, a second app window) and hands out due reminders, for the app.
//
// SQLite's PRAGMA data_version changes whenever a different connection commits,
// so the poller holds one dedicated connection and compares the value every tick.
package watch

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// DefaultInterval is how often the database is polled.
const DefaultInterval = 500 * time.Millisecond

// Config describes what to watch and who to tell. Every callback is optional
// and is called from the Run goroutine, one at a time, so a slow callback
// delays the next poll rather than racing with it.
type Config struct {
	// DB is the database to watch. The watcher pins one of its connections.
	DB *sql.DB
	// Core claims due reminders. Nil disables reminders.
	Core *domain.Core
	// Interval between polls; DefaultInterval when zero.
	Interval time.Duration
	// Now is the clock used for reminders; time.Now when nil.
	Now func() time.Time
	// OnChange is called after the database changed since the previous poll,
	// including changes made through Core by this very process.
	OnChange func()
	// OnReminders is called with reminders that became due. Each reminder is
	// delivered exactly once across all processes (see Core.ClaimDueReminders).
	OnReminders func([]domain.Reminder)
	// OnError is called when a poll fails; the watcher keeps going.
	OnError func(error)
}

// Run polls until ctx is cancelled, then returns nil. It returns an error only
// when it cannot start (no connection or no baseline version).
func Run(ctx context.Context, cfg Config) error {
	if cfg.DB == nil {
		return fmt.Errorf("watch: Config.DB is required")
	}
	conn, err := cfg.DB.Conn(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil // cancelled before it started: nothing to watch
		}
		return fmt.Errorf("watch: acquire connection: %w", err)
	}
	defer func() { _ = conn.Close() }()
	last, err := dataVersion(ctx, conn)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	w := &poller{cfg: cfg, conn: conn, last: last}
	if w.cfg.Interval <= 0 {
		w.cfg.Interval = DefaultInterval
	}
	if w.cfg.Now == nil {
		w.cfg.Now = time.Now
	}
	// A reminder may have come due while nothing was running.
	w.claimReminders(ctx)
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

type poller struct {
	cfg  Config
	conn *sql.Conn
	last int64
}

func (p *poller) poll(ctx context.Context) {
	p.claimReminders(ctx)
	v, err := dataVersion(ctx, p.conn)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	if v != p.last {
		p.last = v
		if p.cfg.OnChange != nil {
			p.cfg.OnChange()
		}
	}
}

func (p *poller) claimReminders(ctx context.Context) {
	if p.cfg.Core == nil || p.cfg.OnReminders == nil {
		return
	}
	now := p.cfg.Now()
	due, err := p.cfg.Core.HasDueReminders(ctx, now)
	if err != nil || !due {
		p.fail(ctx, err)
		return
	}
	claimed, err := p.cfg.Core.ClaimDueReminders(ctx, now)
	if err != nil {
		p.fail(ctx, err)
		return
	}
	if len(claimed) > 0 {
		p.cfg.OnReminders(claimed)
	}
}

// fail reports err unless it is nil or only the result of shutting down.
func (p *poller) fail(ctx context.Context, err error) {
	if err != nil && ctx.Err() == nil && p.cfg.OnError != nil {
		p.cfg.OnError(err)
	}
}

func dataVersion(ctx context.Context, conn *sql.Conn) (int64, error) {
	var v int64
	if err := conn.QueryRowContext(ctx, "PRAGMA data_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("watch: read data_version: %w", err)
	}
	return v, nil
}
