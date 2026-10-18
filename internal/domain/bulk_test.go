package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func statesOf(tasks []domain.Task) []domain.TaskState {
	out := make([]domain.TaskState, len(tasks))
	for i, t := range tasks {
		out[i] = domain.StateOf(t)
	}
	return out
}

func TestUpdateTasksChangesEveryTaskAndReturnsWhatItWasForOneUndo(t *testing.T) {
	c, clk := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a", ProjectID: &p.ID})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b", ProjectID: &p.ID, Status: domain.StatusDoing})
	kid := mustTask(t, c, domain.CreateTaskInput{Title: "kid", ParentID: &b.ID, ProjectID: &p.ID, Status: domain.StatusDoing})
	clk.advance(time.Hour)

	done := domain.StatusDone
	before, err := c.UpdateTasks(ctx, []int64{a.ID, b.ID}, domain.TaskPatch{Status: &done})
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, t := range before {
		ids = append(ids, t.ID)
	}
	if !slices.Equal(ids, []int64{a.ID, b.ID, kid.ID}) {
		t.Fatalf("before holds %v, want a, b and b's open subtask %d", ids, kid.ID)
	}
	for _, id := range []int64{a.ID, b.ID, kid.ID} {
		got := must[*domain.Task](t)(c.GetTask(ctx, id))
		eq(t, "status", got.Status, domain.StatusDone)
	}

	if err := c.RestoreStates(ctx, statesOf(before)); err != nil {
		t.Fatal(err)
	}
	was := map[int64]domain.Status{a.ID: domain.StatusTodo, b.ID: domain.StatusDoing, kid.ID: domain.StatusDoing}
	for id, want := range was {
		got := must[*domain.Task](t)(c.GetTask(ctx, id))
		eq(t, "restored status", got.Status, want)
		if got.CompletedAt != nil {
			t.Errorf("task %d kept completed_at %v after the undo", id, *got.CompletedAt)
		}
	}
}

func TestUpdateTasksIsAllOrNothing(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	done := domain.StatusDone
	_, err := c.UpdateTasks(ctx, []int64{a.ID, 999}, domain.TaskPatch{Status: &done})
	wantFieldError(t, err, "task_id")
	got := must[*domain.Task](t)(c.GetTask(ctx, a.ID))
	eq(t, "status", got.Status, domain.StatusTodo)
}

func TestUpdateTasksMovesToAProjectAndSetsAPriority(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	if _, err := c.UpdateTasks(ctx, []int64{a.ID, b.ID}, domain.TaskPatch{ProjectID: domain.Set(p.ID)}); err != nil {
		t.Fatal(err)
	}
	p0 := domain.P0
	before, err := c.UpdateTasks(ctx, []int64{a.ID, b.ID}, domain.TaskPatch{Priority: &p0})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{a.ID, b.ID} {
		got := must[*domain.Task](t)(c.GetTask(ctx, id))
		eqPtr(t, "project", got.ProjectID, &p.ID)
		eq(t, "priority", got.Priority, domain.P0)
	}
	if err := c.RestoreStates(ctx, statesOf(before)); err != nil {
		t.Fatal(err)
	}
	got := must[*domain.Task](t)(c.GetTask(ctx, a.ID))
	eq(t, "priority", got.Priority, before[0].Priority)
}

func TestHandOffTasksHandsAllOverAndUndoRestoresTheirOldStatus(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a", ProjectID: &p.ID, Status: domain.StatusDoing})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	before, err := c.HandOffTasks(ctx, []int64{a.ID, b.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{a.ID, b.ID} {
		got := must[*domain.Task](t)(c.GetTask(ctx, id))
		eq(t, "source", got.Source, domain.SourceAgent)
		eq(t, "status", got.Status, domain.StatusTodo)
		eqPtr(t, "handed_to", got.HandedTo, ptr(domain.AnyAgent))
	}
	if err := c.RestoreStates(ctx, statesOf(before)); err != nil {
		t.Fatal(err)
	}
	got := must[*domain.Task](t)(c.GetTask(ctx, a.ID))
	eq(t, "source", got.Source, domain.SourceUser)
	eq(t, "status", got.Status, domain.StatusDoing)
	if got.HandedTo != nil || got.HandedAt != nil {
		t.Errorf("handed = %v %v after the undo", got.HandedTo, got.HandedAt)
	}
	_, err = c.HandOffTasks(ctx, []int64{b.ID, 999})
	wantFieldError(t, err, "task_id")
	if again := must[*domain.Task](t)(c.GetTask(ctx, b.ID)); again.HandedTo != nil {
		t.Error("a failed hand-off still handed the first task over")
	}
}

func TestDeleteTasksAndRestoreTasksCarryTheReports(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: a.ID, Agent: "claude-code", Summary: "did a"}))
	snaps, err := c.DeleteTasks(ctx, []int64{a.ID, b.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{a.ID, b.ID} {
		if got := must[*domain.Task](t)(c.GetTask(ctx, id)); got != nil {
			t.Errorf("task %d is still there", id)
		}
	}
	if err := c.RestoreTasks(ctx, snaps); err != nil {
		t.Fatal(err)
	}
	reports := must[[]domain.Report](t)(c.ListReports(ctx, a.ID))
	if len(reports) != 1 || reports[0].Summary != "did a" {
		t.Errorf("reports after the undo: %+v", reports)
	}
	if err := c.RestoreTasks(ctx, snaps); err == nil {
		t.Error("restoring over live tasks succeeded")
	}
}
