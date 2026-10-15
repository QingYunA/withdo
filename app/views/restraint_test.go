package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

// M4d: one title, one input, one list. These are the surfaces that lost things: the header, the
// rows, and the inspector's top line that took over the repository.

func TestProjectHeaderIsTheTitleAndTheViewControlOnly(t *testing.T) {
	tt, _, _ := projectTester(t, 2) // no milestone
	hasNot(t, tt, "QingYunA/answer-me-with-html", "All milestones", "Done 0")
	has(t, tt, "List view", "Board view")
	list, _ := tt.Find("List view")
	board, _ := tt.Find("Board view")
	if list.W != 30 || list.H != 26 || board.W != 30 || board.H != 26 {
		t.Errorf("the cells are %vx%v and %vx%v, want 30x26 icons", list.W, list.H, board.W, board.H)
	}
	if tt.HasText("List") && !tt.HasText("List view") {
		t.Error("the view control has text")
	}
}

func TestHeaderGainsTheMilestoneFilterOnlyWithMilestones(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	has(t, tt, "All milestones")
	filter, _ := tt.Find("All milestones")
	list, _ := tt.Find("List view")
	if filter.X+filter.W > list.X {
		t.Errorf("the filter (right edge %.0f) is not before the view control (%.0f)", filter.X+filter.W, list.X)
	}
	hasNot(t, tt, "QingYunA/adhd-todo", "Done 5")
}

func TestRowsShowAPriorityBadgeOnlyForP0AndP1(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	has(t, tt, "P0", "P1") // Fix tray race (P0); Port core to Go and Decide on the activation policy (P1)
	hasNot(t, tt, "P2", "P3")
	hasNot(t, tt, "Priority: P1  High") // no bars glyph label either
}

func TestAnAgentRowTellsWhoWhereAndWhat(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	has(t, tt, "Claude Code · feat/native-ui · #12 · Board, lists and inspector are in; waiting for your review",
		"Claude Code · fix/tray · Needs a decision on the activation policy")
}

func TestInspectorTopLineHasTheProjectItsRepositoryAndTheNumber(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.OpenTask(3)
	tt.Frame()
	has(t, tt, "adhd-todo · QingYunA/adhd-todo", " · #3")
	m.OpenTask(7) // an Inbox task
	tt.Frame()
	has(t, tt, "Inbox", " · #7")
	hasNot(t, tt, "Inbox · ")
}

func TestInspectorTopLineOfAProjectWithoutARepository(t *testing.T) {
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("local-only", "")
	task, _ := f.CreateTask(data.CreateTaskInput{Title: "A thing", ProjectID: data.IntPtr(p.ID)})
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	m.OpenTask(task.ID)
	tt.Frame()
	has(t, tt, "local-only")
	hasNot(t, tt, "local-only · ")
}
