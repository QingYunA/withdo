package domain_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

func widgetsRepo(t *testing.T, branch string) string {
	t.Helper()
	return testutil.NewRepo(t, t.TempDir(), "widgets", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Branch: branch})
}

func progress(t *testing.T, c *domain.Core, in domain.ProgressInput) *domain.ProgressResult {
	t.Helper()
	if in.Agent == "" {
		in.Agent = "x"
	}
	return must[*domain.ProgressResult](t)(c.ReportProgress(ctx, in))
}

func TestReportProgressCreatesDoingTaskFromCwd(t *testing.T) {
	c, _ := newCore(t)
	r := progress(t, c, domain.ProgressInput{Summary: "started", Cwd: widgetsRepo(t, "feat/login"), Agent: "claude-code"})
	if !r.Created {
		t.Error("first report should create the task")
	}
	eq(t, "title is the first sentence, not the branch", r.Task.Title, "started")
	eqPtr(t, "branch", r.Task.Branch, ptr("feat/login"))
	eq(t, "status", r.Task.Status, domain.StatusDoing)
	eq(t, "source", r.Task.Source, domain.SourceAgent)
	eq(t, "agent", r.Report.Agent, "claude-code")
	if r.Report.Status != nil {
		t.Error("report status should be nil when none was given")
	}
	projects := must[[]domain.Project](t)(c.ListProjects(ctx))
	eq(t, "projects", len(projects), 1)
	eqPtr(t, "task project", r.Task.ProjectID, &projects[0].ID)
}

func TestReportProgressStoresTheCleanedCwdItWasGiven(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	r := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: dir + "/."})
	eq(t, "cwd", r.Report.Cwd, dir)
	got := must[[]domain.Report](t)(c.ListReports(ctx, r.Task.ID))
	eq(t, "stored cwd", got[0].Cwd, dir)
}

// A cwd that only steers the project lookup (the server's default) is not the agent's own.
func TestReportProgressStoresNoCwdWhenOnlyAFallbackResolvedTheProject(t *testing.T) {
	c, _ := newCore(t)
	r := progress(t, c, domain.ProgressInput{Summary: "one", FallbackCwd: widgetsRepo(t, "feat/login")})
	if r.Task.ProjectID == nil {
		t.Error("the fallback cwd should still resolve the project")
	}
	eq(t, "cwd", r.Report.Cwd, "")
	b := progress(t, c, domain.ProgressInput{Summary: "two", TaskID: &r.Task.ID})
	eq(t, "cwd by task id", b.Report.Cwd, "")
}

func TestReportProgressReusesTaskOnSameBranch(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	a := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: dir})
	st := domain.StatusReview
	b := progress(t, c, domain.ProgressInput{Summary: "two", Cwd: dir, Status: &st,
		PRURL: "https://github.com/acme/widgets/pull/12", NextStep: "wait for CI"})
	if b.Created {
		t.Error("second report should reuse the task")
	}
	eq(t, "task id", b.Task.ID, a.Task.ID)
	eq(t, "status", b.Task.Status, domain.StatusReview)
	eqPtr(t, "pr_number", b.Task.PRNumber, ptr(int64(12)))
	eqPtr(t, "report status", b.Report.Status, &st)
	eqPtr(t, "next_step", b.Report.NextStep, ptr("wait for CI"))
	reports := must[[]domain.Report](t)(c.ListReports(ctx, a.Task.ID))
	if len(reports) != 2 || reports[0].Summary != "one" || reports[1].Summary != "two" {
		t.Errorf("reports = %+v", reports)
	}
}

func TestReportProgressBranchRules(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	a := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: dir})
	done := domain.StatusDone
	must[*domain.Task](t)(c.UpdateTask(ctx, a.Task.ID, domain.TaskPatch{Status: &done}))
	b := progress(t, c, domain.ProgressInput{Summary: "again", Cwd: dir})
	if b.Task.ID == a.Task.ID {
		t.Error("a closed task must not be reused")
	}
	other := progress(t, c, domain.ProgressInput{Summary: "other", Cwd: dir, Branch: "feat/other"})
	eqPtr(t, "explicit branch", other.Task.Branch, ptr("feat/other"))
	if other.Task.ID == b.Task.ID {
		t.Error("a different branch gets its own task")
	}
	titled := progress(t, c, domain.ProgressInput{Summary: "s", Cwd: dir, Branch: "feat/titled", Title: "Add login"})
	eq(t, "title overrides branch", titled.Task.Title, "Add login")
	st := domain.StatusBlocked
	blocked := progress(t, c, domain.ProgressInput{Summary: "s", Cwd: dir, Branch: "feat/b", Status: &st})
	eq(t, "initial status follows the report", blocked.Task.Status, domain.StatusBlocked)
}

