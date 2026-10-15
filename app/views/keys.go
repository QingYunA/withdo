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

// typing reports whether a text input has the keyboard. Its keys are the input's own
// (Element.OnTextKey), so the window leaves Esc, the arrows, Enter and Space alone meanwhile.
func (m *Main) typing(c *ui.Context) bool {
	return m.askRef.Focused(c) || m.addRef.Focused(c) || m.inspector.editing()
}

// keys handles the keyboard for one frame: Cmd+K, Cmd+1 and Cmd+2 anywhere; and, unless a text
// input has the keys, Esc to close, then clear, and arrows, Enter and Space on the rows.
func (m *Main) keys(c *ui.Context, ids []int) {
	if m.settings {
		return
	}
	m.runQueued(c, ids)
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyK):
		m.focusAsk(c)
	case c.Shortcut(ui.Cmd, ui.Key1):
		m.goTo(nav{})
	case c.Shortcut(ui.Cmd, ui.Key2):
		m.goTo(nav{screen: screenInbox})
	}
	if m.typing(c) {
		return
	}
	switch {
	case c.Shortcut(0, ui.KeyEscape):
		m.escape()
	case c.Shortcut(0, ui.KeyDown):
		m.move(c, ids, 1)
	case c.Shortcut(0, ui.KeyUp):
		m.move(c, ids, -1)
	case c.Shortcut(0, ui.KeyEnter):
		m.openSelected()
	case c.Shortcut(0, ui.KeySpace):
		m.toggleSelected()
	}
}

// queued is what a text input's key asked for that needs the Context, which OnTextKey does not
// have: it runs between frames. keys carries it out at the start of the next frame.
type queued struct {
	move  int  // arrow: walk the rows by this much
	leave bool // hand the keys from the input to the page
}

// runQueued carries out what the last key of a text input asked for.
func (m *Main) runQueued(c *ui.Context, ids []int) {
	q := m.queue
	m.queue = queued{}
	q.leave = q.leave || m.inspector.reverted()
	switch {
	case q.move != 0:
		m.move(c, ids, q.move)
	case q.leave:
		m.listRef.Focus(c)
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
