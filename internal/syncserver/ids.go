package syncserver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// leaseIDs is POST /v1/ids: it takes the next half-open block of row ids for an entity, so a
// device can create rows offline without collisions. The workspace clock and the counter row make
// concurrent leases, pushes and seeding takeovers serial.
func (s *Server) leaseIDs(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r.Context())
	var req syncproto.IDsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	if _, ok := syncproto.EntityColumns(req.Entity); !ok {
		writeError(w, badRequest("unknown entity %q", req.Entity))
		return
	}
	if req.Count < 1 || req.Count > syncproto.MaxIDsPerRequest {
		writeError(w, badRequest("count must be between 1 and %d", syncproto.MaxIDsPerRequest))
		return
	}

	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		writeError(w, fmt.Errorf("begin id lease: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := lockWorkspaceClock(ctx, tx, sess.workspaceID); err != nil {
		writeError(w, err)
		return
	}
	var ignored int64
	if err := tx.QueryRow(ctx, `SELECT last_mutation_id FROM devices
		WHERE id = $1 AND workspace_id = $2 AND user_id = $3`,
		req.DeviceID, sess.workspaceID, sess.userID).Scan(&ignored); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, deviceNotInAccount())
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
	if seeder != "" && seeder != req.DeviceID {
		writeError(w, conflict("workspace is seeding"))
		return
	}
	var end int64
	if err := tx.QueryRow(ctx, `INSERT INTO id_counters (workspace_id, entity, next) VALUES ($1, $2, $3 + 1)
		ON CONFLICT (workspace_id, entity) DO UPDATE SET next = id_counters.next + $3
		RETURNING next`, sess.workspaceID, req.Entity, req.Count).Scan(&end); err != nil {
		writeError(w, fmt.Errorf("lease ids: %w", err))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, fmt.Errorf("commit id lease: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, syncproto.IDsResponse{Start: end - int64(req.Count), End: end})
}
