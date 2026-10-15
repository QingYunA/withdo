package data

import "testing"

func TestFakeParentDoneClosesItsSubtasks(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	parent, _ := f.CreateTask(CreateTaskInput{Title: "parent", Source: SourceAgent, Status: Doing})
	doing, _ := f.CreateTask(CreateTaskInput{Title: "a", Source: SourceAgent, Status: Doing, ParentID: &parent.ID})
	nested, _ := f.CreateTask(CreateTaskInput{Title: "b", Source: SourceAgent, Status: Review, ParentID: &doing.ID})
	cancelled, _ := f.CreateTask(CreateTaskInput{Title: "c", Source: SourceAgent, Status: Cancelled, ParentID: &parent.ID})
	other, _ := f.CreateTask(CreateTaskInput{Title: "other", Source: SourceAgent, Status: Doing})

	done := Done
	if _, err := f.UpdateTask(parent.ID, Patch{Status: &done}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{doing.ID, nested.ID} {
		d, err := f.GetTask(id)
		if err != nil {
			t.Fatal(err)
		}
		if d.Task.Status != Done || d.Task.CompletedAt == nil {
			t.Errorf("subtask %d: status %v, completed %v", id, d.Task.Status, d.Task.CompletedAt)
		}
		if len(d.Reports) != 1 || d.Reports[0].Agent != "adhd-todo" || d.Reports[0].Summary != "Closed with its parent task" {
			t.Errorf("reports of %d = %+v", id, d.Reports)
		}
	}
	if d, _ := f.GetTask(cancelled.ID); d.Task.Status != Cancelled || len(d.Reports) != 0 {
		t.Errorf("a cancelled subtask is left alone: %+v", d)
	}
	if d, _ := f.GetTask(other.ID); d.Task.Status != Doing {
		t.Errorf("an unrelated task is left alone: %v", d.Task.Status)
	}

	// Reopening the parent does not reopen what was closed with it.
	doingAgain := Doing
	if _, err := f.UpdateTask(parent.ID, Patch{Status: &doingAgain}); err != nil {
		t.Fatal(err)
	}
	if d, _ := f.GetTask(doing.ID); d.Task.Status != Done {
		t.Errorf("subtask reopened: %v", d.Task.Status)
	}
}

func TestRealMapsParentIDBothWays(t *testing.T) {
	r, _ := realPair(t, nil)
	parent, err := r.CreateTask(CreateTaskInput{Title: "parent", Source: SourceUser})
	if err != nil {
		t.Fatal(err)
	}
	kid, err := r.CreateTask(CreateTaskInput{Title: "kid", Source: SourceUser, ParentID: &parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	if kid.ParentID == nil || *kid.ParentID != parent.ID || !kid.IsSubtask() {
		t.Fatalf("created subtask = %+v", kid)
	}
	got, err := r.GetTask(kid.ID)
	if err != nil || got.Task.ParentID == nil || *got.Task.ParentID != parent.ID {
		t.Fatalf("read back %+v, %v", got.Task.ParentID, err)
	}
	all, err := r.ListTasks(Filter{})
	if err != nil || len(Subtasks(all, parent.ID)) != 1 {
		t.Fatalf("Subtasks of the parent in %+v, %v", all, err)
	}
	if _, err := r.CreateTask(CreateTaskInput{Title: "orphan", Source: SourceUser, ParentID: new(int)}); err == nil {
		t.Error("a missing parent should be refused")
	}

	// Closing the parent through the store closes the subtask, as in the fake.
	done := Done
	if _, err := r.UpdateTask(parent.ID, Patch{Status: &done}); err != nil {
		t.Fatal(err)
	}
	d, err := r.GetTask(kid.ID)
	if err != nil || d.Task.Status != Done {
		t.Fatalf("subtask after its parent was closed: %+v, %v", d.Task.Status, err)
	}
}
