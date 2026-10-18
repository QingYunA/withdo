package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// taskTitled finds a task of the app by its title.
func taskTitled(t *testing.T, app *App, title string) data.Task {
	t.Helper()
	for _, task := range app.snap.tasks {
		if task.Title == title {
			return task
		}
	}
	t.Fatalf("no task %q", title)
	return data.Task{}
}

func cmdClick(t *testing.T, tt *ui.Tester, titles ...string) {
	t.Helper()
	for _, title := range titles {
		if err := tt.ClickWith(ui.Cmd, title); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCmdReturnMarksTheSelectionDoneAsOneChangeWithOneUndo(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Key(ui.Cmd, ui.KeyEnter)
	tt.Frame()
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if got := taskTitled(t, app, title).Status; got != data.Done {
			t.Errorf("%q is %s, want done", title, got)
		}
	}
	if !tt.HasText("Marked 2 tasks done") {
		t.Errorf("no toast in %q", tt.Texts())
	}
	if m.multi.active() {
		t.Error("the page is still selecting after the action")
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if got := taskTitled(t, app, title).Status; got == data.Done {
			t.Errorf("%q is still done after ⌘Z", title)
		}
	}
}

func TestCmdShiftAHandsOnlyMyUnfinishedTasksAndCountsTheRest(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist", "Native UI with mygo")
	tt.Key(ui.Cmd|ui.Shift, ui.KeyA)
	tt.Frame()
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		task := taskTitled(t, app, title)
		if task.Source != data.SourceAgent || task.HandedTo != data.AnyAgent {
			t.Errorf("%q = %+v, want handed over", title, task)
		}
	}
	if got := taskTitled(t, app, "Native UI with mygo"); got.HandedTo != "" {
		t.Errorf("the agent's own task was handed over: %+v", got)
	}
	if !tt.HasText("Handed 2 to an agent · 1 skipped") {
		t.Errorf("no toast in %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if task := taskTitled(t, app, title); task.Source != data.SourceUser || task.HandedTo != "" {
			t.Errorf("%q = %+v after ⌘Z, want the user's again", title, task)
		}
	}
}

func TestHandingOnlyAgentTasksSaysNothingCanBeHanded(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Fix tray race on launch", "Native UI with mygo")
	tt.Key(ui.Cmd|ui.Shift, ui.KeyA)
	tt.Frame()
	if !tt.HasText("Only your own unfinished tasks can be handed to an agent") {
		t.Errorf("no toast in %q", tt.Texts())
	}
	if got := taskTitled(t, app, "Fix tray race on launch"); got.HandedTo != "" {
		t.Error("an agent's task was handed over")
	}
}

func TestTheBulkKeysDoNothingWithOneRowSelected(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Book dentist")
	tt.Key(ui.Cmd, ui.KeyEnter)
	if got := taskTitled(t, app, "Book dentist").Status; got == data.Done {
		t.Error("⌘↵ finished a single selected row: the bar is not up")
	}
}
