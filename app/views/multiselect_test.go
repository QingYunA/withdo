package views

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// pickedTitles are the titles of the rows selected together, in page order.
func pickedTitles(m *Main) []string {
	var out []string
	for _, id := range m.multi.IDs() {
		if task, ok := m.app.Task(id); ok {
			out = append(out, task.Title)
		}
	}
	return out
}

func wantPicked(t *testing.T, m *Main, want ...string) {
	t.Helper()
	if got := pickedTitles(m); !slices.Equal(got, want) {
		t.Errorf("selected %q, want %q", got, want)
	}
}

func TestCmdClickAddsAndRemovesRowsWithoutOpeningTheInspector(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if err := tt.ClickWith(ui.Cmd, title); err != nil {
			t.Fatal(err)
		}
	}
	wantPicked(t, m, "Reply to design feedback", "Book dentist")
	if m.openID != 0 {
		t.Error("a ⌘-click opened the inspector")
	}
	if err := tt.ClickWith(ui.Cmd, "Book dentist"); err != nil {
		t.Fatal(err)
	}
	wantPicked(t, m, "Reply to design feedback")
}

func TestShiftClickSelectsFromTheLastClickedRow(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.ClickWith(ui.Cmd, "Reply to design feedback"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ClickWith(ui.Shift, "Book dentist"); err != nil {
		t.Fatal(err)
	}
	wantPicked(t, m, "Reply to design feedback", "Decide on the activation policy", "Book dentist")
}

func TestShiftArrowsExtendFromTheKeyboardRow(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	for range 3 { // Fix tray race, Native UI, Reply to design feedback
		tt.Key(0, ui.KeyDown)
	}
	tt.Key(ui.Shift, ui.KeyDown)
	tt.Key(ui.Shift, ui.KeyDown)
	wantPicked(t, m, "Reply to design feedback", "Decide on the activation policy", "Book dentist")
	tt.Key(ui.Shift, ui.KeyUp)
	wantPicked(t, m, "Reply to design feedback", "Decide on the activation policy")
	tt.Key(0, ui.KeyDown)
	if n := m.multi.Count(); n != 0 {
		t.Errorf("a plain arrow left %d rows selected", n)
	}
}

func TestCmdASelectsEveryRowAndEscClearsThemFirst(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.KeyA)
	if got, want := m.multi.Count(), len(m.rowIDs()); got != want || got < 8 {
		t.Fatalf("⌘A selected %d of %d rows", got, want)
	}
	if err := tt.Click("Book dentist"); err != nil { // a plain click opens it
		t.Fatal(err)
	}
	if m.openID == 0 || m.multi.Count() != 0 {
		t.Errorf("a plain click left openID %d and %d selected", m.openID, m.multi.Count())
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyEscape)
	if m.multi.Count() != 0 || m.openID == 0 {
		t.Errorf("Esc: %d selected, openID %d: it clears the selection first and leaves the inspector", m.multi.Count(), m.openID)
	}
	tt.Key(0, ui.KeyEscape)
	if m.openID != 0 {
		t.Error("the second Esc did not close the inspector")
	}
}

func TestTheSelectionIsClearedWhenThePageChanges(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.Key3)
	if m.multi.Count() != 0 {
		t.Error("the selection followed the user to the Inbox")
	}
}
