// Package selection is the rows a page has selected together: the set, the row a range reaches from,
// and whether the page is in select mode (docs/design/ui-spec.md, Selecting several tasks). It
// decides what a ⌘-click, ⇧-click, ⇧↑ ⇧↓, ⌘A, a plain arrow or a drag does to the set, given the
// order in which the page draws its rows. It is pure: no window, no store. The views ask it
// whether a row is selected and which rows a drag carries, and draw the answer.
//
// The keyboard row (Main.sel) is not part of it; the calls that read or move it take it and give
// back the new one.
package selection

import "slices"

// Selection is the rows selected together on one page. The zero value selects nothing and is not in
// select mode. While it is empty the page behaves as it always did.
type Selection struct {
	order  []int // the rows the page draws, as of the last Sync
	ids    map[int]bool
	anchor int  // the row that was last clicked: ⇧-click and ⇧↑ ⇧↓ reach from it
	mode   bool // select mode (the Select button): a click ticks a row instead of opening it
}

// Sync tells the selection which rows the page draws now, in order, and drops the selected rows that
// are no longer among them (a task that went away, or folded out of sight), and the anchor.
func (s *Selection) Sync(order []int) {
	s.order = order
	if len(s.ids) == 0 {
		return
	}
	keep := make([]int, 0, len(s.ids))
	for _, id := range order {
		if s.ids[id] {
			keep = append(keep, id)
		}
	}
	if len(keep) != len(s.ids) {
		s.set(keep)
	}
	if s.anchor != 0 && !slices.Contains(order, s.anchor) {
		s.anchor = 0
	}
}

// Has reports whether the row is selected.
func (s Selection) Has(id int) bool { return s.ids[id] }

// Count is how many rows are selected.
func (s Selection) Count() int { return len(s.ids) }

// IDs are the selected rows in the order the page draws them.
func (s Selection) IDs() []int {
	var out []int
	for _, id := range s.order {
		if s.ids[id] {
			out = append(out, id)
		}
	}
	return out
}

// Mode reports whether the page is in select mode.
func (s Selection) Mode() bool { return s.mode }

// Active reports whether there is anything to clear: a selection, or select mode.
func (s Selection) Active() bool { return len(s.ids) > 0 || s.mode }

// Bar reports whether the bulk bar is wanted, and with it the bulk actions' keys: two rows are
// selected, or one in select mode.
func (s Selection) Bar() bool { return len(s.ids) >= 2 || (s.mode && len(s.ids) >= 1) }

// EnterMode turns select mode on.
func (s *Selection) EnterMode() { s.mode = true }

// Clear drops the selection and leaves select mode.
func (s *Selection) Clear() { *s = Selection{order: s.order} }

// Deselect drops the selected rows and stays in select mode, if it was on.
func (s *Selection) Deselect() { s.set(nil) }

// Open is a row that was opened: only it is the anchor, nothing is selected, and select mode is over.
func (s *Selection) Open(id int) { *s = Selection{order: s.order, anchor: id} }

// Toggle is a ⌘-click, or a click in select mode, on a row: it ticks or unticks it and becomes the
// anchor. When nothing is ticked yet the keyboard row cursor counts as the first one, as it looks
// selected. It returns the new keyboard row, which is the row.
func (s *Selection) Toggle(id, cursor int) int {
	if len(s.ids) == 0 && cursor != 0 && cursor != id && slices.Contains(s.order, cursor) {
		s.set([]int{cursor})
	}
	next := make(map[int]bool, len(s.ids)+1)
	for k := range s.ids {
		next[k] = true
	}
	if next[id] {
		delete(next, id)
	} else {
		next[id] = true
	}
	s.ids, s.anchor = next, id
	return id
}

// Reach is a ⇧-click on a row: every row from the anchor to it is selected, and it is the new
// keyboard row, which it returns. The range starts at the keyboard row cursor when the anchor is
// not on the page, and at the row itself when neither is.
func (s *Selection) Reach(id, cursor int) int {
	from := s.anchor
	if from == 0 || !slices.Contains(s.order, from) {
		from = cursor
	}
	if from == 0 || !slices.Contains(s.order, from) {
		from = id
	}
	s.set(s.span(from, id))
	s.anchor = from
	return id
}

// Extend is ⇧↑ (dir -1) and ⇧↓ (1): the range from the anchor grows or shrinks with the keyboard
// row cursor, which moves a row. With the cursor on no row it selects the first row, or the last
// for ⇧↑. It returns the new keyboard row, and false when the page has no rows.
func (s *Selection) Extend(dir, cursor int) (int, bool) {
	cur := slices.Index(s.order, cursor)
	if cur < 0 {
		if len(s.order) == 0 {
			return cursor, false
		}
		cur = 0
		if dir < 0 {
			cur = len(s.order) - 1
		}
		s.set([]int{s.order[cur]})
		s.anchor = s.order[cur]
		return s.order[cur], true
	}
	anchor := s.anchor
	if len(s.ids) == 0 || !slices.Contains(s.order, anchor) {
		anchor = cursor
	}
	to := s.order[max(0, min(cur+dir, len(s.order)-1))]
	s.set(s.span(anchor, to))
	s.anchor = anchor
	return to, true
}

// All is ⌘A: every row of the page.
func (s *Selection) All() {
	if len(s.order) == 0 {
		return
	}
	s.set(s.order)
	if s.anchor == 0 {
		s.anchor = s.order[0]
	}
}

// Moved is a plain arrow that moved the keyboard row to cursor (0 for none): it leaves the rows
// ticked together, unless the page is in select mode, where they stay. The row becomes the anchor.
func (s *Selection) Moved(cursor int) {
	if !s.mode {
		s.set(nil)
		s.anchor = 0
	}
	if cursor != 0 {
		s.anchor = cursor
	}
}

// Stack is the rows a drag of the row carries, the one rule for every drag. When the row is one of
// several selected, it is the selected rows in page order that accepts takes (every one, when
// accepts is nil), provided the row is among them and they are at least two; else only the row.
// What the drop target takes is the target's to say: the Agents tray hands over only the user's
// unfinished tasks, a board lane moves every selected card.
func (s Selection) Stack(id int, accepts func(row int) bool) []int {
	if !s.ids[id] || len(s.ids) < 2 {
		return []int{id}
	}
	var out []int
	for _, row := range s.IDs() {
		if accepts == nil || accepts(row) {
			out = append(out, row)
		}
	}
	if len(out) < 2 || !slices.Contains(out, id) {
		return []int{id}
	}
	return out
}

// Dragged is a drag that began on the row: a row that was not selected is taken alone, so the
// selection is dropped.
func (s *Selection) Dragged(id int) {
	if !s.ids[id] && len(s.ids) > 0 {
		s.Clear()
	}
}

// set makes ids the selected rows.
func (s *Selection) set(ids []int) {
	next := make(map[int]bool, len(ids))
	for _, id := range ids {
		next[id] = true
	}
	s.ids = next
}

// span is the rows between two rows of the page, both included, in the order they are drawn; none
// when either is not on the page.
func (s Selection) span(a, b int) []int {
	i, j := slices.Index(s.order, a), slices.Index(s.order, b)
	if i < 0 || j < 0 {
		return nil
	}
	if i > j {
		i, j = j, i
	}
	return s.order[i : j+1]
}
