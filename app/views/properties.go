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
	ui.Column(c).Gap(2).Children(func() {
		m.statusRow(c, t, task)
		m.priorityRow(c, t, task)
		m.milestoneRow(c, t, task)
		m.dateRow(c, t, "Due", task.DueAt, &m.inspector.dueOpen, func(tm time.Time) data.Patch { return data.Patch{DueAt: &tm} }, false)
		m.dateRow(c, t, "Remind", task.RemindAt, &m.inspector.remOpen, func(tm time.Time) data.Patch { return data.Patch{RemindAt: &tm} }, true)
		if task.Source == data.SourceAgent || task.Branch != "" {
			m.branchRow(c, t, task)
		}
		if task.PRNumber > 0 {
			m.prRow(c, t, task)
		}
	})
}

// propRow lays out a label and a value on one 32 pt line.
func propRow(c *ui.Context, t *theme.Tokens, label string, value func()) {
	ui.Row(c.Key("prop-" + label)).MinHeight(theme.RowOneLine).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, label).Width(labelWidth).Shrink(0)
		value()
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
	propRow(c, t, "Status", func() {
		btn := valueButton(c, t, "status").Label("Status: " + task.Status.Label())
		btn.Menu(func(mn *ui.Menu) {
			for _, s := range data.Statuses {
				if mn.Item(s.Label()).Checked(task.Status == s).Chosen() {
					m.app.SetStatus(task, s)
				}
			}
		})
		btn.Children(func() {
			theme.StatusGlyph(c, t, task.Status, 0)
			theme.Text(c, theme.Body, t.Text, task.Status.Label())
		})
	})
}

var priorityNames = map[data.Priority]string{data.P0: "Urgent", data.P1: "High", data.P2: "Normal", data.P3: "Low"}

func (m *Main) priorityRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	propRow(c, t, "Priority", func() {
		label := string(task.Priority) + "  " + priorityNames[task.Priority]
		btn := valueButton(c, t, "priority").Label("Priority: " + label)
		btn.Menu(func(mn *ui.Menu) {
			for _, p := range data.Priorities {
				if mn.Item(string(p) + "  " + priorityNames[p]).Checked(task.Priority == p).Chosen() {
					_ = m.app.Update(task.ID, data.Patch{Priority: data.PriorityPtr(p)})
				}
			}
		})
		btn.Children(func() {
			theme.PriorityBars(c, t, task.Priority)
			theme.Text(c, theme.Body, t.Text, label)
		})
	})
}

func (m *Main) milestoneRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	if task.ProjectID == nil {
		return
	}
	ms := m.app.snap.milestones[*task.ProjectID]
	propRow(c, t, "Milestone", func() {
		label, color := "None", t.TextTertiary
		for _, x := range ms {
			if task.MilestoneID != nil && x.ID == *task.MilestoneID {
				label, color = x.Title, t.Text
			}
		}
		btn := valueButton(c, t, "milestone").Label("Milestone: " + label)
		btn.Menu(func(mn *ui.Menu) {
			if mn.Item("None").Checked(task.MilestoneID == nil).Chosen() {
				_ = m.app.Update(task.ID, data.Patch{MilestoneID: data.IntPtr(0)})
			}
			for _, x := range ms {
				on := task.MilestoneID != nil && *task.MilestoneID == x.ID
				if mn.Item(milestoneLabel(x, m.app.Now())).Checked(on).Chosen() {
					_ = m.app.Update(task.ID, data.Patch{MilestoneID: data.IntPtr(x.ID)})
				}
			}
		})
		btn.Children(func() { theme.Text(c, theme.Body, color, label) })
	})
}

func (m *Main) branchRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	propRow(c, t, "Branch", func() {
		box := ui.Box(c).Grow(1).MinWidth(0).Height(28).Justify(ui.Center)
		box.Children(func() {
			branch, ok := textField(c, t, m.inspector.fields, fieldSpec{
				name: "Branch", value: task.Branch, style: theme.Mono, color: t.Text, placeholder: "None",
			})
			if ok {
				_ = m.app.Update(task.ID, data.Patch{Branch: &branch})
			}
		})
	})
}

func (m *Main) prRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	propRow(c, t, "Pull request", func() {
		btn := valueButton(c, t, "pr").Label("Open pull request #" + itoa(task.PRNumber)).Tooltip(task.PRURL)
		btn.Children(func() {
			theme.Text(c, theme.Mono, t.Accent, "#"+itoa(task.PRNumber))
			ui.Icon(c, theme.IconArrowOut).Size(13, 13).TextColor(t.Accent)
		})
		if btn.Clicked() {
			m.app.Act.OpenURL(task.PRURL)
		}
	})
}

// dateRow is a due or reminder time. It opens a calendar (and a time of day for reminders) in a
// popover and saves as soon as a day or time is picked.
func (m *Main) dateRow(c *ui.Context, t *theme.Tokens, label string, current *time.Time, open *bool, patch func(time.Time) data.Patch, withTime bool) {
	task, _ := m.app.Task(m.inspector.lastOpen)
	propRow(c, t, label, func() {
		text, color := "None", t.TextTertiary
		if current != nil {
			text, color = formatMoment(*current, m.app.Now(), withTime), t.Text
		}
		btn := valueButton(c, t, "date-"+label).Label(label + ": " + text)
		btn.Children(func() {
			theme.Text(c, theme.Body, color, text)
		})
		if btn.Clicked() {
			*open = !*open
			m.inspector.when = startingMoment(current, m.app.Now(), withTime)
		}
		ui.Popover(c, btn, open, func() {
			m.datePicker(c, t, task, patch, withTime, open)
		})
	})
}

func (m *Main) datePicker(c *ui.Context, t *theme.Tokens, task data.Task, patch func(time.Time) data.Patch, withTime bool, open *bool) {
	ui.Column(c).Padding(8).Gap(8).Children(func() {
		changed := ui.Calendar(c, &m.inspector.when).Label("Day").Changed()
		if withTime {
			changed = ui.TimeInput(c, &m.inspector.when).Label("Time").Changed() || changed
		}
		if changed {
			_ = m.app.Update(task.ID, patch(m.inspector.when))
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, "Clear").Clicked() {
				_ = m.app.Update(task.ID, patch(time.Time{}))
				*open = false
			}
			if ui.PrimaryButton(c, "Done").Clicked() {
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

// formatMoment writes "Tomorrow" or "Tomorrow, 10:00".
func formatMoment(t, now time.Time, withTime bool) string {
	day := data.DayWord(t, now)
	day = string(upperFirst(day))
	if withTime || t.Hour() != 0 && t.Minute() != 0 {
		return day + ", " + t.Format("15:04")
	}
	return day
}

func upperFirst(s string) []byte {
	b := []byte(s)
	if len(b) > 0 && b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return b
}
