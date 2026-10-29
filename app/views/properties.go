package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const labelWidth float32 = 96

// properties is the compact list of what can be set on a task. Labels sit on the left in
// secondary, values on the right and edit in place.
func (m *Main) properties(c *ui.Context, t *theme.Tokens, task data.Task) {
	if mine(task) {
		m.mineProperties(c, t, task)
		return
	}
	ui.Column(c).Gap(2).Children(func() {
		m.statusRow(c, t, task)
		m.fromRow(c, t, task)
		m.parentRow(c, t, task)
		m.priorityRow(c, t, task)
		m.categoryRow(c, t, task)
		m.taskProjectRow(c, t, task)
		m.milestoneRow(c, t, task)
		m.dateRow(c, t, "due", m.say.PropDue(), task.DueAt, &m.inspector.dueOpen, func(tm time.Time) data.Patch { return data.Patch{DueAt: &tm} }, false)
		m.dateRow(c, t, "remind", m.say.PropRemind(), task.RemindAt, &m.inspector.remOpen, func(tm time.Time) data.Patch { return data.Patch{RemindAt: &tm} }, true)
		if task.Source == data.SourceAgent || task.Branch != "" {
			m.branchRow(c, t, task)
		}
		if task.PRNumber > 0 {
			m.prRow(c, t, task)
		}
	})
}

// propIcons is the line icon before each property's label, by the row's key.
var propIcons = map[string]*ui.SVG{
	"status": theme.IconStatus, "from": theme.IconAgent, "priority": theme.IconFlag, "category": theme.IconList, "project": theme.IconFolder,
	"milestone": theme.IconDiamond, "due": theme.IconCalendar, "remind": theme.IconBell, "branch": theme.IconBranch,
	"pr": theme.IconPR, "parent": theme.IconBranch, "when": theme.IconClock, "where": theme.IconFolder,
}

// propRow lays out a label (a line icon and its words) and a value on one 32 pt line.
func (m *Main) propRow(c *ui.Context, t *theme.Tokens, key, label string, value func()) {
	m.propRowAt(c, t, key, label, ui.Center, value)
}

// propRowAt is propRow with the label level with the value's first line (ui.Start) for a value of
// chips that may wrap, or centered (ui.Center).
func (m *Main) propRowAt(c *ui.Context, t *theme.Tokens, key, label string, align ui.Align, value func()) {
	labelH := float32(0)
	if align == ui.Start {
		labelH = 28 // the height of a chip
	}
	ui.Row(c.Key("prop-" + key)).MinHeight(theme.RowOneLine).AlignItems(align).Children(func() {
		ui.Row(c).Width(labelWidth).MinHeight(labelH).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
			if icon := propIcons[key]; icon != nil {
				ui.Icon(c, icon).Size(theme.IconProp, theme.IconProp).TextColor(t.TextTertiary).Shrink(0)
			}
			theme.Text(c, theme.Meta, t.TextSecondary, label).Shrink(1).MinWidth(0).Ellipsis("…")
		})
		value()
		m.assistMark(c, t, key)
	})
}

// valueButton is a value that opens a menu or popover. Its text lines up with the other values.
func valueButton(c *ui.Context, t *theme.Tokens, key string) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Height(28).Padding(0, 8).Margin(0, 0, 0, -8).Gap(8).Radius(theme.RadiusControl).
		AlignItems(ui.Center).Shrink(1).MinWidth(0)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	return btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
}

func (m *Main) statusRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "status", m.say.PropStatus(), func() {
		btn := valueButton(c, t, "status").Label(m.say.A11yStatus(statusLabel(m.say, task.Status)))
		if btn.Clicked() {
			m.openMenuBelow(btn, task.ID, pageStatus)
		}
		btn.Children(func() {
			theme.StatusGlyph(c, t, task.Status, 0)
			theme.Text(c, theme.Compact, t.Text, statusLabel(m.say, task.Status))
		})
	})
}

// fromRow says who wrote the task: its avatar and the agent's name, or "You".
func (m *Main) fromRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "from", m.say.PropFrom(), func() {
		ui.Row(c.Key("from")).Gap(8).Shrink(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
			theme.AvatarIcon(c, m.app.taskWho(task), theme.AvatarRow).Shrink(0)
			theme.Text(c, theme.Compact, t.Text, m.taskWhoName(task)).Shrink(1).MinWidth(0).Ellipsis("…")
		})
	})
}

func (m *Main) priorityRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "priority", m.say.PropPriority(), func() {
		label := priorityLabel(m.say, task.Priority)
		btn := valueButton(c, t, "priority").Label(m.say.A11yPriority(label))
		if btn.Clicked() {
			m.openMenuBelow(btn, task.ID, pagePriority)
		}
		btn.Children(func() { theme.Text(c, theme.Compact, t.Text, label) })
	})
}

// categoryRow is the task's own category: Work, Life, or None. A task in a repository's project
// counts as work without one; the row shows only what was set.
func (m *Main) categoryRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "category", m.say.PropCategory(), func() {
		label, color := m.categoryText(task.CategoryID), t.Text
		if task.CategoryID == 0 {
			color = t.TextTertiary
		}
		btn := valueButton(c, t, "category").Label(m.say.A11yCategory(label))
		if btn.Clicked() {
			m.openMenuBelow(btn, task.ID, pageCategory)
		}
		btn.Children(func() { m.categoryValue(c, task.CategoryID, color) })
	})
}

