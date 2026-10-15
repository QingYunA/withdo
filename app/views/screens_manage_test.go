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
					subs:   map[string]bool{m.say.PropStatus(): true, m.say.PropPriority(): true, m.say.ManageMoveTo(): true},
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

func TestScreensManageSearch(t *testing.T) {
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			query := map[string]string{"en": "port", "zh-Hans": "port"}[l.tag]
			manageShot(t, "search-results"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				tt.Key(ui.Cmd, ui.KeyF)
				tt.Frame()
				tt.Type(query)
				tt.Key(0, ui.KeyDown)
				tt.Key(0, ui.KeyDown)
				return nil
			})
			manageShot(t, "search-none"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				tt.Key(ui.Cmd, ui.KeyF)
				tt.Frame()
				tt.Type("kubernetes")
				return nil
			})
		}
	}
}

func TestScreensManageLogbook(t *testing.T) {
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, store, _ := newTestApp(t)
			_, _ = store.UpdateTask(19, data.Patch{Status: data.StatusPtr(data.Cancelled)})
			_, _ = store.UpdateTask(8, data.Patch{Status: data.StatusPtr(data.Done)})
			app.Refresh()
			manageShot(t, "logbook"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenLogbook})
				return nil
			})
			empty, _ := newEmptyApp(t)
			manageShot(t, "logbook-empty"+suffix, empty, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenLogbook})
				return nil
			})
		}
	}
}

func TestScreensManageMilestones(t *testing.T) {
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app, _, _ := newTestApp(t)
			manageShot(t, "milestone-menu"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				r, _ := tt.Find(m.say.A11yMilestone(m.say.MilestoneAll()))
				x, y := r.X, r.Y+r.H+4
				if err := tt.Click(m.say.A11yMilestone(m.say.MilestoneAll())); err != nil {
					t.Fatal(err)
				}
				edit := m.say.ManageMilestoneEdit("v0.2 native")
				return &menuSpec{
					x: x, y: y, items: tt.Menu(),
					subs:    map[string]bool{edit: true, m.say.ManageMilestoneEdit("v0.1 spike"): true},
					checked: map[string]bool{m.say.MilestoneAll(): true},
					flyout: &flyoutSpec{parent: edit, items: []string{
						m.say.ManageMilestoneRename(), m.say.ManageMilestoneDue(), m.say.ManageMilestoneClose()}},
				}
			})
			manageShot(t, "milestone-new"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				m.newMilestone(1)
				tt.Frame()
				tt.Type("v0.3 polish")
				return nil
			})
		}
	}
}
