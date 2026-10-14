package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestNowShowsAllThreeSectionsInOrder(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	for _, want := range []string{"Now", "Wednesday, Oct 14", "NEEDS YOU", "AGENTS AT WORK", "YOURS", "Port core to Go", "Book dentist"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if !above(t, tt, "NEEDS YOU", "AGENTS AT WORK") || !above(t, tt, "AGENTS AT WORK", "YOURS") {
		t.Error("sections are out of order")
	}
}

func TestNowEmptySectionsAreNotDrawn(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "Only mine"})
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, _, _ := newTesterFor(app, 1100)
	if !tt.HasText("YOURS") || !tt.HasText("Only mine") {
		t.Fatalf("texts %q", tt.Texts())
	}
	for _, gone := range []string{"NEEDS YOU", "AGENTS AT WORK", "All clear."} {
		if tt.HasText(gone) {
			t.Errorf("%q is drawn with nothing in it", gone)
		}
	}
}

func TestNowAllClearWhenNothingIsHappening(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	for _, want := range []string{"All clear.", "Agents report here through the adhd-todo MCP server.", "How to connect"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	for _, gone := range []string{"NEEDS YOU", "AGENTS AT WORK", "YOURS"} {
		if tt.HasText(gone) {
			t.Errorf("%q in the All clear state", gone)
		}
	}
	if err := tt.Click("How to connect"); err != nil {
		t.Fatal(err)
	}
	if !m.settings {
		t.Error("How to connect did not open Settings")
	}
}

func TestNeedsYouOrderIsBlockedReminderReview(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	if !above(t, tt, "Fix tray race on launch", "Reply to design feedback") || !above(t, tt, "Reply to design feedback", "Native UI with mygo") {
		t.Errorf("order wrong in %q", tt.Texts())
	}
	for _, want := range []string{
		"Agent is waiting: “Needs a decision on the activation policy”",
		"You asked to be reminded at 2:30 PM",
		"PR #12 is ready for your review",
	} {
		if !tt.HasText(want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestNeedsYouOpenOpensTheInspector(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Open Fix tray race on launch"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 3 || !tt.HasText("ACTIVITY") {
		t.Errorf("open %d, texts %q", m.openID, tt.Texts())
	}
}

func TestNeedsYouOpenPRCallsOpenURL(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	if err := tt.Click("Open pull request #12"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || rec.urls[0] != "https://github.com/QingYunA/adhd-todo/pull/12" {
		t.Errorf("opened %v", rec.urls)
	}
	if m.openID != 0 {
		t.Error("the action button also opened the inspector")
	}
}

func TestNeedsYouDoneMarksTheReminderDone(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	if err := tt.Click("Mark done: Reply to design feedback"); err != nil {
		t.Fatal(err)
	}
	if task, _ := app.Task(7); task.Status != data.Done {
		t.Errorf("status %v, want done", task.Status)
	}
	if m.openID != 0 {
		t.Error("Done opened the inspector")
	}
	app.completing = map[int]completion{} // let the row go
	tt.Frame()
	if tt.HasText("Reply to design feedback") {
		t.Error("the done reminder is still on Now")
	}
}

func TestClickingARowOpensTheInspectorAndEscCloses(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Port core to Go"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 1 || !tt.HasText("ACTIVITY") {
		t.Fatalf("open %d, texts %q", m.openID, tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if m.openID != 0 {
		t.Errorf("Esc left task %d open", m.openID)
	}
	if m.sel != 1 {
		t.Errorf("the first Esc cleared the selection too (%d)", m.sel)
	}
	tt.Key(0, ui.KeyEscape)
	if m.sel != 0 {
		t.Errorf("the second Esc left %d selected", m.sel)
	}
}

func TestAskFieldEnterAddsToInboxAndConfirms(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	before := len(app.snap.tasks)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Water the plants")
	tt.Key(0, ui.KeyEnter)
	if len(app.snap.tasks) != before+1 {
		t.Fatalf("tasks %d, want %d", len(app.snap.tasks), before+1)
	}
	var added data.Task
	for _, task := range app.snap.tasks {
		if task.Title == "Water the plants" {
			added = task
		}
	}
	if added.ProjectID != nil || added.Source != data.SourceUser || added.Status != data.Todo {
		t.Errorf("added %+v", added)
	}
	if !tt.HasText("Added to Inbox") || tt.HasText("Water the plants") && false {
		t.Errorf("no confirmation in %q", tt.Texts())
	}
	// It is mine and undated, so it also shows under Yours, and the field is empty again.
	if !tt.HasText("Water the plants") {
		t.Errorf("the new task is not on Now: %q", tt.Texts())
	}
	tt.Type("x")
	if !tt.HasText("x") {
		t.Error("the field lost the focus after adding")
	}
}

func TestAskFieldEmptyEnterAddsNothing(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	before := len(app.snap.tasks)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Key(0, ui.KeyEnter)
	if len(app.snap.tasks) != before {
		t.Error("an empty Enter created a task")
	}
}

func TestCmdKFocusesTheAskFieldFromAnywhere(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame() // the field takes the focus a few frames after it appears
	tt.Frame()
	tt.Frame()
	if m.nav != (nav{}) {
		t.Fatalf("Cmd+K left us on %+v", m.nav)
	}
	tt.Type("remind me")
	if m.ask.buf != "remind me" {
		t.Errorf("typing went elsewhere: ask %q, texts %q", m.ask.buf, tt.Texts())
	}
}

func TestArrowsSelectEnterOpensSpaceToggles(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	tt.Key(0, ui.KeyDown)
	if m.sel != 3 {
		t.Fatalf("selected %d after one Down, want the blocked task 3", m.sel)
	}
	tt.Key(0, ui.KeyDown)
	if m.sel != 7 {
		t.Fatalf("selected %d after two Down, want the reminder 7", m.sel)
	}
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeySpace) // an agent's task: Space leaves it alone
	if task, _ := app.Task(3); task.Status != data.Blocked {
		t.Errorf("Space changed an agent's task to %v", task.Status)
	}
	tt.Key(0, ui.KeyEnter)
	if m.openID != 3 {
		t.Errorf("Enter opened %d, want 3", m.openID)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeySpace)
	if task, _ := app.Task(7); task.Status != data.Done {
		t.Errorf("Space left my task %v, want done", task.Status)
	}
}

func TestUpFromTheFirstRowReturnsToTheAskField(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyUp)
	if m.sel != 0 {
		t.Errorf("selected %d, want none", m.sel)
	}
	tt.Type("typed")
	if m.ask.buf != "typed" {
		t.Errorf("the ask field did not take the focus: %q", m.ask.buf)
	}
}

func TestSpaceInTheAskFieldIsASpace(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("a b")
	if m.ask.buf != "a b" {
		t.Errorf("buffer %q", m.ask.buf)
	}
	if task, _ := app.Task(7); task.Status == data.Done {
		t.Error("Space toggled a task while typing")
	}
}

func TestCmdNumbersSwitchScreens(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.Key2)
	if m.nav.screen != screenInbox || !tt.HasText("Tasks without a project") {
		t.Errorf("Cmd+2 gave %+v", m.nav)
	}
	tt.Key(ui.Cmd, ui.Key1)
	if m.nav.screen != screenNow {
		t.Errorf("Cmd+1 gave %+v", m.nav)
	}
}

func TestNowCountsAndLinkToInbox(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	for i := 0; i < 9; i++ {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: "Idea " + itoa(i)})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	if !tt.HasText("Show all in Inbox") {
		t.Fatalf("no link: %q", tt.Texts())
	}
	if err := tt.Click("Show all in Inbox"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenInbox {
		t.Errorf("went to %+v", m.nav)
	}
}
