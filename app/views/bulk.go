package views

import (
	"slices"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The bulk actions of the rows selected together. Each is one change in the store and has one Undo,
// which the toast offers and ⌘Z does (offerUndo). When it has been done the page leaves select mode
// and forgets the selection.

// tasksOf are the tasks that are still there among ids.
func (m *Main) tasksOf(ids []int) []data.Task {
	var out []data.Task
	for _, id := range ids {
		if t, ok := m.app.Task(id); ok {
			out = append(out, t)
		}
	}
	return out
}

// finishBulk is what follows every bulk action: no rows are selected and select mode is over.
func (m *Main) finishBulk() {
	m.multi.Clear()
	m.sel = 0
}

// bulkUpdate applies a patch to tasks as one change, says so in the toast and offers one Undo.
func (m *Main) bulkUpdate(c *ui.Context, tasks []data.Task, p data.Patch, title string) {
	if len(tasks) == 0 {
		return
	}
	before, err := m.app.UpdateTasks(idsOf(tasks), p)
	if err != nil {
		return
	}
	if p.Status != nil && *p.Status == data.Done {
		for _, t := range tasks {
			m.app.startCompletion(t, byGlyph) // the rows fold into Done as they do for Space
		}
	}
	m.finishBulk()
	m.offerUndo(c, title, func() { _ = m.app.RestoreStates(before) })
}

// runAction does an action of the selection ids that runs at once: Done, Cancel, Hand to an agent
// or Delete. Move and Priority open a page of the menu, which the caller does.
func (m *Main) runAction(c *ui.Context, kind actionKind, ids []int) {
	a := actionOf(m.tasksOf(ids), kind)
	switch kind {
	case actDone:
		m.bulkUpdate(c, m.tasksOf(a.Ids), data.Patch{Status: data.StatusPtr(data.Done)}, m.say.BulkDoneToast(len(a.Ids)))
	case actCancel:
		m.bulkUpdate(c, m.tasksOf(a.Ids), data.Patch{Status: data.StatusPtr(data.Cancelled)}, m.say.BulkCancelToast(len(a.Ids)))
	case actHandOff:
		m.handOffMany(c, ids)
	case actDelete:
		m.askBulkDelete(ids)
	}
}

// bulkMove moves the selected tasks to a project, or to the Inbox for project 0.
func (m *Main) bulkMove(c *ui.Context, ids []int, project int) {
	tasks := movable(m.tasksOf(ids), project)
	where := m.say.NavInbox()
	if p, ok := m.app.Project(project); ok {
		where = p.Name
	}
	m.bulkUpdate(c, tasks, data.Patch{ProjectID: data.IntPtr(project)}, m.say.BulkMoveToast(len(tasks), where))
}

// bulkPriority gives the selected tasks a priority.
func (m *Main) bulkPriority(c *ui.Context, ids []int, p data.Priority) {
	tasks := repriced(m.tasksOf(ids), p)
	m.bulkUpdate(c, tasks, data.Patch{Priority: data.PriorityPtr(p)}, m.say.BulkPriorityToast(len(tasks), string(p)))
}

// handOffMany hands tasks to an agent as one change, whatever of them is the user's own and unfinished. Each waits in its own project's Waiting for an
// agent, the Inbox's in the Inbox. The toast offers Copy for the agent and Undo.
func (m *Main) handOffMany(c *ui.Context, ids []int) {
	own := only(m.tasksOf(ids), canHandOff)
	skipped := len(ids) - len(own)
	if len(own) == 0 {
		m.finishBulk()
		c.AddToast(ui.Toast{ID: handOffToast, Title: m.say.HandOffNone(), Timeout: undoWithin})
		return
	}
	before, err := m.app.HandOffTasks(idsOf(own))
	if err != nil {
		return
	}
	m.finishBulk()
	m.ho.landed = landing{ids: idsOf(own), at: m.app.Now()}
	m.offerHandOffUndo(c, own, skipped, func() {
		_ = m.app.RestoreStates(before)
		m.ho.landed = landing{}
	})
}

// askBulkDelete asks before the tasks are deleted.
func (m *Main) askBulkDelete(ids []int) {
	if len(ids) > 0 {
		m.manage.confirm = confirmation{open: true, tasks: slices.Clone(ids)}
	}
}

// bulkDelete deletes the tasks, with their reports, as one change.
func (m *Main) bulkDelete(c *ui.Context, ids []int) {
	snaps, err := m.app.DeleteTasks(ids)
	if err != nil {
		return
	}
	if slices.Contains(ids, m.openID) {
		m.openID = 0
	}
	m.finishBulk()
	m.offerUndo(c, m.say.BulkDeleteToast(len(ids)), func() { _ = m.app.RestoreTasks(snaps) })
}
