package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func boardTester(t *testing.T, app *App, project, width int) (*ui.Tester, *Main) {
	t.Helper()
	tt, m, _ := newTesterFor(app, width)
	m.goTo(nav{screen: screenProject, projectID: project})
	m.board[project] = true
	tt.Frame()
	return tt, m
}

func statusesOf(cols []boardColumn) map[data.Status]bool {
	out := map[data.Status]bool{}
	for _, c := range cols {
		out[c.status] = true
	}
	return out
}

func TestBoardShowsOnlyNonEmptyColumnsPlusToDo(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100) // answer-me-with-html: one doing, one backlog
	got := statusesOf(m.boardColumns())
	if len(got) != 2 || !got[data.Todo] || !got[data.Doing] {
		t.Errorf("columns %v, want to do and doing", got)
	}
	for _, gone := range []string{"Blocked", "In review", "Done"} {
		if tt.HasText(gone) {
			t.Errorf("empty column %q is drawn", gone)
		}
	}
	if !tt.HasText("To do") || !tt.HasText("Nothing waiting. Drag a card here to plan it.") {
		t.Errorf("To do is missing or has no hint: %q", tt.Texts())
	}
}

func TestBoardKeepsBacklogAndDoneBehindTheFoldLine(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 1, 1100)
	if got := statusesOf(m.boardColumns()); len(got) != 4 || got[data.Backlog] || got[data.Done] {
		t.Fatalf("columns %v, want to do, doing, blocked, review", got)
	}
	if err := tt.Click("show Done 5 and Backlog 2"); err != nil {
		t.Fatal(err)
	}
	if got := statusesOf(m.boardColumns()); len(got) != 6 {
		t.Errorf("after show: %v", got)
	}
	if !tt.HasText("Backlog") || !tt.HasText("Spike the mygo shell") {
		t.Errorf("Backlog and Done are not drawn: %q", tt.Texts())
	}
}

func TestBoardToDoIsThereOnAnEmptyColumnSet(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("scratch", "me/scratch")
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "In flight", Status: data.Doing, ProjectID: data.IntPtr(p.ID)})
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	_, m := boardTester(t, app, p.ID, 1100)
	cols := m.boardColumns()
	if len(cols) != 2 || cols[0].status != data.Todo || cols[1].status != data.Doing {
		t.Errorf("columns %+v", cols)
	}
}

func TestBoardHasNoCollapsedStrips(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 1, 1840)
	m.expanded[m.nav] = true
	tt.Frame()
	if len(m.boardColumns()) != 6 {
		t.Fatalf("columns %d, want all six that hold tasks", len(m.boardColumns()))
	}
	for _, want := range []string{"Backlog", "To do", "Doing", "Blocked", "In review", "Done"} {
		if !tt.HasText(want) {
			t.Errorf("missing column %q", want)
		}
	}
}

// The header and the composer are one block, and the columns fill exactly that block, centered;
// only when the narrowest columns do not fit does the board scroll under a full-width header.
func TestBoardColumnsFillTheBlockOfTheHeaderAndTheComposer(t *testing.T) {
	const list = 684 // the lists' column: 780 less 2 x 48
	for _, tc := range []struct {
		avail  float32
		n      int
		scroll bool
	}{{883, 4, false}, {1500, 4, false}, {1500, 3, false}, {1500, 2, false}, {900, 6, true}, {1500, 1, false}} {
		got := boardMetricsFor(tc.avail, list, tc.n)
		gaps := float32(tc.n-1) * 12
		if tc.scroll {
			if got.col != 192 || got.head != tc.avail-48 {
				t.Errorf("%+v: %+v, want the narrowest columns under a full-width header", tc, got)
			}
			continue
		}
		if got.col < 192 || got.col*float32(tc.n)+gaps != got.head || 2*got.edge+got.head != tc.avail {
			t.Errorf("%+v: %+v, columns do not fill a centered block", tc, got)
		}
		if got.edge < 24 {
			t.Errorf("%+v: %+v, closer than 24 pt to the window", tc, got)
		}
	}
	if got := boardMetricsFor(1500, list, 3); got.col != 264 {
		t.Errorf("three columns in a wide window: %v wide, want the 264 maximum", got.col)
	}
}

