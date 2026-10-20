package syncclient_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// mergeFixture is the two-sided start of criteria 3 and 4: A has an account with Reading and
// acme/app, B has its own rows including a lower-case reading, its own acme/app and a task image.
type mergeFixture struct {
	h *harness
	a *device
	b *device

	reading  *domain.Category
	project  *domain.Project
	aParent  *domain.Task
	bLower   *domain.Category
	bProject *domain.Project
	bOther   *domain.Project
	bParent  *domain.Task
	bChild   *domain.Task
	bSolo    *domain.Task
}

func newMergeFixture(t *testing.T) *mergeFixture {
	t.Helper()
	h := newHarness(t)
	f := &mergeFixture{h: h, a: h.newDevice(t, nil), b: h.newDevice(t, nil)}

	f.reading = must[*domain.Category](t)(f.a.core.CreateCategory(testCtx,
		domain.CreateCategoryInput{Name: "Reading", Icon: dotIcon}))
	f.project = must[*domain.Project](t)(f.a.core.CreateProject(testCtx,
		domain.CreateProjectInput{Name: "app", RepoSlug: "acme/app"}))
	execSQL(t, f.a.db, "UPDATE projects SET repo_path = ? WHERE id = ?", "/a/acme/app", f.project.ID)
	milestone := must[*domain.Milestone](t)(f.a.core.CreateMilestone(testCtx,
		domain.CreateMilestoneInput{ProjectID: f.project.ID, Title: "v1"}))
	f.aParent = newTask(t, f.a.core, domain.CreateTaskInput{
		Title: "A parent", ProjectID: &f.project.ID, MilestoneID: &milestone.ID, CategoryID: &f.reading.ID,
	})
	newTask(t, f.a.core, domain.CreateTaskInput{Title: "A child", ParentID: &f.aParent.ID})
	must[*domain.Report](t)(f.a.core.AddReport(testCtx,
		domain.AddReportInput{TaskID: f.aParent.ID, Agent: "agent", Summary: "A report"}))
	f.a.link(t, "A")

	f.bLower = must[*domain.Category](t)(f.b.core.CreateCategory(testCtx,
		domain.CreateCategoryInput{Name: "reading", Icon: dotIcon}))
	f.bProject = must[*domain.Project](t)(f.b.core.CreateProject(testCtx,
		domain.CreateProjectInput{Name: "app", RepoSlug: "acme/app"}))
	execSQL(t, f.b.db, "UPDATE projects SET repo_path = ? WHERE id = ?", "/b/acme/app", f.bProject.ID)
	f.bOther = must[*domain.Project](t)(f.b.core.CreateProject(testCtx,
		domain.CreateProjectInput{Name: "solo"}))
	bMilestone := must[*domain.Milestone](t)(f.b.core.CreateMilestone(testCtx,
		domain.CreateMilestoneInput{ProjectID: f.bProject.ID, Title: "B milestone"}))
	f.bParent = newTask(t, f.b.core, domain.CreateTaskInput{
		Title: "B parent", ProjectID: &f.bProject.ID, MilestoneID: &bMilestone.ID, CategoryID: &f.bLower.ID,
	})
	f.bChild = newTask(t, f.b.core, domain.CreateTaskInput{Title: "B child", ParentID: &f.bParent.ID})
	f.bSolo = newTask(t, f.b.core, domain.CreateTaskInput{Title: "B solo", ProjectID: &f.bOther.ID})
	must[*domain.Report](t)(f.b.core.AddReport(testCtx,
		domain.AddReportInput{TaskID: f.bChild.ID, Agent: "agent", Summary: "B report"}))
	must[*domain.TaskImage](t)(f.b.core.AttachImage(testCtx, f.bParent.ID, domain.AttachImageInput{
		Image: domain.Image{Hash: strings.Repeat("a", 64), Ext: "png", MIME: "image/png", Bytes: 12},
		Name:  "shot.png", Width: 1, Height: 1,
	}))

	// The ordinary Link names the conflict; the two ways on are criteria 3 and 4.
	if err := f.b.client.Link(testCtx, "B"); !errors.Is(err, syncclient.ErrLocalData) {
		t.Fatalf("B's Link = %v, want ErrLocalData", err)
	}
	return f
}

// bTaskIDs returns every local task of B by title, for the merge's old-to-new map.
func (f *mergeFixture) bTaskIDs(t *testing.T) map[string]int64 {
	t.Helper()
	db := f.b.db
	return map[string]int64{
		"B parent": int64Value(t, db, "SELECT id FROM tasks WHERE title = ?", "B parent"),
		"B child":  int64Value(t, db, "SELECT id FROM tasks WHERE title = ?", "B child"),
		"B solo":   int64Value(t, db, "SELECT id FROM tasks WHERE title = ?", "B solo"),
	}
}

