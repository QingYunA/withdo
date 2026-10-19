package selection

import (
	"slices"
	"testing"
)

// page is a Selection on a page that draws rows 10 to 60.
func page() *Selection {
	s := &Selection{}
	s.Sync([]int{10, 20, 30, 40, 50, 60})
	return s
}

func wantIDs(t *testing.T, s *Selection, want ...int) {
	t.Helper()
	if got := s.IDs(); !slices.Equal(got, want) {
		t.Errorf("selected %v, want %v", got, want)
	}
}

func TestToggleCountsTheKeyboardRowAsTheFirst(t *testing.T) {
	s := page()
	if cur := s.Toggle(30, 10); cur != 30 {
		t.Errorf("keyboard row %d, want 30", cur)
	}
	wantIDs(t, s, 10, 30)
	s.Toggle(30, 30)
	wantIDs(t, s, 10)
	s.Toggle(10, 10)
	wantIDs(t, s)
	if s.Active() {
		t.Error("an emptied selection is still active")
	}
}

func TestToggleIgnoresAKeyboardRowOffThePage(t *testing.T) {
	s := page()
	s.Toggle(30, 99)
	wantIDs(t, s, 30)
}

func TestReachSelectsFromTheAnchor(t *testing.T) {
	s := page()
	s.Toggle(20, 0)
	if cur := s.Reach(50, 20); cur != 50 {
		t.Errorf("keyboard row %d, want 50", cur)
	}
	wantIDs(t, s, 20, 30, 40, 50)
	s.Reach(10, 50) // backwards, still from the anchor
	wantIDs(t, s, 10, 20)
}

func TestReachFallsBackToTheKeyboardRowThenToItself(t *testing.T) {
	s := page()
	s.Reach(40, 20)
	wantIDs(t, s, 20, 30, 40)
	s = page()
	s.Reach(40, 0)
	wantIDs(t, s, 40)
}

func TestExtendGrowsAndShrinksFromTheAnchor(t *testing.T) {
	s := page()
	cur, ok := s.Extend(1, 20) // nothing ticked: the anchor is the keyboard row
	if !ok || cur != 30 {
		t.Fatalf("Extend ↓ gave %d, %v", cur, ok)
	}
	wantIDs(t, s, 20, 30)
	cur, _ = s.Extend(1, cur)
	wantIDs(t, s, 20, 30, 40)
	cur, _ = s.Extend(-1, cur)
	cur, _ = s.Extend(-1, cur)
	wantIDs(t, s, 20)
	cur, _ = s.Extend(-1, cur)
	if cur != 10 {
		t.Errorf("keyboard row %d, want 10", cur)
	}
	wantIDs(t, s, 10, 20)
}

func TestExtendStopsAtTheEnds(t *testing.T) {
	s := page()
	cur, _ := s.Extend(1, 60)
	if cur != 60 {
		t.Errorf("keyboard row %d, want 60", cur)
	}
	wantIDs(t, s, 60)
}

func TestExtendWithNoKeyboardRowTakesTheFirstOrLast(t *testing.T) {
	s := page()
	if cur, ok := s.Extend(1, 0); !ok || cur != 10 {
		t.Errorf("⇧↓ gave %d, %v, want 10", cur, ok)
	}
	wantIDs(t, s, 10)
	s = page()
	if cur, ok := s.Extend(-1, 0); !ok || cur != 60 {
		t.Errorf("⇧↑ gave %d, %v, want 60", cur, ok)
	}
	empty := &Selection{}
	if _, ok := empty.Extend(1, 0); ok {
		t.Error("an empty page extended")
	}
}

func TestAllSelectsThePageInOrder(t *testing.T) {
	s := page()
	s.All()
	wantIDs(t, s, 10, 20, 30, 40, 50, 60)
	s.Reach(30, 0) // the anchor became the first row
	wantIDs(t, s, 10, 20, 30)
}