func TestReportProgressByTaskID(t *testing.T) {
	c, _ := newCore(t)
	a := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: widgetsRepo(t, "feat/login")})
	done := domain.StatusDone
	must[*domain.Task](t)(c.UpdateTask(ctx, a.Task.ID, domain.TaskPatch{Status: &done}))

	doing := domain.StatusDoing
	r := progress(t, c, domain.ProgressInput{Summary: "more", TaskID: &a.Task.ID, Cwd: t.TempDir(), Status: &doing, Branch: "feat/moved"})
	if r.Created {
		t.Error("task_id never creates")
	}
	eq(t, "id", r.Task.ID, a.Task.ID)
	eq(t, "reopened", r.Task.Status, domain.StatusDoing)
	if r.Task.CompletedAt != nil {
		t.Error("reopening clears completed_at")
	}
	eqPtr(t, "task_id takes the branch", r.Task.Branch, ptr("feat/moved"))
	eq(t, "no extra project from the unrelated cwd", len(must[[]domain.Project](t)(c.ListProjects(ctx))), 1)

	_, err := c.ReportProgress(ctx, domain.ProgressInput{Summary: "s", TaskID: ptr(int64(99)), Agent: "x"})
	wantFieldError(t, err, "task_id")
}

func TestReportProgressOutsideGit(t *testing.T) {
	c, clk := newCore(t)
	r := progress(t, c, domain.ProgressInput{Summary: "did a thing", Cwd: t.TempDir()})
	if r.Task.ProjectID != nil || r.Task.Branch != nil {
		t.Errorf("should land in the inbox without a branch: %+v", r.Task)
	}
	eq(t, "title from summary", r.Task.Title, "did a thing")
	eq(t, "projects", len(must[[]domain.Project](t)(c.ListProjects(ctx))), 0)

	// With no branch there is nothing to match on, so each report is its own task.
	clk.advance(time.Second)
	again := progress(t, c, domain.ProgressInput{Summary: "did a thing", Cwd: t.TempDir()})
	if again.Task.ID == r.Task.ID {
		t.Error("branchless reports must not merge")
	}
}

func TestReportProgressRejectsACwdThatDoesNotExist(t *testing.T) {
	c, _ := newCore(t)
	gone := filepath.Join(t.TempDir(), "gone")
	_, err := c.ReportProgress(ctx, domain.ProgressInput{Summary: "s", Agent: "x", Cwd: gone})
	wantFieldError(t, err, "cwd: no such directory")
	eq(t, "tasks", len(must[[]domain.Task](t)(c.ListTasks(ctx, domain.TaskFilter{}))), 0)

	// With task_id the cwd finds nothing, so a removed worktree can still report it is done.
	task := progress(t, c, domain.ProgressInput{Summary: "s", Cwd: widgetsRepo(t, "feat/a")}).Task
	progress(t, c, domain.ProgressInput{Summary: "done", TaskID: &task.ID, Cwd: gone})
}

func TestReportProgressTitleFromSummary(t *testing.T) {
	c, _ := newCore(t)
	long := progress(t, c, domain.ProgressInput{Summary: strings.Repeat("é", 300)})
	if n := len([]rune(long.Task.Title)); n != 60 {
		t.Errorf("title has %d runes, want 60", n)
	}
	if !strings.HasSuffix(long.Task.Title, "…") {
		t.Error("truncated title should end with an ellipsis")
	}
	multi := progress(t, c, domain.ProgressInput{Summary: "first line\nsecond line"})
	eq(t, "first line only", multi.Task.Title, "first line")
}

func TestReportProgressBumpsUpdatedAt(t *testing.T) {
	c, clk := newCore(t)
	a := progress(t, c, domain.ProgressInput{Summary: "one"})
	clk.advance(time.Hour)
	b := progress(t, c, domain.ProgressInput{Summary: "two", TaskID: &a.Task.ID})
	eq(t, "updated_at", b.Task.UpdatedAt, "2026-10-01T01:00:00.000Z")
	eq(t, "report created_at", b.Report.CreatedAt, b.Task.UpdatedAt)
}

func TestReportProgressValidation(t *testing.T) {
	c, _ := newCore(t)
	bad := domain.Status("weird")
	tests := []struct {
		name  string
		in    domain.ProgressInput
		field string
	}{
		{"empty summary", domain.ProgressInput{Agent: "x"}, "summary"},
		{"blank summary", domain.ProgressInput{Agent: "x", Summary: "  \n"}, "summary"},
		{"bad status", domain.ProgressInput{Agent: "x", Summary: "s", Status: &bad}, "status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.ReportProgress(ctx, tt.in)
			wantFieldError(t, err, tt.field)
		})
	}
}

func TestAddReport(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	r := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "s", CreatedAt: "2026-01-01T00:00:00.000Z"}))
	eq(t, "created_at honoured", r.CreatedAt, "2026-01-01T00:00:00.000Z")
	if _, err := c.AddReport(ctx, domain.AddReportInput{TaskID: 999, Agent: "x", Summary: "s"}); err == nil {
		t.Error("a report for a missing task violates the foreign key")
	}
}
