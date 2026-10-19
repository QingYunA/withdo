package views

import (
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestMineShowsItsSectionsInOrder(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	for _, want := range []string{"Mine", "Wednesday, Oct 14", "From your agents", "Today", "Later", "Fix tray race on launch", "Book dentist", "Done today"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if !above(t, tt, "From your agents", "Today") || !above(t, tt, "Today", "Later") || !above(t, tt, "Later", "Done today") {
		t.Error("sections are out of order")
	}
	for _, gone := range []string{"Port core to Go", "Port the MCP tools", "Needs you", "Now"} {
		if tt.HasText(gone) {
			t.Errorf("%q is on Mine: agents' work in progress and their subtasks are on Agents", gone)
		}
	}
}

func TestMineEmptySectionsAreNotDrawn(t *testing.T) {
	f := demoStore(t, fixedNow, false)
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "Only mine"})
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, _, _ := newTesterFor(app, 1100)
	if !tt.HasText("Later") || !tt.HasText("Only mine") {
		t.Fatalf("texts %q", tt.Texts())
	}
	for _, gone := range []string{"From your agents", "Today", "Nothing needs you right now"} {
		if tt.HasText(gone) {
			t.Errorf("%q is drawn with nothing in it", gone)
		}
	}
}

func TestMineAllClearWhenNothingIsHappening(t *testing.T) {
	app, rec := newEmptyApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	for _, want := range []string{"Nothing needs you right now", "Your agents are at work. Whatever happens shows up here.", "How to connect"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	for _, gone := range []string{"From your agents", "Today", "Later"} {
		if tt.HasText(gone) {
			t.Errorf("%q in the All clear state", gone)
		}
	}
	if err := tt.Click("How to connect"); err != nil {
		t.Fatal(err)
	}
	if len(rec.panes) != 1 || rec.panes[0] != PaneAgents {
		t.Errorf("How to connect asked for %v, want the Agents pane", rec.panes)
	}
}

func TestFromYourAgentsOrderIsBlockedThenReview(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	if !above(t, tt, "Fix tray race on launch", "Native UI with mygo") {
		t.Errorf("order wrong in %q", tt.Texts())
	}
	for _, want := range []string{
		"Claude Code is stuck: “Needs a decision on the activation policy” · adhd-todo",
		"PR #12 is ready for your review · adhd-todo",
		"Answer", "Open PR",
	} {
		if !tt.HasText(want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestMineRowsLeaveTheBranchOutAndAgentsShowsIt(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if tt.HasText("fix/tray") || tt.HasText("feat/native-ui") {
		t.Errorf("a handed-over row shows a branch: %q", tt.Texts())
	}
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	if !tt.HasText("Claude Code · feat/go-core · Ported the store and migrations; next: port the MCP server") {
		t.Errorf("a Working row lost its branch: %q", tt.Texts())
	}
}

func TestTightRowsDropTheirCompanions(t *testing.T) {
	tt, m, _ := newMainTester(t, 860)
	m.goTo(nav{screen: screenAgents})
	m.openRow(2)
	tt.Frame()
	for _, s := range tt.Texts() {
		if strings.Contains(s, "feat/go-core") || strings.Contains(s, "claude/issue-218") {
			t.Errorf("a tight row keeps its companions: %q", s)
		}
	}
	if !tt.HasText("Port core to Go") {
		t.Error("a tight row lost its title")
	}
}

func TestClickingAHandedOverRowOpensTheInspector(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Fix tray race on launch"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 3 || !tt.HasText("Activity") {
		t.Errorf("open %d, texts %q", m.openID, tt.Texts())
	}
}

func TestHandedOverOpenPRCallsOpenURL(t *testing.T) {
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

func TestNotNowPutsAHandedOverRowAsideUntilTheAgentReportsAgain(t *testing.T) {
	app, store, _ := newTestApp(t)
	kept := &data.MemorySeen{}
	app.Dismissed = kept
	tt, m, _ := newTesterFor(app, 1100)
	before := m.mine.Count()
	if err := tt.Click("Not now: Fix tray race on launch"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("Fix tray race on launch") || m.mine.Count() != before-1 {
		t.Errorf("count %d, want %d, and the row gone: %q", m.mine.Count(), before-1, tt.Texts())
	}
	if m.openID != 0 {
		t.Error("Not now also opened the inspector")
	}
	if task, _ := app.Task(3); task.Status != data.Blocked {
		t.Errorf("status %v: putting a row aside must not change the task", task.Status)
	}
	if saved, _ := kept.LoadSeen(); saved[3].IsZero() {
		t.Errorf("saved %v: the dismissal was not kept", saved)
	}
	// The record is read again by a new launch.
	again := NewApp(store, &recorder{}, testConfig())
	again.Now, again.Dismissed = fixedNow, kept
	tt2, _, _ := newTesterFor(again, 1100)
	if tt2.HasText("Fix tray race on launch") {
		t.Error("a dismissed row is back after a restart")
	}
	// The agent reports again: the row comes back.
	store.SeedReportAs(3, "claude-code", "Still stuck", fixedNow().Add(time.Minute))
	app.Refresh()
	tt.Frame()
	if !tt.HasText("Fix tray race on launch") {
		t.Errorf("a newer report should bring the row back: %q", tt.Texts())
	}
}

func TestNotNowLeavesTheTaskOnTheAgentsScreen(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Not now: Fix tray race on launch"); err != nil {
		t.Fatal(err)
	}
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	if !tt.HasText("Fix tray race on launch") {
		t.Errorf("Agents lost the task: %q", tt.Texts())
	}
}

func TestAcceptOnAPullRequestRowMarksTheTaskDone(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	if !tt.HasText("Open PR") || !tt.HasText("Accept") {
		t.Fatalf("a review row with a pull request offers Open PR and Accept: %q", tt.Texts())
	}
	if err := tt.Click("Accept: Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	if task, _ := app.Task(2); task.Status != data.Done {
		t.Errorf("status %v, want done", task.Status)
	}
	if m.openID != 0 {
		t.Error("Accept opened the inspector")
	}
}

func TestAReviewRowWithoutAPullRequestOffersReviewAndAccept(t *testing.T) {
	app, store, _ := newTestApp(t)
	task, err := store.CreateTask(data.CreateTaskInput{Title: "Tidy the docs", Source: data.SourceAgent, Status: data.Review})
	if err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, "Review", "Accept")
	if err := tt.Click("Accept: Tidy the docs"); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.Task(task.ID); got.Status != data.Done {
		t.Errorf("status %v, want done", got.Status)
	}
}

func TestTheGlyphOfMyDueReminderMarksItDone(t *testing.T) {
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
	if !tt.HasText("Done today") || !above(t, tt, "Done today", "Reply to design feedback") {
		t.Error("the done reminder should have moved from Today to Done today")
	}
}

func TestClickingARowOpensTheInspectorAndEscCloses(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	if err := tt.Click("Port core to Go"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 1 || !tt.HasText("Activity") {
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
	if !tt.HasText("Added to Inbox") {
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

func TestArrowsSelectEnterOpensSpaceToggles(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	tt.Key(0, ui.KeyDown)
	if m.sel != 3 {
		t.Fatalf("selected %d after one Down, want the blocked task 3", m.sel)
	}
	tt.Key(0, ui.KeyDown)
	if m.sel != 2 {
		t.Fatalf("selected %d after two Down, want the review task 2", m.sel)
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
	tt.Key(0, ui.KeyDown)
	if m.sel != 7 {
		t.Fatalf("selected %d, want the reminder 7", m.sel)
	}
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
	if m.cmp[surfacePage].buf != "typed" {
		t.Errorf("the ask field did not take the focus: %q", m.cmp[surfacePage].buf)
	}
}

func TestSpaceInTheAskFieldIsASpace(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("a b")
	if m.cmp[surfacePage].buf != "a b" {
		t.Errorf("buffer %q", m.cmp[surfacePage].buf)
	}
	if task, _ := app.Task(7); task.Status == data.Done {
		t.Error("Space toggled a task while typing")
	}
}

func TestCmdNumbersSwitchScreens(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.Key2)
	if m.nav.screen != screenAgents {
		t.Errorf("Cmd+2 gave %+v", m.nav)
	}
	tt.Key(ui.Cmd, ui.Key3)
	if m.nav.screen != screenInbox || !tt.HasText("Book dentist") || tt.HasText("Tasks without a project") {
		t.Errorf("Cmd+3 gave %+v", m.nav)
	}
	tt.Key(ui.Cmd, ui.Key1)
	if m.nav.screen != screenMine {
		t.Errorf("Cmd+1 gave %+v", m.nav)
	}
}

func TestMineLinksToTheInboxWhenUndatedTasksDoNotFit(t *testing.T) {
	f := demoStore(t, fixedNow, false)
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
