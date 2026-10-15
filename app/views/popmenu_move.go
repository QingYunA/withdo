package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// maxMoveRows is how many projects the Move to list shows before it scrolls.
const maxMoveRows = 8

// movePage replaces the menu with a filter field and the Inbox and the projects, the task's own
// checked. Typing narrows the list, the arrows walk it and Enter moves the task.
func (m *Main) movePage(c *ui.Context, t *theme.Tokens, task data.Task) {
	pm := &m.pop
	if pm.filter != pm.seen {
		pm.seen, pm.hl = pm.filter, 0
	}
	ui.Row(c.Key("pop-filter")).Height(popRowH).Shrink(0).Margin(0, 0, 4).Padding(0, 10).Gap(8).Radius(7).
		Background(t.BgSubtle).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, theme.IconSearch).Size(14, 14).TextColor(t.TextTertiary).Shrink(0)
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			theme.Compact.Apply(ui.TextInputBase(c.Key("pop-filter-input"), &pm.filter)).TextColor(t.Text).
				Label(m.say.MenuFindProject()).AutoFocus().OnTextKey(m.filterKey)
			if pm.filter == "" {
				theme.Text(c, theme.Compact, t.TextTertiary, m.say.MenuFindProject()).Absolute().Left(0).Top(0).PassThrough()
			}
		})
	})
	choices := m.moveChoices(task)
	if len(choices) == 0 {
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.MenuNoProjectMatch()).Padding(8, 10, 10)
		return
	}
	list := ui.Scroll(c.Key("pop-list")).Shrink(0).Height(float32(min(len(choices), maxMoveRows)) * popRowH)
	list.Children(func() {
		for _, ch := range choices {
			m.popRow(c, t, ch)
		}
	})
}

// moveChoices are the Inbox and the projects whose name holds what is typed.
func (m *Main) moveChoices(task data.Task) []popSpec {
	q := strings.ToLower(strings.TrimSpace(m.pop.filter))
	hit := func(name string) bool { return q == "" || strings.Contains(strings.ToLower(name), q) }
	move := func(project int) func() {
		return func() {
			m.closeMenu()
			m.app.MoveTask(task, project)
		}
	}
	var out []popSpec
	if inbox := m.say.NavInbox(); hit(inbox) {
		out = append(out, popSpec{key: "move-inbox", icon: theme.IconInbox, label: inbox, a11y: m.say.A11yMenuMoveTo(inbox), checked: task.ProjectID == nil, run: move(0)})
	}
	for _, p := range m.app.snap.projects {
		if hit(p.Name) {
			on := task.ProjectID != nil && *task.ProjectID == p.ID
			out = append(out, popSpec{key: "move-" + itoa(p.ID), mark: "#", label: p.Name, a11y: m.say.A11yMenuMoveTo(p.Name), checked: on, run: move(p.ID)})
		}
	}
	return out
}

// filterKey is the keyboard of the filter field: the arrows walk the list, Enter chooses and Esc
// goes back, all asked of the next frame, which has the Context.
func (m *Main) filterKey(k ui.TextKey) bool {
	if k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyDown:
		m.pop.hl++
	case ui.KeyUp:
		m.pop.hl = max(0, m.pop.hl-1)
	case ui.KeyEnter:
		m.pop.enter = true
	case ui.KeyEscape:
		m.pop.esc = true
	default:
		return false
	}
	return true
}

// remindPage is the ready-made reminders, each with the moment it means, and Custom….
func (m *Main) remindPage(c *ui.Context, t *theme.Tokens, task data.Task) {
	pm := &m.pop
	m.menuTitle(c, t, m.say.MenuRemindTitle())
	now := m.app.Now()
	set := func(at time.Time) func() {
		return func() {
			m.closeMenu()
			_ = m.app.Update(task.ID, data.Patch{RemindAt: &at})
		}
	}
	for _, p := range remindPresets(now) {
		var label string
		switch p.kind {
		case remindLater:
			label = m.say.MenuLaterToday()
		case remindTomorrow:
			label = m.say.MenuTomorrow()
		case remindMonday:
			label = m.say.MenuNextMonday()
		default:
			label = m.say.MenuThisWeekend()
		}
		hint := m.clock.When(p.at, now) // "Mon 9:00 AM"; today and tomorrow are in the label already
		if p.kind == remindLater || p.kind == remindTomorrow {
			hint = m.clock.Time(p.at)
		}
		m.popRow(c, t, popSpec{key: "remind-" + itoa(int(p.kind)), label: label, hint: hint, run: set(p.at)})
	}
	menuRule(c, t)
	m.popRow(c, t, popSpec{key: "remind-none", label: m.say.MenuNoReminder(), run: set(time.Time{})})
	m.popRow(c, t, popSpec{key: "remind-custom", label: m.say.MenuCustom(), run: func() {
		pm.when = startingMoment(task.RemindAt, now, true)
		pm.page, pm.hl = pageCalendar, -1
	}})
}

// calendarPage is Custom…: a day and a time of day, saved with Done.
func (m *Main) calendarPage(c *ui.Context, t *theme.Tokens, task data.Task) {
	pm := &m.pop
	m.menuTitle(c, t, m.say.MenuRemindTitle())
	ui.Column(c).Padding(4, 6, 6).Gap(8).Children(func() {
		ui.Calendar(c, &pm.when).Label(m.say.PickerDay())
		ui.TimeInput(c, &pm.when).Label(m.say.PickerTime())
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c.Key("pop-clear"), m.say.PickerClear()).Clicked() {
				m.closeMenu()
				_ = m.app.Update(task.ID, data.Patch{RemindAt: data.TimePtr(time.Time{})})
			}
			if ui.PrimaryButton(c.Key("pop-done"), m.say.MenuSetReminder()).Clicked() {
				m.closeMenu()
				_ = m.app.Update(task.ID, data.Patch{RemindAt: data.TimePtr(pm.when)})
			}
		})
	})
}

// menuTitle is the title of a piece of the menu, with a back chevron when it was opened from the
// whole menu.
func (m *Main) menuTitle(c *ui.Context, t *theme.Tokens, title string) {
	ui.Row(c).Height(popRowH).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
		if m.pop.back || m.pop.page == pageCalendar {
			back := ui.ButtonBase(c.Key("pop-back")).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().Radius(theme.RadiusControl).
				Label(m.say.MenuBack()).TextColor(t.TextSecondary).Cursor(ui.CursorPointer)
			if back.Hovered() {
				back.Background(t.BgHover)
			}
			back.Children(func() { ui.Icon(c, theme.IconBack).Size(14, 14) })
			if back.Clicked() {
				m.menuBack()
			}
		}
		theme.Text(c, theme.CompactStrong, t.Text, title)
	})
}
