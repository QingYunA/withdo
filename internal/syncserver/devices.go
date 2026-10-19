package syncserver

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// createDevice is POST /v1/devices: the server makes a random device id in the caller's
// workspace. A workspace with no rows and no seeder starts seeding here, and a registration while
// another device seeds takes the seeding over from it.
func (s *Server) createDevice(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r.Context())
	var req syncproto.CreateDeviceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, badRequest("name is required"))
		return
	}
	id, err := newDeviceID()
	if err != nil {
		writeError(w, err)
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		writeError(w, fmt.Errorf("begin device insert: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := lockWorkspaceClock(ctx, tx, sess.workspaceID); err != nil {
		writeError(w, err)
		return
	}
	var empty bool
	if err := tx.QueryRow(ctx,
		"SELECT NOT EXISTS (SELECT 1 FROM rows WHERE workspace_id = $1)", sess.workspaceID).
		Scan(&empty); err != nil {
		writeError(w, fmt.Errorf("check workspace rows: %w", err))
		return
	}
	seeder, err := workspaceSeeder(ctx, tx, sess.workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	seeding := false
	if seeder != "" {
		// A new registration takes over: the old device stops seeding and this one starts.
		if _, err := tx.Exec(ctx,
			"UPDATE devices SET seeding = false WHERE workspace_id = $1 AND seeding", sess.workspaceID); err != nil {
			writeError(w, fmt.Errorf("stop old seeding device: %w", err))
			return
		}
		seeding = true
	} else if empty {
		seeding = true
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO devices (id, workspace_id, user_id, name, seeding) VALUES ($1, $2, $3, $4, $5)",
		id, sess.workspaceID, sess.userID, name, seeding); err != nil {
		writeError(w, fmt.Errorf("insert device: %w", err))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, fmt.Errorf("commit device insert: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, syncproto.CreateDeviceResponse{
		DeviceID:    id,
		WorkspaceID: sess.workspaceID,
		Empty:       empty,
		Seeding:     seeding,
	})
}

// deviceSeeded is POST /v1/devices/seeded: the seeding device reports that its snapshot is
// complete. Any other device, including one that lost the seeding to a takeover, gets 403.
func (s *Server) deviceSeeded(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r.Context())
	var req syncproto.SeededRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.DeviceID == "" {
		writeError(w, badRequest("device_id is required"))
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		writeError(w, fmt.Errorf("begin end seeding: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := lockWorkspaceClock(ctx, tx, sess.workspaceID); err != nil {
		writeError(w, err)
		return
	}
	var seeding bool
	err = tx.QueryRow(ctx, `SELECT seeding FROM devices
		WHERE id = $1 AND workspace_id = $2 AND user_id = $3 FOR UPDATE`,
		req.DeviceID, sess.workspaceID, sess.userID).Scan(&seeding)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, forbidden("device does not belong to this user"))
		return
	}
	if err != nil {
		writeError(w, fmt.Errorf("read device: %w", err))
		return
	}
	if !seeding {
		writeError(w, forbidden("device is not seeding"))
		return
	}
	if _, err := tx.Exec(ctx, "UPDATE devices SET seeding = false WHERE id = $1", req.DeviceID); err != nil {
		writeError(w, fmt.Errorf("end seeding: %w", err))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, fmt.Errorf("commit end seeding: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, syncproto.SeededResponse{Seeded: true})
}

// newDeviceID returns a random UUID v4.
func newDeviceID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("make device id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
