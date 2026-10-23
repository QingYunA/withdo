package data

import (
	"testing"
	"time"
)

// waitingTask is a task the user handed to an agent that has not reported on it yet, a few days ago.
func waitingTask(mutate func(*Task)) Task {
	handedAt := fixedNow().Add(-100 * time.Hour)
	t := Task{
		ID: 1, Title: "Write the plan", Status: Todo, Source: SourceAgent, Priority: P2,
		CreatedAt: handedAt, UpdatedAt: handedAt, HandedTo: "claude", HandedAt: &handedAt,
	}
	if mutate != nil {
		mutate(&t)
	}
	return t
}

func TestAWaitingTaskIsNotHandedBackToTheUserOnMine(t *testing.T) {
	remind := fixedNow().Add(-time.Hour)
	closed := PRClosed
	failing := PRChecksFailing
	open := PROpen
	cases := map[string]func(*Task){
		"plain":               nil,
		"with a due reminder": func(t *Task) { t.RemindAt = &remind },
		"with a closed pull request": func(t *Task) {
			t.PRURL, t.PRNumber, t.PRState = "https://github.com/a/b/pull/1", 1, closed
		},
		"with failing checks": func(t *Task) {
			t.PRURL, t.PRNumber, t.PRState, t.PRChecks = "https://github.com/a/b/pull/1", 1, open, failing
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			n := mineOfTasks([]Task{waitingTask(mutate)}, nil, nil)
			if len(n.FromAgents)+len(n.Today)+len(n.Later) != 0 || n.Count() != 0 || !n.Empty() {
				t.Errorf("Mine = %+v, want nothing: the task waits for its agent", n)
			}
		})
	}
}

func TestAWaitingTaskIsNeitherStaleNorWorking(t *testing.T) {
	in := mineInputOfTasks([]Task{waitingTask(nil)}, nil, nil)
	if got := newBuilder(in).item(in.Tasks[0]); got.Stale || got.Hours != 100 {
		t.Errorf("item: stale %v after %d hours, want not stale: it is To do, nobody went quiet", got.Stale, got.Hours)
	}
	a := BuildAgents(in)
	if a.Count() != 0 || len(a.Working) != 0 || len(a.Done) != 0 || len(a.Queued) != 1 {
		t.Errorf("Agents = %+v, want it queued only: a waiting task is not at work", a)
	}
}

func TestAWaitingTaskIsNotInTheOverview(t *testing.T) {
	h := newHandRig(t)
	task := must[Task](t)(h.CreateTask(CreateTaskInput{Title: "Write the plan", ProjectID: &h.project.ID, Status: Doing}))
	must[Task](t)(h.HandOff(task.ID))
	if ov := must[Overview](t)(h.Overview()); len(ov.Groups) != 0 {
		t.Errorf("overview = %+v, want no active task while it waits", ov.Groups)
	}
}
