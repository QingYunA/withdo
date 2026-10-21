package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The user's own tasks have three states and one star (v31 decisions 8 and 9); an agent's keep seven
// statuses and P0 to P3.

func TestAStarIsImportantForATaskOfMineAtP1OrP0(t *testing.T) {
	for p, want := range map[data.Priority]bool{data.P0: true, data.P1: true, data.P2: false, data.P3: false} {
		if got := starred(data.Task{Priority: p, Source: data.SourceUser}); got != want {
			t.Errorf("%s starred = %v, want %v", p, got, want)
		}
	}
	if importantPriority(true) != data.P1 || importantPriority(false) != data.P2 {
		t.Error("the star writes P1 when set and P2 when not")
	}
	if !showsPriorityMark(data.Task{Source: data.SourceUser, Priority: data.P1}) || showsPriorityMark(data.Task{Source: data.SourceUser, Priority: data.P3}) {
		t.Error("a task of mine shows the star when important and nothing else")
	}
	if !showsPriorityMark(data.Task{Source: data.SourceAgent, Priority: data.P0}) || showsPriorityMark(data.Task{Source: data.SourceAgent, Priority: data.P2}) {
		t.Error("an agent's task keeps its P0 and P1 badges")
	}
}

func TestThePriorityActionIsTheStarOnlyWhenEverySelectedTaskIsMine(t *testing.T) {
	mineTasks := []data.Task{mkTask(1, data.Todo, data.SourceUser), {ID: 2, Status: data.Todo, Source: data.SourceUser, Priority: data.P1}}
	a := actionOf(mineTasks, actPriority)
	if !a.Star || a.Starred || a.Page != pageMain || len(a.Ids) != 1 || a.Ids[0] != 1 {
		t.Errorf("mine: %+v, want a star that stars task 1 and runs at once", a)
	}
	all := []data.Task{{ID: 2, Source: data.SourceUser, Priority: data.P1}, {ID: 3, Source: data.SourceUser, Priority: data.P0}}
	if a := actionOf(all, actPriority); !a.Star || !a.Starred || len(a.Ids) != 2 {
		t.Errorf("all starred: %+v, want a star that takes both off", a)
	}
	mixed := []data.Task{mkTask(1, data.Todo, data.SourceUser), mkTask(2, data.Todo, data.SourceAgent)}
	if a := actionOf(mixed, actPriority); a.Star || a.Page != pagePriority {
		t.Errorf("mixed: %+v, want the four priorities", a)
	}
}

func TestTheInspectorOfATaskOfMineHasThreeStatesChipsAndNoAgentRows(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.OpenTask(5) // Write the release notes
	tt.Frame()
	has(t, tt, "Status", "To do", "Doing", "Done", "When", "Where", "Hand to an agent…", "Important", "Due", "Remind")
	hasNot(t, tt, "Priority", "Branch", "Pull request", "Activity", "Milestone")
	if !above(t, tt, "Status", "When") || !above(t, tt, "When", "Where") || !above(t, tt, "Where", "Hand to an agent…") {
		t.Errorf("Status, When, Where, then Hand to an agent…: %q", tt.Texts())
	}
	m.OpenTask(3) // an agent's task
	tt.Frame()
	has(t, tt, "Priority", "Branch", "Activity")
	hasNot(t, tt, "When", "Where", "Hand to an agent…")
}

func TestTheStatusSegmentsSetTheStatusOfATaskOfMine(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(5)
	tt.Frame()
	click(t, tt, "Doing")
	if got, _ := app.Task(5); got.Status != data.Doing {
		t.Errorf("status %v, want doing", got.Status)
	}
	click(t, tt, "To do")
	if got, _ := app.Task(5); got.Status != data.Todo {
		t.Errorf("status %v, want to do", got.Status)
	}
}

func TestTheStarBesideTheTitleTogglesImportant(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(5)
	tt.Frame()
	click(t, tt, "Important")
	if got, _ := app.Task(5); got.Priority != data.P1 {
		t.Errorf("priority %v, want P1", got.Priority)
	}
	click(t, tt, "Important")
	if got, _ := app.Task(5); got.Priority != data.P2 {
		t.Errorf("priority %v, want P2", got.Priority)
	}
	_ = app.Update(5, data.Patch{Priority: data.PriorityPtr(data.P0)}) // a P0 task of mine reads as starred
	tt.Frame()
	click(t, tt, "Important")
	if got, _ := app.Task(5); got.Priority != data.P2 {
		t.Errorf("a P0 task is %v after the star was clicked, want P2", got.Priority)
	}
}

func TestTheTaskMenuOfATaskOfMineHasTheStarInPlaceOfThePriorityStrip(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	rightClick(t, tt, "Reply to design feedback")
	has(t, tt, "Mark important", "Set status to Cancelled", "Set status to Backlog")
	hasNot(t, tt, "Set priority to Urgent", "P0")
	click(t, tt, "Mark important")
	if got := taskTitled(t, app, "Reply to design feedback").Priority; got != data.P1 {
		t.Errorf("priority %v, want P1", got)
	}
	rightClick(t, tt, "Reply to design feedback")
	click(t, tt, "Not important")
	if got := taskTitled(t, app, "Reply to design feedback").Priority; got != data.P2 {
		t.Errorf("priority %v, want P2", got)
	}
}

func TestPTogglesTheStarOfASelectedTaskOfMine(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.sel = 7
	tt.Frame()
	tt.Key(0, ui.KeyP)
	tt.Frame()
	if m.pop.open {
		t.Error("P opened a page for a task of mine")
	}
	if got, _ := app.Task(7); got.Priority != data.P1 {
		t.Errorf("priority %v, want P1", got.Priority)
	}
}

func TestTheBulkBarStarsTheSelectedTasksOfMineWithOneUndo(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	click(t, tt, "Mark important")
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if got := taskTitled(t, app, title).Priority; got != data.P1 {
			t.Errorf("%q is %s, want P1", title, got)
		}
	}
	has(t, tt, "Marked 2 tasks important")
	tt.Key(ui.Cmd, ui.KeyZ)
	for _, title := range []string{"Reply to design feedback", "Book dentist"} {
		if got := taskTitled(t, app, title).Priority; got != data.P2 {
			t.Errorf("%q is %s after ⌘Z, want P2", title, got)
		}
	}
}

func TestTheBulkBarOfAMixedSelectionKeepsPriority(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Fix tray race on launch")
	tt.Frame()
	has(t, tt, "Priority")
	hasNot(t, tt, "Mark important")
}

func TestHandToAnAgentFromTheInspectorHandsTheTaskOver(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(5)
	tt.Frame()
	click(t, tt, "Hand to an agent…")
	if got, _ := app.Task(5); got.Source != data.SourceAgent {
		t.Errorf("source %v, want the task handed over", got.Source)
	}
	has(t, tt, "Activity") // it is an agent's task now
	hasNot(t, tt, "Hand to an agent…")
}
