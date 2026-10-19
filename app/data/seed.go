package data

import (
	"fmt"
	"slices"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// seed fills a demo store with a small, realistic workspace that matches docs/design/mockups/now.html:
// three projects, a blocked agent task with a next step, a review task with a PR, a reminder that
// just came due, two agents at work in two projects, the user's own tasks with due labels, and
// inbox items. It is written through the domain layer, with the clock moved to when each row
// happened; ids follow the order below, which the tests rely on.
func (r *Real) seed(now time.Time) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	day := 24 * time.Hour

	for _, p := range []struct {
		name, slug, path string
		created, updated time.Time
	}{
		{"adhd-todo", "QingYunA/adhd-todo", "~/Code/adhd-todo", ago(30 * day), ago(time.Hour)},
		{"answer-me-with-html", "QingYunA/answer-me-with-html", "~/Code/answer-me-with-html", ago(20 * day), ago(14 * time.Minute)},
		{"picpocket", "QingYunA/picpocket", "~/Code/picpocket", ago(90 * day), ago(5 * day)},
	} {
		release := r.holdClock(p.created)
		made := r.SeedProject(p.name, p.slug)
		release()
		r.exec(`UPDATE projects SET repo_path = ?, updated_at = ? WHERE id = ?`, p.path, domain.FormatTime(p.updated), made.ID)
	}
	r.seedMilestones(now)
	r.seedTasks(now)
	r.seedReports(now)
	return nil
}

func (r *Real) seedMilestones(now time.Time) {
	day := 24 * time.Hour
	due := domain.FormatTime(now.Add(14 * day))
	for _, m := range []struct {
		title   string
		due     *string
		created time.Time
		closed  bool
	}{
		{"v0.2 native", &due, now.Add(-10 * day), false},
		{"v0.1 spike", nil, now.Add(-28 * day), true},
	} {
		release := r.holdClock(m.created)
		made, err := r.core.CreateMilestone(r.ctx(), domain.CreateMilestoneInput{ProjectID: 1, Title: m.title, DueAt: m.due})
		release()
		if err != nil {
			panic(err)
		}
		if m.closed {
			closed := domain.MilestoneClosed
			if _, err := r.core.UpdateMilestone(r.ctx(), made.ID, domain.MilestonePatch{Status: &closed}); err != nil {
				panic(err)
			}
		}
	}
}

