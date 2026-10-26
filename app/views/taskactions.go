package views

import "github.com/egoist/mygo/ui"

// deleteTask deletes a task at once, without asking, and offers Undo for six seconds: the toast's
// button and ⌘Z restore it with its id, fields and reports.
func (m *Main) deleteTask(c *ui.Context, id int) {
	task, ok := m.app.Task(id)
	if !ok {
		return
	}
	snap, err := m.app.SnapshotTask(id)
	if err != nil || m.app.DeleteTask(id) != nil {
		return
	}
	if m.openID == id {
		m.openID = 0
	}
	if m.sel == id {
		m.sel = 0
	}
	m.offerUndo(c, undoEntry{title: m.say.MenuDeleted(task.Title), undo: func() { _ = m.app.RestoreTask(snap) }})
}

// askAboutTask opens the task in the inspector and asks for the keyboard to go to the inspector's
// composer, where the user can say what to change about it. The composer reads
// m.focusInspectorComposer and clears it once it has taken the focus.
func (m *Main) askAboutTask(id int) {
	if _, ok := m.app.Task(id); !ok {
		return
	}
	m.openRow(id)
	m.focusInspectorComposer = true
}

// taskKeys are the keys of a selected row: S, P, M and R open the task menu on that piece, under the
// row, and Cmd+J and Cmd+Backspace ask about the task and delete it. ⌘J works while typing
// elsewhere; the plain letters do not.
func (m *Main) taskKeys(c *ui.Context, typing bool) {
	if m.multi.Count() >= 2 && !typing {
		m.manyTaskKeys(c)
		return
	}
	id := m.sel
	if id == 0 {
		id = m.openID
	}
	task, ok := m.app.Task(id)
	if !ok {
		return
	}
	if keyAsk.down(c) {
		m.askAboutTask(id)
		return
	}
	if typing {
		return
	}
	if keyToday.down(c) {
		if m.canDoToday(task) {
			m.doToday(c, task)
		}
		return
	}
	pages := map[binding]menuPage{keyStatus: pageStatus, keyPriority: pagePriority, keyMove: pageMove, keyRemind: pageRemind}
	for key, page := range pages {
		if key.down(c) {
			if key == keyPriority && mine(task) { // the star toggles at once
				m.toggleImportant(task)
				return
			}
			m.openMenuForSelection(id, page)
			return
		}
	}
	if keyDelete.down(c) {
		m.deleteTask(c, id)
	}
}

// openMenuForSelection opens the menu under the selected row, on one piece.
func (m *Main) openMenuForSelection(id int, page menuPage) {
	r := m.selRect
	m.openMenu(id, page, r.X+24, r.Y+r.H, false)
}
