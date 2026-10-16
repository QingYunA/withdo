package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
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

func TestSubRowLabelsLineUpWithTheirParents(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	mine, _ := tt.Find("Mine")
	work, _ := tt.Find("Work")
	agents, _ := tt.Find("Agents")
	claude, _ := tt.Find("Claude Code")
	if work.X != mine.X || claude.X != agents.X {
		t.Errorf("labels at Mine %v, Work %v, Agents %v, Claude Code %v: a sub-row's label starts where its parent's does", mine.X, work.X, agents.X, claude.X)
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
	if m.nav != (nav{screen: screenMine, sub: subWork}) {
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
	m.goTo(nav{screen: screenMine, sub: subLife})
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
	if _, err := store.UpdateTask(18, data.Patch{Category: data.CategoryPtr(data.CategoryWork)}); err != nil {
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
	m.goTo(nav{screen: screenMine, sub: subUnsorted}) // it stays while it is the page
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
	m.goTo(nav{screen: screenMine, sub: subLife})
	tt.Frame()
	if place := m.placeOf(surfacePage); place.Kind != data.PlaceNow || place.Category != data.CategoryLife {
		t.Fatalf("place %+v", place)
	}
	m.goTo(nav{screen: screenMine, sub: subUnsorted})
	if place := m.placeOf(surfacePage); place.Category != "" {
		t.Errorf("Unsorted page place %+v", place)
	}
	m.goTo(nav{screen: screenMine, sub: subWork})
	tt.Frame()
	m.cmp[surfacePage].buf = "Call the landlord"
	m.submit(surfacePage)
	var made data.Task
	for _, task := range app.snap.tasks {
		if task.Title == "Call the landlord" {
			made = task
		}
	}
	if made.Category != data.CategoryWork {
		t.Errorf("a task typed on the Work page: %+v", made)
	}
}
