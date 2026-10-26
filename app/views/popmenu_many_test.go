package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestRightClickOnASelectedRowOpensTheMenuOfAllSelectedRows(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if err := tt.RightClick("Book dentist"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"2 tasks", "Mark done", "Cancel", "Move to…", "Mark important", "Hand to an agent", "Delete 2…"} {
		if !tt.HasText(want) {
			t.Errorf("the menu lacks %q in %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if got := taskTitled(t, app, title).Status; got != data.Cancelled {
			t.Errorf("%q is %s, want cancelled", title, got)
		}
	}
	if !tt.HasText("Cancelled 2 tasks") {
		t.Errorf("no toast in %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := taskTitled(t, app, "Book dentist").Status; got == data.Cancelled {
		t.Error("⌘Z did not undo the cancel")
	}
}

func TestRightClickOnAnUnselectedRowActsOnThatRowAndKeepsTheSelection(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if err := tt.RightClick("Write the release notes"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("2 tasks") {
		t.Errorf("the selection's menu opened on an unselected row: %q", tt.Texts())
	}
	if !tt.HasText("Remind me…") {
		t.Errorf("no single-task menu in %q", tt.Texts())
	}
	wantPicked(t, m, "Reply to design feedback", "Book dentist")
}

func TestPAndMOpenTheSelectionsPagesAndCmdBackspaceAsksBeforeDeleting(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	cmdClick(t, tt, "Fix tray race on launch", "Native UI with mygo") // agents' tasks keep the four priorities
	tt.Key(0, ui.KeyP)
	tt.Frame()
	if !tt.HasText("2 tasks") || !tt.HasText("Set priority to Urgent") {
		t.Errorf("P did not open the selection's priority page: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyBackspace)
	tt.Frame()
	if !tt.HasText("Delete 2 tasks?") {
		t.Errorf("⌘⌫ did not ask: %q", tt.Texts())
	}
}
