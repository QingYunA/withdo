package syncclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// errPullConflict marks the 409 a pull answers: the cookie is ahead of the workspace, or another
// device is seeding. It rolls the pull's write back so the caller can reset the cookie and retry.
var errPullConflict = errors.New("syncclient: pull conflict")

// pullAll pages through the server's changes from the stored cookie until there are none left, in
// one ApplyChanges write for the whole pull. A 409 with a cookie of 0 can only be the workspace
// seeding, because 0 is never ahead of the workspace version; a 409 with a real cookie means the
// cookie is from a reset workspace, so it starts again from 0. It returns how many changes it
// applied.
func (c *Client) pullAll(ctx context.Context) (int, error) {
	deviceID, err := c.deviceID(ctx)
	if err != nil {
		return 0, err
	}
	applied := 0
	for {
		cookie, err := c.cookie(ctx)
		if err != nil {
			return applied, err
		}
		err = c.core.ApplyChanges(ctx, func(q store.Querier) error {
			if _, err := q.ExecContext(ctx, "PRAGMA defer_foreign_keys = ON"); err != nil {
				return fmt.Errorf("defer foreign keys: %w", err)
			}
			for {
				page, err := c.pull(ctx, deviceID, cookie, syncproto.DefaultPullLimit)
				if err != nil {
					if isStatus(err, http.StatusConflict) {
						return errPullConflict
					}
					return err
				}
				if err := applyChanges(ctx, q, page.Changes); err != nil {
					return err
				}
				if err := setState(ctx, q, stateCookie, strconv.FormatInt(page.Cookie, 10)); err != nil {
					return err
				}
				cookie = page.Cookie
				applied += len(page.Changes)
				if !page.More {
					return nil
				}
			}
		})
		if err == nil {
			return applied, nil
		}
		if !errors.Is(err, errPullConflict) {
			return applied, fmt.Errorf("pull: %w", err)
		}
		if cookie == 0 {
			return applied, ErrSeeding
		}
		if err := c.setState(ctx, stateCookie, "0"); err != nil {
			return applied, err
		}
	}
}

// applyChanges writes one page's changes inside the pull's write: a change to a row with outbox
// entries is skipped, because the local change wins until it is uploaded and the server's merged
// row comes back on a later pull.
func applyChanges(ctx context.Context, q store.Querier, changes []syncproto.Change) error {
	for _, change := range changes {
		pending, err := hasOutbox(ctx, q, change.Entity, change.RowID)
		if err != nil {
			return err
		}
		if pending {
			continue
		}
		if change.Deleted {
			if err := deleteRow(ctx, q, change.Entity, change.RowID); err != nil {
				return err
			}
			continue
		}
		if err := upsertRow(ctx, q, change); err != nil {
			return err
		}
	}
	return nil
}

// hasOutbox reports whether the row still has a local change waiting to upload.
func hasOutbox(ctx context.Context, q store.Querier, entity string, rowID int64) (bool, error) {
	var found int
	err := q.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM sync_outbox WHERE entity = ? AND row_id = ?)", entity, rowID).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("read outbox for %s %d: %w", entity, rowID, err)
	}
	return found != 0, nil
}

// deleteRow removes the local row of a tombstone.
func deleteRow(ctx context.Context, q store.Querier, entity string, rowID int64) error {
	if _, ok := syncproto.EntityColumns(entity); !ok {
		return fmt.Errorf("syncclient: pull names unknown entity %q", entity)
	}
	if _, err := q.ExecContext(ctx, "DELETE FROM "+quoteIdent(entity)+" WHERE id = ?", rowID); err != nil {
		return fmt.Errorf("delete %s %d: %w", entity, rowID, err)
	}
	return nil
}

// upsertRow writes the columns a pull change carries. The row's device-local columns and every
// column the change does not name stay as they are.
func upsertRow(ctx context.Context, q store.Querier, change syncproto.Change) error {
	known, ok := syncproto.EntityColumns(change.Entity)
	if !ok {
		return fmt.Errorf("syncclient: pull names unknown entity %q", change.Entity)
	}
	allowed := make(map[string]struct{}, len(known))
	for _, column := range known {
		allowed[column] = struct{}{}
	}
	columns := make([]string, 0, len(change.Data))
	for column := range change.Data {
		if _, ok := allowed[column]; !ok {
			return fmt.Errorf("syncclient: pull names unknown column %q of %s", column, change.Entity)
		}
		columns = append(columns, column)
	}
	if len(columns) == 0 {
		return nil
	}
	sort.Strings(columns)

	names := make([]string, len(columns))
	marks := make([]string, len(columns))
	values := make([]any, len(columns))
	for i, column := range columns {
		names[i] = quoteIdent(column)
		marks[i] = "?"
		value, err := writeValue(change.Data[column])
		if err != nil {
			return fmt.Errorf("syncclient: %s %d column %s: %w", change.Entity, change.RowID, column, err)
		}
		values[i] = value
	}
	updates := make([]string, len(columns))
	for i, column := range columns {
		updates[i] = quoteIdent(column) + " = excluded." + quoteIdent(column)
	}
	query := "INSERT INTO " + quoteIdent(change.Entity) +
		" (id, " + strings.Join(names, ", ") + ") VALUES (?, " + strings.Join(marks, ", ") + ")" +
		" ON CONFLICT(id) DO UPDATE SET " + strings.Join(updates, ", ")
	args := append([]any{change.RowID}, values...)
	if _, err := q.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("write %s %d: %w", change.Entity, change.RowID, err)
	}
	return nil
}

// writeValue turns a JSON value into one SQLite stores in the synced tables: numbers are integers
// (the protocol only carries integer ids and counts), strings and booleans stay themselves, and a
// nested value is kept as the JSON text a TEXT column holds (reports.links).
func writeValue(value any) (any, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case json.Number:
		n, err := strconv.ParseInt(v.String(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("number %q is not an integer", v.String())
		}
		return n, nil
	case string:
		return v, nil
	case bool:
		return v, nil
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return string(raw), nil
	}
}

// quoteIdent quotes a table or column name. Every name is checked against syncproto's lists before
// it gets here; the quoting keeps SQLite's own keywords out of the way.
func quoteIdent(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }
