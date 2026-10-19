package store_test

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// TestMigration014AddsSyncTablesAndKeepsData is acceptance criterion 5: a database at version 13
// with data migrates to 14, keeps its rows, and starts recording changes from then on.
func TestMigration014AddsSyncTablesAndKeepsData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v13.db")
	db, err := sql.Open("sqlite", store.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := store.MigrateTo(ctx, db, 13); err != nil {
		t.Fatal(err)
	}
	if v := scalar[int](t, db, "PRAGMA user_version"); v != 13 {
		t.Fatalf("user_version = %d, want 13 before the migration", v)
	}
	for _, q := range []string{
		`INSERT INTO projects (id, name, created_at, updated_at) VALUES (1, 'keep', 't', 't')`,
		`INSERT INTO tasks (id, project_id, title, source, created_at, updated_at) VALUES (1, 1, 'old', 'user', 't', 't')`,
		`INSERT INTO reports (id, task_id, agent, summary, created_at) VALUES (1, 1, 'x', 'old', 't')`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx,
		"INSERT INTO categories (id, name, icon, color, position, created_at) VALUES (3, 'Reading', ?, 'red', 3, 't')",
		strings.Repeat(".", 144)); err != nil {
		t.Fatal(err)
	}

	if err := store.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if v := scalar[int](t, db, "PRAGMA user_version"); v != 14 {
		t.Fatalf("user_version = %d, want 14", v)
	}
	for query, want := range map[string]string{
		"SELECT name FROM projects WHERE id = 1":   "keep",
		"SELECT title FROM tasks WHERE id = 1":     "old",
		"SELECT summary FROM reports WHERE id = 1": "old",
		"SELECT name FROM categories WHERE id = 3": "Reading",
	} {
		got := scalar[string](t, db, query)
		if got != want {
			t.Errorf("%s = %q, want %q", query, got, want)
		}
	}
	if n := scalar[int](t, db, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('sync_state','sync_outbox','sync_id_blocks')"); n != 3 {
		t.Errorf("sync tables = %d, want 3", n)
	}
	if n := scalar[int](t, db, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'sync_%'"); n != 15 {
		t.Errorf("sync triggers = %d, want 15", n)
	}
	if n := scalar[int](t, db, "SELECT COUNT(*) FROM sync_outbox"); n != 0 {
		t.Errorf("the migration recorded %d changes for old rows, want 0", n)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO sync_state (key, value) VALUES ('enabled', '1')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET title = 'new' WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	if got := scalar[string](t, db, "SELECT columns FROM sync_outbox"); got != `["title"]` {
		t.Errorf("columns = %s, want [\"title\"]", got)
	}
}
