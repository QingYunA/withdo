package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

// A task made by its first report takes the summary as its title; the line under the title then
// says who it is and not the same words again.
func TestARowsSecondLineDoesNotRepeatItsTitle(t *testing.T) {
	task := data.Task{ID: 999, Title: "missing dir", Source: data.SourceAgent, Status: data.Doing}
	echo := data.OverviewTask{Task: task, LatestReport: &data.Report{Agent: "dogfood-probe", Summary: "  missing dir "}}
	other := data.OverviewTask{Task: task, LatestReport: &data.Report{Agent: "dogfood-probe", Summary: "found the dir"}}

	if got := listMeta(task, echo); got != "dogfood-probe" {
		t.Errorf("project list line = %q, want only the agent", got)
	}
	if got := listMeta(task, other); got != "dogfood-probe · found the dir" {
		t.Errorf("project list line = %q", got)
	}

	task.PRNumber = 12
	if got := agentSaid(task, echo); got != "#12" {
		t.Errorf("Mine line = %q, want only the pull request", got)
	}
	if got := agentSaid(task, other); got != "#12 · found the dir" {
		t.Errorf("Mine line = %q", got)
	}
}

func TestTheAgentsLineDoesNotRepeatTheTitleButKeepsTheNextStep(t *testing.T) {
	app, _, _ := newTestApp(t)
	m := NewMain(app)
	row := data.AgentRow{Item: data.Item{Task: data.Task{ID: 999, Title: "missing dir"}, Summary: "missing dir"}}
	if got := m.workingMeta(row); got != "Agent" {
		t.Errorf("Agents line = %q, want only the agent", got)
	}
	row.NextStep = "Add the check"
	if got := m.workingMeta(row); got == "Agent" || got == "" {
		t.Errorf("Agents line = %q lost the next step", got)
	}
	row.Summary, row.NextStep = "found the dir", ""
	if got := m.workingMeta(row); got != "Agent · found the dir" {
		t.Errorf("Agents line = %q", got)
	}
}
