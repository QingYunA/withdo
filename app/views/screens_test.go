package views

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The design review: every screen in light and dark at 2x, on three shapes of data. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreens
//
// The real-shape screens read a copy of a real database named by ADHD_REAL_DB (never the
// original) through the real adapter; they are skipped without it.

// mainShot renders the window at 2x.
func mainShot(t *testing.T, name string, app *App, dark bool, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	if setup != nil {
		setup(tt, m)
		tt.Frame()
	}
	settle(tt)
	shot(t, name, tt.Image(), false, dark)
}

func appearances() []struct {
	suffix string
	dark   bool
} {
	return []struct {
		suffix string
		dark   bool
	}{{"light", false}, {"dark", true}}
}

func toProject(id int) func(*ui.Tester, *Main) {
	return func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenProject, projectID: id}) }
}

func TestScreensSeed(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := newTestApp(t)
		mainShot(t, "now-"+a.suffix+".png", app, a.dark, nil)
		mainShot(t, "project-list-"+a.suffix+".png", app, a.dark, toProject(1))
		mainShot(t, "project-board-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			toProject(1)(tt, m)
			m.board[1] = true
		})
		mainShot(t, "inbox-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenInbox}) })
		mainShot(t, "inspector-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.OpenTask(3) })
		mainShot(t, "inspector-narrow-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			tt.SetSize(860, 540)
			m.OpenTask(2)
		})
		mainShot(t, "now-selection-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			tt.Key(0, ui.KeyDown)
			tt.Key(0, ui.KeyDown)
		})
		asking, _, _ := newTestApp(t) // adding a task changes the data, so not the shared store
		mainShot(t, "now-asking-"+a.suffix+".png", asking, a.dark, func(tt *ui.Tester, m *Main) {
			tt.Key(ui.Cmd, ui.KeyK)
			tt.Type("Water the plants")
			tt.Key(0, ui.KeyEnter)
		})
		mainShot(t, "project-list-expanded-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			toProject(1)(tt, m)
			m.expanded[m.nav] = true
			tt.SetSize(1100, 1000)
		})
	}
}

func TestScreensEmpty(t *testing.T) {
	for _, a := range appearances() {
		empty, _ := newEmptyApp(t)
		mainShot(t, "now-empty-"+a.suffix+".png", empty, a.dark, nil)
		withProject, p := emptyProjectApp(t)
		mainShot(t, "project-empty-"+a.suffix+".png", withProject, a.dark, toProject(p.ID))
	}
}

// TestScreensDragging renders a card in flight over the board, with the hidden statuses offered.
func TestScreensDragging(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := newTestApp(t)
		m := NewMain(app)
		m.goTo(nav{screen: screenProject, projectID: 2})
		m.board[2] = true
		tt := ui.NewTester(m.View, 1100, 720)
		tt.SetScale(2)
		tt.SetDark(a.dark)
		tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
		still(tt)
		from, _ := tt.Find("Issue #218 adaptation")
		fx, fy := center(from)
		tt.Press(fx, fy)
		tt.Move(fx+8, fy+8)
		tt.Move(fx+60, fy+80)
		settle(tt)
		shot(t, "project-board-dragging-"+a.suffix+".png", tt.Image(), false, a.dark)
	}
}

func TestScreensRealShape(t *testing.T) {
	path := os.Getenv("ADHD_REAL_DB")
	if path == "" {
		t.Skip("set ADHD_REAL_DB to a copy of a real database")
	}
	// The clock is pinned an hour after the newest row of the user's database, so ages read the
	// same on every run.
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	for _, a := range appearances() {
		r, err := data.OpenReal(context.Background(), path, data.RealOptions{Now: func() time.Time { return now }})
		if err != nil {
			t.Fatal(err)
		}
		app := NewApp(r, &recorder{}, testConfig())
		app.Now = func() time.Time { return now }
		mainShot(t, "now-real-"+a.suffix+".png", app, a.dark, nil)
		ps, _ := r.ListProjects()
		for _, p := range ps {
			if p.RepoSlug != "" {
				mainShot(t, "project-real-"+a.suffix+".png", app, a.dark, toProject(p.ID))
			}
		}
		_ = r.Close()
	}
}
