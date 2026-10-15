package domain_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// worktreeRepo is a widgets repo on branch feat/x with one commit; add makes linked worktrees.
type worktreeRepo struct {
	t    *testing.T
	root string
}

func newWorktreeRepo(t *testing.T) *worktreeRepo {
	t.Helper()
	root := testutil.NewRepo(t, t.TempDir(), "widgets",
		testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Branch: "feat/x", Commit: true})
	return &worktreeRepo{t: t, root: root}
}

// add makes a linked worktree at rel (relative to the repo) on a new branch and returns its path.
func (r *worktreeRepo) add(rel, branch string) string {
	r.t.Helper()
	path := filepath.Join(r.root, rel)
	testutil.Git(r.t, r.root, "worktree", "add", "-q", "-b", branch, path)
	return testutil.Real(r.t, path)
}

// commit makes a commit in dir whose committer date is at.
func commit(t *testing.T, dir, file string, at time.Time) {
	t.Helper()
	t.Setenv("GIT_COMMITTER_DATE", at.Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(dir, file), []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, dir, "add", ".")
	testutil.Git(t, dir, "commit", "-q", "-m", file)
}

func subtaskIDs(t *testing.T, c *domain.Core, parent int64) []int64 {
	t.Helper()
	var ids []int64
	for _, s := range must[[]domain.Task](t)(c.ListSubtasks(ctx, parent)) {
		ids = append(ids, s.ID)
	}
	return ids
}

func reportsOf(t *testing.T, c *domain.Core, id int64) []domain.Report {
	t.Helper()
	return must[[]domain.Report](t)(c.ListReports(ctx, id))
}

func statusOf(t *testing.T, c *domain.Core, id int64) domain.Status {
	t.Helper()
	return must[*domain.Task](t)(c.GetTask(ctx, id)).Status
}

func TestCreateTaskWithParent(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	parent := mustTask(t, c, domain.CreateTaskInput{Title: "parent", ProjectID: &p.ID})
	kid := mustTask(t, c, domain.CreateTaskInput{Title: "kid", ParentID: &parent.ID, ProjectID: &p.ID})
	eqPtr(t, "parent_id", kid.ParentID, &parent.ID)
	if parent.ParentID != nil {
		t.Error("a top-level task has no parent")
	}
	inherited := mustTask(t, c, domain.CreateTaskInput{Title: "inherits", ParentID: &parent.ID})
	eqPtr(t, "project inherited from the parent", inherited.ProjectID, &p.ID)

	_, err := c.CreateTask(ctx, domain.CreateTaskInput{Title: "x", ParentID: ptr(int64(999))})
	wantFieldError(t, err, "parent_id")
	other := mustProject(t, c, "acme/other")
	_, err = c.CreateTask(ctx, domain.CreateTaskInput{Title: "x", ParentID: &parent.ID, ProjectID: &other.ID})
	wantFieldError(t, err, "parent_id")
}

func TestDeletingTheParentLeavesTheSubtask(t *testing.T) {
	c, _ := newCore(t)
	parent := mustTask(t, c, domain.CreateTaskInput{Title: "parent"})
	kid := mustTask(t, c, domain.CreateTaskInput{Title: "kid", ParentID: &parent.ID})
	if err := c.DeleteTask(ctx, parent.ID); err != nil {
		t.Fatal(err)
	}
	got := must[*domain.Task](t)(c.GetTask(ctx, kid.ID))
	if got.ParentID != nil {
		t.Errorf("parent_id = %v, want nil", *got.ParentID)
	}
}

func TestReportProgressParentTaskID(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	parent := progress(t, c, domain.ProgressInput{Summary: "main work", Cwd: dir})
	kid := progress(t, c, domain.ProgressInput{Summary: "sub work", Cwd: dir, Branch: "worktree-agent-1", ParentTaskID: &parent.Task.ID})
	if !kid.Created {
		t.Fatal("the subagent's report should create a task")
	}
	eqPtr(t, "parent_id", kid.Task.ParentID, &parent.Task.ID)
	eqPtr(t, "same project", kid.Task.ProjectID, parent.Task.ProjectID)
	again := progress(t, c, domain.ProgressInput{Summary: "more", Cwd: dir, Branch: "worktree-agent-1", ParentTaskID: &parent.Task.ID})
	eq(t, "same task", again.Task.ID, kid.Task.ID)
	if ids := subtaskIDs(t, c, parent.Task.ID); len(ids) != 1 || ids[0] != kid.Task.ID {
		t.Errorf("subtasks = %v", ids)
	}
}

