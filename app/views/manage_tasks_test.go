package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

func taskProject(app *App, id int) string {
	t, _ := app.Task(id)
	return app.projectName(t.ProjectID)
}

func TestTaskContextMenuOnAListRow(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	if err := tt.RightClick("Native UI with mygo"); err != nil { // has pull request #12
		t.Fatal(err)
	}
	if got, want := tt.Menu(), []string{"Status", "Priority", "Move to", "Open PR", "-", "Delete…"}; !slices.Equal(got, want) {
		t.Errorf("menu %q, want %q", got, want)
	}
	tt.CloseMenu()
	_ = tt.RightClick("Fix tray race on launch") // no pull request
	if got, want := tt.Menu(), []string{"Status", "Priority", "Move to", "-", "Delete…"}; !slices.Equal(got, want) {
		t.Errorf("menu without a PR %q, want %q", got, want)
	}
}

func TestTaskContextMenuChangesStatusPriorityAndOpensThePR(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()

	_ = tt.RightClick("Fix tray race on launch")
	if err := tt.ChooseMenuItem("Status", "Review"); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.Task(3); got.Status != data.Review {
		t.Errorf("status %v", got.Status)
	}
	_ = tt.RightClick("Fix tray race on launch")
	if err := tt.ChooseMenuItem("Priority", "P3  Low"); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.Task(3); got.Priority != data.P3 {
		t.Errorf("priority %v", got.Priority)
	}
	_ = tt.RightClick("Native UI with mygo")
	if err := tt.ChooseMenuItem("Open PR"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || rec.urls[0] != "https://github.com/QingYunA/adhd-todo/pull/12" {
		t.Errorf("opened %v", rec.urls)
	}
}

func TestTaskContextMenuMovesToAProjectOrTheInbox(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	_ = tt.RightClick("Fix tray race on launch")
	if err := tt.ChooseMenuItem("Move to", "picpocket"); err != nil {
		t.Fatal(err)
	}
	if got := taskProject(app, 3); got != "picpocket" {
		t.Errorf("project %q", got)
	}
	if tt.HasText("Fix tray race on launch") {
		t.Error("the task is still listed in the project it left")
	}
	m.goTo(nav{screen: screenProject, projectID: 3})
	tt.Frame()
	_ = tt.RightClick("Fix tray race on launch")
	if err := tt.ChooseMenuItem("Move to", "Inbox"); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.Task(3); got.ProjectID != nil {
		t.Errorf("project %v, want the Inbox", got.ProjectID)
	}
}

func TestTaskContextMenuDeleteAsksFirst(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	_ = tt.RightClick("Fix tray race on launch")
	if err := tt.ChooseMenuItem("Delete…"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Delete “Fix tray race on launch”?", "Its progress reports go with it. It can't be undone."} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.Task(3); !ok {
		t.Fatal("Cancel deleted the task")
	}
	_ = tt.RightClick("Fix tray race on launch")
	_ = tt.ChooseMenuItem("Delete…")
	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.Task(3); ok {
		t.Error("the task is still there")
	}
	if tt.HasText("Fix tray race on launch") {
		t.Error("the task is still listed")
	}
}

func TestTaskContextMenuOnBoardCardsAndNowRows(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	m.board[1] = true
	tt.Frame()
	if err := tt.RightClick("Fix tray race on launch"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Menu(); len(got) != 5 || got[0] != "Status" || got[4] != "Delete…" {
		t.Errorf("card menu %q", got)
	}
	tt.CloseMenu()

	m.goTo(nav{})
	tt.Frame()
	if err := tt.RightClick("Book dentist"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Menu(); len(got) != 5 || got[2] != "Move to" {
		t.Errorf("Now row menu %q", got)
	}
}

func TestInspectorMovesTheTaskThroughItsProjectRow(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(8) // Book dentist, in the Inbox
	tt.Frame()
	if !tt.HasText("Project") {
		t.Fatalf("no Project row: %q", tt.Texts())
	}
	if err := tt.Click("Project: Inbox"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.Menu(), []string{"Inbox", "-", "adhd-todo", "answer-me-with-html", "picpocket"}; !slices.Equal(got, want) {
		t.Errorf("menu %q, want %q", got, want)
	}
	if err := tt.ChooseMenuItem("picpocket"); err != nil {
		t.Fatal(err)
	}
	if got := taskProject(app, 8); got != "picpocket" {
		t.Errorf("project %q", got)
	}
	if !tt.HasText("Project: picpocket") {
		t.Errorf("the row does not show the new project: %q", tt.Texts())
	}

	// A task with a milestone loses it when it moves, and the Milestone row follows the project.
	m.OpenTask(5)
	tt.Frame()
	if err := tt.Click("Milestone: v0.2 native"); err != nil {
		t.Fatalf("task 5 has no milestone row: %v", err)
	}
	tt.CloseMenu()
	if err := tt.Click("Project: adhd-todo"); err != nil {
		t.Fatal(err)
	}
	_ = tt.ChooseMenuItem("picpocket")
	if got, _ := app.Task(5); got.MilestoneID != nil {
		t.Errorf("the milestone came along: %v", *got.MilestoneID)
	}
}

func TestInspectorDeletesTheTaskAfterAsking(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(8)
	tt.Frame()
	if err := tt.Click("Delete task…"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Delete “Book dentist”?") {
		t.Fatalf("no question: %q", tt.Texts())
	}
	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.Task(8); ok {
		t.Error("the task is still there")
	}
	tt.Frame()
	if m.openID != 0 {
		t.Errorf("the inspector stays open on a deleted task: %d", m.openID)
	}
}
