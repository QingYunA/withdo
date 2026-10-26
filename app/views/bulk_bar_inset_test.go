package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// barTop is where the bar's edge is: its buttons are barPad inside it.
func barTop(t *testing.T, tt *ui.Tester) float32 {
	t.Helper()
	r, ok := tt.Find("Clear selection")
	if !ok {
		t.Fatalf("no bar in %q", tt.Texts())
	}
	return r.Y - barPad
}

// scrollDown scrolls the page under (x, y) as far as it goes.
func scrollDown(tt *ui.Tester, x, y float32) {
	for range 12 {
		tt.Scroll(x, y, 0, 400)
		tt.Frame()
	}
}

func TestTheLastRowOfAPageCanBeScrolledAboveTheBar(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	scrollDown(tt, 700, 400)
	last, ok := tt.Find("Send the packaging answer")
	if !ok {
		t.Fatalf("the last row is not drawn in %q", tt.Texts())
	}
	if top := barTop(t, tt); last.Y+last.H > top {
		t.Errorf("the last row ends at %v, under the bar that starts at %v", last.Y+last.H, top)
	}
}

func TestTheLastRowOfALongListScrollsAboveTheBarToo(t *testing.T) {
	f := demoStore(t, fixedNow, false)
	for i := range 14 {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: "Inbox task " + itoa(i+1), Source: data.SourceUser, Status: data.Todo})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	app.noDateOpen = true // the undated tasks of mine with no project are in the No date fold
	app.Refresh()
	m.goTo(nav{})
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Frame()
	scrollDown(tt, 700, 400)
	ids := m.pg.IDs()
	last, _ := m.app.Task(ids[len(ids)-1])
	rect, ok := tt.Find(last.Title)
	if !ok {
		t.Fatalf("the last row %q is not drawn in %q", last.Title, tt.Texts())
	}
	if top := barTop(t, tt); rect.Y+rect.H > top {
		t.Errorf("the last row ends at %v, under the bar that starts at %v", rect.Y+rect.H, top)
	}
}

func TestTheBoardsLanesScrollTheirLastCardAboveTheBar(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 460)
	m.goTo(nav{screen: screenProject, projectID: 1})
	m.board[1] = true
	tt.Frame()
	cmdClick(t, tt, "Write the release notes", "Port the reminder watcher")
	tt.Frame()
	for _, col := range m.pg.Sections {
		if col.Status != "done" {
			continue
		}
		x, _ := tt.Find("Done")
		scrollDown(tt, x.X+20, 300)
		last := col.Tasks[len(col.Tasks)-1]
		rect, ok := tt.Find(last.Title)
		if !ok {
			t.Fatalf("the last card %q is not drawn in %q", last.Title, tt.Texts())
		}
		if top := barTop(t, tt); rect.Y+rect.H > top {
			t.Errorf("the last card ends at %v, under the bar that starts at %v", rect.Y+rect.H, top)
		}
	}
}

func TestThereIsNoInsetWhenNoBarIsUp(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	if got := m.barInset(); got != 0 {
		t.Errorf("inset %v with nothing selected", got)
	}
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if got, want := m.barInset(), barHeight+barBottom; got != want {
		t.Errorf("inset %v with the bar up, want %v: its height and its 22 pt margin", got, want)
	}
}
