package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/egoist/mygo/ui"
)

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
	return m.cmp[surfacePage].ref.Focused(c) || m.cmp[surfaceInspector].ref.Focused(c) || m.searchRef.Focused(c) || m.renameRef.Focused(c) || m.inspector.editing()
}

// textUndoFree reports whether no text field would take Cmd+Z for its own undo (a composer with
// nothing typed has none).
func (m *Main) textUndoFree(c *ui.Context) bool {
	empty := func(s surface) bool { return m.cmp[s].ref.Focused(c) && m.cmp[s].buf == "" }
	return !m.typing(c) || empty(surfacePage) || empty(surfaceInspector)
}

// keys handles the keyboard for one frame: Cmd+K, and Cmd+1 to Cmd+7 for the sidebar's rows, anywhere; and, unless a text
// input has the keys, Esc to close, then clear, and arrows, Enter and Space on the rows.
func (m *Main) keys(c *ui.Context, ids []int) {
	if m.sheetOpen() {
		return
	}
	m.runQueued(c, ids)
	bar, barHit := m.barKey(c)
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyK):
		m.focusPageComposer(c)
	case c.Shortcut(ui.Cmd, ui.KeyF):
		m.searchRef.Focus(c)
	case c.Shortcut(ui.Cmd, ui.Key1):
		m.goTo(nav{})
	case c.Shortcut(ui.Cmd, ui.Key2):
		m.goTo(nav{screen: screenAgents})
	case c.Shortcut(ui.Cmd, ui.Key3):
		m.goTo(nav{screen: screenInbox})
	case c.Shortcut(ui.Cmd, ui.Key4):
		m.goTo(nav{screen: screenCalendar})
	case c.Shortcut(ui.Cmd, ui.Key5):
		m.goTo(nav{screen: screenCompleted})
	case c.Shortcut(ui.Cmd, ui.Key6):
		m.goTo(nav{screen: screenSummary})
	case c.Shortcut(ui.Cmd, ui.Key7) && m.app.RecentChangesOn():
		m.goTo(nav{screen: screenRecent})
	case barHit:
		m.runAction(c, bar, m.multi.IDs())
	case c.Shortcut(ui.Cmd, ui.KeyZ) && m.undoKey(c):
	}
	typing := m.typing(c)
	m.taskKeys(c, typing)
	if typing || m.pop.open { // a key of the selected row may have opened the task menu
		return
	}
	if m.pg.Kind == page.KindCalendar && m.calendarKey(c) {
		return
	}
	switch {
	case c.Shortcut(0, ui.KeyEscape):
		m.escape()
	case c.Shortcut(ui.Shift, ui.KeyDown):
		m.extendSelection(c, 1)
	case c.Shortcut(ui.Shift, ui.KeyUp):
		m.extendSelection(c, -1)
	case c.Shortcut(ui.Cmd, ui.KeyA):
		m.multi.All()
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

// barKey is the action of the bulk bar whose key is down, when the bar is up and no text input
// has the keys.
func (m *Main) barKey(c *ui.Context) (actionKind, bool) {
	if !m.barUp() || m.typing(c) {
		return 0, false
	}
	a, ok := pressedAction(actionsFor(m.tasksOf(m.multi.IDs())), func(b binding) bool { return b.down(c) }, actDone, actHandOff)
	return a.Kind, ok
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

// escape clears the rows selected together, and leaves select mode, first; then cancels a running
// request, closes the reply sheet and the inspector, and clears the selection; with nothing left to close a further Esc makes the
// assistant forget the conversation.
func (m *Main) escape() {
	switch {
	case m.multi.Active():
		m.multi.Clear()
	case m.convo.thinking:
		m.cancelAsk()
	case m.convo.open:
		m.closeSheet()
	case m.openID != 0:
		m.openID = 0
	case m.sel != 0:
		m.sel = 0
	case m.searching():
		m.clearSearch()
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
		m.multi.Moved(0)
		if m.pg.Composer {
			m.focusPageComposer(c)
		}
		return
	}
	m.sel, m.moved = ids[next], true
	m.multi.Moved(m.sel)
	m.listRef.Focus(c) // takes the keys from the composer
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
	if m.multi.Mode() { // there a row is ticked, not finished
		if m.sel != 0 {
			m.sel = m.multi.Toggle(m.sel, m.sel)
		}
		return
	}
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
