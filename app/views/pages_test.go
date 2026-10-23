package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The chip of a row's category narrows the list to that category, and a bar above the list undoes
// it; Agents groups its rows under a header for each agent.

func TestAChipNarrowsTodayToItsCategoryAndTheBarClearsIt(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	has(t, tt, "Show only Work", "Show only Life", "Reply to design feedback", "Book dentist")
	hasNot(t, tt, "Showing Work", "Clear filter")
	click(t, tt, "Show only Life")
	tt.Frame()
	if m.catFilter != lifeID {
		t.Fatalf("filter %d, want Life %d", m.catFilter, lifeID)
	}
	has(t, tt, "Showing Life", "Clear filter", "Book dentist")
	hasNot(t, tt, "Reply to design feedback", "Write the release notes")
	if !above(t, tt, "Showing Life", "Book dentist") {
		t.Error("the bar is not above the list")
	}
	click(t, tt, "Clear filter")
	tt.Frame()
	if m.catFilter != 0 || tt.HasText("Showing Life") {
		t.Errorf("the bar stays: filter %d", m.catFilter)
	}
	has(t, tt, "Reply to design feedback", "Book dentist")
}

func TestEscClearsTheCategoryFilter(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	click(t, tt, "Show only Work")
	tt.Frame()
	if m.catFilter != workID {
		t.Fatalf("filter %d", m.catFilter)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if m.catFilter != 0 || tt.HasText("Showing Work") {
		t.Errorf("Esc left the filter: %d", m.catFilter)
	}
}

func TestTheFilterEndsWhenAnotherPageOpensAndNewTasksGetItsCategory(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	click(t, tt, "Show only Life")
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Buy stamps")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	var made data.Task
	for _, task := range app.snap.tasks {
		if task.Title == "Buy stamps" {
			made = task
		}
	}
	if made.CategoryID != lifeID {
		t.Errorf("a task typed under the Life filter has category %d, want %d", made.CategoryID, lifeID)
	}
	click(t, tt, "Agents")
	if m.catFilter != 0 {
		t.Errorf("filter %d after leaving Today", m.catFilter)
	}
}

func TestAChipOnAProjectPageNarrowsThatListToo(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	if err := app.Update(5, data.Patch{Category: data.IntPtr(lifeID)}); err != nil { // Write the release notes
		t.Fatal(err)
	}
	tt.Frame()
	hover(t, tt, "Show only Life") // a row of a project can be dragged: the pointer arrives before the press
	click(t, tt, "Show only Life")
	tt.Frame()
	has(t, tt, "Showing Life", "Write the release notes")
	hasNot(t, tt, "Decide on the activation policy")
	click(t, tt, "Clear filter")
	tt.Frame()
	has(t, tt, "Decide on the activation policy")
	if m.catFilter != 0 {
		t.Errorf("filter %d", m.catFilter)
	}
}

func TestAgentsGroupsItsRowsUnderAHeaderForEachAgent(t *testing.T) {
	app, store := mineApp(t)
	store.SeedReportAs(15, "codex", "Writing tests for the reply parser", fixedNow().Add(-time.Minute))
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1200)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	has(t, tt, "Claude Code", "Codex", "Port core to Go", "Fix tray race on launch", "Issue #218 adaptation")
	// The agent that reported last comes first.
	order := []string{"Codex", "Issue #218 adaptation", "Claude Code", "Port core to Go", "Fix tray race on launch"}
	for i := 1; i < len(order); i++ {
		if !above(t, tt, order[i-1], order[i]) {
			t.Errorf("%q should be above %q in %q", order[i-1], order[i], tt.Texts())
		}
	}
	hasNot(t, tt, "Working")
	// The page's rows are the ones drawn, in the order the arrow keys walk them.
	var drawn []int
	for _, r := range m.agentsPage.Working {
		drawn = append(drawn, r.ID)
	}
	if got := m.pg.IDs()[:len(drawn)]; len(got) != len(drawn) || got[0] != drawn[0] {
		t.Errorf("ids %v do not start with the grouped rows %v", got, drawn)
	}
}

func TestAnAgentsRowsStayTogetherWhateverTheOrderOfTheirReports(t *testing.T) {
	name := func(t data.Task) string { return map[int]string{1: "a", 2: "b", 3: "a"}[t.ID] }
	row := func(id int) data.AgentRow { return data.AgentRow{Item: data.Item{Task: data.Task{ID: id}}} }
	got := groupByAgent(data.AgentsScreen{Working: []data.AgentRow{row(1), row(2), row(3)}}, name)
	var ids []int
	for _, r := range got.Working {
		ids = append(ids, r.ID)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 3 || ids[2] != 2 {
		t.Errorf("ids %v, want the rows of the first agent together: 1 3 2", ids)
	}
}

func TestTodayFoldsTheBacklogOfTasksWithNoProjectSoNothingIsOutOfReach(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.SetSize(1100, 1400)
	tt.Frame()
	has(t, tt, "Backlog 1", "show Backlog 1")
	hasNot(t, tt, "Renew passport") // parked, no project: the seed's
	click(t, tt, "show Backlog 1")
	tt.Frame()
	has(t, tt, "Renew passport", "hide Backlog 1")
	hasNot(t, tt, "Poll gh for PR status") // a project's backlog is on its page
}
