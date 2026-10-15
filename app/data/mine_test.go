package data

import (
	"testing"
	"time"
)

// mineInputOf is what the views hand to BuildMine, read from a fake store.
func mineInputOf(t *testing.T, f *Fake) MineInput {
	t.Helper()
	tasks, err := f.ListTasks(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	projects, _ := f.ListProjects()
	names := map[int]string{}
	for _, p := range projects {
		names[p.ID] = p.Name
	}
	return MineInput{
		Now: f.Now(), Tasks: tasks,
		Latest: func(id int) *Report { return firstReport(f, id, false) },
		Event:  func(id int) *Report { return firstReport(f, id, true) },
		ProjectName: func(id *int) string {
			if id == nil {
				return ""
			}
			return names[*id]
		},
	}
}

// mineOf builds the Mine screen of a fake store the way the views do.
func mineOf(t *testing.T, f *Fake) Mine {
	t.Helper()
	return BuildMine(mineInputOf(t, f))
}

// firstReport is the newest report of a task from GitHub or, with github false, from an agent.
func firstReport(f *Fake, id int, github bool) *Report {
	d, err := f.GetTask(id)
	if err != nil {
		return nil
	}
	for _, r := range d.Reports {
		if (r.Agent == GitHubAgent) == github {
			return &r
		}
	}
	return nil
}

func titles(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildMineSeedMatchesTheMockup(t *testing.T) {
	n := mineOf(t, NewFake(fixedNow))
	wantHanded := []string{"Fix tray race on launch", "Native UI with mygo"}
	if got := titles(n.FromAgents); !equal(got, wantHanded) {
		t.Errorf("from agents %q, want %q (blocked, review)", got, wantHanded)
	}
	if got := titles(n.Today); !equal(got, []string{"Reply to design feedback", "Decide on the activation policy"}) {
		t.Errorf("today %q: the reminder that came due, then what is due in five hours", got)
	}
	if got := titles(n.Later); !equal(got, []string{"Book dentist", "Write the release notes", "Look into Liquid Glass for the sidebar"}) {
		t.Errorf("later %q", got)
	}
	if n.Count() != 4 || n.Empty() || len(n.IDs()) != 4+3+len(n.Done) {
		t.Errorf("count %d", n.Count())
	}
}

func TestBuildMineSaysWhyAndWhatTheAgentSaid(t *testing.T) {
	n := mineOf(t, NewFake(fixedNow))
	it := n.FromAgents[0]
	if it.Kind != NeedBlocked || it.Words != "Needs a decision on the activation policy" {
		t.Errorf("blocked row: kind %v, words %q", it.Kind, it.Words)
	}
	if !it.Stale || it.Hours != 72 {
		t.Errorf("blocked task stale %v hours %d", it.Stale, it.Hours)
	}
	if got := n.FromAgents[1]; got.Kind != NeedReview || got.PRNumber != 12 {
		t.Errorf("review row: %+v", got)
	}
}

func TestBuildAgentsSeedWorkingTasksWithTheirSubtasks(t *testing.T) {
	a := buildAgentsOf(t, NewFake(fixedNow))
	var working []string
	for _, r := range a.Working {
		working = append(working, r.Title)
	}
	if !equal(working, []string{"Port core to Go", "Issue #218 adaptation", "Fix tray race on launch"}) {
		t.Errorf("working %q, newest report first; a blocked task is at work too", working)
	}
	first := a.Working[0]
	if first.Summary != "Ported the store and migrations" || first.NextStep != "Port the MCP server" || !first.Live || a.Working[1].Live {
		t.Errorf("first row %+v", first)
	}
	if got := titles(first.Subs); !equal(got, []string{"Port the store and migrations", "Port the MCP tools", "Watch the database for changes"}) {
		t.Errorf("subtasks %q, oldest first", got)
	}
	if first.Subs[0].Status != Done || first.Subs[1].Status != Doing || first.Subs[2].Status != Doing {
		t.Errorf("subtask statuses %v %v %v", first.Subs[0].Status, first.Subs[1].Status, first.Subs[2].Status)
	}
	if a.Count() != 3 || a.Empty() {
		t.Errorf("count %d", a.Count())
	}
	if len(a.Done) != 1 || a.Done[0].Title != "Port the MCP server" || len(a.Done[0].Subs) != 1 {
		t.Errorf("done today %+v: only agent tasks, with their subtasks", a.Done)
	}
	if got := a.IDs(); len(got) != 3+3+1+1 {
		t.Errorf("ids %v: a task, then its subtasks", got)
	}
}

func buildAgentsOf(t *testing.T, f *Fake) AgentsScreen {
	t.Helper()
	return BuildAgents(mineInputOf(t, f))
}

func TestSubtasksAreNeverTopLevelRows(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	parent, _ := f.CreateTask(CreateTaskInput{Title: "Parent", Source: SourceAgent, Status: Doing})
	for _, st := range []Status{Blocked, Review, Doing, Todo, Done} {
		f.CreateTask(CreateTaskInput{Title: "child " + string(st), ParentID: IntPtr(parent.ID), Source: SourceAgent, Status: st})
	}
	f.CreateTask(CreateTaskInput{Title: "my child", ParentID: IntPtr(parent.ID), Status: Doing})
	m := mineOf(t, f)
	if len(m.FromAgents)+len(m.Today)+len(m.Later)+len(m.Done) != 0 {
		t.Errorf("subtasks leaked into Mine: %+v", m)
	}
	a := buildAgentsOf(t, f)
	if len(a.Working) != 1 || len(a.Working[0].Subs) != 6 || len(a.Done) != 0 {
		t.Errorf("agents: %d working, %d subtasks", len(a.Working), len(a.Working[0].Subs))
	}
}

func TestACancelledSubtaskIsLeftOutOfItsParent(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	parent, _ := f.CreateTask(CreateTaskInput{Title: "Parent", Source: SourceAgent, Status: Doing})
	f.CreateTask(CreateTaskInput{Title: "dropped", ParentID: IntPtr(parent.ID), Source: SourceAgent, Status: Cancelled})
	f.CreateTask(CreateTaskInput{Title: "kept", ParentID: IntPtr(parent.ID), Source: SourceAgent, Status: Doing})
	if got := titles(buildAgentsOf(t, f).Working[0].Subs); !equal(got, []string{"kept"}) {
		t.Errorf("subtasks %q", got)
	}
}

func TestBuildMineEmptySectionsAndAllClear(t *testing.T) {
	if !mineOf(t, NewEmptyFake(fixedNow)).Empty() {
		t.Error("an empty store is not All clear")
	}
	f := NewEmptyFake(fixedNow)
	f.CreateTask(CreateTaskInput{Title: "Only mine"})
	n := mineOf(t, f)
	if len(n.FromAgents) != 0 || len(n.Today) != 0 || len(n.Later) != 1 {
		t.Errorf("one undated user task gave %d/%d/%d", len(n.FromAgents), len(n.Today), len(n.Later))
	}
}

func TestBuildMineStaleAgentWithoutReports(t *testing.T) {
	now := fixedNow()
	f := NewEmptyFake(func() time.Time { return now })
	f.CreateTask(CreateTaskInput{Title: "Quiet one", Source: SourceAgent, Status: Doing})
	now = now.Add(80 * time.Hour)
	n := mineOf(t, f)
	if len(n.FromAgents) != 1 || n.FromAgents[0].Kind != NeedStale || n.FromAgents[0].Hours != 80 {
		t.Errorf("%+v", n.FromAgents)
	}
	if a := buildAgentsOf(t, f); len(a.Working) != 1 {
		t.Errorf("a quiet agent task is still at work on the Agents screen: %d", len(a.Working))
	}
}

func TestBuildMineReviewWithoutPRAndMyBlockedTask(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	f.CreateTask(CreateTaskInput{Title: "Reviewable", Source: SourceAgent, Status: Review})
	f.CreateTask(CreateTaskInput{Title: "Stuck", Source: SourceUser, Status: Blocked})
	f.CreateTask(CreateTaskInput{Title: "Waiting on someone", Source: SourceUser, Status: Review})
	n := mineOf(t, f)
	if got := titles(n.FromAgents); !equal(got, []string{"Reviewable"}) || n.FromAgents[0].Kind != NeedReview || n.FromAgents[0].PRNumber != 0 {
		t.Fatalf("from agents %q: only an agent hands things over", got)
	}
	if got := titles(n.Today); !equal(got, []string{"Stuck"}) {
		t.Errorf("today %q: my blocked task is mine to unblock; a review task of mine waits on others", got)
	}
}

func TestBuildMineTodayAndLaterLimitsAndHorizon(t *testing.T) {
	now := fixedNow()
	f := NewEmptyFake(func() time.Time { return now })
	for i := 0; i < 10; i++ {
		f.CreateTask(CreateTaskInput{Title: "Undated " + itoaT(i)})
	}
	far, _ := f.CreateTask(CreateTaskInput{Title: "Far away"})
	f.UpdateTask(far.ID, Patch{DueAt: TimePtr(now.Add(30 * 24 * time.Hour))})
	over, _ := f.CreateTask(CreateTaskInput{Title: "Overdue"})
	f.UpdateTask(over.ID, Patch{DueAt: TimePtr(now.Add(-26 * time.Hour))})
	soon, _ := f.CreateTask(CreateTaskInput{Title: "Next week"})
	f.UpdateTask(soon.ID, Patch{DueAt: TimePtr(now.Add(3 * 24 * time.Hour))})
	doing, _ := f.CreateTask(CreateTaskInput{Title: "In progress", Status: Doing})
	n := mineOf(t, f)
	if got := titles(n.Today); !equal(got, []string{"Overdue", "In progress"}) {
		t.Errorf("today %q: overdue first, then what I am doing", got)
	}
	if n.Later[0].Title != "Next week" || len(n.Later) != 1+MaxUndated || n.MoreLater != 3 {
		t.Errorf("%d later rows (first %q), %d more", len(n.Later), n.Later[0].Title, n.MoreLater)
	}
	for _, it := range append(n.Today, n.Later...) {
		if it.Title == "Far away" {
			t.Error("a task due in a month is on Mine")
		}
	}
	_ = doing
}

func TestBuildMineMyReminderDueNowIsTodayNotFromAgents(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	task, _ := f.CreateTask(CreateTaskInput{Title: "Ping me"})
	f.UpdateTask(task.ID, Patch{RemindAt: TimePtr(fixedNow().Add(-time.Minute))})
	n := mineOf(t, f)
	if len(n.FromAgents) != 0 || titles(n.Today)[0] != "Ping me" {
		t.Errorf("from agents %d, today %q", len(n.FromAgents), titles(n.Today))
	}
}

func TestNextMoment(t *testing.T) {
	now := fixedNow()
	a, b := now.Add(time.Hour), now.Add(2*time.Hour)
	if at, ok := NextMoment(Task{DueAt: &b, RemindAt: &a}); !ok || !at.Equal(a) {
		t.Errorf("next moment %v %v", at, ok)
	}
	if _, ok := NextMoment(Task{}); ok {
		t.Error("a task with no times has a moment")
	}
}

func itoaT(n int) string { return string(rune('0' + n)) }

func TestBuildMineDoneTodayListsWhatAnyoneFinishedSinceMidnightNewestFirst(t *testing.T) {
	n := mineOf(t, NewFake(fixedNow)) // 2:30 PM
	if got := titles(n.Done); !equal(got, []string{"Port the MCP server", "Send the packaging answer"}) {
		t.Errorf("done today %q: want what finished at 1:50 PM and 11:30 AM; yesterday's and older stay out", got)
	}
	if n.MoreDone != 0 || n.Count() != 4 || len(n.IDs()) != 4+3+2 {
		t.Errorf("more %d, count %d (done tasks are no part of it), ids %d (the rows drawn)", n.MoreDone, n.Count(), len(n.IDs()))
	}
}

func TestBuildMineDoneTodayNamesWhoFinishedIt(t *testing.T) {
	n := mineOf(t, NewFake(fixedNow))
	by := map[string]string{}
	words := map[string]string{}
	for _, it := range n.Done {
		by[it.Title], words[it.Title] = it.By, it.Summary
	}
	if by["Port the MCP server"] != "claude-code" || words["Port the MCP server"] != "PR #14 merged" {
		t.Errorf("merged by an agent's pull request: by %q, said %q", by["Port the MCP server"], words["Port the MCP server"])
	}
	if by["Send the packaging answer"] != "" || words["Send the packaging answer"] != "" {
		t.Errorf("my own task: by %q, said %q, want neither", by["Send the packaging answer"], words["Send the packaging answer"])
	}
}

func TestBuildMineDoneTodayStopsAtFiveAndCountsTheRest(t *testing.T) {
	now := fixedNow()
	f := NewEmptyFake(func() time.Time { return now })
	for i := 0; i < 7; i++ {
		task, _ := f.CreateTask(CreateTaskInput{Title: "t" + itoa(i), Status: Todo})
		f.mu.Lock()
		at := now.Add(-time.Duration(i+1) * time.Hour)
		tt, _ := f.find(task.ID)
		tt.Status, tt.CompletedAt = Done, &at
		f.mu.Unlock()
	}
	n := mineOf(t, f)
	if got := titles(n.Done); !equal(got, []string{"t0", "t1", "t2", "t3", "t4"}) || n.MoreDone != 2 {
		t.Errorf("done %q, more %d; want the five newest and 2 more", got, n.MoreDone)
	}
	if !n.Empty() {
		t.Error("Done today made Now not empty: All clear looks at what needs attention")
	}
}

func TestBuildMineDoneTodayLeavesOutCancelledAndYesterday(t *testing.T) {
	now := fixedNow()
	f := NewEmptyFake(func() time.Time { return now })
	a, _ := f.CreateTask(CreateTaskInput{Title: "cancelled"})
	b, _ := f.CreateTask(CreateTaskInput{Title: "just after midnight"})
	c, _ := f.CreateTask(CreateTaskInput{Title: "just before midnight"})
	f.UpdateTask(a.ID, Patch{Status: StatusPtr(Cancelled)})
	f.UpdateTask(b.ID, Patch{Status: StatusPtr(Done)})
	f.UpdateTask(c.ID, Patch{Status: StatusPtr(Done)})
	y, m, d := now.Date()
	after, before := time.Date(y, m, d, 0, 0, 1, 0, now.Location()), time.Date(y, m, d, 0, 0, 0, 0, now.Location()).Add(-time.Second)
	f.mu.Lock()
	tb, _ := f.find(b.ID)
	tb.CompletedAt = &after
	tc, _ := f.find(c.ID)
	tc.CompletedAt = &before
	f.mu.Unlock()
	if got := titles(mineOf(t, f).Done); !equal(got, []string{"just after midnight"}) {
		t.Errorf("done today %q", got)
	}
}

func TestCompletedBy(t *testing.T) {
	done := fixedNow()
	task := Task{CompletedAt: &done}
	rep := func(agent string, ago time.Duration, summary string) *Report {
		return &Report{Agent: agent, Summary: summary, CreatedAt: done.Add(-ago)}
	}
	for _, c := range []struct {
		name          string
		agent, github *Report
		by, said      string
	}{
		{"no reports is me", nil, nil, "", ""},
		{"an agent's report as it closed", rep("claude-code", time.Second, "Spike merged"), nil, "claude-code", "Spike merged"},
		{"an agent's report long before is not the finish", rep("claude-code", time.Hour, "Planned"), nil, "", ""},
		{"a merge reported by GitHub after the agent's word", rep("claude-code", time.Hour, "Planned"), rep("github", 0, "PR #9 merged"), "claude-code", "PR #9 merged"},
		{"GitHub alone", nil, rep("github", 0, "PR #9 merged"), "github", "PR #9 merged"},
		{"a later report still counts", rep("claude-code", -time.Minute, "Cleaned up"), nil, "claude-code", "Cleaned up"},
	} {
		by, said := CompletedBy(task, c.agent, c.github)
		if by != c.by || said != c.said {
			t.Errorf("%s: (%q, %q), want (%q, %q)", c.name, by, said, c.by, c.said)
		}
	}
	if by, _ := CompletedBy(Task{}, rep("claude-code", 0, "x"), nil); by != "" {
		t.Errorf("a task that is not done was finished by %q", by)
	}
}
