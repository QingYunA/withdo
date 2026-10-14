package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// panelShot renders the quick panel at 2x.
func panelShot(t *testing.T, name string, app *App, dark bool, setup func(tt *ui.Tester, p *Panel)) {
	t.Helper()
	p := NewPanel(app)
	tt := ui.NewTester(p.View, 440, 560)
	tt.SetScale(2)
	tt.SetDark(dark)
	still(tt)
	if setup != nil {
		setup(tt, p)
	}
	shot(t, name, tt.Image(), true, dark)
}

func TestScreensPanel(t *testing.T) {
	for _, dark := range []bool{false, true} {
		suffix := "light"
		if dark {
			suffix = "dark"
		}
		app, _, _ := newTestApp(t)
		panelShot(t, "panel-data-"+suffix+".png", app, dark, nil)
		empty, _ := newEmptyApp(t)
		panelShot(t, "panel-empty-"+suffix+".png", empty, dark, nil)
	}
}

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
		mainShotSized(t, "main-board-wide-"+suffix+".png", app, dark, 1760, nil)
		mainShot(t, "main-inspector-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) { m.OpenTask(3) })
		mainShot(t, "main-inspector-user-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) { m.OpenTask(7) })
		mainShot(t, "main-settings-"+suffix+".png", app, dark, func(tt *ui.Tester, m *Main) { m.OpenSettings() })
		empty, _ := newEmptyApp(t)
		mainShot(t, "main-empty-"+suffix+".png", empty, dark, nil)
	}
}
