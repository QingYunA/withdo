package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// These tests press real keys in a focused input: the input's OnTextKey takes Enter, Esc and
// Down, the window shortcuts stay out of its way.

func TestDownFromTheComposerWalksIntoTheRows(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	first := m.sel
	if first == 0 {
		t.Fatal("Down in the composer did not select a row")
	}
	tt.Key(0, ui.KeyDown)
	if m.sel == first {
		t.Error("the second Down did not move on: the keys stayed in the field")
	}
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyUp)
	if m.sel != 0 {
		t.Errorf("Up from the first row left %d selected", m.sel)
	}
}

func TestTypingSpaceInTheComposerStaysInTheField(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.KeyK)
	m.sel = 5 // a highlighted user task: Space on the page would complete it
	tt.Type("a b")
	if m.cmp[surfacePage].buf != "a b" {
		t.Errorf("field holds %q", m.cmp[surfacePage].buf)
	}
}

func TestEnterInTheComposerSendsAndEmptiesIt(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Type("Water the plants")
	r.tt.Key(0, ui.KeyEnter)
	if r.pageBuf() != "" || r.m.openID != 0 {
		t.Errorf("field %q, opened %d", r.pageBuf(), r.m.openID)
	}
	r.tt.Type("Feed the cat")
	r.tt.Key(0, ui.KeyEnter)
	titles := map[string]bool{}
	for _, task := range r.app.snap.tasks {
		titles[task.Title] = true
	}
	if !titles["Water the plants"] || !titles["Feed the cat"] {
		t.Errorf("tasks %v: the second line was not typed into an empty field", titles)
	}
}

func TestEscRevertsAnInspectorFieldThenClosesThePane(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(5)
	tt.Frame()
	if err := tt.Click("Title"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("A different title")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if m.openID != 5 {
		t.Fatal("the first Esc closed the pane; it should only revert the field")
	}
	if got, _ := app.Task(5); got.Title != "Write the release notes" {
		t.Errorf("title %q was saved by Esc", got.Title)
	}
	if !tt.HasText("Write the release notes") {
		t.Errorf("the field does not show the stored title: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if m.openID != 0 {
		t.Error("the second Esc did not close the pane")
	}
}

func TestEnterInTheNotesBreaksALineAndDoesNotSave(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(7)
	tt.Frame()
	if err := tt.Click("Notes"); err != nil {
		t.Fatal(err)
	}
	tt.Type("one")
	tt.Key(0, ui.KeyEnter)
	tt.Type("two")
	if got, _ := app.Task(7); got.Notes != "" {
		t.Fatalf("Enter saved the notes: %q", got.Notes)
	}
	if err := tt.Click("Title"); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.Task(7); got.Notes != "one\ntwo" {
		t.Errorf("notes %q", got.Notes)
	}
}
