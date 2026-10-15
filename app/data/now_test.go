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
