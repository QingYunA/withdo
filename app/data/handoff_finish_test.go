package data

import (
	"testing"
	"time"
)

// A task finished any way but a report stops waiting; a status change that does not finish it keeps
// it waiting. Both stores behave alike.
func TestFinishingAWaitingTaskEndsTheWaiting(t *testing.T) {
	for _, status := range []Status{Done, Cancelled} {
		t.Run(string(status), func(t *testing.T) {
			withStore(t, func(t *testing.T, s *Real, _ *time.Time) {
				task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan"}))
				must[Task](t)(s.HandOff(task.ID))

				blocked := Blocked
				kept := must[Task](t)(s.UpdateTask(task.ID, Patch{Status: &blocked}))
				if kept.HandedTo != AnyAgent || kept.HandedAt == nil {
					t.Fatalf("a status change that does not finish ended the waiting: %+v", kept)
				}
				got := must[Task](t)(s.UpdateTask(task.ID, Patch{Status: &status}))
				if got.HandedTo != "" || got.HandedAt != nil {
					t.Errorf("after finishing = %+v", got)
				}
				stored := must[TaskDetail](t)(s.GetTask(task.ID)).Task
				if stored.HandedTo != "" || stored.HandedAt != nil {
					t.Errorf("stored = %+v", stored)
				}
			})
		})
	}
}
