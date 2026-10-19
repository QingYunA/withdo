package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// The sidebar's sub-rows: Work, Life and Unsorted under Mine, one row per agent under Agents.

// onlyInSidebar fails when a text is drawn anywhere but in its row of the sidebar, which lists every
// agent at work; none is there when no agent works.
func onlyInSidebar(t *testing.T, tt *ui.Tester, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if n := countText(tt, text); n > 1 {
			t.Errorf("%q is drawn %d times: only its sidebar row may say it, in %q", text, n, tt.Texts())
		}
	}
}

func TestSidebarSplitsMineIntoWorkLifeAndUnsorted(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	for _, want := range []string{"Work", "Life", "Unsorted"} {
		if !tt.HasText(want) {
			t.Fatalf("no %q in %q", want, tt.Texts())
		}
	}
	for _, pair := range [][2]string{{"Mine", "Work"}, {"Work", "Life"}, {"Life", "Unsorted"}, {"Unsorted", "Agents"}} {
		if !above(t, tt, pair[0], pair[1]) {
			t.Errorf("%q should be above %q", pair[0], pair[1])
		}
	}
}

func TestSubRowsAreIndentedOneStepUnderTheirParents(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	mine, _ := tt.Find("Mine")
	work, _ := tt.Find("Work")
	agents, _ := tt.Find("Agents")
	claude, _ := tt.Find("Claude Code")
	// The sub-row's 12 pt slot starts where the parent's label does, and its label follows the slot.
	in := theme.AvatarSmall + sidebarRowGap
	if work.X != mine.X+in || claude.X != agents.X+in {
		t.Errorf("labels at Mine %v, Work %v, Agents %v, Claude Code %v: a sub-row's label starts %v pt in from its parent's",
			mine.X, work.X, agents.X, claude.X, in)
	}
}

func TestSubRowCountsAreTheSlicesOfTheParentCount(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	// Mine's 4 are two handed-over rows and two due today, all work: the seed's life task is later
	// and its unsorted one is undated.
	got := tt.Texts()
	idx := func(s string) int {
		for i, x := range got {
			if x == s {
				return i
			}
		}
		return -1
	}
	if got[idx("Mine")+1] != "4" || got[idx("Work")+1] != "4" || got[idx("Life")+1] != "Unsorted" {
		t.Errorf("counts beside Mine, Work and Life: %q", got[:12])
	}
}

func TestWorkOpensMineShowingOnlyWork(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Work"); err != nil {
		t.Fatal(err)
	}
	if m.nav != (nav{screen: screenMine, cat: workID}) {
		t.Fatalf("nav %+v", m.nav)
	}
	for _, want := range []string{"Reply to design feedback", "Decide on the activation policy", "Fix tray race on launch"} {
		if !tt.HasText(want) {
			t.Errorf("Work lacks %q in %q", want, tt.Texts())
		}
	}
	for _, not := range []string{"Book dentist", "Look into Liquid Glass for the sidebar"} {
		if tt.HasText(not) {
			t.Errorf("Work shows %q", not)
		}
	}
	if err := tt.Click("Life"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Book dentist") || tt.HasText("Reply to design feedback") {
		t.Errorf("Life shows %q", tt.Texts())
	}
	if err := tt.Click("Unsorted"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Look into Liquid Glass for the sidebar") || tt.HasText("Book dentist") {
		t.Errorf("Unsorted shows %q", tt.Texts())
	}
}

func TestASliceTitlesItsPageAndCommandOneGoesToTheWholeOfMine(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenMine, cat: lifeID})
	tt.Frame()
	if n := countText(tt, "Life"); n != 2 {
		t.Errorf("%d times Life: the sidebar row and the page title, in %q", n, tt.Texts())
	}
	tt.Key(ui.Cmd, ui.Key1)
	if m.nav != (nav{}) || !tt.HasText("Reply to design feedback") || !tt.HasText("Book dentist") {
		t.Errorf("Cmd+1 went to %+v", m.nav)
	}
}

