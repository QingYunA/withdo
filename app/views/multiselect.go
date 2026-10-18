package views

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// Selecting several tasks (docs/design/ui-spec.md, Selecting several tasks). The keyboard row (m.sel)
// is one row; the set below is the rows the user ticked together. While the set is empty the page
// behaves as it always did, and the keyboard row is the only one that is selected. The set belongs
// to one page: it is cleared when the page changes, and held to the rows the page draws.

// multiSel is the rows selected together, the row a range starts from, and whether the page is in
// select mode (the Select button): there a click ticks a row instead of opening it.
type multiSel struct {
	ids    map[int]bool
	anchor int  // the row that was last clicked: ⇧-click and ⇧↑ ⇧↓ reach from it
	mode   bool // select mode
}

// count is how many rows are selected.
func (s multiSel) count() int { return len(s.ids) }

// active reports whether there is anything to clear: a selection, or select mode.
func (s multiSel) active() bool { return len(s.ids) > 0 || s.mode }

// has reports whether the row is selected.
func (s multiSel) has(id int) bool { return s.ids[id] }

// with returns the set with ids as its members.
func (s multiSel) with(ids ...int) multiSel {
	next := make(map[int]bool, len(ids))
	for _, id := range ids {
		next[id] = true
	}
	s.ids = next
	return s
}

// toggled returns the set with id added, or removed when it was in.
func (s multiSel) toggled(id int) multiSel {
	next := make(map[int]bool, len(s.ids)+1)
	for k := range s.ids {
		next[k] = true
	}
	if next[id] {
		delete(next, id)
	} else {
		next[id] = true
	}
	s.ids = next
	return s
}

// clearMulti drops the selection and leaves select mode.
func (m *Main) clearMulti() { m.multi = multiSel{} }

// pruneMulti keeps only the rows the page draws, so a task that went away, or folded out of sight,
// is no longer counted.
func (m *Main) pruneMulti(order []int) {
	if len(m.multi.ids) == 0 {
		return
	}
	var keep []int
	for _, id := range order {
		if m.multi.ids[id] {
			keep = append(keep, id)
		}
	}
	if len(keep) != len(m.multi.ids) {
		m.multi = m.multi.with(keep...)
	}
	if m.multi.anchor != 0 && !slices.Contains(order, m.multi.anchor) {
		m.multi.anchor = 0
	}
}

// selectedRows are the selected rows in the order the page draws them.
func (m *Main) selectedRows() []int {
	var out []int
	for _, id := range m.order {
		if m.multi.ids[id] {
			out = append(out, id)
		}
	}
	return out
}

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
		m.toggleRow(id)
	case mods&ui.Shift != 0:
		m.reachTo(id)
	case m.multi.mode:
		m.toggleRow(id)
	default:
		return true
	}
	return false
}

// toggleRow ticks or unticks a row. When nothing is ticked yet, the row the keyboard stands on
// counts as the first one, as it looks selected.
func (m *Main) toggleRow(id int) {
	if len(m.multi.ids) == 0 && m.sel != 0 && m.sel != id && slices.Contains(m.order, m.sel) {
		m.multi = m.multi.with(m.sel)
	}
	m.multi = m.multi.toggled(id)
	m.multi.anchor, m.sel = id, id
}

// reachTo selects every row from the anchor to this one, and moves the keyboard here.
func (m *Main) reachTo(id int) {
	from := m.multi.anchor
	if from == 0 || !slices.Contains(m.order, from) {
		from = m.sel
	}
	if from == 0 || !slices.Contains(m.order, from) {
		from = id
	}
	m.multi = m.multi.with(m.span(from, id)...)
	m.multi.anchor, m.sel = from, id
}

// span is the rows between two rows of the page, both included, in the order they are drawn.
func (m *Main) span(a, b int) []int {
	i, j := slices.Index(m.order, a), slices.Index(m.order, b)
	if i < 0 || j < 0 {
		return nil
	}
	if i > j {
		i, j = j, i
	}
	return slices.Clone(m.order[i : j+1])
}

// extendSelection is ⇧↑ and ⇧↓: the range from the anchor grows or shrinks with the keyboard row.
func (m *Main) extendSelection(c *ui.Context, dir int) {
	cur := slices.Index(m.order, m.sel)
	if cur < 0 {
		if len(m.order) == 0 {
			return
		}
		cur = 0
		if dir < 0 {
			cur = len(m.order) - 1
		}
		m.sel = m.order[cur]
		m.multi = m.multi.with(m.sel)
		m.multi.anchor, m.moved = m.sel, true
		m.listRef.Focus(c)
		return
	}
	anchor := m.multi.anchor
	if len(m.multi.ids) == 0 || !slices.Contains(m.order, anchor) {
		anchor = m.sel
	}
	next := max(0, min(cur+dir, len(m.order)-1))
	m.multi = m.multi.with(m.span(anchor, m.order[next])...)
	m.multi.anchor, m.sel, m.moved = anchor, m.order[next], true
	m.listRef.Focus(c)
}

// selectAll is ⌘A: every row of the page.
func (m *Main) selectAll() {
	if len(m.order) == 0 {
		return
	}
	m.multi = m.multi.with(m.order...)
	if m.multi.anchor == 0 {
		m.multi.anchor = m.order[0]
	}
}

// nudgeSelectButton is made in the next commit: the Select button flashes on the first ⌘-click.
func (m *Main) nudgeSelectButton() {}
