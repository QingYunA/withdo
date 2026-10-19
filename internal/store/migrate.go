package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrations returns the SQL of each migration in order. The file name
// prefix (001_, 002_) fixes the order; position + 1 is the schema version
// stored in PRAGMA user_version, matching the TypeScript core.
func migrations() ([]string, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, n := range names {
		b, err := migrationFiles.ReadFile("migrations/" + n)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", n, err)
		}
		out = append(out, string(b))
	}
	return out, nil
}

// SchemaVersion is the user_version this build writes.
func SchemaVersion() int {
	m, _ := migrations()
	return len(m)
}

// Migrate applies pending migrations. It is idempotent and safe to run from
// several processes at once: each step re-checks user_version under BEGIN
// IMMEDIATE. A database newer than this build is refused.
func Migrate(ctx context.Context, db *sql.DB) error {
	return migrateTo(ctx, db, SchemaVersion())
}

// migrateTo applies the pending migrations up to version limit, so a test can stop short of the
// latest and write the rows a migration has to carry over.
func migrateTo(ctx context.Context, db *sql.DB, limit int) error {
	all, err := migrations()
	if err != nil {
		return err
	}
	for i := 0; i < limit && i < len(all); i++ {
		version := i + 1
		err := Immediate(ctx, db, func(q Querier) error {
			current, err := userVersion(ctx, q)
			if err != nil {
				return err
			}
			if current > len(all) {
				return fmt.Errorf("database schema version %d is newer than this build supports (%d); upgrade adhd-todo", current, len(all))
			}
			if current >= version {
				return nil
			}
			if _, err := q.ExecContext(ctx, all[i]); err != nil {
				return fmt.Errorf("apply migration %d: %w", version, err)
			}
			if _, err := q.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
				return fmt.Errorf("record schema version %d: %w", version, err)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func userVersion(ctx context.Context, q Querier) (int, error) {
	var v int
	if err := q.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return v, nil
}
