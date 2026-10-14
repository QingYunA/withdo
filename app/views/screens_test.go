package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

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

func TestScreensSeed(t *testing.T) {
	for _, dark := range []bool{false, true} {
		suffix := "light"
		if dark {
			suffix = "dark"
		}
		app, _, _ := newTestApp(t)
		mainShot(t, "now-"+suffix+".png", app, dark, nil)
		mainShot(t, "project-list-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenProject, projectID: 1}) })
	}
}
