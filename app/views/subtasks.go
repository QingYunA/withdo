package views

import (
	"sort"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// subtasksOf are a task's subtasks in the order they were made, cancelled ones included: the
// inspector lists everything, where the Agents screen drops what was called off.
func (a *App) subtasksOf(id int) []data.Task {
	subs := data.Subtasks(a.snap.tasks, id)
	sort.SliceStable(subs, func(i, j int) bool { return subs[i].ID < subs[j].ID })
	return subs
}

// subtasks is the inspector's list of the subtasks of a parent: avatar, title and where each stands.
// Clicking one opens it. A task with none draws nothing.
func (m *Main) subtasks(c *ui.Context, t *theme.Tokens, task data.Task) {
	subs := m.app.subtasksOf(task.ID)
	if len(subs) == 0 {
		return
	}
	ui.Column(c.Key("subtasks")).Gap(2).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, m.say.InspectorSubtasks()).Margin(0, 0, 4)
		for _, s := range subs {
			row := ui.Row(c.Key("sub-"+itoa(s.ID))).Shrink(0).Margin(0, -8).Padding(6, 8).Gap(10).Radius(theme.RadiusCard).
				AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(m.say.A11yOpenTask(s.Title))
			if row.Hovered() {
				row.Background(t.BgHover)
			}
			row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
			row.Children(func() {
				theme.AvatarIcon(c.Key("author"), m.app.taskWho(s), theme.AvatarSmall).Shrink(0)
				theme.Text(c, theme.Body, t.Text, s.Title).Grow(1).Shrink(1).MinWidth(0).Ellipsis("…")
				m.statusPill(c, t, "pill", s.Status)
			})
			if row.Clicked() {
				m.openRow(s.ID)
			}
		}
	})
}

// parentRow is the property of a subtask that names the task it belongs to, as a link that opens it.
func (m *Main) parentRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	parent, ok := m.app.parentOf(task)
	if !ok {
		return
	}
	m.propRow(c, t, "parent", m.say.PropParent(), func() {
		btn := valueButton(c, t, "parent").Label(m.say.A11yOpenTask(parent.Title))
		btn.Children(func() {
			theme.Text(c, theme.Compact, t.Accent, parent.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		})
		if btn.Clicked() {
			m.openRow(parent.ID)
		}
	})
}
