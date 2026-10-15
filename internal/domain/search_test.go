package domain_test

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func taskTitles(ts []domain.Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Title
	}
	return out
}

func sameTitles(t *testing.T, name string, got []domain.Task, want ...string) {
	t.Helper()
	g := taskTitles(got)
	if len(g) != len(want) {
		t.Errorf("%s = %q, want %q", name, g, want)
		return
	}
	for i := range g {
		if g[i] != want[i] {
			t.Errorf("%s = %q, want %q", name, g, want)
			return
		}
	}
}

func search(t *testing.T, c *domain.Core, q string, limit int) []domain.Task {
	t.Helper()
	return must[[]domain.Task](t)(c.SearchTasks(ctx, q, limit))
}

func TestSearchTasksMatchesTitleNotesBranchAndPullRequest(t *testing.T) {
	c, clk := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	mk := func(in domain.CreateTaskInput) { mustTask(t, c, in); clk.advance(time.Minute) }
	mk(domain.CreateTaskInput{Title: "Fix the Login form", ProjectID: &p.ID})
	mk(domain.CreateTaskInput{Title: "Docs", Notes: ptr("mention login throttling")})
	mk(domain.CreateTaskInput{Title: "Refactor", ProjectID: &p.ID, Branch: ptr("feat/LOGIN-v2")})
	mk(domain.CreateTaskInput{Title: "Ship it", ProjectID: &p.ID, PRURL: ptr("https://github.com/acme/widgets/pull/42"), Status: domain.StatusDone})
	mk(domain.CreateTaskInput{Title: "Cancelled idea", Status: domain.StatusCancelled})
	mk(domain.CreateTaskInput{Title: "Unrelated"})

	sameTitles(t, "login, newest first", search(t, c, "login", 0), "Refactor", "Docs", "Fix the Login form")
	sameTitles(t, "by pull request number", search(t, c, "#42", 0), "Ship it")
	sameTitles(t, "bare number", search(t, c, "42", 0), "Ship it")
	sameTitles(t, "all statuses", search(t, c, "idea", 0), "Cancelled idea")
	sameTitles(t, "every word must match", search(t, c, "fix login", 0), "Fix the Login form")
	sameTitles(t, "nothing", search(t, c, "zzz", 0))
}

func TestSearchTasksTreatsTheQueryAsPlainText(t *testing.T) {
	c, _ := newCore(t)
	mustTask(t, c, domain.CreateTaskInput{Title: "100% done"})
	mustTask(t, c, domain.CreateTaskInput{Title: "snake_case"})
	mustTask(t, c, domain.CreateTaskInput{Title: "other"})
	mustTask(t, c, domain.CreateTaskInput{Title: "重构登录页"})
	sameTitles(t, "percent", search(t, c, "%", 0), "100% done")
	sameTitles(t, "underscore", search(t, c, "e_c", 0), "snake_case")
	sameTitles(t, "Chinese", search(t, c, "登录", 0), "重构登录页")
	sameTitles(t, "blank query", search(t, c, "  ", 0))
}

func TestSearchTasksLimitAndArchivedProjects(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	for i := 0; i < 5; i++ {
		mustTask(t, c, domain.CreateTaskInput{Title: "task"})
	}
	mustTask(t, c, domain.CreateTaskInput{Title: "task in project", ProjectID: &p.ID})
	eq(t, "limit", len(search(t, c, "task", 3)), 3)
	eq(t, "all", len(search(t, c, "task", 0)), 6)
	must[*domain.Project](t)(c.ArchiveProject(ctx, p.ID))
	eq(t, "archived projects are not searched", len(search(t, c, "task", 0)), 5)
}

func TestListLogbookIsDoneAndCancelledNewestFirst(t *testing.T) {
	c, clk := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	open := mustTask(t, c, domain.CreateTaskInput{Title: "open"})
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a", ProjectID: &p.ID})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	cancelled := mustTask(t, c, domain.CreateTaskInput{Title: "c"})
	_ = open
	for _, step := range []struct {
		id int64
		st domain.Status
	}{{a.ID, domain.StatusDone}, {cancelled.ID, domain.StatusCancelled}, {b.ID, domain.StatusDone}} {
		clk.advance(time.Hour)
		must[*domain.Task](t)(c.UpdateTask(ctx, step.id, domain.TaskPatch{Status: &step.st}))
	}
	sameTitles(t, "by completion", must[[]domain.Task](t)(c.ListLogbook(ctx, 0)), "b", "c", "a")
	sameTitles(t, "limited", must[[]domain.Task](t)(c.ListLogbook(ctx, 2)), "b", "c")

	// Reopening takes a task out; editing a finished task does not move it.
	todo := domain.StatusTodo
	must[*domain.Task](t)(c.UpdateTask(ctx, b.ID, domain.TaskPatch{Status: &todo}))
	clk.advance(time.Hour)
	must[*domain.Task](t)(c.UpdateTask(ctx, a.ID, domain.TaskPatch{Notes: domain.Set("later")}))
	sameTitles(t, "after reopening", must[[]domain.Task](t)(c.ListLogbook(ctx, 0)), "c", "a")

	must[*domain.Project](t)(c.ArchiveProject(ctx, p.ID))
	sameTitles(t, "archived projects are left out", must[[]domain.Task](t)(c.ListLogbook(ctx, 0)), "c")
}
