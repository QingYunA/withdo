package views

import (
	"slices"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
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

func laneOfTitles(m *Main, lane data.Status) []string {
	for _, col := range m.pg.Sections {
		if col.Status == lane {
			var out []string
			for _, t := range col.Tasks {
				out = append(out, t.Title)
			}
			return out
		}
	}
	return nil
}

func TestBoardHasFourFixedLanesWhateverTheProjectHolds(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100) // answer-me-with-html: one doing task, one backlog task
	var got []data.Status
	for _, col := range m.pg.Sections {
		got = append(got, col.Status)
	}
	if want := []data.Status{data.Todo, data.Doing, data.Review, data.Done}; !slices.Equal(got, want) {
		t.Fatalf("lanes %v, want %v", got, want)
	}
	has(t, tt, "To do", "Doing", "In review", "Done", "Issue #218 adaptation")
	hasNot(t, tt, "Blocked", "Backlog", "Cover the reply parser edge cases")
}

func TestEmptyLanesSayWhatTheyAreFor(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 2, 1100)
	has(t, tt, "Nothing to review", "Nothing yet")
	hasNot(t, tt, "Drag here to start") // Doing holds a card
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("scratch", "me/scratch")
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "Waiting", ProjectID: data.IntPtr(p.ID)})
	empty := NewApp(f, &recorder{}, testConfig())
	empty.Now = fixedNow
	tt2, _ := boardTester(t, empty, p.ID, 1100)
	has(t, tt2, "Drag here to start", "Nothing to review", "Nothing yet")
}

func TestBlockedCardsSitFirstInDoingAndBacklogIsNotOnTheBoard(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 1, 1100)
	doing := laneOfTitles(m, data.Doing)
	if want := []string{"Fix tray race on launch", "Port core to Go", "Decide on the activation policy"}; len(doing) != 3 || doing[0] != want[0] {
		t.Fatalf("Doing lane %v, want the blocked task first among %v", doing, want)
	}
	if !above(t, tt, "Fix tray race on launch", "Port core to Go") {
		t.Error("the blocked card is not above the others")
	}
	hasNot(t, tt, "Blocked", "Backlog", "Poll gh for PR status", "Add a Windows build")
	if got := statusLaneCount(m, data.Doing); got != 3 {
		t.Errorf("the Doing count is %d, want the blocked task counted", got)
	}
}

func statusLaneCount(m *Main, lane data.Status) int {
	for _, col := range m.pg.Sections {
		if col.Status == lane {
			return col.Total
		}
	}
	return -1
}

func TestDoneLaneShowsTheNewestFiveAndShowAll(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("scratch", "me/scratch")
	for i := 0; i < 8; i++ {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: "Finished " + itoa(i), Status: data.Done, ProjectID: data.IntPtr(p.ID)})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m := boardTester(t, app, p.ID, 1100)
	finished := func() (n int) {
		for _, s := range tt.Texts() {
			if strings.HasPrefix(s, "Finished ") {
				n++
			}
		}
		return n
	}
	if n := finished(); n != 5 || !tt.HasText("Show all 8") || statusLaneCount(m, data.Done) != 8 {
		t.Fatalf("%d cards and no link in %q", n, tt.Texts())
	}
	if err := tt.Click("Show all 8"); err != nil {
		t.Fatal(err)
	}
	if n := finished(); n != 8 || tt.HasText("Show all 8") {
		t.Errorf("%d cards, or the link is still shown", n)
	}
}

func TestCardsShowAPriorityBadgeOnlyForP0AndP1(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1100)
	has(t, tt, "P0", "P1") // Fix tray race (P0), Port core to Go and Decide on the activation policy (P1)
	hasNot(t, tt, "P2", "P3")
}

func TestCardsAreTitleAndOneMetaLineWithNoStatusGlyph(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1100)
	if err := tt.Click("Status: To do"); err == nil {
		t.Error("a card has a status glyph button")
	}
	has(t, tt, "feat/watch", "5h", "Fri")
}

// A narrow card cannot hold a badge and a branch: the branch would be cut down to "fix…".
func TestCardsDropTheBranchWhenAMarkOrBadgeIsThere(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1100)
	has(t, tt, "feat/watch")                  // no badge: the branch shows
	hasNot(t, tt, "fix/tray", "feat/go-core") // blocked with P0, and P1
}

