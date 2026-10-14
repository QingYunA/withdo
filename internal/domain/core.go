package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/QingYunA/adhd-todo/internal/gitctx"
	"github.com/QingYunA/adhd-todo/internal/store"
)

// Core runs every operation against one database. It is safe for concurrent
// use; each write takes the SQLite write lock for its whole transaction.
type Core struct {
	db  *sql.DB
	git gitctx.Resolver
	now func() time.Time
}

// Options customise a Core. The zero value uses real git and the real clock.
type Options struct {
	Git gitctx.Resolver
	Now func() time.Time
}

// New returns a Core over db, which must come from store.Open.
func New(db *sql.DB, opts Options) *Core {
	c := &Core{db: db, git: opts.Git, now: opts.Now}
	if c.git == nil {
		c.git = gitctx.Resolve
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c
}

func (c *Core) stamp() string { return FormatTime(c.now()) }

// write runs fn in an immediate transaction.
func (c *Core) write(ctx context.Context, fn func(q store.Querier) error) error {
	return store.Immediate(ctx, c.db, fn)
}

// scanner is satisfied by *sql.Row and *sql.Rows.
type scanner interface{ Scan(dest ...any) error }

// one adapts a QueryRow result: no rows becomes (nil, nil).
func one[T any](row scanner, scan func(scanner) (T, error)) (*T, error) {
	v, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// collect reads every row with scan and closes rows.
func collect[T any](rows *sql.Rows, scan func(scanner) (T, error)) ([]T, error) {
	defer func() { _ = rows.Close() }()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	return out, nil
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
