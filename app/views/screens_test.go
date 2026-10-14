package views

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// mainShot renders the main window at 2x.
func mainShot(t *testing.T, name string, app *App, dark bool, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	mainShotSized(t, name, app, dark, 1100, setup)
}

func mainShotSized(t *testing.T, name string, app *App, dark bool, width int, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	m := NewMain(app)
	tt := ui.NewTester(m.View, width, 720)
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

func TestScreensMain(t *testing.T) {
	for _, dark := range []bool{false, true} {
		suffix := "light"
		if dark {
			suffix = "dark"
		}
		app, _, _ := newTestApp(t)
		mainShot(t, "main-board-"+suffix+".png", app, dark, nil)
		mainShotSized(t, "main-board-wide-"+suffix+".png", app, dark, 1840, nil)
		mainShot(t, "main-inspector-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) { m.OpenTask(3) })
		mainShot(t, "main-inspector-user-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) { m.OpenTask(7) })
		mainShot(t, "main-settings-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) { m.OpenSettings() })
		empty, _ := newEmptyApp(t)
		mainShot(t, "main-empty-"+suffix+".png", empty, dark, nil)
		mainShot(t, "main-adding-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) {
			_ = tt.Click("Add task to Todo")
			tt.Type("Buy a standing desk")
		})
		mainShot(t, "main-column-hover-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) {
			r, _ := tt.Find("Blocked")
			x, y := center(r)
			tt.Move(x+60, y)
		})
	}
}

// TestScreensDragging renders a card in flight over the Doing column, with motion on.
func TestScreensDragging(t *testing.T) {
	for _, dark := range []bool{false, true} {
		suffix := "light"
		if dark {
			suffix = "dark"
		}
		app, _, _ := newTestApp(t)
		m := NewMain(app)
		tt := ui.NewTester(m.View, 1840, 720)
		tt.SetScale(2)
		tt.SetDark(dark)
		tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
		from, _ := tt.Find("Write the release notes")
		to, _ := tt.Find("Doing")
		fx, fy := center(from)
		tx, ty := center(to)
		tt.Press(fx, fy)
		tt.Move(fx+8, fy+8)
		tt.Move(tx+40, ty+110)
		time.Sleep(450 * time.Millisecond)
		tt.Frame()
		shot(t, "main-dragging-"+suffix+".png", tt.Image(), false, dark)
	}
}

// TestScreensDraggingOverAStrip renders a card in flight over the collapsed Done column, which
// opens to take it.
func TestScreensDraggingOverAStrip(t *testing.T) {
	for _, dark := range []bool{false, true} {
		suffix := "light"
		if dark {
			suffix = "dark"
		}
		app, _, _ := newTestApp(t)
		m := NewMain(app)
		tt := ui.NewTester(m.View, 1100, 720)
		tt.SetScale(2)
		tt.SetDark(dark)
		tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
		from, _ := tt.Find("Port core to Go")
		to, _ := tt.Find("Done")
		fx, fy := center(from)
		tx, ty := center(to)
		tt.Press(fx, fy)
		tt.Move(fx+8, fy+8)
		tt.Move(tx, ty+110)
		time.Sleep(450 * time.Millisecond)
		tt.Frame()
		shot(t, "main-dragging-strip-"+suffix+".png", tt.Image(), false, dark)
	}
}
