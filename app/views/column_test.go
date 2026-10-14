package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

func expandedSet(plans map[data.Status]columnPlan) map[data.Status]bool {
	out := map[data.Status]bool{}
	for s, p := range plans {
		if p.expanded {
			out[s] = true
		}
	}
	return out
}

func TestPlanColumnsCollapsesEmptyAndDone(t *testing.T) {
	counts := map[data.Status]int{data.Doing: 1, data.Blocked: 2, data.Done: 5}
	plans := planColumns(2000, counts, nil)
	want := map[data.Status]bool{data.Doing: true, data.Blocked: true}
	got := expandedSet(plans)
	if len(got) != 2 || !got[data.Doing] || !got[data.Blocked] {
		t.Errorf("expanded = %v, want %v", got, want)
	}
	if plans[data.Doing].width != theme.ColumnWidth {
		t.Errorf("width = %v, want the full %v", plans[data.Doing].width, theme.ColumnWidth)
	}
}

func TestPlanColumnsForcedColumnsOpen(t *testing.T) {
	counts := map[data.Status]int{data.Doing: 1, data.Done: 5}
	plans := planColumns(2000, counts, map[data.Status]bool{data.Done: true, data.Todo: true})
	got := expandedSet(plans)
	if len(got) != 3 || !got[data.Done] || !got[data.Todo] || !got[data.Doing] {
		t.Errorf("expanded = %v", got)
	}
}

func TestPlanColumnsNarrowsThenGivesWay(t *testing.T) {
	all := map[data.Status]int{data.Backlog: 1, data.Todo: 1, data.Doing: 1, data.Blocked: 1, data.Review: 1, data.Done: 1}
	// Everything fits at full width.
	if got := expandedSet(planColumns(1700, all, nil)); len(got) != 5 {
		t.Errorf("1700: %v", got)
	}
	// Narrower: the five open columns shrink, none gives way yet.
	plans := planColumns(1250, all, nil)
	if len(expandedSet(plans)) != 5 || plans[data.Doing].width >= theme.ColumnWidth || plans[data.Doing].width < theme.MinColumnWidth {
		t.Errorf("1250: %v width %v", expandedSet(plans), plans[data.Doing].width)
	}
	// A 1100 pt window without the inspector: Backlog and Todo give way, the agent columns stay.
	got := expandedSet(planColumns(boardWidth(1100, false), all, nil))
	if len(got) != 3 || !got[data.Doing] || !got[data.Blocked] || !got[data.Review] {
		t.Errorf("1100 board: %v", got)
	}
	// With the inspector there is room for one column only: the forced one.
	got = expandedSet(planColumns(boardWidth(1100, true), all, map[data.Status]bool{data.Blocked: true}))
	if len(got) != 1 || !got[data.Blocked] {
		t.Errorf("1100 with inspector: %v", got)
	}
	// Forced columns never give way, even when they overflow.
	got = expandedSet(planColumns(300, all, map[data.Status]bool{data.Backlog: true, data.Todo: true}))
	if len(got) != 2 {
		t.Errorf("300: %v", got)
	}
}

func TestPlanColumnsEmptyBoard(t *testing.T) {
	if got := expandedSet(planColumns(1000, nil, nil)); len(got) != 0 {
		t.Errorf("an empty board has no full column: %v", got)
	}
}

func TestBoardWidth(t *testing.T) {
	if got := boardWidth(1100, false); got != 1100-221 {
		t.Errorf("no inspector: %v", got)
	}
	if got := boardWidth(1100, true); got != 1100-221-361 {
		t.Errorf("inspector: %v", got)
	}
}

func TestStripsAtDefaultWidthKeepAttentionColumns(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	for _, want := range []string{"Port core to Go", "Fix tray race on launch", "Native UI with mygo"} {
		if !tt.HasText(want) {
			t.Errorf("no card %q at 1100 wide: %q", want, tt.Texts())
		}
	}
	if tt.HasText("Write the release notes") {
		t.Error("Todo should be a strip at 1100 wide")
	}
}

func TestClickingAStripOpensItAndItsNameClosesIt(t *testing.T) {
	tt, m, _ := newMainTester(t, 1840)
	if tt.HasText("Spike the mygo shell") {
		t.Fatal("Done must start as a strip")
	}
	if err := tt.Click("Done"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Spike the mygo shell") || !m.pinned[m.nav][data.Done] {
		t.Fatalf("Done did not open: %q", tt.Texts())
	}
	if err := tt.Click("Done"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Spike the mygo shell") {
		t.Error("clicking the name did not collapse Done")
	}
}

func TestDroppingOnAStripChangesTheStatus(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	dragTo(t, tt, "Port core to Go", "Done")
	if task, _ := app.Task(1); task.Status != data.Done {
		t.Errorf("status %v after dropping on the Done strip", task.Status)
	}
}

func TestPlusAddsATaskToTheColumnInTheCurrentProject(t *testing.T) {
	tt, m, app := newMainTester(t, 1840)
	if err := tt.Click("Add task to Todo"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Task title…") {
		t.Fatalf("no inline input: %q", tt.Texts())
	}
	tt.Type("Buy a standing desk")
	tt.Key(0, ui.KeyEnter)
	var found *data.Task
	for _, task := range app.snap.tasks {
		if task.Title == "Buy a standing desk" {
			found = &task
		}
	}
	if found == nil {
		t.Fatal("task not created")
	}
	if found.Status != data.Todo || found.ProjectID == nil || *found.ProjectID != m.nav.projectID || found.Source != data.SourceUser {
		t.Errorf("created %+v", *found)
	}
	if m.adding != "" || tt.HasText("Task title…") {
		t.Error("the input stays open after Enter")
	}
}

func TestPlusInTheInboxAddsWithoutAProject(t *testing.T) {
	tt, _, app := newMainTester(t, 1840)
	if err := tt.Click("Inbox"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Add task to Backlog"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Sort the photos")
	tt.Key(0, ui.KeyEnter)
	for _, task := range app.snap.tasks {
		if task.Title == "Sort the photos" {
			if task.ProjectID != nil || task.Status != data.Backlog {
				t.Errorf("created %+v", task)
			}
			return
		}
	}
	t.Error("task not created")
}

func TestEscapeCancelsAddingAndKeepsTheInspector(t *testing.T) {
	tt, m, app := newMainTester(t, 1840)
	m.OpenTask(7)
	tt.Frame()
	before := len(app.snap.tasks)
	if err := tt.Click("Add task to Todo"); err != nil {
		t.Fatal(err)
	}
	tt.Type("never mind")
	tt.Key(0, ui.KeyEscape)
	if m.adding != "" || tt.HasText("Task title…") || len(app.snap.tasks) != before {
		t.Errorf("adding %q, tasks %d -> %d", m.adding, before, len(app.snap.tasks))
	}
	if m.openID != 7 {
		t.Error("the first Esc must only cancel the input")
	}
	tt.Key(0, ui.KeyEscape)
	if m.openID != 0 {
		t.Error("the second Esc closes the inspector")
	}
}

func TestEmptyTitleIsNotAdded(t *testing.T) {
	tt, m, app := newMainTester(t, 1840)
	before := len(app.snap.tasks)
	if err := tt.Click("Add task to Todo"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if len(app.snap.tasks) != before || m.adding != data.Todo {
		t.Error("an empty title must neither create a task nor close the input")
	}
}

func TestPlusOnAStripOpensTheColumnForAdding(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Add task to Todo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Task title…") || !tt.HasText("Write the release notes") || m.adding != data.Todo {
		t.Errorf("the Todo strip did not open for adding: %q", tt.Texts())
	}
}
