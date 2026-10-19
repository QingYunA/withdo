package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Handing a task to an agent: the user drags a row of Mine onto the Agents tray. The view asks the
// store to hand the task over, which starts nothing, and says so in a toast with Undo (and ⌘Z).
// Dragging an agent's row onto Mine takes it back. The sprites that play along are in handoff_drag.go.

// HandOff hands a task to an agent and refreshes. An error goes to the banner.
func (a *App) HandOff(id int) (data.Task, error) {
	var task data.Task
	err := a.do(func() (err error) { task, err = a.Store.HandOff(id); return })
	return task, a.fail(err)
}

// TakeBack makes an agent's task the user's again, and refreshes.
func (a *App) TakeBack(id int) (data.Task, error) {
	var task data.Task
	err := a.do(func() (err error) { task, err = a.Store.TakeBack(id); return })
	return task, a.fail(err)
}

// RestoreHandOff puts a task's Source and hand-off back as they were.
func (a *App) RestoreHandOff(id int, agent string, at *time.Time, source data.Source) error {
	return a.fail(a.do(func() error { _, err := a.Store.RestoreHandOff(id, agent, at, source); return err }))
}

// handOff is the drop on the tray: the task becomes the agent's and waits for it, an agent's header
// robot catches it, and a toast says so.
func (m *Main) handOff(c *ui.Context, id int) {
	task, ok := m.app.Task(id)
	if !ok || !canHandOff(task) {
		return
	}
	if _, err := m.app.HandOff(id); err != nil {
		return
	}
	m.ho.landed = landing{ids: []int{id}, at: m.app.Now()}
	m.offerUndo(c, undoEntry{title: m.say.HandOffDone(), extra: handOffCopy{copyForAgent([]int{id})}, undo: m.undoHandOff(id, task.Status)})
}

// undoHandOff takes the task back and gives it the status it had.
func (m *Main) undoHandOff(id int, was data.Status) func() {
	return func() {
		if _, err := m.app.TakeBack(id); err != nil {
			return
		}
		_ = m.app.Update(id, data.Patch{Status: data.StatusPtr(was)})
		if m.ho.landed.has(id) {
			m.ho.landed = landing{}
		}
	}
}

// takeBack is the drop of an agent's row on Mine: the task is the user's again, with no sprite.
func (m *Main) takeBack(c *ui.Context, id int) {
	task, ok := m.app.Task(id)
	if !ok || !canTakeBack(task) {
		return
	}
	if _, err := m.app.TakeBack(id); err != nil {
		return
	}
	m.offerUndo(c, undoEntry{title: m.say.HandOffTakenBack(), undo: func() {
		_ = m.app.RestoreHandOff(id, task.HandedTo, task.HandedAt, task.Source)
	}})
}

// offerHandOffUndo says in a toast that tasks were handed over, and how many of the selected ones
// were left out, with Undo.
func (m *Main) offerHandOffUndo(c *ui.Context, tasks []data.Task, skipped int, undo func()) {
	title := m.say.HandOffDone()
	switch {
	case skipped > 0:
		title = m.say.HandOffManySkipped(len(tasks), skipped)
	case len(tasks) > 1:
		title = m.say.HandOffMany(len(tasks))
	}
	m.offerUndo(c, undoEntry{title: title, extra: handOffCopy{copyForAgent(idsOf(tasks))}, undo: undo})
}