func TestUnsortedIsOnlyThereWhileItHasRows(t *testing.T) {
	app, store, _ := newTestApp(t)
	if _, err := store.UpdateTask(18, data.Patch{Category: data.IntPtr(workID)}); err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	if tt.HasText("Unsorted") {
		t.Fatalf("Unsorted with nothing in it: %q", tt.Texts())
	}
	if !tt.HasText("Work") || !tt.HasText("Life") {
		t.Errorf("Work and Life are always there: %q", tt.Texts())
	}
	m.goTo(nav{screen: screenMine, cat: catUnsorted}) // it stays while it is the page
	tt.Frame()
	if !tt.HasText("Unsorted") {
		t.Errorf("the open Unsorted page lost its row: %q", tt.Texts())
	}
}

func TestAgentsGetOneSubRowEach(t *testing.T) {
	app, store, _ := newTestApp(t)
	store.ReportAs(15, "codex", "Looking at the parser", fixedNow().Add(-time.Minute))
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	if !tt.HasText("Claude Code") || !tt.HasText("Codex") {
		t.Fatalf("agents in the sidebar: %q", tt.Texts())
	}
	if !above(t, tt, "Agents", "Codex") || !above(t, tt, "Codex", "Claude Code") {
		t.Error("Codex reported last, so it is first, under Agents")
	}
	if err := tt.Click("Codex"); err != nil {
		t.Fatal(err)
	}
	if m.nav != (nav{screen: screenAgents, sub: "Codex"}) {
		t.Fatalf("nav %+v", m.nav)
	}
	if !tt.HasText("Issue #218 adaptation") || tt.HasText("Port core to Go") {
		t.Errorf("Codex page: %q", tt.Texts())
	}
	if err := tt.Click("Claude Code"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Port core to Go") || tt.HasText("Issue #218 adaptation") {
		t.Errorf("Claude Code page: %q", tt.Texts())
	}
	// A subtask goes where its parent goes.
	if !tt.HasText("Port the MCP tools") {
		t.Errorf("subtasks of Port core to Go: %q", tt.Texts())
	}
}

func TestOneAgentUnderTwoSpellingsIsOneRow(t *testing.T) {
	app, store, _ := newTestApp(t)
	store.ReportAs(15, "Claude Code", "Writing tests", fixedNow().Add(-time.Minute))
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	if n := countText(tt, "Claude Code"); n != 1 {
		t.Errorf("%d rows for Claude Code: %q", n, tt.Texts())
	}
}

func TestAnAgentSubRowCountsItsTasksAtWork(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	got := tt.Texts()
	for i, s := range got {
		if s == "Claude Code" {
			if got[i+1] != "3" {
				t.Errorf("count beside Claude Code: %q", got[i:i+3])
			}
			return
		}
	}
	t.Fatalf("no Claude Code row in %q", got)
}

func TestTheComposerOfASlicePutsNewTasksThere(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.goTo(nav{screen: screenMine, cat: lifeID})
	tt.Frame()
	if place := m.placeOf(surfacePage); place.Kind != data.PlaceNow || place.Category != lifeID {
		t.Fatalf("place %+v", place)
	}
	m.goTo(nav{screen: screenMine, cat: catUnsorted})
	if place := m.placeOf(surfacePage); place.Category != 0 {
		t.Errorf("Unsorted page place %+v", place)
	}
	m.goTo(nav{screen: screenMine, cat: workID})
	tt.Frame()
	m.cmp[surfacePage].buf = "Call the landlord"
	m.submit(surfacePage)
	var made data.Task
	for _, task := range app.snap.tasks {
		if task.Title == "Call the landlord" {
			made = task
		}
	}
	if made.CategoryID != workID {
		t.Errorf("a task typed on the Work page: %+v", made)
	}
}

