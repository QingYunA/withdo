package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// ErrNoSyncIDs says a linked database has no leased id block left for an entity. The write that
// needed the id fails and changes nothing; S3 refills the blocks from the server.
var ErrNoSyncIDs = errors.New("no sync ids left")

// SyncChange is one row of the outbox: a local change waiting for the server. Columns is the JSON
// array of the synced columns an update changed, and nil for an insert (every column) or a delete.
type SyncChange struct {
	Seq       int64
	Entity    string
	RowID     int64
	Op        string
	Columns   []string
	CreatedAt string
}

// syncID returns the id a new row of entity takes, or nil when the database is not linked and
// SQLite assigns it. A linked database spends the block with the smallest next: next is the id,
// then advanced, and the block row goes when it is used up.
func (c *Core) syncID(ctx context.Context, q store.Querier, entity string) (*int64, error) {
	on, err := syncEnabled(ctx, q)
	if err != nil {
		return nil, err
	}
	if !on {
		return nil, nil
	}
	for {
		var rowid, next, end int64
		err := q.QueryRowContext(ctx,
			"SELECT rowid, next, end FROM sync_id_blocks WHERE entity = ? ORDER BY next, rowid LIMIT 1",
			entity).Scan(&rowid, &next, &end)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w for %s", ErrNoSyncIDs, entity)
		}
		if err != nil {
			return nil, fmt.Errorf("read %s id block: %w", entity, err)
		}
		if next >= end {
			// An exhausted block from an earlier build: drop it and move on.
			if _, err := q.ExecContext(ctx, "DELETE FROM sync_id_blocks WHERE rowid = ?", rowid); err != nil {
				return nil, fmt.Errorf("drop spent %s id block: %w", entity, err)
			}
			continue
		}
		if _, err := q.ExecContext(ctx, "UPDATE sync_id_blocks SET next = ? WHERE rowid = ?", next+1, rowid); err != nil {
			return nil, fmt.Errorf("advance %s id block: %w", entity, err)
		}
		if next+1 >= end {
			if _, err := q.ExecContext(ctx, "DELETE FROM sync_id_blocks WHERE rowid = ?", rowid); err != nil {
				return nil, fmt.Errorf("drop spent %s id block: %w", entity, err)
			}
		}
		return &next, nil
	}
}

// syncEnabled reports whether the database is linked to a server.
func syncEnabled(ctx context.Context, q store.Querier) (bool, error) {
	var value string
	err := q.QueryRowContext(ctx, "SELECT value FROM sync_state WHERE key = 'enabled'").Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read sync enabled: %w", err)
	}
	return value == "1", nil
}

// ApplyChanges runs fn as one write with the applying flag set, for S3 to write changes from the
// server: the triggers record nothing for the rows it writes. The flag is cleared before the
// transaction commits, and a failing fn rolls it and every change back.
func (c *Core) ApplyChanges(ctx context.Context, fn func(q store.Querier) error) error {
	return c.write(ctx, func(q store.Querier) error {
		if err := setSyncState(ctx, q, "applying", "1"); err != nil {
			return err
		}
		if err := fn(q); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, "DELETE FROM sync_state WHERE key = 'applying'"); err != nil {
			return fmt.Errorf("clear sync applying: %w", err)
		}
		return nil
	})
}

func setSyncState(ctx context.Context, q store.Querier, key, value string) error {
	_, err := q.ExecContext(ctx, `INSERT INTO sync_state (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("set sync %s: %w", key, err)
	}
	return nil
}

// Outbox returns the recorded changes with seq up to and including upTo, oldest first.
func (c *Core) Outbox(ctx context.Context, upTo int64) ([]SyncChange, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT seq, entity, row_id, op, columns, created_at FROM sync_outbox WHERE seq <= ? ORDER BY seq", upTo)
	if err != nil {
		return nil, fmt.Errorf("read outbox: %w", err)
	}
	return collect(rows, scanSyncChange)
}

// DeleteOutbox removes the changes Outbox returned through upTo: the uploader's acknowledgement.
func (c *Core) DeleteOutbox(ctx context.Context, upTo int64) error {
	if _, err := c.db.ExecContext(ctx, "DELETE FROM sync_outbox WHERE seq <= ?", upTo); err != nil {
		return fmt.Errorf("delete outbox through %d: %w", upTo, err)
	}
	return nil
}

func scanSyncChange(s scanner) (SyncChange, error) {
	var ch SyncChange
	var columns sql.NullString
	if err := s.Scan(&ch.Seq, &ch.Entity, &ch.RowID, &ch.Op, &columns, &ch.CreatedAt); err != nil {
		return ch, err
	}
	if columns.Valid {
		if err := json.Unmarshal([]byte(columns.String), &ch.Columns); err != nil {
			return ch, fmt.Errorf("decode outbox columns: %w", err)
		}
	}
	return ch, nil
}
