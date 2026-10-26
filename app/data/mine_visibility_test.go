package data

import (
	"testing"
)

// With no Inbox and no Completed page, every open task is on Today, Agents or its project's page.

func TestBuildMineShowsMyTasksInReviewInToday(t *testing.T) {
	f := emptyDemo(t)
	f.CreateTask(CreateTaskInput{Title: "Check the draft", Status: Review})
	f.CreateTask(CreateTaskInput{Title: "Agent finished", Status: Review, Source: SourceAgent}) // From your agents, not here
	n := mineOf(t, f)
	if got := titles(n.Today); !equal(got, []string{"Check the draft"}) {
		t.Errorf("today %q, want my task in review", got)
	}
	if got := titles(n.FromAgents); !equal(got, []string{"Agent finished"}) {
		t.Errorf("from agents %q", got)
	}
	if n.Count() != 2 {
		t.Errorf("count %d: my review task waits for me", n.Count())
	}
}

func TestBuildAgentsListsQueuedAndParkedTasksWithNoProject(t *testing.T) {
	f := emptyDemo(t)
	p := f.SeedProject("scratch", "me/scratch")
	f.CreateTask(CreateTaskInput{Title: "Queued old", Status: Todo, Source: SourceAgent})
	f.CreateTask(CreateTaskInput{Title: "Queued new", Status: Todo, Source: SourceAgent})
	f.CreateTask(CreateTaskInput{Title: "Queued in a project", Status: Todo, Source: SourceAgent, ProjectID: IntPtr(p.ID)})
	f.CreateTask(CreateTaskInput{Title: "Parked", Status: Backlog, Source: SourceAgent})
	f.CreateTask(CreateTaskInput{Title: "Parked in a project", Status: Backlog, Source: SourceAgent, ProjectID: IntPtr(p.ID)})
	f.CreateTask(CreateTaskInput{Title: "Mine, queued", Status: Todo})
	in := mineInputOf(t, f)
	a := BuildAgents(in)
	if got := rowTitles(a.Queued); !equal(got, []string{"Queued new", "Queued old"}) {
		t.Errorf("queued %q, want the newest first and none of a project's", got)
	}
	if a.BacklogTotal != 1 || len(a.Backlog) != 0 || a.Empty() || a.Count() != 0 {
		t.Errorf("backlog %d (%d rows), empty %v, count %d: the fold is closed and nothing is at work", a.BacklogTotal, len(a.Backlog), a.Empty(), a.Count())
	}
	in.BacklogOpen = true
	if got := rowTitles(BuildAgents(in).Backlog); !equal(got, []string{"Parked"}) {
		t.Errorf("backlog rows %q", got)
	}
	if n := mineOf(t, f); n.NoDateTotal != 1 {
		t.Errorf("%d in No date, want only my queued task: an agent's backlog task is on the Agents page", n.NoDateTotal)
	}
}

func rowTitles(rows []AgentRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Title
	}
	return out
}
