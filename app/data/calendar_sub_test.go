package data

import (
	"testing"
	"time"
)

func TestCalendarLeavesSubtasksToTheirParents(t *testing.T) {
	now := fixedNow()
	due := now.Add(time.Hour)
	parent := Task{ID: 1, Title: "parent", Status: Doing, DueAt: &due}
	child := Task{ID: 2, ParentID: IntPtr(1), Title: "child", Status: Doing, DueAt: &due}
	days := BuildCalendar(CalendarInput{Now: now, Month: now, Tasks: []Task{parent, child}})
	n := 0
	for _, d := range days {
		for _, it := range d.Items {
			n++
			if it.Task.Title != "parent" {
				t.Errorf("the calendar shows %q", it.Task.Title)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d items, want the parent's only", n)
	}
}
