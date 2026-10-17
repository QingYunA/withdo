package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/store"
)

var ctx = context.Background()

func open(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "dir", "t.db")
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, path
}

func scalar[T any](t *testing.T, db *sql.DB, query string) T {
	t.Helper()
	var v T
	if err := db.QueryRowContext(ctx, query).Scan(&v); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return v
}

func TestOpenCreatesDirectoriesAppliesPragmasAndMigrations(t *testing.T) {
	db, path := open(t)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	checks := map[string]any{
		"PRAGMA journal_mode": "wal",
		"PRAGMA busy_timeout": int64(5000),
		"PRAGMA foreign_keys": int64(1),
		"PRAGMA user_version": int64(8),
	}
	for q, want := range checks {
		var got any
		if err := db.QueryRowContext(ctx, q).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s = %v, want %v", q, got, want)
		}
	}
	if store.SchemaVersion() != 8 {
		t.Errorf("SchemaVersion = %d, want 8", store.SchemaVersion())
	}
}

func TestPragmasApplyToEveryPooledConnection(t *testing.T) {
	db, _ := open(t)
	for i := 0; i < 3; i++ {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		var fk int
		if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
			t.Fatalf("connection %d foreign_keys = %d, %v", i, fk, err)
		}
	}
}

func TestSchemaTablesAndConstraints(t *testing.T) {
	db, _ := open(t)
	tables := scalar[int](t, db, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('projects','milestones','tasks','reports')")
	if tables != 4 {
		t.Fatalf("expected 4 schema tables, got %d", tables)
	}
	cols, err := db.QueryContext(ctx, "PRAGMA table_info(tasks)")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cols.Close() }()
	found := false
	for cols.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt sql.NullString
		if err := cols.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		if name == "reminded_at" {
			found = notnull == 0
		}
	}
	if !found {
		t.Error("tasks.reminded_at should exist and be nullable")
	}
	bad := map[string]string{
		"foreign key":  "INSERT INTO tasks (project_id, title, source, created_at, updated_at) VALUES (999,'x','user','t','t')",
		"bad status":   "INSERT INTO tasks (title, status, source, created_at, updated_at) VALUES ('x','nope','user','t','t')",
		"bad priority": "INSERT INTO tasks (title, priority, source, created_at, updated_at) VALUES ('x','P9','user','t','t')",
	}
	for name, q := range bad {
		if _, err := db.ExecContext(ctx, q); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestMigrateIsIdempotentAndKeepsData(t *testing.T) {
	db, path := open(t)
	if _, err := db.ExecContext(ctx, "INSERT INTO projects (name, created_at, updated_at) VALUES ('keep','t','t')"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := store.Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()
	again, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = again.Close() }()
	if name := scalar[string](t, again, "SELECT name FROM projects"); name != "keep" {
		t.Errorf("name = %q", name)
	}
}

func TestVersionOneDatabaseUpgradesInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	v1, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{string(v1), "PRAGMA user_version = 1",
		"INSERT INTO tasks (title, source, created_at, updated_at) VALUES ('old','user','t','t')"} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = raw.Close()
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var title string
	var reminded sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT title, reminded_at FROM tasks").Scan(&title, &reminded); err != nil {
		t.Fatal(err)
	}
	if title != "old" || reminded.Valid {
		t.Errorf("got %q, %v", title, reminded)
	}
	if v := scalar[int](t, db, "PRAGMA user_version"); v != 8 {
		t.Errorf("user_version = %d", v)
	}
}

func TestVersionThreeDatabaseGainsParentID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v3.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{}
	for _, f := range []string{"001_init.sql", "002_reminded_at.sql", "003_pull_requests.sql"} {
		b, err := os.ReadFile("migrations/" + f)
		if err != nil {
			t.Fatal(err)
		}
		stmts = append(stmts, string(b))
	}
	stmts = append(stmts, "PRAGMA user_version = 3",
		"INSERT INTO tasks (title, source, created_at, updated_at) VALUES ('old','agent','t','t')")
	for _, q := range stmts {
		if _, err := raw.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = raw.Close()
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var parent sql.NullInt64
	if err := db.QueryRowContext(ctx, "SELECT parent_id FROM tasks WHERE title = 'old'").Scan(&parent); err != nil || parent.Valid {
		t.Fatalf("old task parent_id = %v, %v; want NULL", parent, err)
	}
	if v := scalar[int](t, db, "PRAGMA user_version"); v != 8 {
		t.Errorf("user_version = %d", v)
	}
	if n := scalar[int](t, db, "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='tasks_parent'"); n != 1 {
		t.Error("index tasks_parent missing")
	}
	// A parent that is deleted leaves its subtasks standing.
	if _, err := db.ExecContext(ctx, "INSERT INTO tasks (title, source, parent_id, created_at, updated_at) VALUES ('kid','agent',1,'t','t')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM tasks WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT parent_id FROM tasks WHERE title = 'kid'").Scan(&parent); err != nil || parent.Valid {
		t.Errorf("kid parent_id = %v, %v; want NULL after the parent is deleted", parent, err)
	}
}

