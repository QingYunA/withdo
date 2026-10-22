package syncserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// MoveSubject renames the user whose subject is from to to, so the same workspace is found when
// the account signs in with a new identity (a Zitadel user id becoming a PocketBase `pb:` id).
// It runs in one transaction. It refuses when from has no user, and when to already has a
// workspace holding devices or rows; a workspace that only the first request created, empty, is
// dropped to make room.
func (s *Server) MoveSubject(ctx context.Context, from, to string) error {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	switch {
	case from == "" || to == "":
		return errors.New("move subject: both subjects are required")
	case from == to:
		return errors.New("move subject: the subjects are the same")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("move subject: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var fromID int64
	if err := tx.QueryRow(ctx, "SELECT id FROM users WHERE subject = $1 FOR UPDATE", from).Scan(&fromID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("move subject: no user with subject %q", from)
		}
		return fmt.Errorf("move subject: find the source user: %w", err)
	}
	var toID int64
	err = tx.QueryRow(ctx, "SELECT id FROM users WHERE subject = $1 FOR UPDATE", to).Scan(&toID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return fmt.Errorf("move subject: find the target user: %w", err)
	default:
		if err := dropEmptyUser(ctx, tx, toID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE users SET subject = $1 WHERE id = $2", to, fromID); err != nil {
		return fmt.Errorf("move subject: update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("move subject: commit: %w", err)
	}
	return nil
}

// dropEmptyUser deletes a user and the empty workspace the first request made for it, or refuses
// when that workspace holds a device or a row.
func dropEmptyUser(ctx context.Context, tx pgx.Tx, userID int64) error {
	var used bool
	if err := tx.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM devices WHERE user_id = $1)
		OR EXISTS (SELECT 1 FROM rows r JOIN workspaces w ON w.id = r.workspace_id WHERE w.owner_id = $1)`,
		userID).Scan(&used); err != nil {
		return fmt.Errorf("move subject: inspect the target: %w", err)
	}
	if used {
		return errors.New("move subject: the target already has a workspace with data")
	}
	for _, stmt := range []string{
		"DELETE FROM id_counters WHERE workspace_id IN (SELECT id FROM workspaces WHERE owner_id = $1)",
		"DELETE FROM workspace_versions WHERE workspace_id IN (SELECT id FROM workspaces WHERE owner_id = $1)",
		"DELETE FROM workspaces WHERE owner_id = $1",
		"DELETE FROM users WHERE id = $1",
	} {
		if _, err := tx.Exec(ctx, stmt, userID); err != nil {
			return fmt.Errorf("move subject: drop the empty target: %w", err)
		}
	}
	return nil
}
