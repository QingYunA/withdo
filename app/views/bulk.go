package views

import (
	"slices"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The bulk actions of the rows selected together. Each is one change in the store and has one Undo,
// which the toast offers and ⌘Z does (offerUndo). When it has been done the page leaves select mode
// and forgets the selection.

// bulkTasks are the selected tasks, in the order the page draws them.
func (m *Main) bulkTasks() []data.Task {
	var out []data.Task
	for _, id := range m.selectedRows() {
		if t, ok := m.app.Task(id); ok {
			out = append(out, t)
		}
	}
	return out
}

// idsOf are the ids of tasks, in order.
func idsOf(tasks []data.Task) []int {
	out := make([]int, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
	}
	return out
}

// only keeps the tasks keep accepts.
func only(tasks []data.Task, keep func(data.Task) bool) []data.Task {
	var out []data.Task
	for _, t := range tasks {
		if keep(t) {
			out = append(out, t)
		}
	}
	return out
}

// finishBulk is what follows every bulk action: no rows are selected and select mode is over.
func (m *Main) finishBulk() {
	m.clearMulti()
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

// bulkSetStatus is Done and Cancel: every selected task that is not in that status yet.
func (m *Main) bulkSetStatus(c *ui.Context, s data.Status) {
	tasks := only(m.bulkTasks(), func(t data.Task) bool { return t.Status != s })
	title := m.say.BulkDoneToast(len(tasks))
	if s == data.Cancelled {
		title = m.say.BulkCancelToast(len(tasks))
	}
	m.bulkUpdate(c, tasks, data.Patch{Status: data.StatusPtr(s)}, title)
}

// bulkMove moves the selected tasks to a project, or to the Inbox for project 0. A subtask stays
// with its parent, and a task that is there already is left alone.
func (m *Main) bulkMove(c *ui.Context, project int) {
	tasks := only(m.bulkTasks(), func(t data.Task) bool {
		there := (t.ProjectID == nil && project == 0) || (t.ProjectID != nil && *t.ProjectID == project)
		return !t.IsSubtask() && !there
	})
	where := m.say.NavInbox()
	if p, ok := m.app.Project(project); ok {
		where = p.Name
	}
	m.bulkUpdate(c, tasks, data.Patch{ProjectID: data.IntPtr(project)}, m.say.BulkMoveToast(len(tasks), where))
}

// bulkPriority gives the selected tasks a priority.
func (m *Main) bulkPriority(c *ui.Context, p data.Priority) {
	tasks := only(m.bulkTasks(), func(t data.Task) bool { return t.Priority != p })
	m.bulkUpdate(c, tasks, data.Patch{Priority: data.PriorityPtr(p)}, m.say.BulkPriorityToast(len(tasks), string(p)))
}

// bulkHandOff hands the selected tasks to an agent: only the user's own unfinished ones go, and the
// toast says how many were skipped.
func (m *Main) bulkHandOff(c *ui.Context) {
	m.handOffMany(c, idsOf(m.bulkTasks()))
}

// handOffMany hands tasks to an agent as one change. Each waits in its own project's Waiting for an
// agent, the Inbox's in the Inbox. The toast offers Copy for the agent and Undo.
func (m *Main) handOffMany(c *ui.Context, ids []int) {
	var own []data.Task
	for _, id := range ids {
		if t, ok := m.app.Task(id); ok && t.Source == data.SourceUser && !t.Status.Terminal() {
			own = append(own, t)
		}
	}
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

// askBulkDelete asks before the selected tasks are deleted.
func (m *Main) askBulkDelete() {
	if len(m.selectedRows()) > 0 {
		m.manage.confirm = confirmation{open: true, tasks: m.selectedRows()}
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
