package views

import "github.com/egoist/mygo/ui"

// undoToast is the one toast of Undo, so a newer change replaces the one before.
const undoToast = "undo"

// offerUndo says what happened in the toast with Undo, which ⌘Z does as well for as long as it shows.
func (m *Main) offerUndo(c *ui.Context, e undoEntry) {
	ticket := m.undos.offer(e)
	c.AddToast(ui.Toast{
		ID: undoToast, Title: e.title, Action: m.say.MenuUndo(), Timeout: undoWithin, OnAction: e.undo, Data: e.extra,
		OnClose: func() { m.undos.closed(ticket) },
	})
}

// tell shows a toast with nothing to undo in the place of the Undo toast, which it replaces.
func (m *Main) tell(c *ui.Context, title string) {
	m.undos.clear()
	c.AddToast(ui.Toast{ID: undoToast, Title: title, Timeout: undoWithin})
}

// undoKey is ⌘Z: it takes back the assistant's last change, else the one the toast offers, and
// leaves the key to a text field that has its own undo. It reports whether it took the key.
func (m *Main) undoKey(c *ui.Context) bool {
	var assistant *undoEntry
	if m.convo.canUndo() {
		assistant = &undoEntry{undo: func() { m.undoLast() }}
	}
	e := pickUndo(m.textUndoFree(c), assistant, m.undos.cur)
	if e == nil {
		return false
	}
	if e == m.undos.cur {
		m.undos.clear()
		c.CloseToast(undoToast)
	}
	e.undo()
	return true
}
