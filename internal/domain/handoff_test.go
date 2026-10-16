package domain_test

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// handedTask is a done task of the user's, handed to claude.
func handedTask(t *testing.T, c *domain.Core) *domain.Task {
	t.Helper()
	p := mustProject(t, c, "acme/widgets")
	task := mustTask(t, c, domain.CreateTaskInput{Title: "ship it", ProjectID: &p.ID, Status: domain.StatusDone})
	return must[*domain.Task](t)(c.HandOff(ctx, task.ID, "claude"))
}

func TestHandOffMakesTheTaskTheAgentsAndWaiting(t *testing.T) {
	c, clk := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	task := mustTask(t, c, domain.CreateTaskInput{Title: "ship it", ProjectID: &p.ID, Status: domain.StatusDone})
	if task.CompletedAt == nil {
		t.Fatal("a done task has completed_at")
	}
	clk.advance(time.Hour)
	got := must[*domain.Task](t)(c.HandOff(ctx, task.ID, "codex"))
	eq(t, "source", got.Source, domain.SourceAgent)
	eq(t, "status", got.Status, domain.StatusTodo)
	if got.CompletedAt != nil {
		t.Errorf("completed_at = %v, want nil", *got.CompletedAt)
	}
	eqPtr(t, "handed_to", got.HandedTo, ptr("codex"))
	eqPtr(t, "handed_at", got.HandedAt, ptr(domain.FormatTime(t0.Add(time.Hour))))
	eq(t, "updated_at", got.UpdatedAt, domain.FormatTime(t0.Add(time.Hour)))
	again := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	eqPtr(t, "stored handed_to", again.HandedTo, ptr("codex"))
}

func TestHandOffRejectsBlankAgentAndMissingTask(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{})
	_, err := c.HandOff(ctx, task.ID, " ")
	wantFieldError(t, err, "agent")
	_, err = c.HandOff(ctx, 999, "claude")
	wantFieldError(t, err, "task_id")
}

func TestTakeBackReturnsTheTaskToTheUserKeepingItsStatus(t *testing.T) {
	c, _ := newCore(t)
	task := handedTask(t, c)
	got := must[*domain.Task](t)(c.TakeBack(ctx, task.ID))
	eq(t, "source", got.Source, domain.SourceUser)
	eq(t, "status", got.Status, domain.StatusTodo)
	if got.HandedTo != nil || got.HandedAt != nil {
		t.Errorf("handed = %v %v, want cleared", got.HandedTo, got.HandedAt)
	}
	_, err := c.TakeBack(ctx, 999)
	wantFieldError(t, err, "task_id")
}

func TestRestoreHandOffUndoesATakeBack(t *testing.T) {
	c, _ := newCore(t)
	task := handedTask(t, c)
	back := must[*domain.Task](t)(c.TakeBack(ctx, task.ID))
	got := must[*domain.Task](t)(c.RestoreHandOff(ctx, back.ID, "claude", task.HandedAt, domain.SourceAgent))
	eq(t, "source", got.Source, domain.SourceAgent)
	eq(t, "status", got.Status, domain.StatusTodo)
	eqPtr(t, "handed_to", got.HandedTo, ptr("claude"))
	eqPtr(t, "handed_at", got.HandedAt, task.HandedAt)

	none := must[*domain.Task](t)(c.RestoreHandOff(ctx, back.ID, "", nil, domain.SourceUser))
	eq(t, "source", none.Source, domain.SourceUser)
	if none.HandedTo != nil || none.HandedAt != nil {
		t.Errorf("handed = %v %v, want cleared", none.HandedTo, none.HandedAt)
	}
	_, err := c.RestoreHandOff(ctx, back.ID, "", nil, "robot")
	wantFieldError(t, err, "source")
}

func TestAnyReportClearsTheHandOff(t *testing.T) {
	status := domain.StatusDoing
	cases := map[string]func(t *testing.T, c *domain.Core, id int64) domain.Task{
		"ReportProgress": func(t *testing.T, c *domain.Core, id int64) domain.Task {
			r := progress(t, c, domain.ProgressInput{Summary: "on it", TaskID: &id, Status: &status})
			eq(t, "returned task status", r.Task.Status, domain.StatusDoing)
			return r.Task
		},
		"AddReport": func(t *testing.T, c *domain.Core, id int64) domain.Task {
			must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: id, Agent: "claude-code", Summary: "hello"}))
			return *must[*domain.Task](t)(c.GetTask(ctx, id))
		},
	}
	for name, report := range cases {
		t.Run(name, func(t *testing.T) {
			c, _ := newCore(t)
			task := handedTask(t, c)
			got := report(t, c, task.ID)
			if got.HandedTo != nil || got.HandedAt != nil {
				t.Errorf("returned handed = %v %v, want cleared", got.HandedTo, got.HandedAt)
			}
			stored := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
			if stored.HandedTo != nil || stored.HandedAt != nil {
				t.Errorf("stored handed = %v %v, want cleared", stored.HandedTo, stored.HandedAt)
			}
			eq(t, "source", stored.Source, domain.SourceAgent)
		})
	}
}

func TestReportProgressMovesAWaitingTaskToDoing(t *testing.T) {
	c, _ := newCore(t)
	task := handedTask(t, c)
	status := domain.StatusDoing
	r := progress(t, c, domain.ProgressInput{Summary: "on it", TaskID: &task.ID, Status: &status, Agent: "claude-code"})
	eq(t, "status", r.Task.Status, domain.StatusDoing)
	if r.Task.HandedTo != nil {
		t.Error("handed_to should clear")
	}
}

func TestRestoreTaskKeepsTheHandOff(t *testing.T) {
	c, _ := newCore(t)
	task := handedTask(t, c)
	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, task.ID))
	if err := c.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	got := must[*domain.Task](t)(c.RestoreTask(ctx, *snap))
	eqPtr(t, "handed_to", got.HandedTo, ptr("claude"))
	eqPtr(t, "handed_at", got.HandedAt, task.HandedAt)
}

func TestLastReportAgentIsTheNewestOfTheProject(t *testing.T) {
	c, clk := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	other := mustProject(t, c, "acme/other")
	a := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID})
	b := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID})
	o := mustTask(t, c, domain.CreateTaskInput{ProjectID: &other.ID})

	got := must[string](t)(c.LastReportAgent(ctx, p.ID))
	eq(t, "no report yet", got, "")

	add := func(task *domain.Task, agent string) {
		clk.advance(time.Minute)
		must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: agent, Summary: "s"}))
	}
	add(a, "claude-code")
	add(b, "codex")
	add(o, "gemini")           // another project
	add(a, domain.GitHubAgent) // the poller is not an agent
	eq(t, "newest on any task of the project", must[string](t)(c.LastReportAgent(ctx, p.ID)), "codex")
	eq(t, "other project", must[string](t)(c.LastReportAgent(ctx, other.ID)), "gemini")
}