func TestSyncDropsRowsThatLeftThePage(t *testing.T) {
	s := page()
	s.All()
	s.Sync([]int{20, 40, 99})
	wantIDs(t, s, 20, 40)
	if s.Has(10) || s.Count() != 2 {
		t.Errorf("%d selected, Has(10) %v", s.Count(), s.Has(10))
	}
	s.Toggle(40, 0)
	s.Sync([]int{20})
	s.Reach(20, 0) // the anchor, 40, left: the range starts at the row itself
	wantIDs(t, s, 20)
}

func TestMovedClearsUnlessInSelectMode(t *testing.T) {
	s := page()
	s.All()
	s.Moved(30)
	if s.Count() != 0 {
		t.Errorf("a plain arrow left %d selected", s.Count())
	}
	s.Extend(1, 30) // the anchor is the row the arrow moved to
	wantIDs(t, s, 30, 40)

	s = page()
	s.EnterMode()
	s.Toggle(20, 0)
	s.Toggle(30, 20)
	s.Moved(40)
	wantIDs(t, s, 20, 30)
	if !s.Mode() {
		t.Error("an arrow left select mode")
	}
	s.Reach(60, 40)
	wantIDs(t, s, 40, 50, 60)
}

func TestClearOpenAndDeselect(t *testing.T) {
	s := page()
	s.EnterMode()
	s.Toggle(20, 0)
	s.Deselect()
	if s.Count() != 0 || !s.Mode() {
		t.Errorf("Deselect left %d selected, mode %v", s.Count(), s.Mode())
	}
	s.Toggle(20, 0)
	s.Clear()
	if s.Active() {
		t.Error("Clear left the selection active")
	}
	s.Toggle(20, 0)
	s.EnterMode()
	s.Open(30)
	if s.Active() {
		t.Error("opening a row left the selection active")
	}
	s.Extend(1, 30)
	wantIDs(t, s, 30, 40)
	s.Clear()
	s.All()
	wantIDs(t, s, 10, 20, 30, 40, 50, 60) // Clear keeps the page's rows
}

func TestBar(t *testing.T) {
	s := page()
	s.Toggle(10, 0)
	if s.Bar() {
		t.Error("the bar is up for one row")
	}
	s.EnterMode()
	if !s.Bar() {
		t.Error("the bar is down for one row in select mode")
	}
	s.Deselect()
	if s.Bar() {
		t.Error("the bar is up for no rows")
	}
	s = page()
	s.Toggle(10, 0)
	s.Toggle(20, 10)
	if !s.Bar() {
		t.Error("the bar is down for two rows")
	}
}

func TestStack(t *testing.T) {
	s := page()
	s.Toggle(50, 0)
	s.Toggle(20, 50)
	s.Toggle(40, 20)
	if got := s.Stack(20, nil); !slices.Equal(got, []int{20, 40, 50}) {
		t.Errorf("stack %v, want the selected rows in page order", got)
	}
	if got := s.Stack(30, nil); !slices.Equal(got, []int{30}) {
		t.Errorf("a row that is not selected carried %v", got)
	}
	odd := func(row int) bool { return row != 40 }
	if got := s.Stack(50, odd); !slices.Equal(got, []int{20, 50}) {
		t.Errorf("stack %v, want the rows the target accepts", got)
	}
	only20 := func(row int) bool { return row == 20 }
	if got := s.Stack(20, only20); !slices.Equal(got, []int{20}) {
		t.Errorf("stack %v: fewer than two accepted is the row alone", got)
	}
	if got := s.Stack(40, odd); !slices.Equal(got, []int{40}) {
		t.Errorf("stack %v: a row the target does not accept is carried alone", got)
	}
	one := page()
	one.Toggle(20, 0)
	if got := one.Stack(20, nil); !slices.Equal(got, []int{20}) {
		t.Errorf("one selected row carried %v", got)
	}
}

func TestDraggedDropsTheSelectionOnlyForAnOutsider(t *testing.T) {
	s := page()
	s.Toggle(10, 0)
	s.Toggle(20, 10)
	s.Dragged(20)
	if s.Count() != 2 {
		t.Errorf("dragging a selected row left %d selected", s.Count())
	}
	s.Dragged(30)
	if s.Active() {
		t.Error("dragging a row outside the selection kept it")
	}
}
