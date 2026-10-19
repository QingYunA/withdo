package data

import (
	"reflect"
	"testing"
	"time"
)

func TestUpdateTasksChangesAllAndRestoreStatesPutsThemBack(t *testing.T) {
	withStore(t, func(t *testing.T, s *Real, now *time.Time) {
		p := must[Project](t)(s.CreateProject(NewProject{Name: "scratch"}))
		a := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a", Priority: P2}))
		b := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "b", Status: Doing, ProjectID: &p.ID}))
		kid := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "kid", Status: Doing, ProjectID: &p.ID, ParentID: &b.ID}))

		before := must[[]Task](t)(s.UpdateTasks([]int{a.ID, b.ID}, BulkPatch{Status: StatusPtr(Done)}))
		if len(before) != 3 || before[0].ID != a.ID || before[1].ID != b.ID || before[2].ID != kid.ID {
			t.Fatalf("before = %+v, want a, b and b's open subtask", before)
		}
		for _, id := range []int{a.ID, b.ID, kid.ID} {
			if got := must[TaskDetail](t)(s.GetTask(id)); got.Task.Status != Done || got.Task.CompletedAt == nil {
				t.Errorf("task %d = %+v, want done", id, got.Task)
			}
		}
		if err := s.RestoreStates(before); err != nil {
			t.Fatal(err)
		}
		for id, want := range map[int]Status{a.ID: Todo, b.ID: Doing, kid.ID: Doing} {
			if got := must[TaskDetail](t)(s.GetTask(id)); got.Task.Status != want || got.Task.CompletedAt != nil {
				t.Errorf("task %d after the undo = %+v, want %s and open", id, got.Task, want)
			}
		}

		moved := must[[]Task](t)(s.UpdateTasks([]int{a.ID, b.ID}, BulkPatch{ProjectID: IntPtr(0), Priority: PriorityPtr(P0)}))
		if got := must[TaskDetail](t)(s.GetTask(b.ID)); got.Task.ProjectID != nil || got.Task.Priority != P0 {
			t.Errorf("moved task = %+v, want the Inbox and P0", got.Task)
		}
		if err := s.RestoreStates(moved); err != nil {
			t.Fatal(err)
		}
		if got := must[TaskDetail](t)(s.GetTask(b.ID)); got.Task.ProjectID == nil || *got.Task.ProjectID != p.ID || got.Task.Priority != P2 {
			t.Errorf("task after the undo = %+v, want project %d and P2", got.Task, p.ID)
		}
	})
}

func TestUpdateTasksChangesNothingWhenOneIsMissing(t *testing.T) {
	withStore(t, func(t *testing.T, s *Real, now *time.Time) {
		a := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a"}))
		if _, err := s.UpdateTasks([]int{a.ID, 9999}, BulkPatch{Status: StatusPtr(Done)}); err == nil {
			t.Fatal("updated a task that does not exist")
		}
		if got := must[TaskDetail](t)(s.GetTask(a.ID)); got.Task.Status != Todo {
			t.Errorf("a = %+v, want it untouched", got.Task)
		}
		if _, err := s.UpdateTasks(nil, BulkPatch{Status: StatusPtr(Done)}); err == nil {
			t.Error("updated no tasks without a word")
		}
	})
}

func TestHandOffTasksAndRestoreStates(t *testing.T) {
	withStore(t, func(t *testing.T, s *Real, now *time.Time) {
		a := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a", Status: Doing}))
		b := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "b"}))
		before := must[[]Task](t)(s.HandOffTasks([]int{a.ID, b.ID}))
		for _, id := range []int{a.ID, b.ID} {
			got := must[TaskDetail](t)(s.GetTask(id)).Task
			if got.Source != SourceAgent || got.Status != Todo || got.HandedTo != AnyAgent || got.HandedAt == nil {
				t.Errorf("handed task = %+v", got)
			}
		}
		ta, tb := must[TaskDetail](t)(s.GetTask(a.ID)).Task, must[TaskDetail](t)(s.GetTask(b.ID)).Task
		if !ta.HandedAt.After(*tb.HandedAt) {
			t.Errorf("handed at %v and %v: the first of the tasks is the newest, so Waiting reads in their order", ta.HandedAt, tb.HandedAt)
		}
		if err := s.RestoreStates(before); err != nil {
			t.Fatal(err)
		}
		got := must[TaskDetail](t)(s.GetTask(a.ID)).Task
		if got.Source != SourceUser || got.Status != Doing || got.HandedTo != "" || got.HandedAt != nil {
			t.Errorf("after the undo = %+v, want the user's, Doing, not waiting", got)
		}
		if _, err := s.HandOffTasks([]int{b.ID, 9999}); err == nil {
			t.Error("handed off a task that does not exist")
		}
		if got := must[TaskDetail](t)(s.GetTask(b.ID)).Task; got.HandedTo != "" {
			t.Error("a failed hand-off still handed the first task over")
		}
	})
}

