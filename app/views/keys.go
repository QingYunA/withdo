package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// rowIDs lists the task ids of the current screen in the order they are drawn, which is the order
// the arrow keys walk.
func (m *Main) rowIDs(now data.Now) []int {
	switch {
	case m.nav.screen == screenNow:
		return now.IDs()
	case m.boardActive():
		var ids []int
		for _, col := range m.boardColumns() {
			for _, t := range col.tasks {
				ids = append(ids, t.ID)
			}
		}
		return ids
	}
	return m.listIDs()
}

// moveSelection moves the highlight by dir through n rows. -1 means no row is highlighted;
// moving up from the first row leaves the list.
func moveSelection(cur, dir, n int) int {
	if n == 0 {
		return -1
	}
	return max(-1, min(cur+dir, n-1))
}

// keys handles the keyboard for one frame: Cmd+K, Cmd+1 and Cmd+2 anywhere; Esc to close, then
// clear; and, unless a text field has the keys, arrows, Enter and Space on the rows.
func (m *Main) keys(c *ui.Context, ids []int) {
	if m.settings {
		return
	}
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyK):
		m.focusAsk(c)
	case c.Shortcut(ui.Cmd, ui.Key1):
		m.goTo(nav{})
	case c.Shortcut(ui.Cmd, ui.Key2):
		m.goTo(nav{screen: screenInbox})
	case c.Shortcut(0, ui.KeyEscape):
		m.escape()
	}
	if m.adding || m.inspector.editing() {
		return
	}
	switch {
	case c.Shortcut(0, ui.KeyDown):
		m.move(c, ids, 1)
	case c.Shortcut(0, ui.KeyUp):
		m.move(c, ids, -1)
	case m.askRef.Focused(c):
		// Enter belongs to the field; a Space is a space.
	case c.Shortcut(0, ui.KeyEnter):
		m.openSelected()
	case c.Shortcut(0, ui.KeySpace):
		m.toggleSelected()
	}
}

// escape cancels a running request, then closes the conversation sheet, the inline input and the
// inspector, then clears the selection; with nothing left to close a further Esc makes the
// assistant forget the conversation.
func (m *Main) escape() {
	switch {
	case m.convo.thinking:
		m.cancelAsk()
	case m.convo.open:
		m.closeSheet()
	case m.adding:
		m.cancelAdding()
	case m.openID != 0:
		m.openID = 0
	case m.sel != 0:
		m.sel = 0
	case m.convo.live:
		m.forgetConversation()
	}
}

func (m *Main) move(c *ui.Context, ids []int, dir int) {
	cur := -1
	for i, id := range ids {
		if id == m.sel {
			cur = i
		}
	}
	next := moveSelection(cur, dir, len(ids))
	if next < 0 {
		m.sel = 0
		if m.nav.screen == screenNow {
			m.focusAsk(c)
		}
		return
	}
	m.sel, m.moved = ids[next], true
	m.listRef.Focus(c) // takes the keys from the ask field
}

func (m *Main) openSelected() {
	if _, ok := m.app.Task(m.sel); !ok {
		return
	}
	m.openID = m.sel
	m.inspector.reset()
}

// toggleSelected marks one of my own tasks done, or reopens it. An agent's task is left alone.
func (m *Main) toggleSelected() {
	task, ok := m.app.Task(m.sel)
	if !ok || task.Source != data.SourceUser {
		return
	}
	if task.Status.Terminal() {
		m.app.SetStatus(task, data.Todo)
		return
	}
	m.app.SetStatus(task, data.Done)
}
