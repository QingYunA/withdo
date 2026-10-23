package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// titleSpot is a point on the page title: at the left of the composer's column, on the line of
// the List | Board switch (the sidebar has the same name, so Find would hit that first).
func titleSpot(t *testing.T, tt *ui.Tester) (x, y float32) {
	t.Helper()
	c, ok := tt.Find("Add a task, or ask me anything")
	if !ok {
		t.Fatalf("no composer in %q", tt.Texts())
	}
	v, ok := tt.Find("List view")
	if !ok {
		t.Fatalf("no List | Board switch in %q", tt.Texts())
	}
	return c.X + 10, v.Y + v.H/2
}

func doubleClickAt(tt *ui.Tester, x, y float32) {
	tt.ClickAt(x, y)
	tt.ClickAt(x, y)
}

// doubleClickTitle double clicks the page title.
func doubleClickTitle(t *testing.T, tt *ui.Tester) {
	t.Helper()
	x, y := titleSpot(t, tt)
	doubleClickAt(tt, x, y)
}

func TestDoubleClickingTheProjectTitleRenamesItInPlace(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	doubleClickTitle(t, tt)
	if m.manage.rename.id != 1 || !m.manage.rename.page {
		t.Fatalf("rename %+v", m.manage.rename)
	}
	tt.Type("-x")
	tt.Key(0, ui.KeyEnter)
	if p, _ := app.Project(1); p.Name != "adhd-todo-x" {
		t.Errorf("name %q", p.Name)
	}
	if m.manage.rename.id != 0 {
		t.Error("still renaming after Enter")
	}
}

func TestEscapeKeepsTheProjectNameWhenRenamingFromTheTitle(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	doubleClickTitle(t, tt)
	tt.Type("zzz")
	tt.Key(0, ui.KeyEscape)
	if p, _ := app.Project(1); p.Name != "adhd-todo" || m.manage.rename.id != 0 {
		t.Errorf("name %q, rename %+v", p.Name, m.manage.rename)
	}
}

func TestDoubleClickingTheInboxTitleDoesNothing(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	_, y := titleSpot(t, tt) // the Inbox has no switch to measure by; its title is on the same line
	m.goTo(nav{})
	tt.Frame()
	c, _ := tt.Find("Add a task, or ask me anything")
	doubleClickAt(tt, c.X+10, y)
	if m.manage.rename.id != 0 {
		t.Errorf("rename %+v", m.manage.rename)
	}
}
