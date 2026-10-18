package views

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The v10 screens: Accept stamped on Mine, a hovered Agents row and a hovered board card. They are
// frames of the animation on the test's clock. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensAccept

// acceptShot opens the screen, runs setup with motion on, moves the clock to in after it and shoots
// the frame there.
func acceptShot(t *testing.T, name string, zh, dark bool, in time.Duration, setup func(tt *ui.Tester, m *Main, app *App)) {
	t.Helper()
	app, _, _ := newTestApp(t)
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	if zh {
		inChinese(tt)
	}
	moving(tt)
	tt.Frame()
	setup(tt, m, app)
	at(tt, app, in)
	settle(tt) // the hover and color transitions run on the wall clock; let them end
	shot(t, name, tt.Image(), false, dark)
}

func TestScreensAccept(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			zh := l.tag == "zh-Hans"
			name := func(what string) string { return what + l.suffix + "-" + a.suffix + ".png" }
			accept := func(tt *ui.Tester, m *Main) {
				if err := tt.Click(m.say.A11yAcceptTask("Native UI with mygo")); err != nil {
					t.Fatal(err)
				}
			}
			// Mine, 50 ms after the click: the third frame of the check (6 of 14 pixels).
			acceptShot(t, name("mine-accepting"), zh, a.dark, 50*time.Millisecond, func(tt *ui.Tester, m *Main, app *App) {
				accept(tt, m)
			})
			// The project's Agents row under the pointer.
			acceptShot(t, name("project-agents-hover"), zh, a.dark, 0, func(tt *ui.Tester, m *Main, app *App) {
				toProject(1)(tt, m)
				tt.Frame()
				hover(t, tt, "Native UI with mygo")
			})
			// The board card of a task in review under the pointer.
			acceptShot(t, name("board-card-hover"), zh, a.dark, 0, func(tt *ui.Tester, m *Main, app *App) {
				toProject(1)(tt, m)
				m.board[1] = true
				tt.Frame()
				hover(t, tt, "Native UI with mygo")
			})
		}
		// Mid-fold: the row and the hairline above it are closing, the Done row is opening.
		acceptShot(t, "mine-accepting-fold-"+a.suffix+".png", false, a.dark, 300*time.Millisecond, func(tt *ui.Tester, m *Main, app *App) {
			if err := tt.Click(m.say.A11yAcceptTask("Native UI with mygo")); err != nil {
				t.Fatal(err)
			}
		})
	}
}
