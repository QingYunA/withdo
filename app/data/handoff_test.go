package data

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestHandOffGivesTheTaskToTheAgentAndTakeBackReturnsIt(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, now *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan", Status: Doing}))
		got := must[Task](t)(s.HandOff(task.ID))
		if got.Source != SourceAgent || got.Status != Todo || got.HandedTo != AnyAgent ||
			got.HandedAt == nil || got.CompletedAt != nil {
			t.Errorf("handed task = %+v", got)
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
		handed := must[Task](t)(s.HandOff(task.ID))
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
		must[Task](t)(s.HandOff(task.ID))
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
		handed := must[Task](t)(s.HandOff(task.ID))
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

// handRig is a Real store with a project.
type handRig struct {
	*Real
	project Project
}

func newHandRig(t *testing.T) *handRig {
	t.Helper()
	r, err := OpenReal(bg, filepath.Join(t.TempDir(), "t.db"), RealOptions{})
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

func TestHandOffGoesToAnyAgentWhoeverReportedLast(t *testing.T) {
	for _, reported := range []string{"", "codex", "claude-code", "gemini-cli", "some-unknown-agent"} {
		h := newHandRig(t)
		if reported != "" {
			h.reportAs(t, reported)
		}
		if got := must[Task](t)(h.HandOff(h.task(t).ID)); got.HandedTo != AnyAgent || got.Source != SourceAgent {
			t.Errorf("after a report from %q: handed to %q, want %q", reported, got.HandedTo, AnyAgent)
		}
	}
	h := newHandRig(t)
	loose := must[Task](t)(h.CreateTask(CreateTaskInput{Title: "loose"}))
	if got := must[Task](t)(h.HandOff(loose.ID)); got.HandedTo != AnyAgent {
		t.Errorf("a task of the Inbox: handed to %q", got.HandedTo)
	}
}

func TestHandedOffTaskIsBackFromTheDatabaseWithItsFields(t *testing.T) {
	h := newHandRig(t)
	handed := must[Task](t)(h.HandOff(h.task(t).ID))
	list := must[[]Task](t)(h.ListTasks(Filter{ProjectID: &h.project.ID, Source: SourceAgent}))
	if len(list) != 1 || list[0].HandedTo != AnyAgent || list[0].HandedAt == nil || !list[0].HandedAt.Equal(*handed.HandedAt) {
		t.Errorf("list = %+v", list)
	}
}
