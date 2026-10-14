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
		Latest: func(id int) *Report {
			d, err := f.GetTask(id)
			if err != nil || len(d.Reports) == 0 {
				return nil
			}
			return &d.Reports[0]
		},
		ProjectName: func(id *int) string {
			if id == nil {
				return ""
			}
			return names[*id]
		},
	})
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
	if n.Count() != 9 || n.Empty() || len(n.IDs()) != 9 {
		t.Errorf("count %d", n.Count())
	}
}

func TestBuildNowSummariesAndLabels(t *testing.T) {
	n := nowOf(t, NewFake(fixedNow))
	wants := map[string]string{
		"Fix tray race on launch":  "Agent is waiting: “Needs a decision on the activation policy”",
		"Reply to design feedback": "You asked to be reminded at 2:30 PM",
		"Native UI with mygo":      "PR #12 is ready for your review",
	}
	for _, it := range n.Needs {
		if it.Summary != wants[it.Title] {
			t.Errorf("%s: summary %q, want %q", it.Title, it.Summary, wants[it.Title])
		}
	}
	if !n.Needs[0].Stale || n.Needs[0].Hours != 72 {
		t.Errorf("blocked task stale %v hours %d", n.Needs[0].Stale, n.Needs[0].Hours)
	}
	if got := n.Agents[0].Summary; got != "Ported the store and migrations; next: port the MCP server" {
		t.Errorf("agent summary %q", got)
	}
	if !n.Agents[0].Live || n.Agents[1].Live {
		t.Errorf("live dots: %v %v, want the 2 minute old report only", n.Agents[0].Live, n.Agents[1].Live)
	}
	labels := []string{"Today", "Tomorrow", "Fri", ""}
	for i, it := range n.Yours {
		if it.Label != labels[i] {
			t.Errorf("%s label %q, want %q", it.Title, it.Label, labels[i])
		}
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
	if len(n.Needs) != 1 || n.Needs[0].Kind != NeedStale || n.Needs[0].Summary != "No update for 3d" || len(n.Agents) != 0 {
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
	if n.Needs[0].Summary != "Marked as blocked" || n.Needs[1].Summary != "Ready for your review" {
		t.Errorf("summaries %q, %q", n.Needs[0].Summary, n.Needs[1].Summary)
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
	if n.Yours[0].Title != "Overdue" || n.Yours[0].Label != "Yesterday" {
		t.Errorf("first row %q %q, want the overdue task", n.Yours[0].Title, n.Yours[0].Label)
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

func TestReminderWording(t *testing.T) {
	now := fixedNow()
	cases := map[time.Duration]string{
		-time.Hour:       "at 1:30 PM",
		-24 * time.Hour:  "yesterday at 2:30 PM",
		-72 * time.Hour:  "on Oct 11 at 2:30 PM",
		-time.Second * 5: "at 2:29 PM",
	}
	for d, want := range cases {
		if got := reminderWhen(now.Add(d), now); got != want {
			t.Errorf("%v: %q, want %q", d, got, want)
		}
	}
}

func TestLowerFirstKeepsAcronyms(t *testing.T) {
	for in, want := range map[string]string{"Port the MCP server": "port the MCP server", "MCP tools": "MCP tools", "next": "next", "": ""} {
		if got := lowerFirst(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestNextMomentAndShort(t *testing.T) {
	now := fixedNow()
	a, b := now.Add(time.Hour), now.Add(2*time.Hour)
	if at, ok := NextMoment(Task{DueAt: &b, RemindAt: &a}); !ok || !at.Equal(a) {
		t.Errorf("next moment %v %v", at, ok)
	}
	if _, ok := NextMoment(Task{}); ok {
		t.Error("a task with no times has a moment")
	}
	if Short(0) != "1h" || Short(5) != "5h" || Short(72) != "3d" {
		t.Errorf("short %q %q %q", Short(0), Short(5), Short(72))
	}
	if Capital("") != "" || Capital("fri") != "Fri" {
		t.Error("capital")
	}
}

func itoaT(n int) string { return string(rune('0' + n)) }
