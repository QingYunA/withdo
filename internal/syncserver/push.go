package syncserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// push is POST /v1/push. All mutations of a request are one transaction, together with the
// device's last_mutation_id and the workspace clock, so a failed push changes nothing.
func (s *Server) push(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r.Context())
	var req syncproto.PushRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	if len(req.Mutations) > syncproto.MaxPushMutations {
		writeError(w, badRequest("at most %d mutations per push", syncproto.MaxPushMutations))
		return
	}
	last, version, changed, err := s.applyPush(r.Context(), sess, &req)
	if err != nil {
		writeError(w, err)
		return
	}
	if changed {
		s.hub.publish(sess.workspaceID, version)
	}
	writeJSON(w, http.StatusOK, syncproto.PushResponse{LastMutationID: last})
}

// applyPush runs the push rules in one transaction. A mutation at or below last_mutation_id was
// already applied and is skipped; the remaining ids must rise. A put merges its columns into the
// row (a delete wins over a later put), and a new row needs an id the workspace has already
// handed out, except while its seeding device writes the initial snapshot.
func (s *Server) applyPush(ctx context.Context, sess *session, req *syncproto.PushRequest) (last, version int64, changed bool, err error) {
	if req.DeviceID == "" {
		return 0, 0, false, badRequest("device_id is required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, false, fmt.Errorf("begin push: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The workspace clock is the lock every seeding decision shares; take it before the device.
	version, err = lockWorkspaceClock(ctx, tx, sess.workspaceID)
	if err != nil {
		return 0, 0, false, err
	}
	if err := tx.QueryRow(ctx, `SELECT last_mutation_id FROM devices
		WHERE id = $1 AND workspace_id = $2 AND user_id = $3 FOR UPDATE`,
		req.DeviceID, sess.workspaceID, sess.userID).Scan(&last); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, false, forbidden("device does not belong to this user")
		}
		return 0, 0, false, fmt.Errorf("read device: %w", err)
	}
	seeder, err := workspaceSeeder(ctx, tx, sess.workspaceID)
	if err != nil {
		return 0, 0, false, err
	}
	if seeder != "" && seeder != req.DeviceID {
		return 0, 0, false, conflict("workspace is seeding")
	}
	seeding := seeder == req.DeviceID

	maxID := last
	prev := last
	seedingMax := map[string]int64{}
	for _, m := range req.Mutations {
		if m.ID <= last {
			continue
		}
		if m.ID <= prev {
			return 0, 0, false, badRequest("mutation ids must rise within a request")
		}
		if _, ok := syncproto.EntityColumns(m.Entity); !ok {
			return 0, 0, false, badRequest("unknown entity %q", m.Entity)
		}
		if m.RowID <= 0 {
			return 0, 0, false, badRequest("row_id must be positive")
		}
		if m.Op != syncproto.OpPut && m.Op != syncproto.OpDelete {
			return 0, 0, false, badRequest("unknown op %q", m.Op)
		}
		if m.Op == syncproto.OpPut && !syncproto.ValidData(m.Entity, m.Data) {
			return 0, 0, false, badRequest("unknown column for entity %q", m.Entity)
		}

		var raw sql.NullString
		var deleted bool
		err := tx.QueryRow(ctx, `SELECT data::text, deleted FROM rows
			WHERE workspace_id = $1 AND entity = $2 AND row_id = $3`,
			sess.workspaceID, m.Entity, m.RowID).Scan(&raw, &deleted)
		exists := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, false, fmt.Errorf("read row: %w", err)
		}
		if m.Op == syncproto.OpPut && exists && deleted {
			// A delete wins: the row stays a tombstone and takes no new version.
			prev = m.ID
			if m.ID > maxID {
				maxID = m.ID
			}
			continue
		}

		if m.Op == syncproto.OpPut && !exists && !seeding {
			var next int64
			if err := tx.QueryRow(ctx, `SELECT next FROM id_counters
				WHERE workspace_id = $1 AND entity = $2`,
				sess.workspaceID, m.Entity).Scan(&next); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return 0, 0, false, conflict("row id %d is outside a leased id block", m.RowID)
				}
				return 0, 0, false, fmt.Errorf("read id counter: %w", err)
			}
			if m.RowID >= next {
				return 0, 0, false, conflict("row id %d is outside a leased id block", m.RowID)
			}
		}

		version++
		changed = true
		if m.Op == syncproto.OpDelete {
			if _, err := tx.Exec(ctx, `INSERT INTO rows (workspace_id, entity, row_id, data, deleted, version)
				VALUES ($1, $2, $3, NULL, true, $4)
				ON CONFLICT (workspace_id, entity, row_id) DO UPDATE
				SET data = NULL, deleted = true, version = EXCLUDED.version`,
				sess.workspaceID, m.Entity, m.RowID, version); err != nil {
				return 0, 0, false, fmt.Errorf("write delete: %w", err)
			}
		} else {
			merged := map[string]any{}
			if raw.Valid {
				merged, err = decodeData(raw.String)
				if err != nil {
					return 0, 0, false, fmt.Errorf("decode row data: %w", err)
				}
			}
			for column, value := range m.Data {
				merged[column] = value
			}
			encoded, err := json.Marshal(merged)
			if err != nil {
				return 0, 0, false, fmt.Errorf("encode row data: %w", err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO rows (workspace_id, entity, row_id, data, deleted, version)
				VALUES ($1, $2, $3, $4::jsonb, false, $5)
				ON CONFLICT (workspace_id, entity, row_id) DO UPDATE
				SET data = EXCLUDED.data, deleted = false, version = EXCLUDED.version`,
				sess.workspaceID, m.Entity, m.RowID, string(encoded), version); err != nil {
				return 0, 0, false, fmt.Errorf("write put: %w", err)
			}
		}
		if seeding && m.RowID > seedingMax[m.Entity] {
			seedingMax[m.Entity] = m.RowID
		}
		prev = m.ID
		if m.ID > maxID {
			maxID = m.ID
		}
	}

	if changed {
		if _, err := tx.Exec(ctx, "UPDATE workspace_versions SET version = $1 WHERE workspace_id = $2",
			version, sess.workspaceID); err != nil {
			return 0, 0, false, fmt.Errorf("advance workspace version: %w", err)
		}
	}
	for entity, maxRowID := range seedingMax {
		// Every seeding push raises the counters above the largest id the snapshot wrote, so later
		// leases can never collide with it.
		if _, err := tx.Exec(ctx, `INSERT INTO id_counters (workspace_id, entity, next)
			VALUES ($1, $2, $3)
			ON CONFLICT (workspace_id, entity) DO UPDATE
			SET next = GREATEST(id_counters.next, EXCLUDED.next)`,
			sess.workspaceID, entity, maxRowID+1); err != nil {
			return 0, 0, false, fmt.Errorf("raise id counter: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE devices SET last_mutation_id = $1 WHERE id = $2",
		maxID, req.DeviceID); err != nil {
		return 0, 0, false, fmt.Errorf("update device mutation id: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, false, fmt.Errorf("commit push: %w", err)
	}
	return maxID, version, changed, nil
}
