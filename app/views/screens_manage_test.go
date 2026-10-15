package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The screens of managing projects, tasks and milestones by hand, in both languages and both
// appearances. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensManage

// projectMenuAt right-clicks a project in the sidebar and returns the menu to draw.
func projectMenuAt(tt *ui.Tester, name string) *menuSpec {
	r, _ := tt.Find(name)
	x, y := r.X+24, r.Y+r.H/2
	tt.RightClickAt(x, y)
	return &menuSpec{x: x, y: y, items: tt.Menu()}
}

func TestScreensManageProjects(t *testing.T) {
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			manageShot(t, "sidebar-new-project"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				if err := tt.Click(m.say.ManageNewProjectTip()); err != nil {
					t.Fatal(err)
				}
				tt.Type("Garden")
				_ = tt.Click(m.say.ManageRepoPlaceholder())
				tt.Type("QingYunA/adhd-todo")
				tt.Key(0, ui.KeyEnter)
				return nil
			})
			manageShot(t, "project-menu"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				return projectMenuAt(tt, "adhd-todo")
			})
			manageShot(t, "project-delete"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				m.askDeleteProject(1)
				return nil
			})
			archived, _, _ := newTestApp(t)
			_ = archived.Store.ArchiveProject(3)
			archived.Refresh()
			manageShot(t, "settings-archived"+suffix, archived, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.OpenSettings()
				return nil
			})
		}
	}
}

func TestScreensManageTasks(t *testing.T) {
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			manageShot(t, "task-menu"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				r, _ := tt.Find("Fix tray race on launch")
				x, y := r.X+40, r.Y+r.H/2
				tt.RightClickAt(x, y)
				var statuses []string
				for _, s := range data.Statuses {
					statuses = append(statuses, statusLabel(m.say, s))
				}
				return &menuSpec{
					x: x, y: y, items: tt.Menu(),
					subs: map[string]bool{m.say.PropStatus(): true, m.say.PropPriority(): true, m.say.ManageMoveTo(): true},
					flyout: &flyoutSpec{parent: m.say.PropStatus(), items: statuses, checked: map[string]bool{statusLabel(m.say, data.Blocked): true}},
				}
			})
			manageShot(t, "task-delete"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				m.askDeleteTask(3)
				return nil
			})
			manageShot(t, "inspector-project"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.OpenTask(5)
				return nil
			})
		}
	}
}
