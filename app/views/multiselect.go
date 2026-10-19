package views

import (
	"github.com/egoist/mygo/ui"
)

// Selecting several tasks (docs/design/ui-spec.md, Selecting several tasks). The keyboard row (m.sel)
// is one row; m.multi, a selection.Selection, is the rows the user ticked together, and decides
// what every click and key does to them. While it is empty the page behaves as it always did, and
// the keyboard row is the only one that is selected. It belongs to one page: it is cleared when the
// page changes, and held to the rows the page draws (m.multi.Sync).

// rowClicked reports whether a click on the row should open its task. A ⌘-click or ⇧-click, and any
// click in select mode, select instead and report false.
func (m *Main) rowClicked(row ui.Element, id int) bool {
	if !row.Clicked() {
		return false
	}
	mods := row.ClickModifiers()
	switch {
	case mods&ui.Cmd != 0:
		m.nudgeSelectButton()
		m.sel = m.multi.Toggle(id, m.sel)
	case mods&ui.Shift != 0:
		m.sel = m.multi.Reach(id, m.sel)
	case m.multi.Picks(id):
		m.sel = m.multi.Toggle(id, m.sel)
	default:
		return true
	}
	return false
}

// extendSelection is ⇧↑ and ⇧↓: the range from the anchor grows or shrinks with the keyboard row.
func (m *Main) extendSelection(c *ui.Context, dir int) {
	sel, ok := m.multi.Extend(dir, m.sel)
	if !ok {
		return
	}
	m.sel, m.moved = sel, true
	m.listRef.Focus(c)
}
