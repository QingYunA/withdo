package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// richTask is a task with every kind of field and two reports, as an agent would leave it.
func richTask(t *testing.T, c *domain.Core, projectID, milestoneID *int64) *domain.Task {
	t.Helper()
	task := mustTask(t, c, domain.CreateTaskInput{
		Title: "Port the store", Source: domain.SourceAgent, ProjectID: projectID, MilestoneID: milestoneID,
		Status: domain.StatusReview, Priority: domain.P1, Notes: ptr("notes"), Branch: ptr("feat/store"),
		PRURL: ptr("https://github.com/acme/widgets/pull/7"), DueAt: ptr("2026-10-20T00:00:00.000Z"),
		RemindAt: ptr("2026-10-19T09:00:00.000Z"), CategoryID: ptr(int64(1)),
	})
	st := domain.StatusDoing
	must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "claude-code", Summary: "first", Status: &st, NextStep: ptr("second"), Cwd: "/work/tree"}))
	must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "github", Summary: "PR #7 merged"}))
	return must[*domain.Task](t)(c.GetTask(ctx, task.ID))
}

func TestRestoreTaskBringsBackTheSameRowAndReports(t *testing.T) {
	c, clk, db := newCoreDB(t)
	p := mustProject(t, c, "acme/widgets")
	ms := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	task := richTask(t, c, &p.ID, &ms.ID)
	exec(t, db, "UPDATE tasks SET pr_state='merged', pr_checks='passing', pr_review='approved', pr_checked_at=?, reminded_at=? WHERE id=?",
		"2026-10-01T00:00:00.000Z", "2026-10-01T00:00:00.000Z", task.ID)
	want := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	wantReports := must[[]domain.Report](t)(c.ListReports(ctx, task.ID))

	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, task.ID))
	noErr(t, c.DeleteTask(ctx, task.ID))
	clk.advance(5 * time.Second) // the restore is later than the delete; the row keeps its own times

	got := must[*domain.Task](t)(c.RestoreTask(ctx, *snap))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("restored %+v\nwant %+v", got, want)
	}
	gotReports := must[[]domain.Report](t)(c.ListReports(ctx, task.ID))
	if !reflect.DeepEqual(gotReports, wantReports) {
		t.Errorf("reports %+v, want %+v", gotReports, wantReports)
	}
}

func TestSnapshotOfAMissingTaskIsNotFound(t *testing.T) {
	c, _ := newCore(t)
	if _, err := c.SnapshotTask(ctx, 999); !domain.IsNotFound(err) {
		t.Errorf("err = %v, want not found", err)
	}
}

func TestRestoreRefusesATaskThatIsStillThere(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "here"})
	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, task.ID))
	if _, err := c.RestoreTask(ctx, *snap); !domain.IsConflict(err) {
		t.Errorf("err = %v, want conflict", err)
	}
	if got := must[[]domain.Task](t)(c.ListTasks(ctx, domain.TaskFilter{})); len(got) != 1 {
		t.Errorf("%d tasks, want 1", len(got))
	}
}

func TestRestoreFallsBackToTheInboxWhenTheProjectIsGone(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	ms := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	task := mustTask(t, c, domain.CreateTaskInput{Title: "orphan", ProjectID: &p.ID, MilestoneID: &ms.ID})
	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, task.ID))
	noErr(t, c.DeleteTask(ctx, task.ID))
	must[domain.Deleted](t)(c.DeleteProject(ctx, p.ID))

	got := must[*domain.Task](t)(c.RestoreTask(ctx, *snap))
	if got.ProjectID != nil || got.MilestoneID != nil {
		t.Errorf("project %v, milestone %v: want the Inbox with no milestone", got.ProjectID, got.MilestoneID)
	}
	eq(t, "title", got.Title, "orphan")
}

func TestRestoreIsAtomicWhenAReportCannotBeInserted(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "t"})
	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, task.ID))
	noErr(t, c.DeleteTask(ctx, task.ID))
	snap.Reports = []domain.Report{
		{ID: 1, TaskID: task.ID, Agent: "a", Summary: "one", CreatedAt: "2026-10-01T00:00:00.000Z"},
		{ID: 1, TaskID: task.ID, Agent: "a", Summary: "same id", CreatedAt: "2026-10-01T00:00:00.000Z"},
	}
	if _, err := c.RestoreTask(ctx, *snap); err == nil {
		t.Fatal("restore succeeded with two reports of one id")
	}
	if got := must[*domain.Task](t)(c.GetTask(ctx, task.ID)); got != nil {
		t.Errorf("the task was left behind: %+v", got)
	}
}

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRestoreLeavesTheTaskUnsortedWhenItsCategoryIsGone(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "Reading")
	task := mustTask(t, c, domain.CreateTaskInput{Title: "book", CategoryID: &reading.ID})
	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, task.ID))
	noErr(t, c.DeleteTask(ctx, task.ID))
	must[domain.DeletedCategory](t)(c.DeleteCategory(ctx, reading.ID))

	got := must[*domain.Task](t)(c.RestoreTask(ctx, *snap))
	if got.CategoryID != nil || got.Category != nil {
		t.Errorf("category = %v %v, want none", got.CategoryID, got.Category)
	}
}
