package data

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestHandOffGivesTheTaskToTheAgentAndTakeBackReturnsIt(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, now *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan", Status: Doing}))
		res := must[HandOffResult](t)(s.HandOff(task.ID))
		got := res.Task
		if got.Source != SourceAgent || got.Status != Todo || got.HandedTo != res.Agent || got.HandedTo == "" ||
			got.HandedAt == nil || got.CompletedAt != nil || res.AgentName == "" {
			t.Errorf("handed task = %+v, result %+v", got, res)
		}
		back := must[Task](t)(s.TakeBack(task.ID))
		if back.Source != SourceUser || back.Status != Todo || back.HandedTo != "" || back.HandedAt != nil {
			t.Errorf("taken back = %+v", back)
		}
		if _, err := s.HandOff(9999); err == nil {
			t.Error("handed off a task that does not exist")
		}
		if _, err := s.TakeBack(9999); err == nil {
			t.Error("took back a task that does not exist")
		}
	})
}

func TestRestoreHandOffUndoesATakeBack(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, now *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan"}))
		handed := must[HandOffResult](t)(s.HandOff(task.ID)).Task
		must[Task](t)(s.TakeBack(task.ID))

		got := must[Task](t)(s.RestoreHandOff(task.ID, handed.HandedTo, handed.HandedAt, SourceAgent))
		if got.Source != SourceAgent || got.HandedTo != handed.HandedTo || got.Status != Todo ||
			got.HandedAt == nil || !got.HandedAt.Equal(*handed.HandedAt) {
			t.Errorf("restored = %+v, want %+v", got, handed)
		}
		none := must[Task](t)(s.RestoreHandOff(task.ID, "", nil, SourceUser))
		if none.Source != SourceUser || none.HandedTo != "" || none.HandedAt != nil {
			t.Errorf("restored to nothing = %+v", none)
		}
		if _, err := s.RestoreHandOff(9999, "", nil, SourceUser); err == nil {
			t.Error("restored a task that does not exist")
		}
	})
}

func TestAReportOnAWaitingTaskEndsTheWaiting(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, now *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan"}))
		must[HandOffResult](t)(s.HandOff(task.ID))
		reportOn(t, s, task.ID, "on it")
		got := must[TaskDetail](t)(s.GetTask(task.ID)).Task
		if got.HandedTo != "" || got.HandedAt != nil || got.Source != SourceAgent {
			t.Errorf("after the report = %+v", got)
		}
	})
}

func TestRestoreTaskKeepsAWaitingTaskWaiting(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, _ *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan"}))
		handed := must[HandOffResult](t)(s.HandOff(task.ID)).Task
		snap := must[TaskSnapshot](t)(s.SnapshotTask(task.ID))
		if err := s.DeleteTask(task.ID); err != nil {
			t.Fatal(err)
		}
		back := must[Task](t)(s.RestoreTask(snap))
		if back.HandedTo != handed.HandedTo || back.HandedAt == nil || !back.HandedAt.Equal(*handed.HandedAt) {
			t.Errorf("restored = %+v, want waiting like %+v", back, handed)
		}
	})
}

// handRig is a Real store whose Mac has the commands in found, with a project.
type handRig struct {
	*Real
	project Project
}

func newHandRig(t *testing.T, found ...string) *handRig {
	t.Helper()
	env := &agentsetup.Env{Home: t.TempDir(), LookPath: func(name string) (string, error) {
		if slices.Contains(found, name) {
			return "/opt/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}}
	r, err := OpenReal(bg, filepath.Join(t.TempDir(), "t.db"), RealOptions{Agents: env})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	p := must[*domain.Project](t)(r.core.CreateProject(bg, domain.CreateProjectInput{Name: "widgets"}))
	return &handRig{Real: r, project: fromProject(*p)}
}

func (h *handRig) task(t *testing.T) Task {
	t.Helper()
	return must[Task](t)(h.CreateTask(CreateTaskInput{Title: "Write the plan", ProjectID: &h.project.ID}))
}

// reportAs files a report from the named agent on a new task of the project.
func (h *handRig) reportAs(t *testing.T, agent string) {
	t.Helper()
	other := h.task(t)
	must[*domain.Report](t)(h.core.AddReport(bg, domain.AddReportInput{TaskID: int64(other.ID), Agent: agent, Summary: "hi"}))
}

func TestHandOffPicksTheAgentThatLastReportedInTheProject(t *testing.T) {
	for reported, want := range map[string][2]string{
		"codex":       {"codex", "Codex"},
		"Claude Code": {"claude", "Claude Code"},
		"gemini-cli":  {"gemini", "Gemini CLI"},
	} {
		h := newHandRig(t, "claude", "codex", "gemini")
		h.reportAs(t, "claude-code")
		h.reportAs(t, reported)
		res := must[HandOffResult](t)(h.HandOff(h.task(t).ID))
		if res.Agent != want[0] || res.AgentName != want[1] || res.Task.HandedTo != want[0] {
			t.Errorf("after a report from %q: agent %q (%q), want %q", reported, res.Agent, res.AgentName, want[0])
		}
	}
}

func TestHandOffFallsBackToTheFirstAgentFoundThenClaude(t *testing.T) {
	h := newHandRig(t, "gemini", "codex")
	res := must[HandOffResult](t)(h.HandOff(h.task(t).ID))
	if res.Agent != "codex" {
		t.Errorf("no report yet, codex and gemini found: %q, want the first of the list (codex)", res.Agent)
	}
	h.reportAs(t, "some-unknown-agent")
	h.reportAs(t, domain.GitHubAgent)
	if res = must[HandOffResult](t)(h.HandOff(h.task(t).ID)); res.Agent != "codex" {
		t.Errorf("a report from no known agent: %q, want codex", res.Agent)
	}
	none := newHandRig(t)
	res = must[HandOffResult](t)(none.HandOff(none.task(t).ID))
	if res.Agent != "claude" || res.AgentName != "Claude Code" {
		t.Errorf("nothing found: %+v, want claude ", res)
	}
}

func TestHandOffOfATaskWithoutAProjectStillPicksAnAgent(t *testing.T) {
	h := newHandRig(t, "codex")
	task := must[Task](t)(h.CreateTask(CreateTaskInput{Title: "loose"}))
	res := must[HandOffResult](t)(h.HandOff(task.ID))
	if res.Agent != "codex" {
		t.Errorf("inbox task: %+v", res)
	}
}

func TestHandedOffTaskIsBackFromTheDatabaseWithItsFields(t *testing.T) {
	h := newHandRig(t, "claude")
	res := must[HandOffResult](t)(h.HandOff(h.task(t).ID))
	list := must[[]Task](t)(h.ListTasks(Filter{ProjectID: &h.project.ID, Source: SourceAgent}))
	if len(list) != 1 || list[0].HandedTo != "claude" || list[0].HandedAt == nil || !list[0].HandedAt.Equal(*res.Task.HandedAt) {
		t.Errorf("list = %+v", list)
	}
}