func TestReportProgressParentTaskIDValidation(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	parent := progress(t, c, domain.ProgressInput{Summary: "main work", Cwd: dir})

	_, err := c.ReportProgress(ctx, domain.ProgressInput{Summary: "s", Agent: "x", Cwd: dir, Branch: "b", ParentTaskID: ptr(int64(999))})
	wantFieldError(t, err, "parent_task_id")

	otherRepo := testutil.NewRepo(t, t.TempDir(), "gadgets", testutil.RepoOptions{Remote: "git@github.com:acme/gadgets.git", Branch: "main"})
	_, err = c.ReportProgress(ctx, domain.ProgressInput{Summary: "s", Agent: "x", Cwd: otherRepo, ParentTaskID: &parent.Task.ID})
	wantFieldError(t, err, "parent_task_id")

	// A report with no repo of its own joins the parent's project.
	loose := progress(t, c, domain.ProgressInput{Summary: "s", Cwd: t.TempDir(), ParentTaskID: &parent.Task.ID})
	eqPtr(t, "project from the parent", loose.Task.ProjectID, parent.Task.ProjectID)

	// A task cannot be its own parent, nor the parent of its ancestor.
	_, err = c.ReportProgress(ctx, domain.ProgressInput{Summary: "s", Agent: "x", TaskID: &parent.Task.ID, ParentTaskID: &parent.Task.ID})
	wantFieldError(t, err, "parent_task_id")
	kid := progress(t, c, domain.ProgressInput{Summary: "kid", Cwd: dir, Branch: "kid", ParentTaskID: &parent.Task.ID})
	grandkid := progress(t, c, domain.ProgressInput{Summary: "grandkid", Cwd: dir, Branch: "grandkid", ParentTaskID: &kid.Task.ID})
	_, err = c.ReportProgress(ctx, domain.ProgressInput{Summary: "s", Agent: "x", TaskID: &parent.Task.ID, ParentTaskID: &grandkid.Task.ID})
	wantFieldError(t, err, "parent_task_id")
}

func TestReportProgressParentTaskIDAdoptsAnExistingTask(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	parent := progress(t, c, domain.ProgressInput{Summary: "main work", Cwd: dir})
	lone := progress(t, c, domain.ProgressInput{Summary: "alone", Cwd: dir, Branch: "lone"})
	got := progress(t, c, domain.ProgressInput{Summary: "now a subtask", TaskID: &lone.Task.ID, ParentTaskID: &parent.Task.ID})
	eqPtr(t, "parent_id", got.Task.ParentID, &parent.Task.ID)
}

