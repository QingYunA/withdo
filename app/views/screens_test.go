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
