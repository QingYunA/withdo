package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func projectTester(t *testing.T, project int) (*ui.Tester, *Main, *App) {
	t.Helper()
	tt, m, app := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: project})
	tt.Frame()
	return tt, m, app
}

func TestProjectListGroupsInOrderAndHidesEmptyGroups(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	for _, want := range []string{"adhd-todo", "QingYunA/adhd-todo", "Blocked", "In review", "Doing", "To do", "Fix tray race on launch", "What's next in adhd-todo? “fix the login redirect loop, p1, friday”"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if !above(t, tt, "Blocked", "In review") || !above(t, tt, "In review", "Doing") || !above(t, tt, "Doing", "To do") {
		t.Error("groups are out of order")
	}
	if tt.HasText("Spike the mygo shell") || tt.HasText("Poll gh for PR status") {
		t.Error("Done and Backlog tasks are drawn before they are shown")
	}
	// A project with a single doing task draws one group and nothing else.
	tt2, _, _ := projectTester(t, 2)
	if !tt2.HasText("Doing") || tt2.HasText("Blocked") || tt2.HasText("In review") || tt2.HasText("To do") {
		t.Errorf("one task, one group: %q", tt2.Texts())
	}
}

func TestProjectListFoldsDoneAndBacklogUntilShown(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	tt.SetSize(1100, 1600) // room for the opened groups
	tt.Frame()
	if !tt.HasText("Done 4 · Backlog 2") || !tt.HasText("— show") {
		t.Fatalf("no fold line in %q", tt.Texts())
	}
	if tt.HasText("Spike the mygo shell") {
		t.Error("a done task is visible while folded")
	}
	if err := tt.Click("show Done 4 and Backlog 2"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Backlog", "Done", "Spike the mygo shell", "Poll gh for PR status", "— hide"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q after showing: %q", want, tt.Texts())
		}
	}
	if !m.expanded[m.nav] {
		t.Error("state not kept")
	}
	if err := tt.Click("hide Done 4 and Backlog 2"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Spike the mygo shell") {
		t.Error("still shown after hide")
	}
}

func TestProjectListShowsOnlyWhatExistsInTheFoldLine(t *testing.T) {
	tt, _, _ := projectTester(t, 3) // picpocket: one backlog task
	if !tt.HasText("Backlog 1") || tt.HasText("Done 0") {
		t.Errorf("fold line: %q", tt.Texts())
	}
}

func TestProjectListRowsCarryAgentSummaryAndLabels(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	for _, want := range []string{"Needs a decision on the activation policy", "3d stale", "Ported the store and migrations", "Fri", "5h", "fix/tray", "#12"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
}

func TestTheComposerCreatesATodoInTheProjectWithoutAModel(t *testing.T) {
	tt, _, app := projectTester(t, 2)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Handle empty replies")
	tt.Key(0, ui.KeyEnter)
	var added data.Task
	for _, task := range app.snap.tasks {
		if task.Title == "Handle empty replies" {
			added = task
		}
	}
	if added.ProjectID == nil || *added.ProjectID != 2 || added.Status != data.Todo || added.Source != data.SourceUser {
		t.Fatalf("added %+v", added)
	}
	if !tt.HasText("To do") || !tt.HasText("Handle empty replies") {
		t.Errorf("not listed: %q", tt.Texts())
	}
}

func TestInboxIsTheListWithoutTheBoardToggle(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.goTo(nav{screen: screenInbox})
	tt.Frame()
	if !tt.HasText("Inbox") || !tt.HasText("Book dentist") || tt.HasText("Board view") {
		t.Errorf("inbox: %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Call mum")
	tt.Key(0, ui.KeyEnter)
	for _, task := range app.snap.tasks {
		if task.Title == "Call mum" && task.ProjectID != nil {
			t.Error("an inbox task got a project")
		}
	}
}

func TestListKeyboardWalksRowsInOrder(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	ids := m.listIDs()
	if len(ids) != 6 || ids[0] != 3 || ids[1] != 2 {
		t.Fatalf("ids %v", ids)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	if m.sel != 2 {
		t.Fatalf("selected %d, want the review task", m.sel)
	}
	tt.Key(0, ui.KeyEnter)
	if m.openID != 2 {
		t.Errorf("opened %d", m.openID)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Key(0, ui.KeyEscape)
	if m.openID != 0 || m.sel != 0 {
		t.Errorf("Esc left open %d, selected %d", m.openID, m.sel)
	}
	tt.Key(0, ui.KeyDown)
	for i := 0; i < 3; i++ {
		tt.Key(0, ui.KeyDown)
	}
	tt.Key(0, ui.KeySpace) // task 1 is an agent's: nothing happens
	if task, _ := app.Task(m.sel); task.Source == data.SourceUser && task.Status == data.Done {
		t.Error("Space completed the wrong task")
	}
}

func TestSpaceCompletesAUsersTaskInTheList(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	m.sel = 5
	tt.Key(0, ui.KeySpace)
	if task, _ := app.Task(5); task.Status != data.Done {
		t.Errorf("status %v", task.Status)
	}
}

func TestDoneGroupShowsTenAndShowAll(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("scratch", "me/scratch")
	for i := 0; i < 12; i++ {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: "Finished " + itoa(i), Status: data.Done, ProjectID: data.IntPtr(p.ID)})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1600)
	m.goTo(nav{screen: screenProject, projectID: p.ID})
	tt.Frame()
	if err := tt.Click("show Done 12"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Show all 12 done") {
		t.Fatalf("no link: %q", tt.Texts())
	}
	if err := tt.Click("Show all 12 done"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Show all 12 done") {
		t.Error("link still shown")
	}
}