func TestWorktreeHeuristicMakesSubtasks(t *testing.T) {
	c, clk := newCore(t)
	repo := newWorktreeRepo(t)
	parent := progress(t, c, domain.ProgressInput{Summary: "main work", Cwd: repo.root, Agent: "claude-code"})
	eq(t, "parent branch", *parent.Task.Branch, "feat/x")

	under := repo.add(".claude/worktrees/agent-a1", "worktree-a1")
	byName := repo.add("elsewhere-wt", "worktree-agent-b2")
	plain := repo.add("plain-wt", "feat/plain")

	kid := progress(t, c, domain.ProgressInput{Summary: "sub work", Cwd: under, Agent: "claude-code"})
	eqPtr(t, "under .claude/worktrees", kid.Task.ParentID, &parent.Task.ID)
	eqPtr(t, "same project", kid.Task.ProjectID, parent.Task.ProjectID)
	eq(t, "branch stays on the task", *kid.Task.Branch, "worktree-a1")

	kid2 := progress(t, c, domain.ProgressInput{Summary: "sub work", Cwd: byName, Agent: "claude-code"})
	eqPtr(t, "branch worktree-agent-", kid2.Task.ParentID, &parent.Task.ID)

	// Later reports of the same worktree update its task and keep the parent.
	again := progress(t, c, domain.ProgressInput{Summary: "more", Cwd: under, Agent: "claude-code"})
	eq(t, "same task", again.Task.ID, kid.Task.ID)
	eqPtr(t, "parent kept", again.Task.ParentID, &parent.Task.ID)

	other := progress(t, c, domain.ProgressInput{Summary: "other agent", Cwd: repo.add(".claude/worktrees/agent-c3", "worktree-c3"), Agent: "codex"})
	if other.Task.ParentID != nil {
		t.Error("another agent's worktree report must not attach to claude-code's task")
	}
	top := progress(t, c, domain.ProgressInput{Summary: "plain worktree", Cwd: plain, Agent: "claude-code"})
	if top.Task.ParentID != nil {
		t.Error("an ordinary linked worktree is not a subagent")
	}
	root := progress(t, c, domain.ProgressInput{Summary: "main again", Cwd: repo.root, Agent: "claude-code", Branch: "feat/y"})
	if root.Task.ParentID != nil {
		t.Error("the main worktree never makes subtasks")
	}

	// A parent last reported more than 12 hours ago does not count.
	clk.advance(13 * time.Hour)
	late := progress(t, c, domain.ProgressInput{Summary: "late", Cwd: repo.add(".claude/worktrees/agent-d4", "worktree-d4"), Agent: "claude-code"})
	if late.Task.ParentID != nil {
		t.Error("a parent last reported 13 hours ago must not adopt a subtask")
	}
}

func TestExplicitParentBeatsTheHeuristic(t *testing.T) {
	c, _ := newCore(t)
	repo := newWorktreeRepo(t)
	progress(t, c, domain.ProgressInput{Summary: "main work", Cwd: repo.root, Agent: "claude-code"})
	chosen := progress(t, c, domain.ProgressInput{Summary: "another", Cwd: repo.root, Agent: "claude-code", Branch: "feat/z"})
	wt := repo.add(".claude/worktrees/agent-f6", "worktree-f6")
	kid := progress(t, c, domain.ProgressInput{Summary: "sub", Cwd: wt, Agent: "claude-code", ParentTaskID: &chosen.Task.ID})
	eqPtr(t, "parent", kid.Task.ParentID, &chosen.Task.ID)
}

func TestWorktreeHeuristicPicksTheMostRecentOpenParent(t *testing.T) {
	c, clk := newCore(t)
	repo := newWorktreeRepo(t)
	old := progress(t, c, domain.ProgressInput{Summary: "old", Cwd: repo.root, Agent: "claude-code"})
	done := domain.StatusDone
	must[*domain.Task](t)(c.UpdateTask(ctx, old.Task.ID, domain.TaskPatch{Status: &done}))
	clk.advance(time.Hour)
	fresh := progress(t, c, domain.ProgressInput{Summary: "fresh", Cwd: repo.root, Agent: "claude-code"})
	if fresh.Task.ID == old.Task.ID {
		t.Fatal("a closed task must not be reused")
	}
	kid := progress(t, c, domain.ProgressInput{Summary: "sub", Cwd: repo.add(".claude/worktrees/agent-e5", "worktree-e5"), Agent: "claude-code"})
	eqPtr(t, "open parent", kid.Task.ParentID, &fresh.Task.ID)
}

func TestDefaultTitleIsTheFirstSentence(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	tests := []struct{ name, summary, want string }{
		{"period", "Fix the login redirect. Then run the tests.", "Fix the login redirect"},
		{"cjk period", "修复登录跳转。然后跑测试。", "修复登录跳转"},
		{"question mark kept", "Why does it fail? Looking.", "Why does it fail?"},
		{"dots inside words", "Rename foo.go to bar.go and rerun", "Rename foo.go to bar.go and rerun"},
		{"newline", "first line\nsecond", "first line"},
		{"long", strings.Repeat("a", 100), strings.Repeat("a", 59) + "…"},
		{"long cjk", strings.Repeat("字", 100), strings.Repeat("字", 59) + "…"},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := progress(t, c, domain.ProgressInput{Summary: tt.summary, Cwd: dir, Branch: "b" + string(rune('a'+i))})
			eq(t, "title", r.Task.Title, tt.want)
			if n := len([]rune(r.Task.Title)); n > 60 {
				t.Errorf("title has %d runes", n)
			}
		})
	}
	explicit := progress(t, c, domain.ProgressInput{Summary: "Something. Else.", Cwd: dir, Branch: "titled", Title: "Mine"})
	eq(t, "explicit title wins", explicit.Task.Title, "Mine")
}

