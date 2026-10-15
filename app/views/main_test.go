package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestInspectorSavesTheTitleOnEnter(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(5)
	tt.Frame()
	title, ok := tt.Find("Write the release notes")
	if !ok {
		t.Fatalf("no title field: %q", tt.Texts())
	}
	// The inspector's title is the right-most match; click into it and replace the text.
	_ = title
	if err := tt.Click("Title"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Ship the release notes")
	tt.Key(0, ui.KeyEnter)
	if got, _ := app.Task(5); got.Title != "Ship the release notes" {
		t.Errorf("title %q after Enter", got.Title)
	}
}

func TestInspectorSavesNotesOnBlur(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(7)
	tt.Frame()
	if err := tt.Click("Notes"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Start with the panel")
	if got, _ := app.Task(7); got.Notes != "" {
		t.Fatalf("notes saved before blur: %q", got.Notes)
	}
	if err := tt.Click("Title"); err != nil { // move the focus away
		t.Fatal(err)
	}
	if got, _ := app.Task(7); got.Notes != "Start with the panel" {
		t.Errorf("notes %q after blur", got.Notes)
	}
}

func TestInspectorChangesPriorityThroughTheMenu(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(7)
	tt.Frame()
	if err := tt.Click("Priority: P2  Normal"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("P1  High"); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.Task(7); got.Priority != data.P1 {
		t.Errorf("priority %v", got.Priority)
	}
}

func TestInspectorOpensThePullRequest(t *testing.T) {
	app, _, rec := newTestApp(t)
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	still(tt)
	m.OpenTask(2)
	tt.Frame()
	if err := tt.Click("Open pull request #12"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || rec.urls[0] != "https://github.com/QingYunA/adhd-todo/pull/12" {
		t.Errorf("opened %v", rec.urls)
	}
}

func TestSidebarListsNowInboxAndProjects(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	for _, want := range []string{"Now", "Inbox", "PROJECTS", "adhd-todo", "answer-me-with-html", "picpocket", "Settings"} {
		if !tt.HasText(want) {
			t.Errorf("sidebar is missing %q", want)
		}
	}
	if err := tt.Click("answer-me-with-html"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenProject || !tt.HasText("Issue #218 adaptation") || tt.HasText("Port core to Go") {
		t.Errorf("after choosing the project: %q", tt.Texts())
	}
	if err := tt.Click("Inbox"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenInbox || !tt.HasText("Book dentist") {
		t.Errorf("inbox shows %q", tt.Texts())
	}
	if err := tt.Click("Now"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenNow {
		t.Error("Now did not open")
	}
}

func TestSidebarCountsAndTheBlockedDot(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	// Now counts every row of its three sections; Inbox the open inbox tasks; a project with a
	// blocked task shows a dot instead of its count.
	got := tt.Texts()
	idx := func(s string) int {
		for i, x := range got {
			if x == s {
				return i
			}
		}
		return -1
	}
	if got[idx("Now")+1] != "9" || got[idx("Inbox")+1] != "3" {
		t.Errorf("counts after Now and Inbox: %q", got[:8])
	}
	if got[idx("adhd-todo")+1] == "5" || got[idx("answer-me-with-html")+1] != "1" {
		t.Errorf("project counts: %q", got[:14])
	}
}

func TestOpenTaskFromANotificationShowsItOverNow(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 2})
	m.OpenTask(7)
	tt.Frame()
	if m.nav.screen != screenNow || m.openID != 7 || !tt.HasText("ACTIVITY") {
		t.Errorf("nav %+v open %d", m.nav, m.openID)
	}
	m.OpenTask(99999) // a task that no longer exists changes nothing
	if m.openID != 7 {
		t.Error("an unknown task replaced the open one")
	}
}

func TestStoreErrorShowsACalmBannerWithRetry(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	app.snap.err = errTest
	tt.Frame()
	if !tt.HasText("Couldn't reach the data store. Retrying…") {
		t.Fatalf("no banner in %q", tt.Texts())
	}
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	if app.Err() != nil {
		t.Error("the banner stays after a good refresh")
	}
}

func TestMainEmptyProjectAndInbox(t *testing.T) {
	app, p := emptyProjectApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: p.ID})
	tt.Frame()
	for _, want := range []string{"scratch", "me/scratch", "No tasks yet.", "How to connect"} {
		if !tt.HasText(want) {
			t.Errorf("empty project is missing %q in %q", want, tt.Texts())
		}
	}
	m.goTo(nav{screen: screenInbox})
	tt.Frame()
	for _, want := range []string{"Your inbox is clear.", "Tasks without a project land here. Write one in the box above, or press ⌘K."} {
		if !tt.HasText(want) {
			t.Errorf("empty inbox is missing %q in %q", want, tt.Texts())
		}
	}
}

func TestMainFromAnEmptyStore(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	if !tt.HasText("Projects appear when an agent reports from a repository.") {
		t.Errorf("no empty sidebar hint: %q", tt.Texts())
	}
}

var errTest = testError("unreachable")

type testError string

func (e testError) Error() string { return string(e) }

var _ = data.Todo
var _ = ui.KeyA
