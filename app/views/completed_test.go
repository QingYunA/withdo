package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func completedTester(t *testing.T) (*ui.Tester, *Main, *App) {
	t.Helper()
	tt, m, app := newMainTester(t, 1100)
	tt.Frame()
	m.openHistory(0, false) // Home has no way in: See summary opens the Summary
	tt.Frame()
	return tt, m, app
}

func TestTheHistoryHasNoSidebarRowAndHasABackControl(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	hasNot(t, tt, "Completed", "Show all")
	m.openHistory(0, false)
	tt.Frame()
	if m.nav != (nav{screen: screenCompleted}) || m.histBack != (nav{}) {
		t.Fatalf("nav %+v, back %+v", m.nav, m.histBack)
	}
	has(t, tt, "Completed", "Done and cancelled, newest first", "Back to Today")
	if err := tt.Click("Back to Today"); err != nil {
		t.Fatal(err)
	}
	if m.nav != (nav{}) {
		t.Errorf("the back control went to %+v", m.nav)
	}
}

func TestShowAllOfAProjectOpensItsOwnHistoryAndGoesBackToIt(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	tt.SetSize(1100, 1600)
	tt.Frame()
	if err := tt.Click("Show all"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if want := (nav{screen: screenCompleted, projectID: 1}); m.nav != want {
		t.Fatalf("nav %+v, want %+v", m.nav, want)
	}
	for _, task := range m.completedTasks() {
		if task.ProjectID == nil || *task.ProjectID != 1 {
			t.Errorf("task %q of another project is in this project's history", task.Title)
		}
	}
	has(t, tt, "Back to adhd-todo")
	click(t, tt, "Back to adhd-todo")
	if want := (nav{screen: screenProject, projectID: 1}); m.nav != want {
		t.Errorf("back went to %+v", m.nav)
	}
}

func TestShowAllOfAgentsOpensWhatAgentsFinished(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	click(t, tt, "Agents")
	click(t, tt, "Show all")
	tt.Frame()
	if want := (nav{screen: screenCompleted, agents: true}); m.nav != want {
		t.Fatalf("nav %+v, want %+v", m.nav, want)
	}
	for _, task := range m.completedTasks() {
		if task.Source != data.SourceAgent {
			t.Errorf("%q was not written by an agent", task.Title)
		}
	}
	has(t, tt, "Back to Agents")
}

func TestCompletedListsDoneAndCancelledNewestFirstGroupedByDay(t *testing.T) {
	app, store, _ := newTestApp(t)
	// Cancelled today, so there is a Today group; the seed has done tasks 1, 2, 5 and 6 days ago.
	if _, err := store.UpdateTask(19, data.Patch{Status: data.StatusPtr(data.Cancelled)}); err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 900)
	tt.Frame()
	m.openHistory(0, false)
	tt.Frame()
	for _, want := range []string{"Completed", "Done and cancelled, newest first", "Today", "Yesterday", "Oct 12", "Oct 9", "Oct 8",
		"Renew passport", "Draft the UI spec", "Spike the mygo shell", "Map worktrees to the main repo", "Pick a UI toolkit", "2:30 PM"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	order := []string{"Renew passport", "Yesterday", "Draft the UI spec", "Oct 12", "Spike the mygo shell", "Oct 9", "Map worktrees to the main repo", "Oct 8", "Pick a UI toolkit"}
	for i := 1; i < len(order); i++ {
		if !above(t, tt, order[i-1], order[i]) {
			t.Errorf("%q should be above %q", order[i-1], order[i])
		}
	}
	if tt.HasText("Port core to Go") {
		t.Error("an open task is in Completed")
	}
}

func TestCompletedGroupsTasksOfOneDayUnderOneHeader(t *testing.T) {
	app, store, _ := newTestApp(t)
	for _, id := range []int{8, 18} {
		if _, err := store.UpdateTask(id, data.Patch{Status: data.StatusPtr(data.Done)}); err != nil {
			t.Fatal(err)
		}
	}
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenCompleted})
	tt.Frame()
	days := 0
	for _, s := range tt.Texts() {
		if s == "Today" {
			days++
		}
	}
	if days != 3 { // the sidebar row, the back control and the one day header
		t.Errorf("%d Today headers in %q", days, tt.Texts())
	}
}

func TestEmptyCompletedSaysWhatAppearsThere(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenCompleted})
	tt.Frame()
	for _, want := range []string{"Nothing finished yet", "Tasks you mark done or cancel collect here, newest first."} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
}

func TestCompletedRowsReopenOpenAndHaveTheTaskMenu(t *testing.T) {
	tt, m, app := completedTester(t)
	rightClick(t, tt, "Draft the UI spec")
	if !m.pop.open || m.pop.task != 13 || !tt.HasText("Move to…") {
		t.Errorf("menu open %v for %d", m.pop.open, m.pop.task)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if err := tt.Click("Draft the UI spec"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 13 {
		t.Errorf("opened %d", m.openID)
	}
	first := m.completedTasks()[0].ID
	click(t, tt, "Status: Done") // the first glyph: a finished task's glyph opens the menu
	click(t, tt, "Set status to To do")
	if got, _ := app.Task(first); got.Status != data.Todo {
		t.Errorf("status %v, want reopened", got.Status)
	}
	for _, task := range m.completedTasks() {
		if task.ID == first {
			t.Error("a reopened task stays in Completed")
		}
	}
}

func TestCompletedKeyboard(t *testing.T) {
	tt, m, _ := completedTester(t)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	first := m.sel
	if first == 0 {
		t.Fatal("Down selected nothing")
	}
	tt.Key(0, ui.KeyDown)
	if m.sel == first {
		t.Error("the second Down did not move")
	}
	tt.Key(0, ui.KeyEnter)
	if m.openID != m.sel {
		t.Errorf("Enter opened %d, selected %d", m.openID, m.sel)
	}
}

func TestCompletedShowsAHundredAndLoadsOlderOnRequest(t *testing.T) {
	f := demoStore(t, fixedNow, false)
	for i := 0; i < 130; i++ {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: "done " + itoa(i), Status: data.Done})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenCompleted})
	tt.SetSize(1100, 4200) // room for every row, so the link is on screen
	tt.Frame()
	if got := len(m.completedTasks()); got != 100 {
		t.Fatalf("%d rows before asking for more", got)
	}
	if err := tt.Click("Show older"); err != nil {
		t.Fatal(err)
	}
	if got := len(m.completedTasks()); got != 130 {
		t.Errorf("%d rows after Show older", got)
	}
	if tt.HasText("Show older") {
		t.Error("Show older stays when everything is loaded")
	}
}
