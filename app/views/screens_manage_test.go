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

// projectMenuAt right-clicks a project in the sidebar, which opens its drawn menu.
func projectMenuAt(tt *ui.Tester, name string) {
	r, _ := tt.Find(name)
	tt.RightClickAt(r.X+24, r.Y+r.H/2)
}

func TestScreensManageProjects(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			manageShot(t, "sidebar-new-project"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				if err := tt.Click(m.say.ManageNewProjectTip()); err != nil {
					t.Fatal(err)
				}
				tt.Type("Garden")
				_ = tt.Click(m.say.ManageRepoPlaceholder())
				tt.Type("QingYunA/withdo")
				tt.Key(0, ui.KeyEnter)
			})
			manageShot(t, "project-menu"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				projectMenuAt(tt, "withdo")
			})
			manageShot(t, "project-delete"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				m.askDeleteProject(1)
			})
		}
	}
}

func TestScreensManageTasks(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			manageShot(t, "task-menu"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				r, _ := tt.Find("Fix tray race on launch")
				tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
			})
			manageShot(t, "task-deleted"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				r, _ := tt.Find("Fix tray race on launch")
				tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
				if err := tt.Click(m.say.MenuDelete()); err != nil {
					t.Fatal(err)
				}
			})
			manageShot(t, "inspector-project"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.OpenTask(5)
			})
		}
	}
}

func TestScreensManageSearch(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			query := map[string]string{"en": "port", "zh-Hans": "port"}[l.tag]
			manageShot(t, "search-results"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				tt.Key(ui.Cmd, ui.KeyF)
				tt.Frame()
				tt.Type(query)
				tt.Key(0, ui.KeyDown)
				tt.Key(0, ui.KeyDown)
			})
			manageShot(t, "search-none"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				tt.Key(ui.Cmd, ui.KeyF)
				tt.Frame()
				tt.Type("kubernetes")
			})
		}
	}
}

func TestScreensManageCompleted(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, store, _ := newTestApp(t)
			_, _ = store.UpdateTask(19, data.Patch{Status: data.StatusPtr(data.Cancelled)})
			_, _ = store.UpdateTask(8, data.Patch{Status: data.StatusPtr(data.Done)})
			app.Refresh()
			manageShot(t, "completed"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenCompleted})
			})
			empty, _ := newEmptyApp(t)
			manageShot(t, "completed-empty"+suffix, empty, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenCompleted})
			})
		}
	}
}

func TestScreensManageMilestones(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			manageShot(t, "milestone-menu"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				if err := tt.Click(m.say.A11yMilestone(m.say.MilestoneAll())); err != nil {
					t.Fatal(err)
				}
				// Rest on the first Edit item, which opens its submenu.
				r, _ := tt.Find(m.say.ManageMilestoneEdit("v0.2 native"))
				tt.Move(r.X+r.W/2, r.Y+r.H/2)
			})
			manageShot(t, "milestone-new"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				m.newMilestone(1)
				tt.Frame()
				tt.Type("v0.3 polish")
			})
		}
	}
}
