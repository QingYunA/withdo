package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func rightClick(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	if err := tt.RightClick(text); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
}

func click(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	if err := tt.Click(text); err != nil {
		t.Fatalf("%v in %q", err, tt.Texts())
	}
	tt.Frame()
}

func TestRightClickOpensOurTaskMenuAtThePointer(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	row, _ := tt.Find("Fix tray race on launch")
	rightClick(t, tt, "Fix tray race on launch")
	if !m.pop.open || m.pop.task != 3 {
		t.Fatalf("menu open %v for task %d", m.pop.open, m.pop.task)
	}
	for _, want := range []string{"Status", "Priority", "P0", "P3", "Move to…", "Remind me…", "Ask about this task…", "Delete", "⌘J", "⌘⌫",
		"Set status to Doing", "Set status to Cancelled", "Set priority to Urgent"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText("Open PR") {
		t.Error("Open PR for a task with no pull request")
	}
	if got := tt.Menu(); got != nil {
		t.Errorf("the system menu opened too: %q", got)
	}
	// It hangs from the pointer: right under it, in a 268 pt panel.
	move, _ := tt.Find("Move to…")
	if move.X < row.X || move.Y < row.Y+row.H/2 || move.X > row.X+row.W/2+popWidth {
		t.Errorf("menu items at %+v, pointer at the center of %+v", move, row)
	}
	// A task with a pull request has Open PR.
	tt.Key(0, ui.KeyEscape)
	rightClick(t, tt, "Native UI with mygo")
	if !tt.HasText("Open PR") {
		t.Errorf("no Open PR in %q", tt.Texts())
	}
}

func TestTaskMenuStripsSetStatusAndPriorityInOneClick(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Set status to In review")
	if got, _ := app.Task(3); got.Status != data.Review {
		t.Errorf("status %v", got.Status)
	}
	if m.pop.open {
		t.Error("the menu stays open after a choice")
	}
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Set priority to Low")
	if got, _ := app.Task(3); got.Priority != data.P3 {
		t.Errorf("priority %v", got.Priority)
	}
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Set status to Cancelled")
	if got, _ := app.Task(3); got.Status != data.Cancelled {
		t.Errorf("status %v, want a way to cancel", got.Status)
	}
}

func TestTaskMenuMovesToAProjectOrTheInboxFromAFilter(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Move to…")
	for _, want := range []string{"Inbox", "adhd-todo", "answer-me-with-html", "picpocket", "Find a project"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText("Remind me…") {
		t.Error("the main page stays under the Move to list")
	}
	tt.Type("pic")
	tt.Frame()
	if task, _ := app.Task(3); len(m.moveChoices(task)) != 1 || !tt.HasText("Move to picpocket") || tt.HasText("Move to Inbox") {
		t.Errorf("the filter did not narrow the list: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := taskProject(app, 3); got != "picpocket" {
		t.Errorf("project %q after Enter", got)
	}
	if m.pop.open {
		t.Error("the menu stays open after moving")
	}
	if tt.HasText("Fix tray race on launch") {
		t.Error("the task is still listed in the project it left")
	}
	m.goTo(nav{screen: screenProject, projectID: 3})
	tt.Frame()
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Move to…")
	click(t, tt, "Move to Inbox")
	if got, _ := app.Task(3); got.ProjectID != nil {
		t.Errorf("project %v, want the Inbox", got.ProjectID)
	}
}

func TestTaskMenuMoveListIsKeyboardNavigable(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Move to…")
	// Inbox, adhd-todo (the task's own), answer-me-with-html, picpocket: Down, Down, Down, Enter.
	for i := 0; i < 3; i++ {
		tt.Key(0, ui.KeyDown)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := taskProject(app, 3); got != "picpocket" {
		t.Errorf("project %q", got)
	}
}

func TestTaskMenuRemindsFromPresets(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Remind me…")
	for _, want := range []string{"Later today", "6:00 PM", "Tomorrow", "9:00 AM", "Next Monday", "Mon 9:00 AM", "This weekend", "Sat 10:00 AM", "No reminder", "Custom…"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	click(t, tt, "Tomorrow")
	got, _ := app.Task(3)
	want := time.Date(2026, 10, 15, 9, 0, 0, 0, time.Local)
	if got.RemindAt == nil || !got.RemindAt.Equal(want) {
		t.Fatalf("remind at %v, want %v", got.RemindAt, want)
	}
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Remind me…")
	click(t, tt, "No reminder")
	if got, _ := app.Task(3); got.RemindAt != nil {
		t.Errorf("remind at %v, want none", got.RemindAt)
	}
}

func TestTaskMenuCustomReminderOpensACalendar(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Remind me…")
	click(t, tt, "Custom…")
	if m.pop.page != pageCalendar {
		t.Fatalf("page %v", m.pop.page)
	}
	m.pop.when = time.Date(2026, 10, 20, 15, 0, 0, 0, time.Local)
	tt.Frame()
	click(t, tt, "Set reminder")
	got, _ := app.Task(3)
	if got.RemindAt == nil || !got.RemindAt.Equal(m.pop.when) {
		t.Errorf("remind at %v, want %v", got.RemindAt, m.pop.when)
	}
}

func TestTaskMenuOpensThePullRequestAndAsksAboutTheTask(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	rightClick(t, tt, "Native UI with mygo")
	click(t, tt, "Open PR")
	if len(rec.urls) != 1 || rec.urls[0] != "https://github.com/QingYunA/adhd-todo/pull/12" {
		t.Errorf("opened %v", rec.urls)
	}
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Ask about this task…")
	if m.openID != 3 || !m.focusInspectorComposer || m.pop.open {
		t.Errorf("open %d, focus asked %v, menu open %v", m.openID, m.focusInspectorComposer, m.pop.open)
	}
}

func TestTaskMenuDeleteHasNoQuestionAndUndoRestoresTheTask(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	reportsOf3 := func() int { snap, _ := app.Store.Snapshot(); return len(snap.Reports[3]) }
	before := reportsOf3()
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Delete")
	if _, ok := app.Task(3); ok {
		t.Fatal("the task is still there")
	}
	if tt.HasText("Delete “Fix tray race on launch”?") || tt.HasText("Cancel") {
		t.Error("Delete asked first")
	}
	if !tt.HasText("Deleted “Fix tray race on launch”") || !tt.HasText("Undo") {
		t.Fatalf("no toast: %q", tt.Texts())
	}
	if m.pop.open {
		t.Error("the menu stays open after deleting")
	}
	click(t, tt, "Undo")
	got, ok := app.Task(3)
	if !ok || got.Title != "Fix tray race on launch" || got.Status != data.Blocked || got.Priority != data.P0 {
		t.Fatalf("restored %+v (%v)", got, ok)
	}
	if n := reportsOf3(); n != before || n == 0 {
		t.Errorf("reports %d, want %d", n, before)
	}
	if !tt.HasText("Fix tray race on launch") {
		t.Error("the task is not listed again")
	}
}

func TestTaskMenuClosesOnEscapeAndOnAPressOutside(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if m.pop.open {
		t.Error("Esc left the menu open")
	}
	rightClick(t, tt, "Fix tray race on launch")
	if !m.pop.open {
		t.Fatal("did not reopen")
	}
	tt.ClickAt(1050, 700) // empty page
	tt.Frame()
	if m.pop.open {
		t.Error("a press outside left the menu open")
	}
	// A right-click on another row moves the menu there.
	rightClick(t, tt, "Fix tray race on launch")
	rightClick(t, tt, "Write the release notes")
	if !m.pop.open || m.pop.task != 5 {
		t.Errorf("menu open %v for task %d, want it moved to task 5", m.pop.open, m.pop.task)
	}
}

func TestTaskMenuOnBoardCardsNowRowsSearchAndCompleted(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	m.board[1] = true
	tt.Frame()
	rightClick(t, tt, "Fix tray race on launch")
	if !m.pop.open || !tt.HasText("Move to…") {
		t.Errorf("card menu: open %v", m.pop.open)
	}
	tt.Key(0, ui.KeyEscape)
	m.goTo(nav{})
	tt.Frame()
	rightClick(t, tt, "Book dentist")
	if !m.pop.open || m.pop.task != 8 {
		t.Errorf("Now row menu: open %v for %d", m.pop.open, m.pop.task)
	}
	tt.Key(0, ui.KeyEscape)
	m.goTo(nav{screen: screenCompleted})
	tt.Frame()
	rightClick(t, tt, "Draft the UI spec")
	if !m.pop.open || m.pop.task != 13 {
		t.Errorf("Completed row menu: open %v for %d", m.pop.open, m.pop.task)
	}
}

func TestGlyphOfMyOwnTaskCompletesItAndAnAgentsOpensTheMenu(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	tt.SetSize(1100, 1200) // room under the glyph of Done today, which is near the bottom
	click(t, tt, "Mark done: Book dentist")
	if got, _ := app.Task(8); got.Status != data.Done {
		t.Errorf("status %v after one click on the glyph", got.Status)
	}
	if m.pop.open || m.openID != 0 {
		t.Errorf("menu open %v, inspector %d: the click should do one thing", m.pop.open, m.openID)
	}
	click(t, tt, "Status: Done") // Port the MCP server, an agent's, in Done today
	if !m.pop.open || m.pop.task != 20 || !tt.HasText("Set status to Doing") {
		t.Errorf("an agent's glyph: menu open %v for %d", m.pop.open, m.pop.task)
	}
	if got, _ := app.Task(20); got.Status != data.Done {
		t.Errorf("an agent's task changed to %v by a click on its glyph", got.Status)
	}
	// The menu hangs under the glyph.
	g, _ := tt.Find("Status: Done")
	s, _ := tt.Find("Set status to To do")
	if s.Y < g.Y {
		t.Errorf("menu strip at %+v is above the glyph at %+v", s, g)
	}
}

func TestSelectedRowKeysOpenTheMenuOnOnePiece(t *testing.T) {
	for _, c := range []struct {
		key  ui.Key
		page menuPage
		text string
	}{
		{ui.KeyS, pageStatus, "Set status to Doing"},
		{ui.KeyP, pagePriority, "Set priority to High"},
		{ui.KeyM, pageMove, "Find a project"},
		{ui.KeyR, pageRemind, "No reminder"},
	} {
		tt, m, _ := projectTester(t, 1)
		m.sel = 5
		tt.Frame()
		tt.Key(0, c.key)
		tt.Frame()
		if !m.pop.open || m.pop.page != c.page || m.pop.task != 5 || !tt.HasText(c.text) {
			t.Errorf("key %v: open %v page %v task %d, texts %q", c.key, m.pop.open, m.pop.page, m.pop.task, tt.Texts())
		}
		if c.page != pageMove && tt.HasText("Delete") {
			t.Errorf("key %v showed the whole menu", c.key)
		}
	}
}

func TestStatusStripIsKeyboardNavigable(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	m.sel = 5 // To do
	tt.Frame()
	tt.Key(0, ui.KeyS)
	tt.Frame()
	tt.Key(0, ui.KeyRight) // to do → doing
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got, _ := app.Task(5); got.Status != data.Doing {
		t.Errorf("status %v", got.Status)
	}
	if m.pop.open {
		t.Error("Enter left the menu open")
	}
}

func TestCmdJAsksAboutTheSelectedTaskAndCmdBackspaceDeletesIt(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	m.sel = 5
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyJ)
	tt.Frame()
	if m.openID != 5 || !m.focusInspectorComposer {
		t.Errorf("open %d, focus asked %v", m.openID, m.focusInspectorComposer)
	}
	tt.Key(ui.Cmd, ui.KeyBackspace)
	tt.Frame()
	if _, ok := app.Task(5); ok {
		t.Error("Cmd+Backspace left the task")
	}
	if !tt.HasText("Undo") {
		t.Errorf("no Undo toast: %q", tt.Texts())
	}
	if m.openID != 0 {
		t.Error("the inspector stays on a deleted task")
	}
}

func TestInspectorValuesOpenOnlyTheirPieceOfTheMenu(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(8) // Book dentist, in the Inbox
	tt.Frame()
	click(t, tt, "Status: To do")
	if !tt.HasText("Set status to Done") || tt.HasText("Move to…") {
		t.Errorf("the Status value should show the strip alone: %q", tt.Texts())
	}
	click(t, tt, "Set status to Doing")
	if got, _ := app.Task(8); got.Status != data.Doing {
		t.Errorf("status %v", got.Status)
	}
	click(t, tt, "Priority: P2  Normal")
	if !tt.HasText("Set priority to High") || tt.HasText("Set status to Done") {
		t.Errorf("the Priority value should show the priority strip alone: %q", tt.Texts())
	}
	click(t, tt, "Set priority to High")
	if got, _ := app.Task(8); got.Priority != data.P1 {
		t.Errorf("priority %v", got.Priority)
	}
	click(t, tt, "Project: Inbox")
	click(t, tt, "Move to picpocket")
	if got := taskProject(app, 8); got != "picpocket" {
		t.Errorf("project %q", got)
	}
	click(t, tt, "Remind: None")
	click(t, tt, "9:00 AM")
	if got, _ := app.Task(8); got.RemindAt == nil || got.RemindAt.Day() != 15 {
		t.Errorf("remind at %v", got.RemindAt)
	}
}

func TestInspectorMovingATaskDropsItsMilestone(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(5)
	tt.Frame()
	if !tt.HasText("Milestone: v0.2 native") {
		t.Fatalf("task 5 has no milestone row: %q", tt.Texts())
	}
	click(t, tt, "Project: adhd-todo")
	click(t, tt, "Move to picpocket")
	if got, _ := app.Task(5); got.MilestoneID != nil {
		t.Errorf("the milestone came along: %v", *got.MilestoneID)
	}
}

func TestInspectorDeleteTaskIsImmediateWithUndo(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(8)
	tt.Frame()
	click(t, tt, "Delete task…")
	if _, ok := app.Task(8); ok {
		t.Error("the task is still there")
	}
	if tt.HasText("Delete “Book dentist”?") || !tt.HasText("Deleted “Book dentist”") {
		t.Errorf("texts %q", tt.Texts())
	}
	if m.openID != 0 {
		t.Errorf("the inspector stays open on a deleted task: %d", m.openID)
	}
	click(t, tt, "Undo")
	if _, ok := app.Task(8); !ok {
		t.Error("Undo did not bring the task back")
	}
}

func TestTaskMenuLabelsNameTheHoveredChoiceInline(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	rightClick(t, tt, "Write the release notes")
	has(t, tt, "· To do", "· P2 Normal")
	g, _ := tt.Find("Set status to Doing")
	tt.Move(center(g))
	tt.Frame()
	tt.Frame()
	has(t, tt, "· Doing", "· P2 Normal")
	hasNot(t, tt, "· To do")
	p, _ := tt.Find("Set priority to Urgent")
	tt.Move(center(p))
	tt.Frame()
	tt.Frame()
	has(t, tt, "· To do", "· P0 Urgent")
}

func TestMoveToStaysInTheWindowAndDoesNotJumpWhileFiltering(t *testing.T) {
	tt, _, _ := projectTester(t, 1) // 720 pt tall; the last row is near the bottom
	rightClick(t, tt, "Port the reminder watcher")
	click(t, tt, "Move to…")
	last, ok := tt.Find("Move to picpocket")
	if !ok {
		t.Fatalf("no picpocket in %q", tt.Texts())
	}
	if bottom := last.Y + last.H; bottom > 720-popEdge+1 {
		t.Errorf("the list ends at %.0f, past the window's edge", bottom)
	}
	before, _ := tt.Find("Find a project")
	tt.Type("p")
	tt.Frame()
	after, _ := tt.Find("Find a project")
	if before.Y != after.Y {
		t.Errorf("the filter moved from %.0f to %.0f while typing", before.Y, after.Y)
	}
}

func TestCmdZRestoresATaskDeletedOnItsOwn(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	m.sel = 5
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyBackspace)
	tt.Frame()
	if _, ok := app.Task(5); ok {
		t.Fatal("Cmd+Backspace left the task")
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	tt.Frame()
	if _, ok := app.Task(5); !ok {
		t.Error("Cmd+Z did not bring the task back")
	}
	if tt.HasText("Undo") {
		t.Error("the Undo toast stays after Cmd+Z")
	}
}
