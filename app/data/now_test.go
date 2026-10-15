package data

import (
	"testing"
	"time"
)

// nowOf builds the Now screen of a fake store the way the views do.
func nowOf(t *testing.T, f *Fake) Now {
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
	return BuildNow(NowInput{
		Now: f.Now(), Tasks: tasks,
		Latest: func(id int) *Report { return firstReport(f, id, false) },
		Event:  func(id int) *Report { return firstReport(f, id, true) },
		ProjectName: func(id *int) string {
			if id == nil {
				return ""
			}
			return names[*id]
		},
	})
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

func titles(items []NowItem) []string {
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

func TestBuildNowSeedMatchesTheMockup(t *testing.T) {
	n := nowOf(t, NewFake(fixedNow))
	wantNeeds := []string{"Fix tray race on launch", "Reply to design feedback", "Native UI with mygo"}
	if got := titles(n.Needs); !equal(got, wantNeeds) {
		t.Errorf("needs %q, want %q (blocked, reminder, review)", got, wantNeeds)
	}
	if got := titles(n.Agents); !equal(got, []string{"Port core to Go", "Issue #218 adaptation"}) {
		t.Errorf("agents %q, newest report first", got)
	}
	if got := titles(n.Yours); !equal(got, []string{"Decide on the activation policy", "Book dentist", "Write the release notes", "Look into Liquid Glass for the sidebar"}) {
		t.Errorf("yours %q", got)
	}
	if n.Count() != 9 || n.Empty() || len(n.IDs()) != 9+len(n.Done) {
		t.Errorf("count %d", n.Count())
	}
}

func TestBuildNowSaysWhyAndWhatTheAgentSaid(t *testing.T) {
	n := nowOf(t, NewFake(fixedNow))
	words := map[string]string{
		"Fix tray race on launch":  "Needs a decision on the activation policy",
		"Reply to design feedback": "",
		"Native UI with mygo":      "",
	}
	for _, it := range n.Needs {
		if it.Words != words[it.Title] {
			t.Errorf("%s: words %q, want %q", it.Title, it.Words, words[it.Title])
		}
	}
	if !n.Needs[0].Stale || n.Needs[0].Hours != 72 {
		t.Errorf("blocked task stale %v hours %d", n.Needs[0].Stale, n.Needs[0].Hours)
	}
	if got := n.Agents[0]; got.Summary != "Ported the store and migrations" || got.NextStep != "Port the MCP server" {
		t.Errorf("agent said %q, next %q", got.Summary, got.NextStep)
	}
	if !n.Agents[0].Live || n.Agents[1].Live {
		t.Errorf("live dots: %v %v, want the 2 minute old report only", n.Agents[0].Live, n.Agents[1].Live)
	}
}

func TestBuildNowEmptySectionsAndAllClear(t *testing.T) {
	if !nowOf(t, NewEmptyFake(fixedNow)).Empty() {
		t.Error("an empty store is not All clear")
	}
	f := NewEmptyFake(fixedNow)
	f.CreateTask(CreateTaskInput{Title: "Only mine"})
	n := nowOf(t, f)
	if len(n.Needs) != 0 || len(n.Agents) != 0 || len(n.Yours) != 1 {
		t.Errorf("one user task gave %d/%d/%d", len(n.Needs), len(n.Agents), len(n.Yours))
	}
}

func TestBuildNowStaleAgentWithoutReports(t *testing.T) {
	now := fixedNow()
	f := NewEmptyFake(func() time.Time { return now })
	task, _ := f.CreateTask(CreateTaskInput{Title: "Quiet one", Source: SourceAgent, Status: Doing})
	now = now.Add(80 * time.Hour)
	n := nowOf(t, f)
	if len(n.Needs) != 1 || n.Needs[0].Kind != NeedStale || n.Needs[0].Hours != 80 || len(n.Agents) != 0 {
		t.Errorf("%+v / agents %d", n.Needs, len(n.Agents))
	}
	_ = task
}

func TestBuildNowReviewWithoutPRAndUserBlocked(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	f.CreateTask(CreateTaskInput{Title: "Reviewable", Source: SourceAgent, Status: Review})
	f.CreateTask(CreateTaskInput{Title: "Stuck", Source: SourceUser, Status: Blocked})
	f.CreateTask(CreateTaskInput{Title: "Waiting on someone", Source: SourceUser, Status: Review})
	n := nowOf(t, f)
	if got := titles(n.Needs); !equal(got, []string{"Stuck", "Reviewable"}) {
		t.Fatalf("needs %q: a user review task waits on others, not on me", got)
	}
	if n.Needs[0].Kind != NeedBlocked || n.Needs[0].Source != SourceUser || n.Needs[0].Words != "" ||
		n.Needs[1].Kind != NeedReview || n.Needs[1].PRNumber != 0 {
		t.Errorf("needs %+v, %+v", n.Needs[0], n.Needs[1])
	}
}

func TestBuildNowYoursLimitsAndHorizon(t *testing.T) {
	now := fixedNow()
	f := NewEmptyFake(func() time.Time { return now })
	for i := 0; i < 10; i++ {
		f.CreateTask(CreateTaskInput{Title: "Undated " + itoaT(i)})
	}
	far, _ := f.CreateTask(CreateTaskInput{Title: "Far away"})
	f.UpdateTask(far.ID, Patch{DueAt: TimePtr(now.Add(30 * 24 * time.Hour))})
	over, _ := f.CreateTask(CreateTaskInput{Title: "Overdue"})
	f.UpdateTask(over.ID, Patch{DueAt: TimePtr(now.Add(-26 * time.Hour))})
	n := nowOf(t, f)
	if n.Yours[0].Title != "Overdue" {
		t.Errorf("first row %q, want the overdue task", n.Yours[0].Title)
	}
	if len(n.Yours) != 1+MaxUndated || n.MoreUndated != 3 {
		t.Errorf("%d rows, %d more", len(n.Yours), n.MoreUndated)
	}
	for _, it := range n.Yours {
		if it.Title == "Far away" {
			t.Error("a task due in a month is on Now")
		}
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

func TestBuildNowDoneTodayListsWhatAnyoneFinishedSinceMidnightNewestFirst(t *testing.T) {
	n := nowOf(t, NewFake(fixedNow)) // 2:30 PM
	if got := titles(n.Done); !equal(got, []string{"Port the MCP server", "Send the packaging answer"}) {
		t.Errorf("done today %q: want what finished at 1:50 PM and 11:30 AM; yesterday's and older stay out", got)
	}
	if n.MoreDone != 0 || n.Count() != 9 || len(n.IDs()) != 11 {
		t.Errorf("more %d, count %d (done tasks are no part of it), ids %d (the rows drawn)", n.MoreDone, n.Count(), len(n.IDs()))
	}
}

func TestBuildNowDoneTodayNamesWhoFinishedIt(t *testing.T) {
	n := nowOf(t, NewFake(fixedNow))
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

func TestBuildNowDoneTodayStopsAtFiveAndCountsTheRest(t *testing.T) {
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
	n := nowOf(t, f)
	if got := titles(n.Done); !equal(got, []string{"t0", "t1", "t2", "t3", "t4"}) || n.MoreDone != 2 {
		t.Errorf("done %q, more %d; want the five newest and 2 more", got, n.MoreDone)
	}
	if !n.Empty() {
		t.Error("Done today made Now not empty: All clear looks at what needs attention")
	}
}

func TestBuildNowDoneTodayLeavesOutCancelledAndYesterday(t *testing.T) {
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
	if got := titles(nowOf(t, f).Done); !equal(got, []string{"just after midnight"}) {
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
