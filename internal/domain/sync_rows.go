package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// SyncedRow is one local row of a synced entity as the sync client reads it: the row's id and the
// values of the columns it asked for. A NULL column is present with a nil value, so an insert can
// send it and clear the column on the other devices.
type SyncedRow struct {
	ID   int64
	Data map[string]any
}

// SyncState returns the value stored under key in sync_state. found is false when the key is not
// there, which is how an unlinked database looks.
func (c *Core) SyncState(ctx context.Context, key string) (value string, found bool, err error) {
	err = c.db.QueryRowContext(ctx, "SELECT value FROM sync_state WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read sync %s: %w", key, err)
	}
	return value, true, nil
}

// SyncRow reads one row of entity with the given synced columns. ok is false when the row is gone:
// a queued put for it is dropped because its delete is in the outbox behind it.
func (c *Core) SyncRow(ctx context.Context, entity string, id int64, columns []string) (SyncedRow, bool, error) {
	cols, err := syncSelectColumns(entity, columns)
	if err != nil {
		return SyncedRow{}, false, err
	}
	row := SyncedRow{ID: id, Data: make(map[string]any, len(columns))}
	values := make([]any, len(columns))
	dest := make([]any, len(columns)+1)
	dest[0] = &row.ID
	for i := range values {
		dest[i+1] = &values[i]
	}
	err = c.db.QueryRowContext(ctx,
		"SELECT id, "+strings.Join(cols, ", ")+" FROM "+quoteSyncIdent(entity)+" WHERE id = ?", id).
		Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return SyncedRow{}, false, nil
	}
	if err != nil {
		return SyncedRow{}, false, fmt.Errorf("read %s row %d: %w", entity, id, err)
	}
	for i, column := range columns {
		row.Data[column] = syncValue(values[i])
	}
	return row, true, nil
}

// SyncHasLocalData reports whether the database holds user rows a joining device would add to a
// workspace that already has data: any project, milestone, task or report, or a category the user
// made. The built-in Work and Life categories do not count.
func (c *Core) SyncHasLocalData(ctx context.Context) (bool, error) {
	var n int
	err := c.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM projects)
		+ (SELECT COUNT(*) FROM milestones)
		+ (SELECT COUNT(*) FROM tasks)
		+ (SELECT COUNT(*) FROM reports)
		+ (SELECT COUNT(*) FROM categories WHERE builtin IS NULL)`).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("count local rows: %w", err)
	}
	return n > 0, nil
}

// SyncIDsLeft reports how many ids of entity the local blocks still hand out.
func (c *Core) SyncIDsLeft(ctx context.Context, entity string) (int64, error) {
	var n int64
	err := c.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(end - next), 0) FROM sync_id_blocks WHERE entity = ? AND end > next", entity).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count %s id blocks: %w", entity, err)
	}
	return n, nil
}

// syncValue turns a driver value into one encoding/json writes unchanged: TEXT stays a string and
// NULL stays nil. A driver that returns bytes would otherwise marshal them as base64.
func syncValue(v any) any {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}

// syncSelectColumns checks and quotes the column list of a read. The sync client passes the names
// from syncproto, never user input, but a bad name must not reach the SQL text.
func syncSelectColumns(entity string, columns []string) ([]string, error) {
	if err := checkSyncIdent(entity); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("sync read of %s names no columns", entity)
	}
	out := make([]string, len(columns))
	for i, column := range columns {
		if err := checkSyncIdent(column); err != nil {
			return nil, err
		}
		out[i] = quoteSyncIdent(column)
	}
	return out, nil
}

// checkSyncIdent accepts the identifiers the sync protocol uses: lower-case letters, digits and
// underscores, starting with a letter or underscore.
func checkSyncIdent(name string) error {
	if name == "" {
		return errors.New("empty sync identifier")
	}
	for i, r := range name {
		switch {
		case r == '_', r >= 'a' && r <= 'z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return fmt.Errorf("invalid sync identifier %q", name)
		}
	}
	return nil
}

// quoteSyncIdent quotes an identifier that checkSyncIdent accepted.
func quoteSyncIdent(name string) string { return `"` + name + `"` }
