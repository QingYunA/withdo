package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The inspector of a task of the user's own (v31): the title with its star, Status as three
// segments, When and Where as chips, then the images, the notes and Hand to an agent… (inspector.go
// lays them out). An agent's task keeps the property list of properties.go and its Activity.

// mineStates are the segments of the status control, in order. Backlog and Cancelled are reachable
// from the task menu only.
var mineStates = []data.Status{data.Todo, data.Doing, data.Done}

// mineProperties is Status, When and Where, with a task's parent first when it is a subtask.
func (m *Main) mineProperties(c *ui.Context, t *theme.Tokens, task data.Task) {
	ui.Column(c).Gap(2).Children(func() {
		m.mineStatusRow(c, t, task)
		m.parentRow(c, t, task)
		m.whenRow(c, t, task)
		m.whereRow(c, t, task)
	})
}

// mineStatusRow is To do | Doing | Done. A status outside the three (a task that was parked or
// cancelled from the menu) selects none and is named after the control, so the row is not blank.
func (m *Main) mineStatusRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "status", m.say.PropStatus(), func() {
		labels := []string{m.say.StatusTodo(), m.say.StatusDoing(), m.say.SegmentDone()}
		sel := -1
		for i, s := range mineStates {
			if s == task.Status {
				sel = i
			}
		}
		ui.Row(c).Shrink(1).MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			if i := segmented(c, t, "inspector-status", labels, sel); i >= 0 {
				m.app.SetStatus(task, mineStates[i])
			}
			if sel < 0 {
				theme.Text(c, theme.Meta, t.TextSecondary, statusLabel(m.say, task.Status)).Shrink(1).MinWidth(0).Ellipsis("…")
			}
		})
	})
}

// chipButton is a chip of the inspector: a quiet pill that opens a picker, 28 pt tall.
func chipButton(c *ui.Context, t *theme.Tokens, key string) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Height(28).Padding(0, 9).Gap(6).Radius(7).Background(t.BgSubtle).
		AlignItems(ui.Center).Shrink(0).MaxWidth(theme.InspectorWidth - 48 - labelWidth).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	return btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
}

// chipLook draws what is inside a chip: a 12 pt icon and its words, in tertiary while nothing is set.
func chipLook(c *ui.Context, t *theme.Tokens, icon *ui.SVG, text string, color ui.Color) {
	ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary).Shrink(0)
	theme.Text(c, theme.Compact, color, text).Shrink(1).MinWidth(0).Ellipsis("…")
}

// whenRow holds the due day and the reminder as chips; each opens the picker it always opened.
func (m *Main) whenRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRowAt(c, t, "when", m.say.PropWhen(), ui.Start, func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(6).Wrap().Children(func() {
			chip := func(key, label string, icon *ui.SVG, current *time.Time, open *bool, patch func(time.Time) data.Patch, withTime bool) {
				btn := chipButton(c, t, "chip-"+key)
				m.dateValue(c, t, btn, key, label, current, open, patch, withTime, func(text string, color ui.Color, set bool) {
					if !set {
						text = label // a chip with nothing set says what it is for
					}
					chipLook(c, t, icon, text, color)
				})
				m.assistMark(c, t, key)
			}
			chip("due", m.say.PropDue(), theme.IconCalendar, task.DueAt, &m.inspector.dueOpen, func(tm time.Time) data.Patch { return data.Patch{DueAt: &tm} }, false)
			chip("remind", m.say.PropRemind(), theme.IconBell, task.RemindAt, &m.inspector.remOpen, func(tm time.Time) data.Patch { return data.Patch{RemindAt: &tm} }, true)
		})
	})
}

// whereRow holds the category and the project (with its milestone) as chips; they open the Category
// page and the Move to… list of the task menu.
func (m *Main) whereRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRowAt(c, t, "where", m.say.PropWhere(), ui.Start, func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(6).Wrap().Children(func() {
			m.categoryChip(c, t, task)
			m.assistMark(c, t, "category")
			m.projectChip(c, t, task)
			m.assistMark(c, t, "project")
		})
	})
}

func (m *Main) categoryChip(c *ui.Context, t *theme.Tokens, task data.Task) {
	label := m.categoryText(task.CategoryID)
	btn := chipButton(c, t, "chip-category").Label(m.say.A11yCategory(label))
	if btn.Clicked() {
		m.openMenuBelow(btn, task.ID, pageCategory)
	}
	btn.Children(func() {
		if cat, ok := m.app.Category(task.CategoryID); ok {
			categoryGlyph(c, cat, theme.IconSmall)
			theme.Text(c, theme.Compact, t.Text, label).Shrink(1).MinWidth(0).Ellipsis("…")
			return
		}
		chipLook(c, t, theme.IconList, m.say.PropCategory(), t.TextTertiary)
	})
}

func (m *Main) projectChip(c *ui.Context, t *theme.Tokens, task data.Task) {
	name, color := m.app.projectName(task.ProjectID), t.Text
	if task.ProjectID == nil {
		name, color = m.say.NavNoProject(), t.TextTertiary
	} else if task.MilestoneID != nil {
		for _, x := range m.app.snap.milestones[*task.ProjectID] {
			if x.ID == *task.MilestoneID {
				name += " · " + x.Title
			}
		}
	}
	btn := chipButton(c, t, "chip-project").Label(m.say.ManageA11yProject(name))
	if btn.Clicked() {
		m.openMenuBelow(btn, task.ID, pageMove)
	}
	btn.Children(func() { chipLook(c, t, theme.IconFolder, name, color) })
}

// titleWithStar is the title and, at its right, the important star: a click toggles it.
func (m *Main) titleWithStar(c *ui.Context, t *theme.Tokens, task data.Task) {
	ui.Row(c.Key("title-row")).Gap(8).AlignItems(ui.Start).Children(func() {
		ui.Box(c.Key("title-wrap")).Grow(1).MinWidth(0).Children(func() { m.titleField(c, t, task) })
		ui.Row(c.Key("title-star")).Shrink(0).Gap(4).Height(theme.HitTarget).AlignItems(ui.Center).Margin(-2, -6, 0, 0).Children(func() {
			m.assistMark(c, t, "priority") // the assistant just starred it
			label := m.say.A11yImportant()
			btn := ui.ButtonBase(c.Key("important")).Size(theme.HitTarget, theme.HitTarget).Center().Radius(theme.RadiusControl).Shrink(0).
				Label(label).Tooltip(label).Cursor(ui.CursorPointer)
			if btn.Hovered() {
				btn.Background(t.BgHover)
			}
			btn.Children(func() { theme.Star(c, t, starred(task), theme.IconRow) })
			if btn.Clicked() {
				m.toggleImportant(task)
			}
		})
	})
}

// handOffButton is Hand to an agent… at the foot of a task of the user's own that is not finished.
// It is the drop on the Agents tray, by button.
func (m *Main) handOffButton(c *ui.Context, t *theme.Tokens, task data.Task) {
	if !canHandOff(task) {
		return
	}
	label := m.say.InspectorHandOff()
	btn := ui.ButtonBase(c.Key("hand-off")).Height(32).Padding(0, 12).Gap(8).Radius(theme.RadiusControl).Border(theme.Hairline, t.Border).
		AlignItems(ui.Center).Justify(ui.Center).Label(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() {
		ui.Icon(c, theme.IconAgent).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Accent)
		theme.Text(c, theme.Compact, t.Text, label)
	})
	if btn.Clicked() {
		m.handOff(c, task.ID)
	}
}