func TestClosingAParentClosesItsOpenSubtasks(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusDone, domain.StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			c, _ := newCore(t)
			parent := mustTask(t, c, domain.CreateTaskInput{Title: "parent", Status: domain.StatusDoing})
			sub := func(title string, parent int64, st domain.Status) *domain.Task {
				return mustTask(t, c, domain.CreateTaskInput{Title: title, ParentID: &parent, Status: st, Source: domain.SourceAgent})
			}
			doing := sub("a", parent.ID, domain.StatusDoing)
			review := sub("b", parent.ID, domain.StatusReview)
			nested := sub("c", doing.ID, domain.StatusDoing)
			cancelled := sub("d", parent.ID, domain.StatusCancelled)

			must[*domain.Task](t)(c.UpdateTask(ctx, parent.ID, domain.TaskPatch{Status: &status}))

			for _, id := range []int64{doing.ID, review.ID, nested.ID} {
				got := must[*domain.Task](t)(c.GetTask(ctx, id))
				eq(t, "subtask status", got.Status, domain.StatusDone)
				if got.CompletedAt == nil {
					t.Error("completed_at should be set")
				}
				rs := reportsOf(t, c, id)
				if len(rs) != 1 || rs[0].Agent != "adhd-todo" || rs[0].Summary != "Closed with its parent task" ||
					rs[0].Status == nil || *rs[0].Status != domain.StatusDone {
					t.Errorf("reports of %d = %+v", id, rs)
				}
			}
			eq(t, "an already cancelled subtask stays cancelled", statusOf(t, c, cancelled.ID), domain.StatusCancelled)
			eq(t, "and gets no report", len(reportsOf(t, c, cancelled.ID)), 0)
		})
	}
}

func TestClosingAParentThroughEveryPath(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	pr := progress(t, c, domain.ProgressInput{Summary: "main work", Cwd: dir})
	kid := progress(t, c, domain.ProgressInput{Summary: "sub", Cwd: dir, Branch: "kid", ParentTaskID: &pr.Task.ID})
	done := domain.StatusDone
	progress(t, c, domain.ProgressInput{Summary: "finished", TaskID: &pr.Task.ID, Status: &done})
	eq(t, "closed by report_progress", statusOf(t, c, kid.Task.ID), domain.StatusDone)

	// A merged pull request closes the parent through RecordPR.
	p2 := mustTask(t, c, domain.CreateTaskInput{Title: "p2", Status: domain.StatusReview, PRURL: ptr(prURL)})
	k2 := mustTask(t, c, domain.CreateTaskInput{Title: "k2", ParentID: &p2.ID, Status: domain.StatusDoing})
	must[bool](t)(c.RecordPR(ctx, p2.ID, domain.PRUpdate{URL: prURL, Snapshot: domain.PRSnapshot{State: domain.PRMerged}, Done: true}))
	eq(t, "closed by the merge", statusOf(t, c, k2.ID), domain.StatusDone)
}

func TestReopeningAParentDoesNotReopenSubtasks(t *testing.T) {
	c, _ := newCore(t)
	parent := mustTask(t, c, domain.CreateTaskInput{Title: "parent", Status: domain.StatusDoing})
	kid := mustTask(t, c, domain.CreateTaskInput{Title: "kid", ParentID: &parent.ID, Status: domain.StatusDoing})
	done, doing := domain.StatusDone, domain.StatusDoing
	must[*domain.Task](t)(c.UpdateTask(ctx, parent.ID, domain.TaskPatch{Status: &done}))
	must[*domain.Task](t)(c.UpdateTask(ctx, parent.ID, domain.TaskPatch{Status: &doing}))
	eq(t, "kid", statusOf(t, c, kid.ID), domain.StatusDone)
}

