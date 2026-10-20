package syncserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// pushState is the write side of one push transaction: the workspace clock and the seeding rules
// travel together, so every change a reference cascade makes gets its own version and, while a
// snapshot is seeding, raises the id counters above the ids it wrote.
type pushState struct {
	ctx         context.Context
	tx          pgx.Tx
	workspaceID int64
	version     int64
	changed     bool
	seeding     bool
	seedingMax  map[string]int64
}

// nextVersion advances the workspace clock for one row change and returns the new version.
func (p *pushState) nextVersion() int64 {
	p.version++
	p.changed = true
	return p.version
}

// noteRow remembers the largest id a seeding push wrote for entity.
func (p *pushState) noteRow(entity string, rowID int64) {
	if p.seeding && rowID > p.seedingMax[entity] {
		p.seedingMax[entity] = rowID
	}
}

// readRow returns a row's stored JSON and deleted flag. exists is false when the row has no
// tombstone yet.
func (p *pushState) readRow(entity string, rowID int64) (raw sql.NullString, deleted, exists bool, err error) {
	err = p.tx.QueryRow(p.ctx, `SELECT data::text, deleted FROM rows
		WHERE workspace_id = $1 AND entity = $2 AND row_id = $3`,
		p.workspaceID, entity, rowID).Scan(&raw, &deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return raw, false, false, nil
	}
	if err != nil {
		return raw, false, false, fmt.Errorf("read row: %w", err)
	}
	return raw, deleted, true, nil
}

// referenceState reports whether a referenced row is a tombstone. A row that does not exist yet
// is not deleted: a put may name a parent that a later mutation of the same push creates.
func (p *pushState) referenceState(entity string, rowID int64) (deleted, exists bool, err error) {
	err = p.tx.QueryRow(p.ctx, `SELECT deleted FROM rows
		WHERE workspace_id = $1 AND entity = $2 AND row_id = $3`,
		p.workspaceID, entity, rowID).Scan(&deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read reference target: %w", err)
	}
	return deleted, true, nil
}

// liveReferencingIDs lists the live rows of ref.Entity whose ref.Column points at rowID.
func (p *pushState) liveReferencingIDs(ref syncproto.Reference, rowID int64) ([]int64, error) {
	rows, err := p.tx.Query(p.ctx, `SELECT row_id FROM rows
		WHERE workspace_id = $1 AND entity = $2 AND deleted = false AND data->>$3 = $4
		ORDER BY row_id`,
		p.workspaceID, ref.Entity, ref.Column, strconv.FormatInt(rowID, 10))
	if err != nil {
		return nil, fmt.Errorf("find %s.%s references: %w", ref.Entity, ref.Column, err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan %s.%s reference: %w", ref.Entity, ref.Column, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find %s.%s references: %w", ref.Entity, ref.Column, err)
	}
	return ids, nil
}

// setNull clears one referencing column with a version of its own.
func (p *pushState) setNull(ref syncproto.Reference, rowID int64) error {
	version := p.nextVersion()
	if _, err := p.tx.Exec(p.ctx, `UPDATE rows
		SET data = data || jsonb_build_object($3::text, NULL), version = $4
		WHERE workspace_id = $1 AND entity = $2 AND row_id = $5 AND deleted = false`,
		p.workspaceID, ref.Entity, ref.Column, version, rowID); err != nil {
		return fmt.Errorf("clear %s.%s of row %d: %w", ref.Entity, ref.Column, rowID, err)
	}
	p.noteRow(ref.Entity, rowID)
	return nil
}

// deleteCascade applies a delete the way the local schema and domain do: references that delete
// their row are deleted recursively, the others are set NULL, and the target becomes a tombstone
// last. Every change gets a version, so a client sees children before their parent.
func (p *pushState) deleteCascade(entity string, rowID int64, visited map[string]bool) error {
	key := entity + "/" + strconv.FormatInt(rowID, 10)
	if visited[key] {
		return nil
	}
	visited[key] = true
	for _, ref := range syncproto.ReferencesTo(entity) {
		ids, err := p.liveReferencingIDs(ref, rowID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			switch ref.OnDelete {
			case syncproto.DeleteRef:
				if err := p.deleteCascade(ref.Entity, id, visited); err != nil {
					return err
				}
			case syncproto.SetNullRef:
				if err := p.setNull(ref, id); err != nil {
					return err
				}
			}
		}
	}
	version := p.nextVersion()
	if _, err := p.tx.Exec(p.ctx, `INSERT INTO rows (workspace_id, entity, row_id, data, deleted, version)
		VALUES ($1, $2, $3, NULL, true, $4)
		ON CONFLICT (workspace_id, entity, row_id) DO UPDATE
		SET data = NULL, deleted = true, version = EXCLUDED.version`,
		p.workspaceID, entity, rowID, version); err != nil {
		return fmt.Errorf("write delete %s %d: %w", entity, rowID, err)
	}
	p.noteRow(entity, rowID)
	return nil
}

// resolvePutReferences keeps a put from planting an orphan. A reference to a deleted target makes
// the put a delete when the reference deletes its row, and clears the column otherwise. It
// reports deleteInstead for the first case.
func (p *pushState) resolvePutReferences(entity string, data map[string]any) (deleteInstead bool, err error) {
	for _, ref := range syncproto.References {
		if ref.Entity != entity {
			continue
		}
		value, ok := data[ref.Column]
		if !ok || value == nil {
			continue
		}
		rowID, ok := referenceID(value)
		if !ok {
			continue
		}
		deleted, exists, err := p.referenceState(ref.Target, rowID)
		if err != nil {
			return false, err
		}
		if !exists || !deleted {
			continue
		}
		switch ref.OnDelete {
		case syncproto.DeleteRef:
			return true, nil
		case syncproto.SetNullRef:
			data[ref.Column] = nil
		}
	}
	return false, nil
}

// referenceID reads a row id from a JSON value; numbers arrive as json.Number.
func referenceID(value any) (int64, bool) {
	switch v := value.(type) {
	case json.Number:
		n, err := strconv.ParseInt(v.String(), 10, 64)
		return n, err == nil
	case float64:
		if v != math.Trunc(v) {
			return 0, false
		}
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	}
	return 0, false
}
