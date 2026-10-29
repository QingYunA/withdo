package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// manageShot renders the window at 2x in a language; setup may open a menu, which is drawn in the window.
func manageShot(t *testing.T, name string, app *App, dark bool, lang string, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	m := NewMain(app)
	tt := newTester(m.View, 1100, 720)
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	if lang == "zh-Hans" {
		inChinese(tt)
	}
	if setup != nil {
		setup(tt, m)
		tt.Frame()
	}
	settle(tt)
	shot(t, name, tt.Image(), false, dark)
}

// languages are the two the screens come in, with the suffix their files carry.
var languages = []struct{ tag, suffix string }{{"en", ""}, {"zh-Hans", "-zh"}}
