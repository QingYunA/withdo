package data

import "time"

// seed fills the fake with a small, realistic workspace that matches docs/design/mockups/now.html:
// three projects, a blocked agent task with a next step, a review task with a PR, a reminder that
// just came due, two agents at work in two projects, the user's own tasks with due labels, and
// inbox items.
func (f *Fake) seed() {
	now := f.Now()
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	in := func(d time.Duration) *time.Time { t := now.Add(d); return &t }
	day := 24 * time.Hour

	f.projects = []Project{
		{ID: 1, Name: "adhd-todo", RepoSlug: "QingYunA/adhd-todo", RepoPath: "~/Code/adhd-todo", CreatedAt: ago(30 * day), UpdatedAt: ago(time.Hour)},
		{ID: 2, Name: "answer-me-with-html", RepoSlug: "QingYunA/answer-me-with-html", RepoPath: "~/Code/answer-me-with-html", CreatedAt: ago(20 * day), UpdatedAt: ago(14 * time.Minute)},
		{ID: 3, Name: "picpocket", RepoSlug: "QingYunA/picpocket", RepoPath: "~/Code/picpocket", CreatedAt: ago(90 * day), UpdatedAt: ago(5 * day)},
	}
	f.milestones = []Milestone{
		{ID: 1, ProjectID: 1, Title: "v0.2 native", DueAt: in(14 * day), Open: true, CreatedAt: ago(10 * day)},
		{ID: 2, ProjectID: 1, Title: "v0.1 spike", DueAt: nil, Open: false, CreatedAt: ago(28 * day)},
	}
	f.seedTasks(now, ago, in)
	f.seedReports(ago)
}

type timeAgo func(time.Duration) time.Time
type timeIn func(time.Duration) *time.Time

func (f *Fake) seedTasks(now time.Time, ago timeAgo, in timeIn) {
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
			t.PRURL = "https://github.com/QingYunA/adhd-todo/pull/" + itoa(pr)
		}
		return t
	}
	done := func(t Task, at time.Time) Task { t.CompletedAt = &at; return t }

	t5 := mk(5, p1, m1, "Write the release notes", Todo, P2, SourceUser, ago(day))
	t5.DueAt = daysAhead(2, 10)
	t5.Notes = "Cover the Go core, the native UI and how to register the MCP server."
	t6 := mk(6, p1, nil, "Decide on the activation policy", Doing, P1, SourceUser, ago(3*time.Hour))
	t6.DueAt = in(5 * time.Hour)
	t7 := mk(7, nil, nil, "Reply to design feedback", Todo, P2, SourceUser, ago(6*time.Hour))
	t7.RemindAt = in(0) // came due just now
	t8 := mk(8, nil, nil, "Book dentist", Todo, P2, SourceUser, ago(2*day))
	t8.DueAt = daysAhead(1, 9)

	f.tasks = []Task{
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
		agent(mk(15, p2, nil, "Issue #218 adaptation", Doing, P2, SourceAgent, ago(14*time.Minute)), "claude/issue-218", 0),
		mk(16, p2, nil, "Cover the reply parser edge cases", Backlog, P3, SourceUser, ago(8*day)),
		mk(17, p3, nil, "Export thumbnails to HEIC", Backlog, P2, SourceUser, ago(9*day)),
		mk(18, nil, nil, "Look into Liquid Glass for the sidebar", Todo, P2, SourceUser, ago(4*time.Hour)),
		mk(19, nil, nil, "Renew passport", Backlog, P2, SourceUser, ago(12*day)),
	}
}

func (f *Fake) seedReports(ago timeAgo) {
	day := 24 * time.Hour
	rep := func(task int, agent string, st Status, at time.Duration, summary, next string) Report {
		return Report{ID: len(f.reports) + 1, TaskID: task, Agent: agent, Status: st, Summary: summary, NextStep: next, CreatedAt: ago(at)}
	}
	add := func(rs ...Report) {
		for _, r := range rs {
			r.ID = len(f.reports) + 1
			f.reports = append(f.reports, r)
		}
	}
	add(
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
		rep(15, "claude-code", Doing, 14*time.Minute, "Writing tests for the reply parser", ""),
		rep(15, "claude-code", Doing, 52*time.Minute, "Read the issue and mapped it to the parser", "Write the tests first"),
	)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