func TestListTasksByParent(t *testing.T) {
	c, _ := newCore(t)
	parent := mustTask(t, c, domain.CreateTaskInput{Title: "parent"})
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a", ParentID: &parent.ID})
	mustTask(t, c, domain.CreateTaskInput{Title: "other"})
	got := must[[]domain.Task](t)(c.ListTasks(ctx, domain.TaskFilter{ParentID: &parent.ID}))
	if len(got) != 1 || got[0].ID != a.ID {
		t.Fatalf("got %+v", got)
	}
}

func TestCloseFinishedSubtasks(t *testing.T) {
	c, clk, db := newCoreDB(t)
	repo := newWorktreeRepo(t)
	// The tasks are created after the repo's first commit, as in real life.
	clk.t = time.Now().Add(time.Minute)
	parent := progress(t, c, domain.ProgressInput{Summary: "main work", Cwd: repo.root, Agent: "claude-code"})
	project := *parent.Task.ProjectID
	sub := func(branch string) *domain.Task {
		in := domain.CreateTaskInput{Title: "sub " + branch, ParentID: &parent.Task.ID, ProjectID: &project,
			Status: domain.StatusDoing, Source: domain.SourceAgent}
		if branch != "" {
			in.Branch = &branch
		}
		return mustTask(t, c, in)
	}

	gone := sub("worktree-gone")
	fresh := sub("worktree-fresh")
	testutil.Git(t, repo.root, "branch", "worktree-fresh")
	merged := sub("worktree-merged")
	mergedDir := repo.add(".claude/worktrees/merged", "worktree-merged")
	commit(t, mergedDir, "merged.txt", clk.t.Add(time.Minute))
	testutil.Git(t, repo.root, "merge", "-q", "--no-edit", "worktree-merged")
	unmerged := sub("worktree-unmerged")
	unmergedDir := repo.add(".claude/worktrees/unmerged", "worktree-unmerged")
	commit(t, unmergedDir, "unmerged.txt", clk.t.Add(time.Minute))
	noBranch := sub("")
	topLevel := mustTask(t, c, domain.CreateTaskInput{Title: "top", ProjectID: &project, Status: domain.StatusDoing, Branch: ptr("worktree-top-gone")})
	finished := sub("worktree-finished-gone")
	done := domain.StatusDone
	must[*domain.Task](t)(c.UpdateTask(ctx, finished.ID, domain.TaskPatch{Status: &done}))

	// A subtask of a project whose repository cannot be reached is left alone.
	exec(t, db, "INSERT INTO projects (name, repo_path, created_at, updated_at) VALUES ('ghost', ?, 't', 't')", filepath.Join(t.TempDir(), "missing"))
	var ghostID int64
	for _, p := range must[[]domain.Project](t)(c.ListProjects(ctx)) {
		if p.Name == "ghost" {
			ghostID = p.ID
		}
	}
	ghostParent := mustTask(t, c, domain.CreateTaskInput{Title: "ghost parent", ProjectID: &ghostID, Status: domain.StatusDoing})
	ghost := mustTask(t, c, domain.CreateTaskInput{Title: "ghost sub", ParentID: &ghostParent.ID, ProjectID: &ghostID, Status: domain.StatusDoing, Branch: ptr("x")})

	n := must[int](t)(c.CloseFinishedSubtasks(ctx))
	eq(t, "closed", n, 2)

	for _, closed := range []*domain.Task{gone, merged} {
		eq(t, "status of "+closed.Title, statusOf(t, c, closed.ID), domain.StatusDone)
		rs := reportsOf(t, c, closed.ID)
		if len(rs) != 1 || rs[0].Agent != "adhd-todo" || rs[0].Summary != "Its branch was merged or deleted" ||
			rs[0].Status == nil || *rs[0].Status != domain.StatusDone {
			t.Errorf("reports of %s = %+v", closed.Title, rs)
		}
	}
	for _, open := range []*domain.Task{fresh, unmerged, noBranch, topLevel, ghost, &parent.Task} {
		eq(t, "status of "+open.Title, statusOf(t, c, open.ID), domain.StatusDoing)
	}
	eq(t, "no extra report for a finished subtask", len(reportsOf(t, c, finished.ID)), 0)

	// Nothing changes the second time.
	eq(t, "second run", must[int](t)(c.CloseFinishedSubtasks(ctx)), 0)
}
