package syncserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// lockWorkspaceClock takes the workspace's clock row lock and returns its current version.
// Registering a device, a push, leasing ids and ending seeding all take this lock first, so a
// seeding takeover cannot race a push or a lease.
func lockWorkspaceClock(ctx context.Context, tx pgx.Tx, workspaceID int64) (int64, error) {
	var version int64
	if err := tx.QueryRow(ctx,
		"SELECT version FROM workspace_versions WHERE workspace_id = $1 FOR UPDATE",
		workspaceID).Scan(&version); err != nil {
		return 0, fmt.Errorf("lock workspace clock: %w", err)
	}
	return version, nil
}

// workspaceSeeder returns the id of the workspace's seeding device, or "" when it has none.
func workspaceSeeder(ctx context.Context, tx pgx.Tx, workspaceID int64) (string, error) {
	var id string
	err := tx.QueryRow(ctx,
		"SELECT id FROM devices WHERE workspace_id = $1 AND seeding LIMIT 1",
		workspaceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read seeding device: %w", err)
	}
	return id, nil
}