func TestDeleteTasksAndRestoreTasks(t *testing.T) {
	withStore(t, func(t *testing.T, s *Real, now *time.Time) {
		a := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a"}))
		b := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "b"}))
		snaps := must[[]TaskSnapshot](t)(s.DeleteTasks([]int{a.ID, b.ID}))
		if len(snaps) != 2 {
			t.Fatalf("snapshots = %d, want 2", len(snaps))
		}
		for _, id := range []int{a.ID, b.ID} {
			if _, err := s.GetTask(id); err == nil {
				t.Errorf("task %d is still there", id)
			}
		}
		if err := s.RestoreTasks(snaps); err != nil {
			t.Fatal(err)
		}
		for _, id := range []int{a.ID, b.ID} {
			if _, err := s.GetTask(id); err != nil {
				t.Errorf("task %d did not come back: %v", id, err)
			}
		}
		if err := s.RestoreTasks(snaps); err == nil {
			t.Error("restored over live tasks")
		}
		if _, err := s.DeleteTasks([]int{a.ID, 9999}); err == nil {
			t.Error("deleted a task that does not exist")
		}
		if _, err := s.GetTask(a.ID); err != nil {
			t.Error("a failed delete still removed the first task")
		}
	})
}

// Every field a BulkPatch can carry has to come back with RestoreStates, whole: the task after the
// Undo is the task before, whatever the change touched. A new field in BulkPatch fails here until
// the case is added.
func TestEveryBulkPatchFieldIsFullyUndone(t *testing.T) {
	zero := 0
	cases := map[string]BulkPatch{
		"Status":    {Status: StatusPtr(Done)},
		"Priority":  {Priority: PriorityPtr(P0)},
		"ProjectID": {ProjectID: &zero},
	}
	if n := reflect.TypeOf(BulkPatch{}).NumField(); n != len(cases) {
		t.Fatalf("BulkPatch has %d fields and the test %d cases", n, len(cases))
	}
	for field, patch := range cases {
		t.Run(field, func(t *testing.T) {
			withStore(t, func(t *testing.T, s *Real, now *time.Time) {
				p := must[Project](t)(s.CreateProject(NewProject{Name: "scratch"}))
				ms := must[Milestone](t)(s.CreateMilestone(p.ID, "v1", nil))
				a := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a", Status: Doing, Priority: P2, ProjectID: &p.ID}))
				must[Task](t)(s.UpdateTask(a.ID, Patch{MilestoneID: IntPtr(ms.ID)}))
				before := must[TaskDetail](t)(s.GetTask(a.ID)).Task

				*now = now.Add(time.Hour)
				undo := must[[]Task](t)(s.UpdateTasks([]int{a.ID}, patch))
				if changed := must[TaskDetail](t)(s.GetTask(a.ID)).Task; reflect.DeepEqual(changed, before) {
					t.Fatalf("%s: the patch changed nothing", field)
				}
				*now = now.Add(time.Hour)
				if err := s.RestoreStates(undo); err != nil {
					t.Fatal(err)
				}
				after := must[TaskDetail](t)(s.GetTask(a.ID)).Task
				after.UpdatedAt, before.UpdatedAt = time.Time{}, time.Time{}
				if !reflect.DeepEqual(after, before) {
					t.Errorf("%s: after the undo\n%+v\nwant\n%+v", field, after, before)
				}
			})
		})
	}
}
