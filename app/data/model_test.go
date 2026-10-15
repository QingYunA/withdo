package data

import "testing"

func TestSubtasksAreTheTasksOfAParent(t *testing.T) {
	parent := 7
	tasks := []Task{{ID: 7}, {ID: 8, ParentID: &parent}, {ID: 9}, {ID: 10, ParentID: &parent}}
	got := Subtasks(tasks, 7)
	if len(got) != 2 || got[0].ID != 8 || got[1].ID != 10 || !got[0].IsSubtask() || tasks[0].IsSubtask() {
		t.Errorf("subtasks of 7: %+v", got)
	}
}
