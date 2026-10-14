package domain_test

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

var (
	now    = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	past   = "2026-06-01T11:00:00.000Z"
	future = "2026-06-01T13:00:00.000Z"
)

func TestClaimDueReminders(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	a := mustTask(t, c, domain.CreateTaskInput{Title: "in project", ProjectID: &p.ID, RemindAt: ptr(past)})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "in inbox", RemindAt: ptr(domain.FormatTime(now))})
	mustTask(t, c, domain.CreateTaskInput{Title: "future", RemindAt: ptr(future)})
	mustTask(t, c, domain.CreateTaskInput{Title: "none"})
	mustTask(t, c, domain.CreateTaskInput{Title: "done", RemindAt: ptr(past), Status: domain.StatusDone})
	mustTask(t, c, domain.CreateTaskInput{Title: "cancelled", RemindAt: ptr(past), Status: domain.StatusCancelled})

	has := must[bool](t)(c.HasDueReminders(ctx, now))
	if !has {
		t.Error("HasDueReminders should see the due tasks")
	}
	due := must[[]domain.Reminder](t)(c.ClaimDueReminders(ctx, now))
	if len(due) != 2 {
		t.Fatalf("due = %+v", due)
	}
	eq(t, "first id (ordered by remind_at)", due[0].TaskID, a.ID)
	eqPtr(t, "first project", due[0].ProjectName, ptr("widgets"))
	eq(t, "second id", due[1].TaskID, b.ID)
	if due[1].ProjectName != nil {
		t.Error("inbox reminder has no project name")
	}
	stored := must[*domain.Task](t)(c.GetTask(ctx, a.ID))
	eqPtr(t, "reminded_at", stored.RemindedAt, ptr(domain.FormatTime(now)))
	if must[bool](t)(c.HasDueReminders(ctx, now)) {
		t.Error("nothing should be due after claiming")
	}
}

func TestRemindersFireOncePerArming(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "x", RemindAt: ptr(past)})
	claim := func() int { return len(must[[]domain.Reminder](t)(c.ClaimDueReminders(ctx, now))) }
	eq(t, "first claim", claim(), 1)
	eq(t, "second claim", claim(), 0)
	eq(t, "far future claim", len(must[[]domain.Reminder](t)(c.ClaimDueReminders(ctx, now.AddDate(1, 0, 0)))), 0)
	must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{RemindAt: domain.Set(past)}))
	eq(t, "re-armed claim", claim(), 1)
	eq(t, "claim after re-arm", claim(), 0)
}

func TestConcurrentClaimsHandOutEachReminderOnce(t *testing.T) {
	c, _ := newCore(t)
	for i := 0; i < 20; i++ {
		mustTask(t, c, domain.CreateTaskInput{Title: "x", RemindAt: ptr(past)})
	}
	results := make(chan int, 8)
	for i := 0; i < 8; i++ {
		go func() {
			due, err := c.ClaimDueReminders(ctx, now)
			if err != nil {
				t.Error(err)
			}
			results <- len(due)
		}()
	}
	total := 0
	for i := 0; i < 8; i++ {
		total += <-results
	}
	eq(t, "total claimed", total, 20)
}
