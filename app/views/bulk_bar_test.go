package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestTheBulkBarAppearsWithTwoRowsSelectedAndLeavesWithOne(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	cmdClick(t, tt, "Book dentist")
	tt.Frame()
	if tt.HasText("1 selected") {
		t.Error("the bar is up with one row selected outside select mode")
	}
	cmdClick(t, tt, "Reply to design feedback")
	tt.Frame()
	for _, want := range []string{"2 selected", "Move to…", "Mark important", "Hand to an agent", "Delete"} {
		if !tt.HasText(want) {
			t.Errorf("the bar lacks %q in %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Clear selection"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("2 selected") {
		t.Error("× did not clear the selection")
	}
}

func TestTheBarIsUpWithOneRowInSelectMode(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	if err := tt.Click("Select"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Write the release notes"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Hand to an agent") {
		t.Errorf("no bar in %q", tt.Texts())
	}
}

func TestTheBarsDoneButtonFinishesTheSelection(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if err := tt.Click("Done"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	done := 0
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if taskTitled(t, app, title).Status == data.Done {
			done++
		}
	}
	if done != 2 || m.multi.Active() {
		t.Errorf("%d of 2 done, selecting %v: the bar's Done finishes them and the page stops selecting", done, m.multi.Active())
	}
}

func TestTheBarsPriorityChoosesForEverySelectedRowAndUndoPutsThemBack(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Fix tray race on launch", "Native UI with mygo")
	tt.Frame()
	before := map[string]data.Priority{}
	for _, title := range []string{"Fix tray race on launch", "Native UI with mygo"} {
		before[title] = taskTitled(t, app, title).Priority
	}
	if err := tt.Click("Priority"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Set priority to Low"); err != nil {
		t.Fatalf("%v in %q", err, tt.Texts())
	}
	tt.Frame()
	for title := range before {
		if got := taskTitled(t, app, title).Priority; got != data.P3 {
			t.Errorf("%q is %s, want P3", title, got)
		}
	}
	if !tt.HasText("Set 2 tasks to P3") {
		t.Errorf("no toast in %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	for title, want := range before {
		if got := taskTitled(t, app, title).Priority; got != want {
			t.Errorf("%q is %s after ⌘Z, want %s", title, got, want)
		}
	}
}

func TestTheBarsMoveToMovesEverySelectedRowToTheProject(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if err := tt.Click("Move to…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Move to picpocket"); err != nil {
		t.Fatalf("%v in %q", err, tt.Texts())
	}
	tt.Frame()
	pic := 0
	for _, p := range app.snap.projects {
		if p.Name == "picpocket" {
			pic = p.ID
		}
	}
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if got := taskTitled(t, app, title); got.ProjectID == nil || *got.ProjectID != pic {
			t.Errorf("%q is in project %v, want %d", title, got.ProjectID, pic)
		}
	}
	if !tt.HasText("Moved 2 tasks to picpocket") {
		t.Errorf("no toast in %q", tt.Texts())
	}
}

func TestTheBarsDeleteAsksFirstThenDeletesAllWithOneUndo(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Delete 2 tasks?") {
		t.Fatalf("no question in %q", tt.Texts())
	}
	if got := taskTitled(t, app, "Book dentist"); got.ID == 0 {
		t.Fatal("deleted before the answer")
	}
	tt.Key(0, ui.KeyTab) // Cancel has the focus; the sheet's Delete is next
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		for _, task := range app.snap.tasks {
			if task.Title == title {
				t.Errorf("%q is still there", title)
			}
		}
	}
	if !tt.HasText("Deleted 2 tasks") {
		t.Errorf("no toast in %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	taskTitled(t, app, "Reply to design feedback")
	taskTitled(t, app, "Book dentist")
}

func TestTheBarsHandOffButtonHandsTheSelectionOver(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if err := tt.Click("Hand to an agent"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if got := taskTitled(t, app, title); got.HandedTo != data.AnyAgent {
			t.Errorf("%q = %+v, want handed over", title, got)
		}
	}
	if !tt.HasText("Handed 2 tasks to an agent") {
		t.Errorf("no toast in %q", tt.Texts())
	}
}

func TestTheBarKeepsItsButtonsByLabelInANarrowWindow(t *testing.T) {
	tt, _, _ := newMainTester(t, 700)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if !tt.HasText("2 selected") {
		t.Fatalf("no bar in %q", tt.Texts())
	}
	if err := tt.Click("Hand to an agent"); err != nil {
		t.Errorf("the icon button lost its label: %v", err)
	}
}
