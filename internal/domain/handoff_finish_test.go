package domain_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// A task finished by any path other than a report (by hand, a merged pull request) stops waiting.
func TestFinishingATaskEndsItsHandOff(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusDone, domain.StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			c, _ := newCore(t)
			task := handedTask(t, c)
			got := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Status: &status}))
			eq(t, "status", got.Status, status)
			if got.HandedTo != nil || got.HandedAt != nil {
				t.Errorf("handed = %v %v, want cleared", got.HandedTo, got.HandedAt)
			}
			stored := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
			if stored.HandedTo != nil || stored.HandedAt != nil {
				t.Errorf("stored handed = %v %v, want cleared", stored.HandedTo, stored.HandedAt)
			}
		})
	}
}

func TestAStatusChangeThatDoesNotFinishKeepsTheHandOff(t *testing.T) {
	c, _ := newCore(t)
	task := handedTask(t, c)
	for _, status := range []domain.Status{domain.StatusBlocked, domain.StatusTodo} {
		got := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Status: &status}))
		eqPtr(t, "handed_to", got.HandedTo, ptr(domain.AnyAgent))
		eqPtr(t, "handed_at", got.HandedAt, task.HandedAt)
	}
}

func TestRestoreHandOffStillRestoresOnAFinishedTask(t *testing.T) {
	c, _ := newCore(t)
	task := handedTask(t, c)
	done := domain.StatusDone
	must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Status: &done}))
	got := must[*domain.Task](t)(c.RestoreHandOff(ctx, task.ID, domain.AnyAgent, task.HandedAt, domain.SourceAgent))
	eqPtr(t, "handed_to", got.HandedTo, ptr(domain.AnyAgent))
	eqPtr(t, "handed_at", got.HandedAt, task.HandedAt)
}

// Closing a parent finishes its open subtasks through the same path.
func TestClosingAParentEndsTheHandOffOfItsSubtasks(t *testing.T) {
	c, _ := newCore(t)
	task := handedTask(t, c)
	sub := mustTask(t, c, domain.CreateTaskInput{Title: "sub", ProjectID: task.ProjectID, ParentID: &task.ID})
	must[*domain.Task](t)(c.HandOff(ctx, sub.ID))
	done := domain.StatusDone
	must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Status: &done}))
	got := must[*domain.Task](t)(c.GetTask(ctx, sub.ID))
	eq(t, "subtask status", got.Status, domain.StatusDone)
	if got.HandedTo != nil || got.HandedAt != nil {
		t.Errorf("subtask handed = %v %v, want cleared", got.HandedTo, got.HandedAt)
	}
}
