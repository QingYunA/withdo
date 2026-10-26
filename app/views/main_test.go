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

// A branch longer than the inspector ends in an ellipsis, as every other long value does, and
// clicking it edits the whole name.
func TestInspectorEllipsizesALongBranchAndEditsItOnClick(t *testing.T) {
	const width = 860
	tt, m, app := newMainTester(t, width)
	long := "worktree-agent-ac6ff63e8ba29afb9-fix-the-remaining-dogfood-findings"
	if err := app.Update(2, data.Patch{Branch: &long}); err != nil {
		t.Fatal(err)
	}
	m.OpenTask(2)
	tt.Frame()
	r, ok := tt.Find(long)
	if !ok {
		t.Fatalf("the branch is not drawn as text: %q", tt.Texts())
	}
	if r.X+r.W > width {
		t.Errorf("the branch runs to %.0f, past the window's %d: it is cut, not ellipsized", r.X+r.W, width)
	}

	if err := tt.Click(long); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("feat/short")
	tt.Key(0, ui.KeyEnter)
	if got, _ := app.Task(2); got.Branch != "feat/short" {
		t.Errorf("branch %q after the click and Enter", got.Branch)
	}
}

func TestInspectorOpensThePullRequest(t *testing.T) {
	app, _, rec := newTestApp(t)
	m := NewMain(app)
	tt := newTester(m.View, 1100, 720)
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

func TestSidebarListsTodayAgentsCalendarAndProjects(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	for _, want := range []string{"Today", "Agents", "Calendar", "Summary", "Projects", "adhd-todo", "answer-me-with-html", "picpocket", "Settings"} {
		if !tt.HasText(want) {
			t.Errorf("sidebar is missing %q", want)
		}
	}
	for _, gone := range []string{"Mine", "Inbox", "Completed", "Unsorted", "Expand Today", "Collapse Today"} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still has %q", gone)
		}
	}
	if err := tt.Click("answer-me-with-html"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenProject || !tt.HasText("Issue #218 adaptation") || tt.HasText("Port core to Go") {
		t.Errorf("after choosing the project: %q", tt.Texts())
	}
	if err := tt.Click("Calendar"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenCalendar {
		t.Errorf("Calendar did not open: %+v", m.nav)
	}
	if err := tt.Click("Agents"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenAgents || !tt.HasText("3 working") {
		t.Errorf("Agents did not open: %+v", m.nav)
	}
	if err := tt.Click("Today"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenMine {
		t.Error("Today did not open")
	}
}

func TestSidebarCountsAndTheBlockedDot(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	// Today counts what is handed over and what is due today (and nothing it has not seen: the first
	// frame is on Today); Agents the tasks at work, never their subtasks; a project with a blocked task
	// shows a dot instead of its count.
	got := tt.Texts()
	idx := func(s string) int {
		for i, x := range got {
			if x == s {
				return i
			}
		}
		return -1
	}
	if got[idx("Today")+1] != "4" || got[idx("Agents")+1] != "3" || got[idx("Calendar")+1] != "Summary" || got[idx("Summary")+1] != "Projects" {
		t.Errorf("counts after Today, Agents, Calendar and Summary: %q", got[:10])
	}
	if got[idx("adhd-todo")+1] == "5" || got[idx("answer-me-with-html")+1] != "1" {
		t.Errorf("project counts: %q", got[:14])
	}
}

func TestOpenTaskFromANotificationShowsItOverMine(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 2})
	m.OpenTask(7)
	tt.Frame()
	if m.nav.screen != screenMine || m.openID != 7 || !tt.HasText("Where") {
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
	for _, want := range []string{"scratch", "No tasks yet.", "How to connect"} {
		if !tt.HasText(want) {
			t.Errorf("empty project is missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText("me/scratch") {
		t.Error("the repository is in the header")
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
