package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// canDoToday says whether the task menu offers Do today: for a task of mine that is not finished
// and is not due today already.
func (m *Main) canDoToday(task data.Task) bool {
	if !mine(task) || task.Status.Terminal() {
		return false
	}
	return task.DueAt == nil || data.DaysFrom(*task.DueAt, m.app.Now()) != 0
}

// doToday sets the due date of a task of mine to today, keeping the time of day it had; a task with
// no due time gets the end of the working day, which reads as the day alone. A toast says so, with
// Undo, which gives the task its old due date back.
func (m *Main) doToday(c *ui.Context, task data.Task) {
	now := m.app.Now()
	hour, minute := defaultDueHour, 0
	if task.DueAt != nil {
		l := task.DueAt.In(now.Location())
		hour, minute = l.Hour(), l.Minute()
	}
	y, mo, d := now.Date()
	due := time.Date(y, mo, d, hour, minute, 0, 0, now.Location())
	before := time.Time{} // no due date: Undo clears it
	if task.DueAt != nil {
		before = *task.DueAt
	}
	if m.app.Update(task.ID, data.Patch{DueAt: data.TimePtr(due)}) != nil {
		return
	}
	m.offerUndo(c, undoEntry{title: m.say.ToastMovedToToday(), undo: func() {
		_ = m.app.Update(task.ID, data.Patch{DueAt: data.TimePtr(before)})
	}})
}
