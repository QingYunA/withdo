package views

import "testing"

func TestUndoSlotKeepsTheNewestEntryUntilItsToastCloses(t *testing.T) {
	var s undoSlot
	first := s.offer(undoEntry{title: "first"})
	second := s.offer(undoEntry{title: "second"})
	if s.cur == nil || s.cur.title != "second" {
		t.Fatalf("slot holds %+v, want the newest", s.cur)
	}
	s.closed(first) // the replaced toast closing must not drop the newer entry
	if s.cur == nil {
		t.Fatal("an old toast closing dropped the new entry")
	}
	s.closed(second)
	if s.cur != nil {
		t.Error("the entry outlived its toast")
	}
}

func TestUndoSlotClearDropsTheEntry(t *testing.T) {
	var s undoSlot
	s.offer(undoEntry{title: "x"})
	s.clear()
	if s.cur != nil {
		t.Error("clear left the entry")
	}
}

func TestPickUndoTakesTheFirstEntryAndLeavesTheKeyToTextFields(t *testing.T) {
	assistant, toast := &undoEntry{title: "assistant"}, &undoEntry{title: "toast"}
	for _, c := range []struct {
		name     string
		textFree bool
		in       []*undoEntry
		want     *undoEntry
	}{
		{"assistant first", true, []*undoEntry{assistant, toast}, assistant},
		{"toast when the assistant has none", true, []*undoEntry{nil, toast}, toast},
		{"nothing to undo", true, []*undoEntry{nil, nil}, nil},
		{"a text field keeps the key", false, []*undoEntry{assistant, toast}, nil},
		{"no entries", true, nil, nil},
	} {
		if got := pickUndo(c.textFree, c.in...); got != c.want {
			t.Errorf("%s: picked %+v, want %+v", c.name, got, c.want)
		}
	}
}
