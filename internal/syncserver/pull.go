package syncserver

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// pull is GET /v1/pull: the changes with a version above cookie, oldest first, at most limit. The
// device's last_mutation_id is read in the same snapshot as the changes.
func (s *Server) pull(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r.Context())
	q := r.URL.Query()
	cookie := int64(0)
	if raw := q.Get("cookie"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v < 0 {
			writeError(w, badRequest("invalid cookie"))
			return
		}
		cookie = v
	}
	limit := syncproto.DefaultPullLimit
	if raw := q.Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			writeError(w, badRequest("invalid limit"))
			return
		}
		limit = min(v, syncproto.MaxPullLimit)
	}
	deviceID := q.Get("device_id")
	if deviceID == "" {
		writeError(w, badRequest("device_id is required"))
		return
	}

	ctx := r.Context()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		writeError(w, fmt.Errorf("begin pull: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var last int64
	if err := tx.QueryRow(ctx, `SELECT last_mutation_id FROM devices
		WHERE id = $1 AND workspace_id = $2 AND user_id = $3`,
		deviceID, sess.workspaceID, sess.userID).Scan(&last); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, forbidden("device does not belong to this user"))
			return
		}
		writeError(w, fmt.Errorf("read device: %w", err))
		return
	}
	seeder, err := workspaceSeeder(ctx, tx, sess.workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	if seeder != "" && seeder != deviceID {
		writeError(w, conflict("workspace is seeding"))
		return
	}
	// The same transaction that reads the changes remembers that this device is current.
	if _, err := tx.Exec(ctx, "UPDATE devices SET last_seen_at = now() WHERE id = $1", deviceID); err != nil {
		writeError(w, fmt.Errorf("mark device seen: %w", err))
		return
	}
	var version int64
	if err := tx.QueryRow(ctx,
		"SELECT version FROM workspace_versions WHERE workspace_id = $1",
		sess.workspaceID).Scan(&version); err != nil {
		writeError(w, fmt.Errorf("read workspace version: %w", err))
		return
	}
	if cookie > version {
		writeError(w, conflict("cookie %d is ahead of workspace version %d", cookie, version))
		return
	}

	rows, err := tx.Query(ctx, `SELECT entity, row_id, deleted, data::text, version FROM rows
		WHERE workspace_id = $1 AND version > $2
		ORDER BY version, entity, row_id
		LIMIT $3`, sess.workspaceID, cookie, limit+1)
	if err != nil {
		writeError(w, fmt.Errorf("read changes: %w", err))
		return
	}
	changes := make([]syncproto.Change, 0, limit)
	for rows.Next() {
		var (
			change syncproto.Change
			raw    sql.NullString
		)
		if err := rows.Scan(&change.Entity, &change.RowID, &change.Deleted, &raw, &change.Version); err != nil {
			rows.Close()
			writeError(w, fmt.Errorf("scan change: %w", err))
			return
		}
		if raw.Valid {
			data, err := decodeData(raw.String)
			if err != nil {
				rows.Close()
				writeError(w, fmt.Errorf("decode change data: %w", err))
				return
			}
			change.Data = data
		}
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		writeError(w, fmt.Errorf("read changes: %w", err))
		return
	}
	rows.Close()

	more := len(changes) > limit
	if more {
		changes = changes[:limit]
	}
	out := syncproto.PullResponse{
		Cookie:         version,
		LastMutationID: last,
		Changes:        changes,
		More:           more,
	}
	if len(changes) > 0 {
		out.Cookie = changes[len(changes)-1].Version
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, fmt.Errorf("commit pull: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}
