package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// taskProjectRow is the Project row of the inspector: the task's project, and under it the Move to
// list of the task menu.
func (m *Main) taskProjectRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	propRow(c, t, "project", m.say.ManagePropProject(), func() {
		name, color := m.app.projectName(task.ProjectID), t.Text
		if task.ProjectID == nil {
			name, color = m.say.NavInbox(), t.TextTertiary
		}
		btn := valueButton(c, t, "project").Label(m.say.ManageA11yProject(name))
		if btn.Clicked() {
			m.openMenuBelow(btn, task.ID, pageMove)
		}
		btn.Children(func() { theme.Text(c, theme.Compact, color, name).Ellipsis("…") })
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
		m.deleteTask(c, task.ID)
	}
}
