package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
)

func TestEachColumnOfAProjectHasItsOwnSelectButtonThatReachesOnlyItsRows(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	if m.pg.Kind != page.KindColumns {
		t.Fatal("the project is not drawn as columns")
	}
	if err := tt.Click("Select"); err != nil { // the first one is the Mine column's
		t.Fatal(err)
	}
	tt.Frame()
	if !m.multi.Mode() || m.multi.Scope() != int(sideMine)+1 {
		t.Fatalf("mode %v, scope %d, want select mode on Mine", m.multi.Mode(), m.multi.Scope())
	}
	if err := tt.Click("Select all"); err != nil {
		t.Fatal(err)
	}
	mine := m.columnIDs(sideMine)
	if m.multi.Count() != len(mine) {
		t.Errorf("Select all ticked %d, want the %d rows of Mine", m.multi.Count(), len(mine))
	}
	for _, id := range m.multi.IDs() {
		if !slices.Contains(mine, id) {
			t.Errorf("task %d is not in Mine but was ticked", id)
		}
	}
	// A click on an agent's row opens it instead of ticking it.
	if err := tt.Click("Fix tray race on launch"); err != nil {
		t.Fatal(err)
	}
	if m.multi.Has(m.columnIDs(sideAgents)[0]) {
		t.Error("a click on an agent's row in Mine's select mode ticked it")
	}
	if m.openID == 0 {
		t.Error("a click on an agent's row in Mine's select mode did not open it")
	}
}

func TestTheAgentsColumnSelectButtonReachesOnlyAgentRows(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	r, ok := tt.Find("Select")
	if !ok {
		t.Fatal("no Select button")
	}
	tray := m.cols.geo.tray
	tt.ClickAt(tray.X+tray.W-theme.TrayPad-30, r.Y+r.H/2)
	tt.Frame()
	if m.multi.Scope() != int(sideAgents)+1 {
		t.Fatalf("scope %d, want the Agents column's", m.multi.Scope())
	}
	m.multi.All()
	agents := m.columnIDs(sideAgents)
	if m.multi.Count() != len(agents) || len(agents) == 0 {
		t.Errorf("ticked %d of %d agent rows", m.multi.Count(), len(agents))
	}
	for _, id := range m.multi.IDs() {
		if !slices.Contains(agents, id) {
			t.Errorf("task %d is not an agent's but was ticked", id)
		}
	}
}