func TestOpenRefusesNewerDatabase(t *testing.T) {
	db, path := open(t)
	if _, err := db.ExecContext(ctx, "PRAGMA user_version = 9"); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if _, err := store.Open(ctx, path); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("err = %v, want a 'newer' error", err)
	}
}

func TestOpenErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(ctx, filepath.Join(file, "sub", "t.db")); err == nil {
		t.Error("a directory below a regular file should fail")
	}
	garbage := filepath.Join(dir, "garbage.db")
	if err := os.WriteFile(garbage, []byte(strings.Repeat("not sqlite ", 200)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(ctx, garbage); err == nil {
		t.Error("a non-SQLite file should fail")
	}
}

func TestImmediateCommitsAndRollsBack(t *testing.T) {
	db, _ := open(t)
	insert := func(q store.Querier, name string) error {
		_, err := q.ExecContext(ctx, "INSERT INTO projects (name, created_at, updated_at) VALUES (?, 't', 't')", name)
		return err
	}
	if err := store.Immediate(ctx, db, func(q store.Querier) error { return insert(q, "kept") }); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := store.Immediate(ctx, db, func(q store.Querier) error {
		if err := insert(q, "lost"); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if n := scalar[int](t, db, "SELECT COUNT(*) FROM projects"); n != 1 {
		t.Errorf("projects = %d, want 1 (rollback must discard 'lost')", n)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := store.Immediate(cancelled, db, func(store.Querier) error { return nil }); err == nil {
		t.Error("a cancelled context should fail")
	}
}

func TestConcurrentWritersDoNotSeeBusy(t *testing.T) {
	_, path := open(t)
	const writers, perWriter = 4, 50
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() { // each writer has its own pool, like separate processes
			defer wg.Done()
			db, err := store.Open(ctx, path)
			if err != nil {
				errs <- err
				return
			}
			defer func() { _ = db.Close() }()
			for i := 0; i < perWriter; i++ {
				err := store.Immediate(ctx, db, func(q store.Querier) error {
					_, err := q.ExecContext(ctx, "INSERT INTO projects (name, created_at, updated_at) VALUES (?, 't', 't')", fmt.Sprintf("w%d-%d", w, i))
					return err
				})
				if err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if n := scalar[int](t, db, "SELECT COUNT(*) FROM projects"); n != writers*perWriter {
		t.Errorf("rows = %d, want %d", n, writers*perWriter)
	}
}

// ts-v2.db was written by the TypeScript core (bun:sqlite, user_version 2): the oldest database a
// user can still have. Opening it runs every migration since.
func TestOpensDatabaseCreatedByTypeScriptCore(t *testing.T) {
	db := openCopyOf(t, "testdata/ts-v2.db")
	if v := scalar[int](t, db, "PRAGMA user_version"); v != 8 {
		t.Fatalf("user_version = %d, want 8 (migrations 005 to 008 applied)", v)
	}
	var title, branch string
	var pr int
	if err := db.QueryRowContext(ctx, "SELECT title, branch, pr_number FROM tasks").Scan(&title, &branch, &pr); err != nil {
		t.Fatal(err)
	}
	if title != "Write docs" || branch != "feat/docs" || pr != 3 {
		t.Errorf("got %q %q %d", title, branch, pr)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO reports (task_id, agent, summary, created_at) VALUES (1,'go','hi','t')"); err != nil {
		t.Errorf("writing to a TS-created database: %v", err)
	}
}

// Migration 005 adds reports.cwd: rows that predate it, and rows written without one, read ”.
func TestMigration005AddsReportCwdDefaultingToEmpty(t *testing.T) {
	db := openCopyOf(t, "testdata/ts-v2.db")
	if _, err := db.ExecContext(ctx, "INSERT INTO reports (task_id, agent, summary, created_at) VALUES (1,'go','hi','t')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO reports (task_id, agent, summary, created_at, cwd) VALUES (1,'go','there','t','/work/tree')"); err != nil {
		t.Fatal(err)
	}
	if got := scalar[string](t, db, "SELECT cwd FROM reports WHERE summary = 'hi'"); got != "" {
		t.Errorf("default cwd = %q, want empty", got)
	}
	if got := scalar[string](t, db, "SELECT cwd FROM reports WHERE summary = 'there'"); got != "/work/tree" {
		t.Errorf("cwd = %q", got)
	}
}

// Migration 003 adds the pull request columns; a task that predates them has none of the five set.
func TestMigration003AddsPullRequestColumnsToExistingRows(t *testing.T) {
	db := openCopyOf(t, "testdata/ts-v2.db")
	var state, checks, review, checkedAt, closedSeen sql.NullString
	err := db.QueryRowContext(ctx,
		"SELECT pr_state, pr_checks, pr_review, pr_checked_at, pr_closed_seen_at FROM tasks").
		Scan(&state, &checks, &review, &checkedAt, &closedSeen)
	if err != nil {
		t.Fatal(err)
	}
	for name, v := range map[string]sql.NullString{"pr_state": state, "pr_checks": checks, "pr_review": review,
		"pr_checked_at": checkedAt, "pr_closed_seen_at": closedSeen} {
		if v.Valid {
			t.Errorf("%s = %q, want NULL for a row from before the migration", name, v.String)
		}
	}
	good := "UPDATE tasks SET pr_state='merged', pr_checks='failing', pr_review='changes_requested', pr_checked_at='t', pr_closed_seen_at='t'"
	if _, err := db.ExecContext(ctx, good); err != nil {
		t.Errorf("valid values: %v", err)
	}
	for _, col := range []string{"pr_state", "pr_checks", "pr_review"} {
		if _, err := db.ExecContext(ctx, "UPDATE tasks SET "+col+" = 'nope'"); err == nil {
			t.Errorf("%s should reject an unknown value", col)
		}
	}
}

// Migration 006 adds the hand-off columns; a task that predates them is not waiting for anyone.
func TestMigration006AddsHandOffColumnsToExistingRows(t *testing.T) {
	db := openCopyOf(t, "testdata/ts-v2.db")
	var to, at sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT handed_to, handed_at FROM tasks").Scan(&to, &at); err != nil {
		t.Fatal(err)
	}
	if to.Valid || at.Valid {
		t.Errorf("handed_to = %v, handed_at = %v, want NULL for a row from before the migration", to, at)
	}
}

// Migration 007 adds tasks.category: a task that predates it is not classified, and only work or
// life can be written.
func TestMigration007AddsCategoryToExistingRows(t *testing.T) {
	db := openCopyOf(t, "testdata/ts-v2.db")
	var category sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT category FROM tasks").Scan(&category); err != nil {
		t.Fatal(err)
	}
	if category.Valid {
		t.Errorf("category = %q, want NULL for a row from before the migration", category.String)
	}
	for _, v := range []string{"work", "life"} {
		if _, err := db.ExecContext(ctx, "UPDATE tasks SET category = ?", v); err != nil {
			t.Errorf("category %q: %v", v, err)
		}
	}
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET category = 'nope'"); err == nil {
		t.Error("category should reject an unknown value")
	}
}

// openCopyOf opens a copy of a database file from testdata, so the original stays at its version.
func openCopyOf(t *testing.T, file string) *sql.DB {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "copy.db")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// Migration 008 adds reports.kind: a report that predates it has none.
func TestMigration008AddsKindToExistingReports(t *testing.T) {
	db := openCopyOf(t, "testdata/ts-v2.db")
	var kind string
	if err := db.QueryRowContext(ctx, "SELECT kind FROM reports LIMIT 1").Scan(&kind); err != nil {
		t.Fatal(err)
	}
	if kind != "" {
		t.Errorf("kind = %q, want empty for a report from before the migration", kind)
	}
}
