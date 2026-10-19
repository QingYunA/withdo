package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// TestScreensFiledToast draws the quiet toast after the assistant sorted a new task on its own.
// Render with scripts/screens.sh TestScreensFiledToast.
func TestScreensFiledToast(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix + ".png"
			if zh {
				sfx = "-zh" + sfx
			}
			fa := &data.FakeAssistant{Ready: true}
			app, store, rec := readyApp(t, fa)
			fa.Script = filedReply(store, true)
			setup := func(tt *ui.Tester, m *Main) {
				tt.Key(ui.Cmd, ui.KeyK)
				tt.Type("pay rent")
				tt.Key(0, ui.KeyEnter)
				rec.drain(t)
				m.closeSheet()
				tt.Frame()
				settle(tt)
			}
			if zh {
				zhShot(t, "mine-filed-toast"+sfx, app, a.dark, setup)
				continue
			}
			mainShot(t, "mine-filed-toast"+sfx, app, a.dark, setup)
		}
	}
}