// TestS3UIa3Merge is S3-UI-a acceptance criterion 3: B's own rows join A's account, natural keys
// map onto the account's rows, every reference still resolves, B's own tasks get ids from its fresh
// blocks, merged projects keep B's device-local repo_path, and the returned map covers every B task.
func TestS3UIa3Merge(t *testing.T) {
	f := newMergeFixture(t)
	oldIDs := f.bTaskIDs(t)
	tasksStart := f.h.serverIDCounter(t, "tasks")

	merged, err := f.b.client.LinkMerge(testCtx, "B")
	if err != nil {
		t.Fatalf("LinkMerge: %v", err)
	}
	if len(merged) != len(oldIDs) {
		t.Fatalf("merge returned %d task ids, want %d: %v", len(merged), len(oldIDs), merged)
	}
	for title, old := range oldIDs {
		got, ok := merged[old]
		if !ok {
			t.Fatalf("merge map has no id for B's %q (%d)", title, old)
		}
		if got == old {
			t.Errorf("B's %q kept its old id %d, want a fresh id", title, old)
		}
		if got < tasksStart {
			t.Errorf("B's %q got id %d, below its fresh block start %d", title, got, tasksStart)
		}
	}

	// A pulls what B uploaded, and the two are equal on every synced column.
	f.a.sync(t)
	assertSyncedEqual(t, f.a, f.b)

	// B's lower-case reading joined A's Reading; its acme/app tasks point at A's project.
	for title, id := range map[string]int64{"B parent": merged[oldIDs["B parent"]]} {
		if got := int64Value(t, f.b.db, "SELECT category_id FROM tasks WHERE id = ?", id); got != f.reading.ID {
			t.Errorf("%s category_id = %d, want A's Reading %d", title, got, f.reading.ID)
		}
	}
	for title, old := range map[string]int64{"B parent": oldIDs["B parent"], "B child": oldIDs["B child"]} {
		if got := int64Value(t, f.b.db, "SELECT project_id FROM tasks WHERE id = ?", merged[old]); got != f.project.ID {
			t.Errorf("%s project_id = %d, want A's acme/app %d", title, got, f.project.ID)
		}
	}
	if got := int64Value(t, f.b.db, "SELECT milestone_id FROM tasks WHERE id = ?", merged[oldIDs["B parent"]]); got == 0 {
		t.Error("B parent lost its milestone")
	}
	if got := int64Value(t, f.b.db, "SELECT parent_id FROM tasks WHERE id = ?", merged[oldIDs["B child"]]); got != merged[oldIDs["B parent"]] {
		t.Errorf("B child parent_id = %d, want %d", got, merged[oldIDs["B parent"]])
	}
	if got := int64Value(t, f.b.db, "SELECT task_id FROM task_images WHERE image_hash = ?", strings.Repeat("a", 64)); got != merged[oldIDs["B parent"]] {
		t.Errorf("B's task image points at task %d, want %d", got, merged[oldIDs["B parent"]])
	}
	if got := str(t, f.b.db, "SELECT repo_path FROM projects WHERE id = ?", f.project.ID); got != "/b/acme/app" {
		t.Errorf("B's merged project repo_path = %q, want B's /b/acme/app", got)
	}
	if got := int64Value(t, f.b.db, "SELECT COUNT(*) FROM projects WHERE name = 'solo'"); got != 1 {
		t.Errorf("B's project without a slug has %d rows, want 1", got)
	}
	if got := count(t, f.b.db, "SELECT COUNT(*) FROM tasks WHERE title LIKE 'B %'"); got != len(oldIDs) {
		t.Errorf("B has %d own tasks, want %d", got, len(oldIDs))
	}
	assertNoBrokenReferences(t, f.b.db)
}

// TestS3UIa4Replace is S3-UI-a acceptance criterion 4: LinkReplace leaves B equal to A, and the
// backup next to the database holds B's old rows.
func TestS3UIa4Replace(t *testing.T) {
	f := newMergeFixture(t)
	backup, err := f.b.client.LinkReplace(testCtx, "B")
	if err != nil {
		t.Fatalf("LinkReplace: %v", err)
	}
	wantDir := filepath.Join(testutil.Real(t, filepath.Dir(f.b.path)), "backups")
	if testutil.Real(t, filepath.Dir(backup)) != wantDir {
		t.Errorf("backup = %s, want a file in %s", backup, wantDir)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup file: %v", err)
	}

	assertSyncedEqual(t, f.a, f.b)
	assertNoBrokenReferences(t, f.b.db)
	if got := count(t, f.b.db, "SELECT COUNT(*) FROM tasks WHERE title = 'B parent'"); got != 0 {
		t.Errorf("B kept %d old tasks after replace", got)
	}

	// The backup holds B's old rows, taken before the delete.
	old, err := sql.Open("sqlite", backup)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer func() { _ = old.Close() }()
	var n int
	if err := old.QueryRow("SELECT COUNT(*) FROM tasks WHERE title = 'B parent'").Scan(&n); err != nil {
		t.Fatalf("read backup tasks: %v", err)
	}
	if n != 1 {
		t.Errorf("backup has %d B parent tasks, want 1", n)
	}
	if err := old.QueryRow("SELECT COUNT(*) FROM tasks WHERE title = 'A parent'").Scan(&n); err != nil {
		t.Fatalf("read backup A tasks: %v", err)
	}
	if n != 0 {
		t.Errorf("backup has %d A parent tasks, want none", n)
	}
}

// assertNoBrokenReferences fails on any row PRAGMA foreign_key_check finds.
func assertNoBrokenReferences(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			table  string
			rowID  int64
			parent string
			fkid   int
		)
		if err := rows.Scan(&table, &rowID, &parent, &fkid); err != nil {
			t.Fatalf("scan foreign_key_check: %v", err)
		}
		t.Errorf("broken reference: %s row %d -> %s (fk %d)", table, rowID, parent, fkid)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
}