func TestBoardHeaderComposerAndColumnsShareTheirEdges(t *testing.T) {
	for _, width := range []int{1100, 1840} {
		app, _, _ := newTestApp(t)
		tt, _ := boardTester(t, app, 1, width)
		lay := boardMetricsFor(float32(width)-217, 684, 4) // the sidebar and its hairline are 217 pt
		left := 217 + lay.edge
		right := left + lay.head
		toggle, _ := tt.Find("Board view")
		send, _ := tt.Find("Send")
		chip, _ := tt.Find("# adhd-todo")
		first, _ := tt.Find("To do")
		near := func(what string, got, want float32) {
			t.Helper()
			if got < want-1.5 || got > want+1.5 {
				t.Errorf("%d pt window: %s at %.1f, want %.1f", width, what, got, want)
			}
		}
		// The offsets are the padding inside each: the card's 16 and border, the chip's 8, the
		// toggle's 2, the column's 8, 6, the glyph and its gap.
		near("composer left", chip.X-8-16-1, left)
		near("composer right", send.X+send.W+16+1, right)
		near("toggle right", toggle.X+toggle.W+2, right)
		near("first column left", first.X-6-8-18-8, left)
	}
}

func TestDroppingACardChangesItsStatus(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1840)
	dragTo(t, tt, "Write the release notes", "Doing")
	if task, _ := app.Task(5); task.Status != data.Doing {
		t.Errorf("status %v after the drop, want doing", task.Status)
	}
}

func TestDroppingOnTheSameColumnChangesNothing(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1840)
	dragTo(t, tt, "Write the release notes", "To do")
	if task, _ := app.Task(5); task.Status != data.Todo {
		t.Errorf("status %v, want todo", task.Status)
	}
}

func TestDraggingRevealsDropTargetsForHiddenStatuses(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100)
	from, _ := tt.Find("Issue #218 adaptation")
	fx, fy := center(from)
	tt.Press(fx, fy)
	tt.Move(fx+8, fy+8)
	tt.Move(fx+20, fy+30)
	tt.Frame()
	if m.dragging == 0 {
		t.Fatal("no drag in progress")
	}
	if !tt.HasText("Blocked") || !tt.HasText("In review") || !tt.HasText("Done") {
		t.Fatalf("hidden statuses are not offered: %q", tt.Texts())
	}
	to, _ := tt.Find("In review")
	tx, ty := center(to)
	tt.Move(tx, ty)
	tt.Release(tx, ty)
	if task, _ := app.Task(15); task.Status != data.Review {
		t.Errorf("status %v after dropping on the hidden Review target", task.Status)
	}
	tt.Frame()
	if m.dragging != 0 {
		t.Error("drag state lingers")
	}
}

func TestBoardClickOpensTheInspectorAndAddsTasks(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100)
	if err := tt.Click("Issue #218 adaptation"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 15 {
		t.Errorf("opened %d", m.openID)
	}
	m.openID = 0
	tt.Frame()                                     // the columns take the room the inspector gave back
	if err := tt.Click("Add a task"); err != nil { // the + of the To do column puts the cursor in the composer
		t.Fatal(err)
	}
	tt.Type("Board added")
	tt.Key(0, ui.KeyEnter)
	found := false
	for _, task := range app.snap.tasks {
		found = found || task.Title == "Board added" && task.Status == data.Todo
	}
	if !found {
		t.Error("the To do column did not add a task")
	}
}

func TestViewToggleSwitchesBetweenListAndBoard(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	if err := tt.Click("Board view"); err != nil {
		t.Fatal(err)
	}
	if !m.board[1] || !tt.HasText("Nothing waiting") && !tt.HasText("Blocked") {
		t.Errorf("not a board: %q", tt.Texts())
	}
	if err := tt.Click("List view"); err != nil {
		t.Fatal(err)
	}
	if m.board[1] || tt.HasText("Nothing waiting. Drag a card here to plan it.") {
		t.Error("not back on the list")
	}
}

func TestMilestoneMenuFiltersTheProject(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	if !tt.HasText("All milestones") {
		t.Error("project 1 has milestones but no menu")
	}
	if err := tt.Click("Milestone: All milestones"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("v0.2 native · due Oct 28"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Decide on the activation policy") {
		t.Error("the milestone filter left a task of no milestone")
	}
}
