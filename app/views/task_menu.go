package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// taskMenu is the context menu of a task, on a row of any list or a card of the board: Status,
// Priority, Move to, Open PR when there is a pull request, and Delete… which asks first.
func (m *Main) taskMenu(mn *ui.Menu, task data.Task) {
	mn.Submenu(m.say.PropStatus(), func(sm *ui.Menu) {
		for _, s := range data.Statuses {
			if sm.Item(statusLabel(m.say, s)).Checked(task.Status == s).Chosen() {
				m.app.SetStatus(task, s)
			}
		}
	})
	mn.Submenu(m.say.PropPriority(), func(sm *ui.Menu) {
		for _, p := range data.Priorities {
			if sm.Item(priorityLabel(m.say, p)).Checked(task.Priority == p).Chosen() {
				_ = m.app.Update(task.ID, data.Patch{Priority: data.PriorityPtr(p)})
			}
		}
	})
	mn.Submenu(m.say.ManageMoveTo(), func(sm *ui.Menu) { m.moveItems(sm, task) })
	if task.PRURL != "" && mn.Item(m.say.ActionOpenPR()).Chosen() {
		m.app.Act.OpenURL(task.PRURL)
	}
	mn.Separator()
	if mn.Item(m.say.ManageDelete()).Chosen() {
		m.askDeleteTask(task.ID)
	}
}

// moveItems lists the Inbox and the projects, the task's own checked. The Project row of the
// inspector shows the same list.
func (m *Main) moveItems(mn *ui.Menu, task data.Task) {
	if mn.Item(m.say.NavInbox()).Checked(task.ProjectID == nil).Chosen() {
		m.app.MoveTask(task, 0)
	}
	mn.Separator()
	for _, p := range m.app.snap.projects {
		if mn.Item(p.Name).Checked(task.ProjectID != nil && *task.ProjectID == p.ID).Chosen() {
			m.app.MoveTask(task, p.ID)
		}
	}
}

// taskProjectRow is the Project row of the inspector: the task's project, and the Move to list.
func (m *Main) taskProjectRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "project", m.say.ManagePropProject(), func() {
		name, color := m.app.projectName(task.ProjectID), t.Text
		if task.ProjectID == nil {
			name, color = m.say.NavInbox(), t.TextTertiary
		}
		btn := valueButton(c, t, "project").Label(m.say.ManageA11yProject(name))
		btn.Menu(func(mn *ui.Menu) { m.moveItems(mn, task) })
		btn.Children(func() { theme.Text(c, theme.Body, color, name).Ellipsis("…") })
	})
}

// deleteTaskButton is the quiet Delete task… at the foot of the inspector.
func (m *Main) deleteTaskButton(c *ui.Context, t *theme.Tokens, task data.Task) {
	btn := ui.ButtonBase(c.Key("delete-task")).Height(28).Padding(0, 8).Margin(8, 0, 0, -8).Radius(theme.RadiusControl).
		AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorPointer).Label(m.say.ManageDeleteTask())
	ink := t.TextSecondary
	if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.Text
	}
	btn.Children(func() { theme.Text(c, theme.Body, ink, m.say.ManageDeleteTask()) })
	if btn.Clicked() {
		m.askDeleteTask(task.ID)
	}
}
