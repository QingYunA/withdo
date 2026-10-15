package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func searchFor(t *testing.T, tt *ui.Tester, q string) {
	t.Helper()
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Frame()
	tt.Type(q)
}

func TestSidebarHasASearchFieldAndCmdFFocusesIt(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	if !tt.HasText("Search") || !tt.HasText("⌘F") {
		t.Fatalf("no search field in the sidebar: %q", tt.Texts())
	}
	m.goTo(nav{screen: screenProject, projectID: 2}) // from any screen
	tt.Frame()
	searchFor(t, tt, "tray")
	if m.manage.search.query != "tray" {
		t.Fatalf("the field holds %q", m.manage.search.query)
	}
}

func TestSearchResultsAreGroupedByProjectAcrossEveryStatus(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	searchFor(t, tt, "the")
	// Matches in three places: the done "Spike the mygo shell" and "Map worktrees..." of adhd-todo,
	// the open "Cover the reply parser edge cases" of answer-me-with-html and the Inbox's "Renew..." none.
	for _, want := range []string{"Search", "Spike the mygo shell", "Cover the reply parser edge cases", "adhd-todo", "answer-me-with-html"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if !above(t, tt, "Spike the mygo shell", "Cover the reply parser edge cases") && !above(t, tt, "Cover the reply parser edge cases", "Spike the mygo shell") {
		t.Error("rows overlap")
	}
	if got := m.searchModel(); len(got.groups) < 2 {
		t.Errorf("groups %+v", got.groups)
	}
	if !tt.HasText("results for “the”") {
		t.Errorf("no count: %q", tt.Texts())
	}
}

func TestSearchFindsTitleNotesBranchAndPullRequest(t *testing.T) {
	for q, want := range map[string]string{
		"tray":     "Fix tray race on launch", // title and branch
		"release":  "Write the release notes", // title
		"mygo":     "Native UI with mygo",     // title; branch is native-ui
		"windows":  "Add a Windows build",     // backlog
		"#12":      "Native UI with mygo",     // pull request
		"dentist":  "Book dentist",            // the Inbox
		"register": "Write the release notes", // notes
		"claude/":  "Issue #218 adaptation",   // branch
	} {
		tt, _, _ := newMainTester(t, 1100)
		tt.Frame()
		searchFor(t, tt, q)
		if !tt.HasText(want) {
			t.Errorf("%q does not find %q: %q", q, want, tt.Texts())
		}
	}
}

func TestSearchPutsTheInboxLastAndNamesIt(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.Frame()
	searchFor(t, tt, "e")
	if !tt.HasText("Book dentist") {
		t.Fatalf("no inbox result: %q", tt.Texts())
	}
	if !above(t, tt, "Write the release notes", "Book dentist") {
		t.Error("the Inbox group is not last")
	}
}

func TestSearchWithNoMatchSaysWhereItLooked(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.Frame()
	searchFor(t, tt, "zzzz")
	for _, want := range []string{"No tasks match “zzzz”", "Search looks in titles, notes, branches and pull request numbers, in every project."} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
}

func TestSearchKeyboard(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	searchFor(t, tt, "port")
	ids := m.rowIDs(m.app.buildNow())
	if len(ids) < 2 {
		t.Fatalf("results %v", ids)
	}
	tt.Key(0, ui.KeyDown)
	if m.sel != ids[0] {
		t.Fatalf("Down selected %d, want the first result %d", m.sel, ids[0])
	}
	tt.Key(0, ui.KeyDown)
	if m.sel != ids[1] {
		t.Errorf("second Down selected %d, want %d", m.sel, ids[1])
	}
	tt.Key(0, ui.KeyEnter)
	if m.openID != ids[1] {
		t.Errorf("Enter opened %d, want %d", m.openID, ids[1])
	}
	if !m.searching() {
		t.Error("opening a result ended the search")
	}
	// Esc: the inspector, then the highlight, then the search.
	tt.Key(0, ui.KeyEscape)
	if m.openID != 0 || m.sel == 0 || !m.searching() {
		t.Errorf("first Esc: open %d sel %d searching %v", m.openID, m.sel, m.searching())
	}
	tt.Key(0, ui.KeyEscape)
	if m.sel != 0 || !m.searching() {
		t.Errorf("second Esc: sel %d searching %v", m.sel, m.searching())
	}
	tt.Key(0, ui.KeyEscape)
	if m.searching() || m.manage.search.query != "" {
		t.Errorf("third Esc left the search %q", m.manage.search.query)
	}
}

func TestEnterInTheSearchFieldOpensTheFirstResult(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	searchFor(t, tt, "dentist")
	tt.Key(0, ui.KeyEnter)
	if m.openID != 8 {
		t.Errorf("opened %d, want Book dentist (8)", m.openID)
	}
}

func TestEscInTheSearchFieldClearsAndLeaves(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	searchFor(t, tt, "tray")
	if !tt.HasText("1 result for “tray”") {
		t.Fatalf("results: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if m.searching() || tt.HasText("results for") {
		t.Errorf("Esc left the results: %q", tt.Texts())
	}
	if m.nav.screen != screenProject || m.nav.projectID != 1 || !tt.HasText("Blocked") {
		t.Errorf("the screen under the search is not back: nav %+v", m.nav)
	}
	tt.Type("x")
	if m.manage.search.query != "" {
		t.Errorf("the field keeps the keyboard: %q", m.manage.search.query)
	}
}

func TestClickingASidebarItemEndsTheSearch(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	searchFor(t, tt, "tray")
	if err := tt.Click("Inbox"); err != nil {
		t.Fatal(err)
	}
	if m.searching() || m.nav.screen != screenInbox {
		t.Errorf("searching %v nav %+v", m.searching(), m.nav)
	}
	if tt.HasText("results for") {
		t.Error("results still shown")
	}
}

func TestSearchRowsOpenTheInspectorAndHaveTheTaskMenu(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	searchFor(t, tt, "dentist")
	rightClick(t, tt, "Book dentist")
	if !m.pop.open || m.pop.task != 8 || !tt.HasText("Move to…") {
		t.Errorf("menu open %v for %d", m.pop.open, m.pop.task)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if err := tt.Click("Book dentist"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 8 {
		t.Errorf("opened %d", m.openID)
	}
}
