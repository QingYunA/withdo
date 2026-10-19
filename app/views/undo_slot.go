package views

import "time"

// The one rule of Undo: every change that shows an Undo toast also answers ⌘Z. A change hands the
// window one undoEntry (offerUndo); the window keeps the newest in its slot, shows it in the one
// toast and takes it back on the toast's button or ⌘Z. This file is the pure part, the glue with
// the toast and the keys is undo.go.

// undoWithin is how long a toast offers Undo.
const undoWithin = 6 * time.Second

// undoEntry is one change the user can take back.
type undoEntry struct {
	title string // what the toast says happened
	extra any    // data for the toast's look, such as the Copy for the agent button of a hand-off
	undo  func() // reverses the change
}

// undoSlot holds the entry that is on offer: the newest, for as long as its toast shows.
type undoSlot struct {
	cur *undoEntry
	seq int
}

// offer puts e in the slot, replacing the one before, and returns the ticket its toast hands back
// to closed.
func (s *undoSlot) offer(e undoEntry) int {
	s.seq++
	s.cur = &e
	return s.seq
}

// closed says the toast of ticket is gone: the entry expires, unless a newer one has replaced it.
func (s *undoSlot) closed(ticket int) {
	if s.seq == ticket {
		s.cur = nil
	}
}

// clear drops the entry, whichever it is.
func (s *undoSlot) clear() { s.cur = nil }

// pickUndo is the entry ⌘Z takes: the first of entries that is not nil, none when a text field
// would take the key for its own undo. The assistant's last change comes before the toast's.
func pickUndo(textFree bool, entries ...*undoEntry) *undoEntry {
	if !textFree {
		return nil
	}
	for _, e := range entries {
		if e != nil {
			return e
		}
	}
	return nil
}