func (r *Real) seedTasks(now time.Time) {
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	in := func(d time.Duration) *time.Time { t := now.Add(d); return &t }
	day := 24 * time.Hour
	p1, p2, p3 := IntPtr(1), IntPtr(2), IntPtr(3)
	m1 := IntPtr(1)
	daysAhead := func(n, h int) *time.Time {
		y, m, d := now.Date()
		t := time.Date(y, m, d+n, h, 0, 0, 0, now.Location())
		return &t
	}
	mk := func(id int, pid, mid *int, title string, st Status, pr Priority, src Source, updated time.Time) Task {
		return Task{ID: id, ProjectID: pid, MilestoneID: mid, Title: title, Status: st, Priority: pr, Source: src,
			CreatedAt: updated.Add(-2 * day), UpdatedAt: updated}
	}
	agent := func(t Task, branch string, pr int) Task {
		t.Branch = branch
		if pr > 0 {
			t.PRNumber = pr
			t.PRURL = fmt.Sprintf("https://github.com/QingYunA/adhd-todo/pull/%d", pr)
		}
		return t
	}
	done := func(t Task, at time.Time) Task { t.CompletedAt = &at; return t }
	sub := func(parent int, t Task, finished time.Time) Task {
		t.ParentID = IntPtr(parent)
		if !finished.IsZero() {
			t.CompletedAt = &finished
		}
		return t
	}

	t5 := mk(5, p1, m1, "Write the release notes", Todo, P2, SourceUser, ago(day))
	t5.DueAt = daysAhead(2, 10)
	t5.Notes = "Cover the Go core, the native UI and how to register the MCP server."
	t6 := mk(6, p1, nil, "Decide on the activation policy", Doing, P1, SourceUser, ago(3*time.Hour))
	t6.DueAt = in(5 * time.Hour)
	t7 := mk(7, nil, nil, "Reply to design feedback", Todo, P2, SourceUser, ago(6*time.Hour))
	t7.CategoryID = 1
	t7.RemindAt = in(0) // came due just now
	t8 := mk(8, nil, nil, "Book dentist", Todo, P2, SourceUser, ago(2*day))
	t8.DueAt = daysAhead(1, 9)
	t8.CategoryID = 2
	t19 := mk(19, nil, nil, "Renew passport", Backlog, P2, SourceUser, ago(12*day))
	t19.CategoryID = 2
	t21 := done(mk(21, nil, nil, "Send the packaging answer", Done, P2, SourceUser, ago(3*time.Hour)), ago(3*time.Hour))
	t21.CategoryID = 1

	tasks := []Task{
		agent(mk(1, p1, m1, "Port core to Go", Doing, P1, SourceAgent, ago(2*time.Minute)), "feat/go-core", 0),
		agent(mk(2, p1, m1, "Native UI with mygo", Review, P2, SourceAgent, ago(25*time.Minute)), "feat/native-ui", 12),
		agent(mk(3, p1, nil, "Fix tray race on launch", Blocked, P0, SourceAgent, ago(3*day)), "fix/tray", 0),
		mk(4, p1, nil, "Poll gh for PR status", Backlog, P3, SourceUser, ago(4*day)),
		t5, t6, t7, t8,
		mk(9, p1, nil, "Add a Windows build", Backlog, P3, SourceUser, ago(6*day)),
		agent(mk(10, p1, m1, "Port the reminder watcher", Todo, P2, SourceAgent, ago(5*time.Hour)), "feat/watch", 0),
		done(agent(mk(11, p1, nil, "Spike the mygo shell", Done, P2, SourceAgent, ago(2*day)), "spike/mygo-shell", 5), ago(2*day)),
		done(agent(mk(12, p1, nil, "Map worktrees to the main repo", Done, P1, SourceAgent, ago(5*day)), "fix/worktree", 1), ago(5*day)),
		done(mk(13, p1, nil, "Draft the UI spec", Done, P2, SourceUser, ago(day)), ago(day)),
		done(mk(14, p1, nil, "Pick a UI toolkit", Done, P2, SourceUser, ago(6*day)), ago(6*day)),
		done(agent(mk(20, p1, m1, "Port the MCP server", Done, P2, SourceAgent, ago(40*time.Minute)), "feat/mcp", 14), ago(40*time.Minute)),
		t21,
		agent(mk(15, p2, nil, "Issue #218 adaptation", Doing, P2, SourceAgent, ago(14*time.Minute)), "claude/issue-218", 0),
		mk(16, p2, nil, "Cover the reply parser edge cases", Backlog, P3, SourceUser, ago(8*day)),
		mk(17, p3, nil, "Export thumbnails to HEIC", Backlog, P2, SourceUser, ago(9*day)),
		mk(18, nil, nil, "Look into Liquid Glass for the sidebar", Todo, P2, SourceUser, ago(4*time.Hour)),
		t19,
		// Subtasks: what the subagents of "Port core to Go" and "Port the MCP server" are doing.
		sub(1, mk(22, p1, nil, "Port the store and migrations", Done, P2, SourceAgent, ago(8*time.Minute)), ago(8*time.Minute)),
		sub(1, mk(23, p1, nil, "Port the MCP tools", Doing, P2, SourceAgent, ago(2*time.Minute)), time.Time{}),
		sub(1, mk(24, p1, nil, "Watch the database for changes", Doing, P2, SourceAgent, ago(time.Minute)), time.Time{}),
		sub(20, mk(25, p1, nil, "Cover the tools with tests", Done, P2, SourceAgent, ago(50*time.Minute)), ago(50*time.Minute)),
	}
	// The ids are the database's, in the order the rows are made.
	slices.SortFunc(tasks, func(a, b Task) int { return a.ID - b.ID })
	for _, t := range tasks {
		r.SeedTask(t)
	}
}

