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
// With keepAlias it also records from as an alias of that user, so a device still signed in with
// the old identity reaches the same workspace. It runs in one transaction. It refuses when from has
// no user, when to is an alias, and when to already has a workspace holding devices or rows; a
// workspace that only the first request created, empty, is dropped to make room.
func (s *Server) MoveSubject(ctx context.Context, from, to string, keepAlias bool) error {
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
	if err := refuseAlias(ctx, tx, to, "move subject"); err != nil {
		return err
	}
	if err := dropShellUser(ctx, tx, to, "move subject"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE users SET subject = $1 WHERE id = $2", to, fromID); err != nil {
		return fmt.Errorf("move subject: update: %w", err)
	}
	if keepAlias {
		if _, err := tx.Exec(ctx, "INSERT INTO subject_aliases (subject, user_id) VALUES ($1, $2)", from, fromID); err != nil {
			return fmt.Errorf("move subject: record the alias: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("move subject: commit: %w", err)
	}
	return nil
}

// AddAlias makes alias sign in as the user whose subject is target. It is for an account that was
// moved without keeping the alias. When the alias subject already has a user of its own, created by
// a device's first request after the move, that user is dropped if it is an empty shell and the
// call refuses otherwise.
func (s *Server) AddAlias(ctx context.Context, alias, target string) error {
	alias, target = strings.TrimSpace(alias), strings.TrimSpace(target)
	switch {
	case alias == "" || target == "":
		return errors.New("add alias: both subjects are required")
	case alias == target:
		return errors.New("add alias: the subjects are the same")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("add alias: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var targetID int64
	if err := tx.QueryRow(ctx, "SELECT id FROM users WHERE subject = $1 FOR UPDATE", target).Scan(&targetID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("add alias: no user with subject %q", target)
		}
		return fmt.Errorf("add alias: find the target user: %w", err)
	}
	if err := refuseAlias(ctx, tx, alias, "add alias"); err != nil {
		return err
	}
	if err := dropShellUser(ctx, tx, alias, "add alias"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO subject_aliases (subject, user_id) VALUES ($1, $2)", alias, targetID); err != nil {
		return fmt.Errorf("add alias: insert: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("add alias: commit: %w", err)
	}
	return nil
}

// DropAlias removes an alias; a device still using it gets a new empty user on its next request.
func (s *Server) DropAlias(ctx context.Context, subject string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM subject_aliases WHERE subject = $1", strings.TrimSpace(subject))
	if err != nil {
		return fmt.Errorf("drop alias: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("drop alias: %q is not an alias", subject)
	}
	return nil
}

// SubjectInfo is one subject and the user it reaches, for `adhd-sync list-subjects`.
type SubjectInfo struct {
	UserID  int64
	Subject string
	Alias   bool
	Devices int64
	Rows    int64
}

// ListSubjects returns every subject, the primary one first within each user, with the user's
// device and row counts. It reads no tokens or secrets.
func (s *Server) ListSubjects(ctx context.Context) ([]SubjectInfo, error) {
	rows, err := s.pool.Query(ctx, `
		WITH subjects AS (
			SELECT id AS user_id, subject, false AS alias FROM users
			UNION ALL SELECT user_id, subject, true FROM subject_aliases)
		SELECT s.user_id, s.subject, s.alias,
			(SELECT count(*) FROM devices d WHERE d.user_id = s.user_id),
			(SELECT count(*) FROM rows r JOIN workspaces w ON w.id = r.workspace_id WHERE w.owner_id = s.user_id)
		FROM subjects s ORDER BY s.user_id, s.alias, s.subject`)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()
	var out []SubjectInfo
	for rows.Next() {
		var i SubjectInfo
		if err := rows.Scan(&i.UserID, &i.Subject, &i.Alias, &i.Devices, &i.Rows); err != nil {
			return nil, fmt.Errorf("list subjects: %w", err)
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// refuseAlias fails when subject is already an alias, which would make it both an alias and a
// primary subject.
func refuseAlias(ctx context.Context, tx pgx.Tx, subject, op string) error {
	var taken bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM subject_aliases WHERE subject = $1)", subject).Scan(&taken); err != nil {
		return fmt.Errorf("%s: check aliases: %w", op, err)
	}
	if taken {
		return fmt.Errorf("%s: %q is already an alias; drop-alias it first", op, subject)
	}
	return nil
}

// dropShellUser drops the user with this subject, if there is one and it is an empty shell.
func dropShellUser(ctx context.Context, tx pgx.Tx, subject, op string) error {
	var id int64
	err := tx.QueryRow(ctx, "SELECT id FROM users WHERE subject = $1 FOR UPDATE", subject).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("%s: find the target user: %w", op, err)
	}
	return dropEmptyUser(ctx, tx, id)
}

// dropEmptyUser deletes a user and the empty workspace the first request made for it, or refuses
// when that workspace holds a device or a row, or the user has aliases.
func dropEmptyUser(ctx context.Context, tx pgx.Tx, userID int64) error {
	var used bool
	if err := tx.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM devices WHERE user_id = $1)
		OR EXISTS (SELECT 1 FROM subject_aliases WHERE user_id = $1)
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
