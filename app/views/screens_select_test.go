package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The screens of selecting several tasks (docs/design/mockups/v20.html), in light and dark, in
// English and Simplified Chinese. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSelect

// selectShot renders the main window at 2x in the language, with the page set up by setup.
func selectShot(t *testing.T, name string, zh bool, app *App, dark bool, size [2]int, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	columnsShot(t, name, zh, app, dark, size, setup)
}

// tick ⌘-clicks the rows, as the user selects several.
func tick(tt *ui.Tester, titles ...string) {
	for _, title := range titles {
		_ = tt.ClickWith(ui.Cmd, title)
	}
}

func TestScreensSelect(t *testing.T) {
	screensTest(t)
	full := [2]int{1100, 720}
	for _, a := range appearances() {
		for _, l := range languages {
			zh := l.tag == "zh-Hans"
			name := func(what string) string { return "select-" + what + l.suffix + "-" + a.suffix + ".png" }

			// Select mode on Mine: the checkboxes, the count, Select all and Done, and the bar.
			app, _, _ := newTestApp(t)
			selectShot(t, name("mode"), zh, app, a.dark, full, func(tt *ui.Tester, m *Main) {
				m.multi.mode = true
				tt.Frame()
				tick(tt, "Reply to design feedback", "Decide on the activation policy")
			})

			// Three rows selected with ⌘-click: joined blocks, the ring on the last one, the bar.
			app, _, _ = newTestApp(t)
			selectShot(t, name("bar"), zh, app, a.dark, full, func(tt *ui.Tester, m *Main) {
				tick(tt, "Reply to design feedback", "Decide on the activation policy", "Book dentist")
			})

			// The menu of the selection, opened by a right-click on one of its rows.
			app, _, _ = newTestApp(t)
			selectShot(t, name("menu"), zh, app, a.dark, full, func(tt *ui.Tester, m *Main) {
				tick(tt, "Reply to design feedback", "Decide on the activation policy", "Book dentist")
				tt.Frame()
				_ = tt.RightClick("Decide on the activation policy")
			})

			// A stack in the hand over the tray, and the rows just landed with the toast.
			size := [2]int{1100, 900}
			rig := func() *handRig {
				r := newHandRig(t, false)
				r.tt.SetScale(2)
				r.tt.SetDark(a.dark)
				if zh {
					inChinese(r.tt)
				}
				r.tt.SetSize(size[0], size[1])
				r.tt.Frame()
				tick(r.tt, "Decide on the activation policy", "Write the release notes", "Draft the UI spec")
				r.tt.Frame()
				return r
			}
			r := rig()
			r.dragOnto(t, "Write the release notes", "Port core to Go", true)
			r.after(330 * time.Millisecond)
			shot(t, name("stack-over"), r.tt.Image(), false, a.dark)

			r = rig()
			r.dragOnto(t, "Write the release notes", "Port core to Go", false)
			r.after(250 * time.Millisecond)
			settle(r.tt)
			shot(t, name("stack-landed"), r.tt.Image(), false, a.dark)
		}

		// The board with two cards selected, and the narrow bar beside the inspector.
		app, _, _ := newTestApp(t)
		selectShot(t, "select-board-"+a.suffix+".png", false, app, a.dark, full, func(tt *ui.Tester, m *Main) {
			toProject(1)(tt, m)
			m.board[1] = true
			tt.Frame()
			tick(tt, "Fix tray race on launch", "Decide on the activation policy", "Write the release notes")
		})
		app, _, _ = newTestApp(t)
		selectShot(t, "select-bar-narrow-"+a.suffix+".png", false, app, a.dark, [2]int{860, 540}, func(tt *ui.Tester, m *Main) {
			m.OpenTask(3)
			tt.Frame()
			tick(tt, "Reply to design feedback", "Decide on the activation policy")
		})
	}
}

// keep the data import for the helpers above if the tests change.
var _ = data.Todo
