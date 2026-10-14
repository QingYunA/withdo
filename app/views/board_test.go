package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func boardTester(t *testing.T, app *App, project, width int) (*ui.Tester, *Main) {
	t.Helper()
	tt, m, _ := newTesterFor(app, width)
	m.goTo(nav{screen: screenProject, projectID: project})
	m.board[project] = true
	tt.Frame()
	return tt, m
}

func TestBoardShowsOnlyNonEmptyColumnsPlusToDo(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100) // answer-me-with-html: one doing, one backlog
	cols := m.boardColumns()
	got := map[data.Status]bool{}
	for _, c := range cols {
		got[c.status] = true
	}
	if len(cols) != 3 || !got[data.Backlog] || !got[data.Todo] || !got[data.Doing] {
		t.Errorf("columns %v, want backlog, todo and doing", got)
	}
	for _, gone := range []string{"Blocked", "Review", "Done"} {
		if tt.HasText(gone) {
			t.Errorf("empty column %q is drawn", gone)
		}
	}
	if !tt.HasText("To do") || !tt.HasText("Nothing waiting. Drag a card here to plan it.") {
		t.Errorf("To do is missing or has no hint: %q", tt.Texts())
	}
}

func TestBoardToDoIsThereOnAnEmptyColumnSet(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("scratch", "me/scratch")
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "In flight", Status: data.Doing, ProjectID: data.IntPtr(p.ID)})
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	_, m := boardTester(t, app, p.ID, 1100)
	cols := m.boardColumns()
	if len(cols) != 2 || cols[0].status != data.Todo || cols[1].status != data.Doing {
		t.Errorf("columns %+v", cols)
	}
}

func TestBoardHasNoCollapsedStrips(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 1, 1840)
	if len(m.boardColumns()) != 6 {
		t.Fatalf("columns %d, want all six that hold tasks", len(m.boardColumns()))
	}
	for _, want := range []string{"Backlog", "To do", "Doing", "Blocked", "Review", "Done"} {
		if !tt.HasText(want) {
			t.Errorf("missing column %q", want)
		}
	}
	if w := columnWidth(900, 6); w != 208 {
		t.Errorf("six columns in 900 pt: width %v, want the 208 minimum (the board scrolls)", w)
	}
	if w := columnWidth(900, 2); w != 264 {
		t.Errorf("two columns: width %v, want the 264 maximum", w)
	}
}

func TestDroppingACardChangesItsStatus(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1840)
	dragTo(t, tt, "Write the release notes", "Doing")
	if task, _ := app.Task(5); task.Status != data.Doing {
		t.Errorf("status %v after the drop, want doing", task.Status)
	}
}

func TestDroppingOnTheSameColumnChangesNothing(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1840)
	dragTo(t, tt, "Write the release notes", "To do")
	if task, _ := app.Task(5); task.Status != data.Todo {
		t.Errorf("status %v, want todo", task.Status)
	}
}

func TestDraggingRevealsDropTargetsForHiddenStatuses(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100)
	from, _ := tt.Find("Issue #218 adaptation")
	fx, fy := center(from)
	tt.Press(fx, fy)
	tt.Move(fx+8, fy+8)
	tt.Move(fx+20, fy+30)
	tt.Frame()
	if m.dragging == 0 {
		t.Fatal("no drag in progress")
	}
	if !tt.HasText("Blocked") || !tt.HasText("Review") || !tt.HasText("Done") {
		t.Fatalf("hidden statuses are not offered: %q", tt.Texts())
	}
	to, _ := tt.Find("Review")
	tx, ty := center(to)
	tt.Move(tx, ty)
	tt.Release(tx, ty)
	if task, _ := app.Task(15); task.Status != data.Review {
		t.Errorf("status %v after dropping on the hidden Review target", task.Status)
	}
	tt.Frame()
	if m.dragging != 0 {
		t.Error("drag state lingers")
	}
}

func TestBoardClickOpensTheInspectorAndAddsTasks(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100)
	if err := tt.Click("Issue #218 adaptation"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 15 {
		t.Errorf("opened %d", m.openID)
	}
	m.openID = 0
	if err := tt.Click("Add a task"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Board added")
	tt.Key(0, ui.KeyEnter)
	found := false
	for _, task := range app.snap.tasks {
		found = found || task.Title == "Board added" && task.Status == data.Todo
	}
	if !found {
		t.Error("the To do column did not add a task")
	}
}

func TestViewToggleSwitchesBetweenListAndBoard(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	if err := tt.Click("Board view"); err != nil {
		t.Fatal(err)
	}
	if !m.board[1] || !tt.HasText("Done") {
		t.Errorf("not a board: %q", tt.Texts())
	}
	if err := tt.Click("List view"); err != nil {
		t.Fatal(err)
	}
	if m.board[1] || tt.HasText("Nothing waiting. Drag a card here to plan it.") {
		t.Error("not back on the list")
	}
}

func TestMilestoneMenuOnlyForProjectsWithMilestones(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	if !tt.HasText("All milestones") {
		t.Error("project 1 has milestones but no menu")
	}
	tt2, _, _ := projectTester(t, 2)
	if tt2.HasText("All milestones") {
		t.Error("project 2 has no milestones but shows the menu")
	}
	if err := tt.Click("Milestone: All milestones"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("v0.2 native · due Oct 28"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Decide on the activation policy") {
		t.Error("the milestone filter left a task of no milestone")
	}
}
