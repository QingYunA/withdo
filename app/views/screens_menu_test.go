package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The screens of the task menu, Done today and the Done group, in both languages and appearances.
// Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensMenu

// menuAt right-clicks the row showing text, at its middle, and draws a frame.
func menuAt(tt *ui.Tester, text string, dx float32) {
	r, _ := tt.Find(text)
	tt.RightClickAt(r.X+dx, r.Y+r.H/2)
	tt.Frame()
}

func TestScreensMenu(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			inProject := func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
			}
			manageShot(t, "task-menu-hover"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				inProject(tt, m)
				menuAt(tt, "Write the release notes", 120)
				g, _ := tt.Find(m.say.A11yMenuStatus(statusLabel(m.say, data.Doing)))
				tt.Move(g.X+g.W/2, g.Y+g.H/2)
				tt.Frame() // the label above the strip names the choice under the pointer
				tt.Frame()
				return nil
			})
			manageShot(t, "task-menu-move"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				inProject(tt, m)
				menuAt(tt, "Write the release notes", 120)
				_ = tt.Click(m.say.MenuMoveTo())
				return nil
			})
			manageShot(t, "task-menu-remind"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				inProject(tt, m)
				menuAt(tt, "Write the release notes", 120)
				_ = tt.Click(m.say.MenuRemindMe())
				return nil
			})
			manageShot(t, "task-menu-custom"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				inProject(tt, m)
				menuAt(tt, "Write the release notes", 120)
				_ = tt.Click(m.say.MenuRemindMe())
				tt.Frame()
				_ = tt.Click(m.say.MenuCustom())
				return nil
			})
			manageShot(t, "task-menu-glyph"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				_ = tt.Click(m.say.A11yStatus(statusLabel(m.say, data.Blocked)))
				return nil
			})
			manageShot(t, "inspector-menu-priority"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.OpenTask(8)
				tt.Frame()
				_ = tt.Click(m.say.A11yPriority(priorityLabel(m.say, data.P2)))
				return nil
			})
			manageShot(t, "mine-done-today"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				tt.SetSize(1100, 1000)
				return nil
			})
			manageShot(t, "project-history"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				tt.SetSize(1100, 1000)
				inProject(tt, m)
				_ = tt.Click(m.say.ShowAll())
				return nil
			})
		}
	}
}
