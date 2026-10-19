package syncclient_test

import (
	"database/sql"
	"fmt"
	"testing"
)

const seedTimestamp = "2026-10-01T00:00:00.000Z"

// seedRich fills db with the snapshot test 1 needs, as an unlinked database would already have it:
// projects with a sub-project, a user category, a milestone, 1500 tasks (some subtasks, some done,
// some sorted under the category), and 300 reports. The device-local columns get values too, so
// the test can prove they do not travel.
func seedRich(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.BeginTx(testCtx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, p := range []struct {
		id     int64
		name   string
		parent any
	}{
		{1, "Alpha", nil},
		{2, "Alpha docs", int64(1)},
	} {
		if _, err := tx.ExecContext(testCtx, `INSERT INTO projects
			(id, name, repo_slug, repo_path, created_at, updated_at, archived_at, parent_id)
			VALUES (?, ?, NULL, NULL, ?, ?, NULL, ?)`, p.id, p.name, seedTimestamp, seedTimestamp, p.parent); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.ExecContext(testCtx, `UPDATE projects SET repo_path = '/repos/alpha' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(testCtx, `INSERT INTO categories
		(id, builtin, name, icon, color, position, created_at) VALUES (3, NULL, 'Reading', ?, 'blue', 3, ?)`,
		dotIcon, seedTimestamp); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(testCtx, `INSERT INTO milestones
		(id, project_id, title, due_at, status, created_at) VALUES (1, 1, 'v1', NULL, 'open', ?)`,
		seedTimestamp); err != nil {
		t.Fatal(err)
	}

	task, err := tx.PrepareContext(testCtx, `INSERT INTO tasks
		(id, project_id, milestone_id, title, notes, status, priority, source, created_at, updated_at,
		 completed_at, reminded_at, parent_id, category_id)
		VALUES (?, 1, ?, ?, ?, ?, 'P2', 'user', ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = task.Close() }()
	for i := int64(1); i <= 1500; i++ {
		var milestone, parent, category, completed, reminded, notes any
		if i%3 == 0 {
			milestone = int64(1)
		}
		if i%10 == 0 && i > 10 {
			parent = i - 1
		}
		if i%2 == 0 {
			category = int64(3)
		}
		status := "todo"
		if i%7 == 0 {
			status = "done"
			completed = seedTimestamp
		}
		if i == 7 {
			reminded = seedTimestamp
		}
		if i == 1 {
			notes = "first note"
		}
		if _, err := task.ExecContext(testCtx, i, milestone, fmt.Sprintf("task %d", i), notes,
			status, seedTimestamp, seedTimestamp, completed, reminded, parent, category); err != nil {
			t.Fatal(err)
		}
	}

	report, err := tx.PrepareContext(testCtx, `INSERT INTO reports
		(id, task_id, agent, status, summary, next_step, created_at, cwd, kind, links)
		VALUES (?, ?, 'agent', NULL, ?, NULL, ?, ?, '', '[]')`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = report.Close() }()
	for i := int64(1); i <= 300; i++ {
		cwd := ""
		if i == 1 {
			cwd = "/work/dir"
		}
		if _, err := report.ExecContext(testCtx, i, i*5, fmt.Sprintf("report %d", i), seedTimestamp, cwd); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// TestS2b1SeedAndJoin is acceptance criterion 1: a device with a rich local database seeds the
// workspace; an empty device joins it; both end equal, and the seeder's device-local columns are
// untouched.
func TestS2b1SeedAndJoin(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	seedRich(t, a.db)
	a.link(t, "A")

	if got := count(t, a.db, "SELECT COUNT(*) FROM tasks"); got != 1500 {
		t.Fatalf("A has %d tasks after linking, want 1500", got)
	}
	for name, check := range map[string]struct {
		query string
		want  string
	}{
		"repo_path":   {"SELECT COALESCE(repo_path, '') FROM projects WHERE id = 1", "/repos/alpha"},
		"cwd":         {"SELECT cwd FROM reports WHERE id = 1", "/work/dir"},
		"reminded_at": {"SELECT COALESCE(reminded_at, '') FROM tasks WHERE id = 7", seedTimestamp},
	} {
		if got := str(t, a.db, check.query); got != check.want {
			t.Errorf("A's %s = %q after linking, want %q", name, got, check.want)
		}
	}

	b := h.newDevice(t, nil)
	b.link(t, "B")
	assertSyncedEqual(t, a, b)

	if got := count(t, b.db, "SELECT COUNT(*) FROM tasks"); got != 1500 {
		t.Errorf("B has %d tasks, want 1500", got)
	}
	if got := count(t, b.db, "SELECT COUNT(*) FROM reports"); got != 300 {
		t.Errorf("B has %d reports, want 300", got)
	}
	for name, check := range map[string]struct {
		query string
		want  string
	}{
		"repo_path":   {"SELECT COALESCE(repo_path, '') FROM projects WHERE id = 1", ""},
		"cwd":         {"SELECT cwd FROM reports WHERE id = 1", ""},
		"reminded_at": {"SELECT COALESCE(reminded_at, '') FROM tasks WHERE id = 7", ""},
	} {
		if got := str(t, b.db, check.query); got != check.want {
			t.Errorf("B's %s = %q, want %q (device-local columns must not sync)", name, got, check.want)
		}
	}

	// The subtask and category links survived the snapshot.
	if got := count(t, b.db, "SELECT COUNT(*) FROM tasks WHERE parent_id IS NOT NULL"); got == 0 {
		t.Error("B has no subtasks")
	}
	if got := count(t, b.db, "SELECT COUNT(*) FROM tasks WHERE category_id = 3"); got != 750 {
		t.Errorf("B has %d sorted tasks, want 750", got)
	}
}