func (r *Real) seedReports(now time.Time) {
	day := 24 * time.Hour
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	rep := func(task int, agent string, st Status, at time.Duration, summary, next string) Report {
		return Report{TaskID: task, Agent: agent, Status: st, Summary: summary, NextStep: next, CreatedAt: ago(at)}
	}
	// Where the agents work: the worktree of each task they reported from. Tasks 10 to 12 left none.
	worktree := func(name string) string { return "/Users/demo/code/" + name }
	cwds := map[int]string{
		1: worktree("adhd-todo/.claude/worktrees/native-ui"), 2: worktree("answer-me-with-html"),
		3: worktree("picpocket/.claude/worktrees/tray-race"), 15: worktree("answer-me-with-html/.claude/worktrees/issue-218"),
		20: worktree("adhd-todo/.claude/worktrees/mcp-port"), 22: worktree("adhd-todo/.claude/worktrees/store"),
		23: worktree("adhd-todo/.claude/worktrees/mcp-tools"), 24: worktree("adhd-todo/.claude/worktrees/watcher"),
		25: worktree("adhd-todo/.claude/worktrees/mcp-tests"),
	}
	for _, x := range []Report{
		rep(1, "claude-code", Doing, 2*time.Minute, "Ported the store and migrations", "Port the MCP server"),
		rep(1, "claude-code", Doing, 41*time.Minute, "Opened the migrations against a copy of the real database", "Port the store"),
		rep(1, "claude-code", Doing, 3*time.Hour, "Read the TypeScript domain tests and listed the cases to port", "Start with the migrations"),
		rep(2, "claude-code", Review, 25*time.Minute, "Board, lists and inspector are in; waiting for your review", "Address the review comments"),
		rep(2, "claude-code", Doing, 2*time.Hour, "Board drag and drop changes status", "Polish the inspector"),
		rep(3, "claude-code", Blocked, 3*day, "The tray icon appears before the app is ready on macOS 27", "Needs a decision on the activation policy"),
		rep(3, "claude-code", Doing, 3*day+2*time.Hour, "Reproduced the race with a slow launch", ""),
		rep(10, "claude-code", Todo, 5*time.Hour, "Planned the watcher around PRAGMA data_version", "Wait for the store port"),
		rep(11, "claude-code", Done, 2*day, "Spike merged; the shell replaces Tauri", ""),
		rep(12, "claude-code", Done, 5*day, "Worktrees of a remote-less repo map to the main project", ""),
		rep(20, "claude-code", Review, 3*time.Hour, "Eight tools ported with their tests", "Open the pull request"),
		rep(20, GitHubAgent, Done, 40*time.Minute, "PR #14 merged", ""),
		rep(15, "claude-code", Doing, 14*time.Minute, "Writing tests for the reply parser", ""),
		rep(15, "claude-code", Doing, 52*time.Minute, "Read the issue and mapped it to the parser", "Write the tests first"),
		rep(22, "claude-code", Done, 8*time.Minute, "Store and migrations ported", ""),
		rep(23, "claude-code", Doing, 2*time.Minute, "Eight of twelve tools ported", "Port the remaining tools"),
		rep(24, "claude-code", Doing, time.Minute, "Watching PRAGMA data_version", ""),
		rep(25, "claude-code", Done, 50*time.Minute, "Tests added", ""),
	} {
		if x.Agent != GitHubAgent {
			x.Cwd = cwds[x.TaskID]
		}
		r.SeedReport(x)
	}
}
