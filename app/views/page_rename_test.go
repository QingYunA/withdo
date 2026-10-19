package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// doubleClickTitle double clicks the page title, which sits above the composer (the sidebar has
// the same name, so Find would hit that first).
func doubleClickTitle(t *testing.T, tt *ui.Tester) {
	t.Helper()
	c, ok := tt.Find("Add a task, or ask me anything")
	if !ok {
		t.Fatalf("no composer in %q", tt.Texts())
	}
	tt.ClickAt(c.X+10, c.Y-c.H-20)
	tt.ClickAt(c.X+10, c.Y-c.H-20)
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
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenInbox})
	tt.Frame()
	doubleClickTitle(t, tt)
	if m.manage.rename.id != 0 {
		t.Errorf("rename %+v", m.manage.rename)
	}
}