// addCategory makes a category of the user's own with one lit pixel and refreshes the app.
func addCategory(t *testing.T, app *App, name, color string) data.Category {
	t.Helper()
	c, err := app.CreateCategory(data.NewCategory{Name: name, Icon: pixmap.Bitmap{}.Set(3, 3, true).Picture(), Color: color})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestEveryCategoryHasASubRowByPositionAndUnsortedComesLast(t *testing.T) {
	app, _, _ := newTestApp(t)
	addCategory(t, app, "Reading", "orange") // nothing is sorted into it: its row is there all the same
	addCategory(t, app, "Health", "red")
	tt, _, _ := newTesterFor(app, 1100)
	for _, pair := range [][2]string{{"Mine", "Work"}, {"Work", "Life"}, {"Life", "Reading"}, {"Reading", "Health"}, {"Health", "Unsorted"}, {"Unsorted", "Agents"}} {
		if !above(t, tt, pair[0], pair[1]) {
			t.Errorf("%q should be above %q in %q", pair[0], pair[1], tt.Texts())
		}
	}
}

func TestACategorySubRowDrawsItsIconInItsColor(t *testing.T) {
	app, _, _ := newTestApp(t)
	addCategory(t, app, "Reading", "orange")
	tt, _, _ := newTesterFor(app, 1100)
	for _, row := range []struct {
		label string
		color ui.Color
	}{{"Work", theme.CategoryColor("blue")}, {"Life", theme.CategoryColor("green")}, {"Reading", theme.CategoryColor("orange")}} {
		r, ok := tt.Find(row.label)
		if !ok {
			t.Fatalf("no %q in %q", row.label, tt.Texts())
		}
		slot := ui.Rect{X: r.X - sidebarRowGap - theme.AvatarSmall, Y: r.Y, W: theme.AvatarSmall, H: r.H}
		if !paintedIn(tt.Image(), 1100, slot, row.color, 30) {
			t.Errorf("%s: no %v pixel in its icon slot", row.label, row.color)
		}
	}
}

func TestACategoryRowOpensItsPageWithItsTasks(t *testing.T) {
	app, _, _ := newTestApp(t)
	reading := addCategory(t, app, "Reading", "orange")
	_ = app.Update(18, data.Patch{Category: data.IntPtr(reading.ID)})
	tt, m, _ := newTesterFor(app, 1100)
	click(t, tt, "Reading")
	if m.nav != (nav{screen: screenMine, cat: reading.ID}) {
		t.Fatalf("nav %+v", m.nav)
	}
	if !tt.HasText("Look into Liquid Glass for the sidebar") || tt.HasText("Reply to design feedback") {
		t.Errorf("Reading page: %q", tt.Texts())
	}
	if n := countText(tt, "Reading"); n != 2 {
		t.Errorf("%d times Reading: the sidebar row and the page title", n)
	}
}

func TestThePageOfADeletedCategoryFallsBackToMine(t *testing.T) {
	app, _, _ := newTestApp(t)
	reading := addCategory(t, app, "Reading", "orange")
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenMine, cat: reading.ID})
	tt.Frame()
	if _, err := app.DeleteCategory(reading.ID); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.nav != (nav{}) {
		t.Errorf("nav %+v: the page of a deleted category is Mine", m.nav)
	}
}

func TestTheComposerOfAUserCategoryFilesTasksThere(t *testing.T) {
	app, _, _ := newTestApp(t)
	reading := addCategory(t, app, "Reading", "orange")
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenMine, cat: reading.ID})
	tt.Frame()
	m.cmp[surfacePage].buf = "Finish the novel"
	m.submit(surfacePage)
	for _, task := range app.snap.tasks {
		if task.Title == "Finish the novel" {
			if task.CategoryID != reading.ID {
				t.Errorf("a task typed on the Reading page: %+v", task)
			}
			return
		}
	}
	t.Error("the task was not created")
}
