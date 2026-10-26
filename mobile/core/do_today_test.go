package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// TestDoTodayMovesATaskOfMineToToday pins the task menu's Do today: it keeps the time of day the
// task had, gives an undated task 18:00, returns the old due date for Undo, and refuses what the
// desktop's menu does not offer it for.
func TestDoTodayMovesATaskOfMineToToday(t *testing.T) {
	c, err := open(demoDatabase(t, demoNow), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	defer func() { _ = c.Close() }()
	create := func(input string) int64 {
		raw, err := c.CreateTask(input)
		if err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
		var task data.Task
		if err := json.Unmarshal([]byte(raw), &task); err != nil {
			t.Fatal(err)
		}
		return int64(task.ID)
	}
	dueOf := func(id int64) *time.Time {
		raw, err := c.Task(id)
		if err != nil {
			t.Fatal(err)
		}
		var page struct{ Task data.Task }
		if err := json.Unmarshal([]byte(raw), &page); err != nil {
			t.Fatal(err)
		}
		return page.Task.DueAt
	}

	undated := create(`{"title":"Buy a monitor arm"}`)
	before, err := c.DoToday(undated)
	if err != nil {
		t.Fatalf("DoToday on an undated task: %v", err)
	}
	if before != "" {
		t.Errorf("previous due of an undated task = %q, want empty", before)
	}
	if got := dueOf(undated); got == nil || !got.Equal(time.Date(2026, 10, 20, 18, 0, 0, 0, demoNow.Location())) {
		t.Errorf("undated task is due %v, want today 18:00", got)
	}

	dated := create(`{"title":"Call the dentist","due_at":"2026-10-23T09:30:00Z"}`)
	before, err = c.DoToday(dated)
	if err != nil {
		t.Fatalf("DoToday on a dated task: %v", err)
	}
	if want := "2026-10-23T09:30:00Z"; before != want {
		t.Errorf("previous due = %q, want %q", before, want)
	}
	if got := dueOf(dated); got == nil || !got.Equal(time.Date(2026, 10, 20, 9, 30, 0, 0, demoNow.Location())) {
		t.Errorf("dated task is due %v, want today 09:30", got)
	}
	if _, err := c.DoToday(dated); err == nil {
		t.Errorf("DoToday on a task already due today should refuse")
	}

	finished := create(`{"title":"Already finished"}`)
	if err := c.Complete(finished); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoToday(finished); err == nil {
		t.Errorf("DoToday on a finished task should refuse")
	}
	if _, err := c.DoToday(999999); err == nil {
		t.Errorf("DoToday on a missing task should refuse")
	}
}
