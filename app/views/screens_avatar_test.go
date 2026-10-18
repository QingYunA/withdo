package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The M5 screens of avatars and empty states, in light and dark at 2x, in both languages. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensAvatars

func TestScreensAvatars(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			shot := mainShot
			if l.tag == "zh-Hans" {
				shot = zhShot
			}
			suffix := l.suffix + "-" + a.suffix + ".png"
			empty, _ := newEmptyApp(t)
			shot(t, "inbox-empty"+suffix, empty, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenInbox}) })
			withProject, p := emptyProjectApp(t)
			shot(t, "project-empty-avatars"+suffix, withProject, a.dark, toProject(p.ID))
			app, _, _ := newTestApp(t)
			shot(t, "inspector-from-you"+suffix, app, a.dark, func(tt *ui.Tester, m *Main) { m.OpenTask(5) })
		}
	}
}
