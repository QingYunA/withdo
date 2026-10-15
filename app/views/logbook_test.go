package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func logbookTester(t *testing.T) (*ui.Tester, *Main, *App) {
	t.Helper()
	tt, m, app := newMainTester(t, 1100)
	tt.Frame()
	if err := tt.Click("Logbook"); err != nil {
		t.Fatal(err)
	}
	return tt, m, app
}

func TestLogbookSitsUnderInboxInTheSidebar(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	if !above(t, tt, "Inbox", "Logbook") || !above(t, tt, "Logbook", "PROJECTS") {
		t.Errorf("sidebar order: %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.Key3)
	if m.nav.screen != screenLogbook {
		t.Errorf("Cmd+3 went to %+v", m.nav)
	}
}

func TestLogbookListsDoneAndCancelledNewestFirstGroupedByDay(t *testing.T) {
	app, store, _ := newTestApp(t)
	// Cancelled today, so there is a Today group; the seed has done tasks 1, 2, 5 and 6 days ago.
	if _, err := store.UpdateTask(19, data.Patch{Status: data.StatusPtr(data.Cancelled)}); err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 900)
	tt.Frame()
	_ = tt.Click("Logbook")
	for _, want := range []string{"Logbook", "Done and cancelled, newest first", "Today", "Yesterday", "Oct 12", "Oct 9", "Oct 8",
		"Renew passport", "Draft the UI spec", "Spike the mygo shell", "Map worktrees to the main repo", "Pick a UI toolkit", "2:30 PM"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	order := []string{"Today", "Renew passport", "Yesterday", "Draft the UI spec", "Oct 12", "Spike the mygo shell", "Oct 9", "Map worktrees to the main repo", "Oct 8", "Pick a UI toolkit"}
	for i := 1; i < len(order); i++ {
		if !above(t, tt, order[i-1], order[i]) {
			t.Errorf("%q should be above %q", order[i-1], order[i])
		}
	}
	if tt.HasText("Port core to Go") {
		t.Error("an open task is in the Logbook")
	}
}

func TestLogbookGroupsTasksOfOneDayUnderOneHeader(t *testing.T) {
	app, store, _ := newTestApp(t)
	for _, id := range []int{8, 18} {
		if _, err := store.UpdateTask(id, data.Patch{Status: data.StatusPtr(data.Done)}); err != nil {
			t.Fatal(err)
		}
	}
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenLogbook})
	tt.Frame()
	days := 0
	for _, s := range tt.Texts() {
		if s == "Today" {
			days++
		}
	}
	if days != 1 {
		t.Errorf("%d Today headers in %q", days, tt.Texts())
	}
}

func TestEmptyLogbookSaysWhatAppearsThere(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenLogbook})
	tt.Frame()
	for _, want := range []string{"Nothing finished yet", "Tasks you mark done or cancel collect here, newest first."} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
}

func TestLogbookRowsReopenOpenAndHaveTheTaskMenu(t *testing.T) {
	tt, m, app := logbookTester(t)
	if err := tt.RightClick("Draft the UI spec"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Menu(); len(got) != 5 || got[0] != "Status" {
		t.Errorf("menu %q", got)
	}
	tt.CloseMenu()
	if err := tt.Click("Draft the UI spec"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 13 {
		t.Errorf("opened %d", m.openID)
	}
	if err := tt.Click("Status: Done"); err != nil { // the first glyph
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Todo"); err != nil {
		t.Fatalf("status menu: %v", err)
	}
	if got, _ := app.Task(13); got.Status != data.Todo {
		t.Errorf("status %v, want reopened", got.Status)
	}
	for _, task := range m.logbookTasks() {
		if task.ID == 13 {
			t.Error("a reopened task stays in the Logbook")
		}
	}
}

func TestLogbookKeyboard(t *testing.T) {
	tt, m, _ := logbookTester(t)
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

func TestLogbookShowsAHundredAndLoadsOlderOnRequest(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	for i := 0; i < 130; i++ {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: "done " + itoa(i), Status: data.Done})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenLogbook})
	tt.SetSize(1100, 4200) // room for every row, so the link is on screen
	tt.Frame()
	if got := len(m.logbookTasks()); got != 100 {
		t.Fatalf("%d rows before asking for more", got)
	}
	if err := tt.Click("Show older"); err != nil {
		t.Fatal(err)
	}
	if got := len(m.logbookTasks()); got != 130 {
		t.Errorf("%d rows after Show older", got)
	}
	if tt.HasText("Show older") {
		t.Error("Show older stays when everything is loaded")
	}
}