func (m *Main) milestoneRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	if task.ProjectID == nil {
		return
	}
	ms := m.app.snap.milestones[*task.ProjectID]
	m.propRow(c, t, "milestone", m.say.PropMilestone(), func() {
		label, color := m.say.ValueNone(), t.TextTertiary
		for _, x := range ms {
			if task.MilestoneID != nil && x.ID == *task.MilestoneID {
				label, color = x.Title, t.Text
			}
		}
		btn := valueButton(c, t, "milestone").Label(m.say.A11yMilestone(label))
		btn.PopoverMenuButton(func(mn *ui.Menu) {
			if mn.Item(m.say.ValueNone()).Checked(task.MilestoneID == nil).Chosen() {
				_ = m.app.Update(task.ID, data.Patch{MilestoneID: data.IntPtr(0)})
			}
			for _, x := range ms {
				on := task.MilestoneID != nil && *task.MilestoneID == x.ID
				if mn.Item(m.milestoneLabel(x)).Checked(on).Chosen() {
					_ = m.app.Update(task.ID, data.Patch{MilestoneID: data.IntPtr(x.ID)})
				}
			}
		})
		btn.Children(func() { theme.Text(c, theme.Compact, color, label) })
	})
}

func (m *Main) branchRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "branch", m.say.FieldBranch(), func() {
		box := ui.Box(c).Grow(1).MinWidth(0).Height(28).Justify(ui.Center)
		box.Children(func() {
			branch, ok := textField(c, t, m.inspector.fields, fieldSpec{
				name: "Branch", label: m.say.FieldBranch(), value: task.Branch, style: theme.Mono, color: t.Text,
				placeholder: m.say.ValueNone(), escReverts: true, ellipsis: true,
			})
			if ok {
				_ = m.app.Update(task.ID, data.Patch{Branch: &branch})
			}
		})
	})
}

func (m *Main) prRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.propRow(c, t, "pr", m.say.PropPullRequest(), func() {
		btn := valueButton(c, t, "pr").Label(m.say.A11yOpenPR(itoa(task.PRNumber))).Tooltip(task.PRURL)
		btn.Children(func() {
			theme.Text(c, theme.Mono, t.Accent, "#"+itoa(task.PRNumber))
			ui.Icon(c, theme.IconArrowOut).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Accent)
		})
		if btn.Clicked() {
			m.app.Act.OpenURL(task.PRURL)
		}
	})
}

// dateRow is a due or reminder time. It opens a calendar (and a time of day for reminders) in a
// popover and saves as soon as a day or time is picked.
func (m *Main) dateRow(c *ui.Context, t *theme.Tokens, key, label string, current *time.Time, open *bool, patch func(time.Time) data.Patch, withTime bool) {
	m.propRow(c, t, key, label, func() {
		btn := valueButton(c, t, "date-"+key)
		m.dateValue(c, t, btn, key, label, current, open, patch, withTime, func(text string, color ui.Color, _ bool) {
			theme.Text(c, theme.Compact, color, text)
		})
	})
}

// dateValue is a due or reminder time as a button (a value of the property list or a chip, which
// draws its own look) and what it opens: a calendar popover, or the reminder presets of the task menu.
func (m *Main) dateValue(c *ui.Context, t *theme.Tokens, btn ui.Element, key, label string, current *time.Time, open *bool,
	patch func(time.Time) data.Patch, withTime bool, look func(text string, color ui.Color, set bool)) {
	task, _ := m.app.Task(m.inspector.lastOpen)
	text, color := m.say.ValueNone(), t.TextTertiary
	if current != nil {
		text, color = m.clock.Moment(*current, m.app.Now(), withTime), t.Text
	}
	btn.Label(m.say.A11yDate(label, text))
	btn.Children(func() { look(text, color, current != nil) })
	switch {
	case key == "remind": // the reminders are the task menu's presets
		if btn.Clicked() {
			m.openMenuBelow(btn, task.ID, pageRemind)
		}
		return
	case btn.Clicked():
		*open = !*open
		m.inspector.when = startingMoment(current, m.app.Now(), withTime)
	}
	ui.Popover(c, btn, open, func() {
		m.datePicker(c, t, task, patch, withTime, open)
	})
}

func (m *Main) datePicker(c *ui.Context, t *theme.Tokens, task data.Task, patch func(time.Time) data.Patch, withTime bool, open *bool) {
	ui.Column(c).Padding(8).Gap(8).Children(func() {
		changed := ui.Calendar(c, &m.inspector.when).Label(m.say.PickerDay()).Changed()
		if withTime {
			changed = ui.TimeInput(c, &m.inspector.when).Label(m.say.PickerTime()).Changed() || changed
		}
		if changed {
			_ = m.app.Update(task.ID, patch(m.inspector.when))
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, m.say.PickerClear()).Clicked() {
				_ = m.app.Update(task.ID, patch(time.Time{}))
				*open = false
			}
			if ui.PrimaryButton(c, m.say.PickerDone()).Clicked() {
				*open = false
			}
		})
	})
}

// startingMoment is where the picker starts: the current value, or tomorrow morning.
func startingMoment(cur *time.Time, now time.Time, withTime bool) time.Time {
	if cur != nil {
		return *cur
	}
	y, mo, d := now.Date()
	hour := 9
	if withTime {
		hour = now.Add(time.Hour).Hour()
	}
	return time.Date(y, mo, d+1, hour, 0, 0, 0, now.Location())
}
