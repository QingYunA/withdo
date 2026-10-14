// Package store opens the adhd-todo SQLite database and keeps its schema up to date.
//
// The schema and the PRAGMA user_version numbering are shared with the
// original TypeScript core, so a database created by either opens in the other.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure Go SQLite driver, registered as "sqlite"
)

// BusyTimeoutMillis is how long a connection waits for another writer's lock.
const BusyTimeoutMillis = 5000

// Querier is the subset of *sql.DB, *sql.Conn and *sql.Tx the data layer needs.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Open creates the database directory if needed, opens the database in WAL
// mode with foreign keys on, and applies pending migrations.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	if err := Migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// dsn builds a file: URI whose pragmas run on every pooled connection.
func dsn(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", BusyTimeoutMillis))
	q.Add("_pragma", "foreign_keys(1)")
	u.RawQuery = q.Encode()
	return u.String()
}

// Immediate runs fn inside BEGIN IMMEDIATE ... COMMIT on a dedicated
// connection. Taking the write lock up front means a concurrent writer can
// never invalidate fn's snapshot, so readers inside fn see what they write.
// fn's error (or a panic) rolls the transaction back.
func Immediate(ctx context.Context, db *sql.DB, fn func(q Querier) error) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		// The caller's context may be cancelled; the rollback must still run.
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")
	}()
	if err := fn(conn); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	return nil
}