// The header and the composer are one block, and the lanes fill exactly that block, centered;
// only when the narrowest lanes do not fit does the board scroll under a full-width header.
func TestBoardLanesFillTheBlockOfTheHeaderAndTheComposer(t *testing.T) {
	const pad = 48
	for _, avail := range []float32{883, 1100, 1500} {
		got := boardMetricsFor(avail, pad)
		gaps := 3 * theme.ColumnGap
		if got.col < theme.MinColumnWidth || got.col*4+gaps != got.head || 2*got.edge+got.head != avail {
			t.Errorf("%v: %+v, lanes do not fill a centered block", avail, got)
		}
		if got.head > theme.BoardMax {
			t.Errorf("%v: the block is %v, wider than %v", avail, got.head, theme.BoardMax)
		}
	}
	if got := boardMetricsFor(1500, pad); got.head != theme.BoardMax {
		t.Errorf("a wide window: the block is %v, want the %v maximum", got.head, theme.BoardMax)
	}
	if got := boardMetricsFor(523, 40); got.col != theme.MinColumnWidth || got.head != 443 || got.edge != 40 {
		t.Errorf("beside the inspector: %+v, want the narrowest lanes under a full-width header", got)
	}
}

func TestBoardHeaderComposerAndLanesShareTheirEdges(t *testing.T) {
	for _, width := range []int{1100, 1840} {
		app, _, _ := newTestApp(t)
		tt, _ := boardTester(t, app, 1, width)
		lay := boardMetricsFor(float32(width)-217, 48) // the sidebar and its hairline are 217 pt
		left := 217 + lay.edge
		right := left + lay.head
		toggle, _ := tt.Find("Board view")
		composer, _ := tt.Find("Ask the assistant")
		first, _ := tt.Find("To do")
		last, _ := tt.Find("Done")
		near := func(what string, got, want float32) {
			t.Helper()
			if got < want-1.5 || got > want+1.5 {
				t.Errorf("%d pt window: %s at %.1f, want %.1f", width, what, got, want)
			}
		}
		// The offsets are the glyph and its gap (18 + 8), and the control's own padding.
		near("composer left", composer.X-1-16-(theme.AvatarRow+10), left) // the border, the padding, the avatar and its gap
		near("toggle right", toggle.X+toggle.W+2, right)
		near("first lane left", first.X-18-8, left)
		near("last lane left", last.X-18-8, left+3*(lay.col+theme.ColumnGap))
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

func TestDroppingABlockedCardOnItsOwnLaneKeepsItBlocked(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1840)
	dragTo(t, tt, "Fix tray race on launch", "Doing")
	if task, _ := app.Task(3); task.Status != data.Blocked {
		t.Errorf("status %v after dropping a blocked card on Doing, want blocked", task.Status)
	}
	dragTo(t, tt, "Fix tray race on launch", "In review")
	if task, _ := app.Task(3); task.Status != data.Review {
		t.Errorf("status %v after dropping it on In review", task.Status)
	}
}

func TestBoardClickOpensTheInspectorAndTheComposerAddsTasks(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 2, 1100)
	if err := tt.Click("Issue #218 adaptation"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 15 {
		t.Errorf("opened %d", m.openID)
	}
	m.openID = 0
	tt.Frame()
	if _, ok := tt.Find("Add a task"); ok {
		t.Error("a lane header has a +")
	}
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Board added")
	tt.Key(0, ui.KeyEnter)
	found := false
	for _, task := range app.snap.tasks {
		found = found || task.Title == "Board added" && task.Status == data.Todo
	}
	if !found {
		t.Error("the composer above the board did not add a task")
	}
}

func TestViewToggleSwitchesBetweenListAndBoard(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	if err := tt.Click("Board view"); err != nil {
		t.Fatal(err)
	}
	if !m.board[1] || !tt.HasText("In review") {
		t.Errorf("not a board: %q", tt.Texts())
	}
	if err := tt.Click("List view"); err != nil {
		t.Fatal(err)
	}
	if m.board[1] || tt.HasText("Nothing to review") {
		t.Error("not back on the list")
	}
}

func TestMilestoneFilterIsHiddenUntilTheProjectHasAMilestone(t *testing.T) {
	tt, _, _ := projectTester(t, 2)
	if tt.HasText("All milestones") {
		t.Error("a project without a milestone has the filter")
	}
	tt, _, _ = projectTester(t, 1)
	has(t, tt, "All milestones")
	if err := tt.Click("Milestone: All milestones"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("v0.2 native · due Oct 28"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Decide on the activation policy") {
		t.Error("the milestone filter left a task of no milestone")
	}
	has(t, tt, "v0.2 native")
}
