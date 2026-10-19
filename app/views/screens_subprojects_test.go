package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The screens of sub-projects: the sidebar nesting them, and the sheet that makes one.
// Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSubprojects

func TestScreensSubprojects(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, store, _ := newTestApp(t)
			parent, err := store.CreateProject(data.NewProject{Name: "Research"})
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Field notes", "Kitchen"} {
				if _, err := store.CreateProject(data.NewProject{Name: name, ParentID: data.IntPtr(parent.ID)}); err != nil {
					t.Fatal(err)
				}
			}
			app.Refresh()
			manageShot(t, "sidebar-subprojects"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				return nil
			})
			manageShot(t, "new-subproject-sheet"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.openNewProject(parent.ID)
				return nil
			})
		}
	}
}
