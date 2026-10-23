package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func milestonesOf(app *App, project int) []data.Milestone { return app.snap.milestones[project] }

func openMilestoneMenu(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click("Milestone: " + label); err != nil {
		t.Fatal(err)
	}
}

// newMilestoneOf opens the New milestone sheet from the project's sidebar menu, where a project
// without a milestone has its only way to one.
func newMilestoneOf(t *testing.T, tt *ui.Tester, project string) {
	t.Helper()
	if err := tt.RightClick(project); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("New milestone…"); err != nil {
		t.Fatal(err)
	}
}

func TestMilestoneMenuAppearsWithTheFirstMilestoneAndKeepsNewMilestone(t *testing.T) {
	tt, _, _ := projectTester(t, 2) // no milestone yet
	hasNot(t, tt, "All milestones")

	tt, _, _ = projectTester(t, 1)
	openMilestoneMenu(t, tt, "All milestones")
	got := tt.Menu()
	for _, want := range []string{"All milestones", "New milestone…", "Edit “v0.2 native”", "Edit “v0.1 spike”"} {
		if !slices.Contains(got, want) {
			t.Errorf("menu %q lacks %q", got, want)
		}
	}
	if !slices.Contains(got, "v0.1 spike · closed") {
		t.Errorf("menu %q: the filter list is gone", got)
	}

	inbox, m, _ := newMainTester(t, 1100)
	m.goTo(nav{})
	inbox.Frame()
	if inbox.HasText("All milestones") {
		t.Error("the Inbox has a milestone menu")
	}
}

func TestNewMilestoneSheetCreatesAnOpenMilestone(t *testing.T) {
	tt, _, app := projectTester(t, 2)
	newMilestoneOf(t, tt, "answer-me-with-html")
	for _, want := range []string{"New milestone", "Title", "Due", "No due date", "Cancel", "Create"} {
		if !tt.HasText(want) {
			t.Errorf("sheet lacks %q in %q", want, tt.Texts())
		}
	}
	_ = tt.Click("Create") // disabled with no title
	if len(milestonesOf(app, 2)) != 0 {
		t.Fatal("a milestone without a title was created")
	}
	tt.Type("v1.0")
	tt.Key(0, ui.KeyEnter)
	ms := milestonesOf(app, 2)
	if len(ms) != 1 || ms[0].Title != "v1.0" || !ms[0].Open || ms[0].DueAt != nil {
		t.Fatalf("milestones %+v", ms)
	}
	if tt.HasText("No due date") {
		t.Error("the sheet stays open")
	}
	if !tt.HasText("Milestone: All milestones") {
		t.Errorf("header: %q", tt.Texts())
	}
}

func TestNewMilestoneWithADueDate(t *testing.T) {
	tt, _, app := projectTester(t, 2)
	newMilestoneOf(t, tt, "answer-me-with-html")
	tt.Type("Launch")
	if err := tt.Click("No due date"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Done"); err != nil { // the popover takes the day it opened on
		t.Fatal(err)
	}
	if err := tt.Click("Create"); err != nil {
		t.Fatal(err)
	}
	ms := milestonesOf(app, 2)
	if len(ms) != 1 || ms[0].DueAt == nil {
		t.Fatalf("milestones %+v", ms)
	}
	if h, mi, _ := ms[0].DueAt.Clock(); h != 0 || mi != 0 {
		t.Errorf("a due date is a day, not %v", ms[0].DueAt)
	}
}

func TestRenameAMilestone(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	openMilestoneMenu(t, tt, "All milestones")
	if err := tt.ChooseMenuItem("Edit “v0.2 native”", "Rename…"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Rename milestone") {
		t.Fatalf("no sheet: %q", tt.Texts())
	}
	if tt.HasText("No due date") {
		t.Error("the rename sheet asks for a date")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("v0.2 shipped")
	tt.Key(0, ui.KeyEnter)
	if got := milestonesOf(app, 1)[0].Title; got != "v0.2 shipped" {
		t.Errorf("title %q", got)
	}
}

func TestMilestoneDueDateSheetSetsAndClears(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	openMilestoneMenu(t, tt, "All milestones")
	if err := tt.ChooseMenuItem("Edit “v0.2 native”", "Due date…"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Due date of “v0.2 native”") || tt.HasText("Title") {
		t.Fatalf("sheet: %q", tt.Texts())
	}
	// The sheet shows the date it has; Clear removes it.
	if tt.HasText("No due date") {
		t.Fatalf("the existing date is not shown: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if milestonesOf(app, 1)[0].DueAt == nil {
		t.Fatal("Esc changed the date")
	}

	openMilestoneMenu(t, tt, "All milestones")
	_ = tt.ChooseMenuItem("Edit “v0.2 native”", "Due date…")
	// The value button under Due shows the day; opening it offers Clear.
	if err := tt.Click("Oct 28"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Clear"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatal(err)
	}
	if milestonesOf(app, 1)[0].DueAt != nil {
		t.Errorf("due not cleared: %v", milestonesOf(app, 1)[0].DueAt)
	}
}

func TestCloseAndReopenAMilestone(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	openMilestoneMenu(t, tt, "All milestones")
	if err := tt.ChooseMenuItem("Edit “v0.2 native”", "Close milestone"); err != nil {
		t.Fatal(err)
	}
	if milestonesOf(app, 1)[0].Open {
		t.Error("not closed")
	}
	openMilestoneMenu(t, tt, "All milestones")
	if err := tt.ChooseMenuItem("Edit “v0.2 native”", "Reopen milestone"); err != nil {
		t.Fatal(err)
	}
	if !milestonesOf(app, 1)[0].Open {
		t.Error("not reopened")
	}
	// A closed milestone offers Reopen, never Close.
	openMilestoneMenu(t, tt, "All milestones")
	if err := tt.ChooseMenuItem("Edit “v0.1 spike”", "Close milestone"); err == nil {
		t.Error("a closed milestone offers Close")
	}
}
