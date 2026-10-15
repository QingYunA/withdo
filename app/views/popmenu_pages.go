package views

import (
	"cmp"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// popSpec describes one line of the menu.
type popSpec struct {
	key, label, hint string
	a11y             string   // what assistive technology and the tests call it, when the label is not enough
	icon             *ui.SVG  // in the 18 pt slot at the left, drawn 15 pt
	tint             ui.Color // the icon's color at rest, when it is not the text's (a project's folder)
	mark             string   // text in that slot instead of an icon
	checked, danger  bool
	run              func()
}

// popRow is a line of the menu. The pointer or the keyboard highlights it with the accent, the way
// the system's menus do.
func (m *Main) popRow(c *ui.Context, t *theme.Tokens, s popSpec) {
	pm := &m.pop
	idx := len(pm.items)
	pm.items = append(pm.items, s.run)
	row := ui.ButtonBase(c.Key("pop-"+s.key)).Height(popRowH).Shrink(0).Padding(0, 8).Gap(10).Radius(7).
		AlignItems(ui.Center).Label(cmp.Or(s.a11y, s.label)).Cursor(ui.CursorPointer)
	ink, quiet := t.Text, t.TextTertiary
	if s.danger {
		ink = t.Blocked()
	}
	if row.Hovered() || pm.hl == idx {
		row.Background(t.Accent)
		ink, quiet = t.AccentText, t.AccentText.Alpha(0.8)
	}
	row.Children(func() {
		if s.icon != nil || s.mark != "" {
			ui.Box(c).Size(18, 18).Shrink(0).Center().Children(func() {
				if s.icon != nil {
					color := ink
					if s.tint != (ui.Color{}) && ink == t.Text {
						color = s.tint
					}
					ui.Icon(c, s.icon).Size(theme.IconMenu, theme.IconMenu).TextColor(color)
				} else {
					theme.Text(c, theme.Compact, quiet, s.mark)
				}
			})
		}
		theme.Text(c, theme.Compact, ink, s.label).Grow(1).MinWidth(0).Ellipsis("…")
		switch {
		case s.checked:
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(ink).Shrink(0)
		case s.hint != "":
			theme.Text(c, theme.Meta, quiet, s.hint).Shrink(0)
		}
	})
	if row.Clicked() {
		s.run()
	}
}

// menuRule is the hairline between groups of lines.
func menuRule(c *ui.Context, t *theme.Tokens) {
	ui.Box(c).Height(1).Shrink(0).Margin(5, 8).Background(t.Border)
}

// mainPage is the whole menu: the strips, then what else can be done with the task.
func (m *Main) mainPage(c *ui.Context, t *theme.Tokens, task data.Task) {
	pm := &m.pop
	m.stripLabel(c, t, "status", m.say.PropStatus(), statusLabel(m.say, task.Status))
	m.statusStrip(c, t, task, false)
	m.stripLabel(c, t, "priority", m.say.PropPriority(), m.priorityText(task.Priority))
	m.priorityStrip(c, t, task, false)
	menuRule(c, t)
	m.popRow(c, t, popSpec{key: "move", icon: theme.IconMove, label: m.say.MenuMoveTo(), hint: "M", run: func() {
		pm.page, pm.back, pm.hl, pm.filter = pageMove, true, 0, ""
	}})
	m.popRow(c, t, popSpec{key: "remind", icon: theme.IconClock, label: m.say.MenuRemindMe(), hint: "R", run: func() {
		pm.page, pm.back, pm.hl = pageRemind, true, -1
	}})
	if task.PRURL != "" {
		m.popRow(c, t, popSpec{key: "pr", icon: theme.IconArrowOut, label: m.say.ActionOpenPR(), run: func() {
			m.closeMenu()
			m.app.Act.OpenURL(task.PRURL)
		}})
	}
	m.popRow(c, t, popSpec{key: "ask", icon: theme.IconSparkle, label: m.say.MenuAsk(), hint: "⌘J", run: func() {
		m.closeMenu()
		m.askAboutTask(task.ID)
	}})
	menuRule(c, t)
	m.popRow(c, t, popSpec{key: "delete", icon: theme.IconTrash, label: m.say.MenuDelete(), hint: "⌘⌫", danger: true, run: func() {
		m.closeMenu()
		m.deleteTask(c, task.ID)
	}})
}

// stripButton is one choice of a strip: a tinted square that shows the current one, tints under
// the pointer and has the label above the strip name it (tip). keyboard lists it for the arrow
// keys.
func (m *Main) stripButton(c *ui.Context, t *theme.Tokens, strip, key, label, tip string, on, keyboard bool, run func(), draw func()) {
	pm := &m.pop
	idx := -1
	if keyboard {
		idx = len(pm.items)
		pm.items = append(pm.items, run)
	}
	btn := ui.ButtonBase(c.Key(key)).Grow(1).MinWidth(0).Height(popStrip).Radius(8).Center().
		Label(label).Cursor(ui.CursorPointer)
	hot := btn.Hovered() || (idx >= 0 && pm.hl == idx)
	if hot {
		pm.tipNext = popTip{strip, tip}
	}
	switch {
	case on && hot:
		btn.Background(t.Accent.Alpha(0.2))
	case on:
		btn.Background(t.BgSelected)
	case hot:
		btn.Background(t.BgHover)
	}
	btn.Children(draw)
	if btn.Clicked() {
		run()
	}
}

// statusStrip is the status glyphs in a row; one click sets the status.
func (m *Main) statusStrip(c *ui.Context, t *theme.Tokens, task data.Task, keyboard bool) {
	ui.Row(c.Key("pop-status")).Gap(2).Padding(0, 2, 4).Shrink(0).Children(func() {
		for _, s := range data.Statuses {
			name := statusLabel(m.say, s)
			m.stripButton(c, t, "status", "pop-status-"+string(s), m.say.A11yMenuStatus(name), name, task.Status == s, keyboard,
				func() {
					m.closeMenu()
					m.app.SetStatus(task, s)
				},
				func() { theme.StatusGlyph(c, t, s, 0) })
		}
	})
}

// priorityStrip is P0 to P3 in a row, P0 in the one red; one click sets the priority.
func (m *Main) priorityStrip(c *ui.Context, t *theme.Tokens, task data.Task, keyboard bool) {
	ui.Row(c.Key("pop-priority")).Gap(2).Padding(0, 2, 4).Shrink(0).Children(func() {
		for _, p := range data.Priorities {
			name := priorityName(m.say, p)
			ink := t.Text
			if p == data.P0 {
				ink = t.Blocked()
			}
			m.stripButton(c, t, "priority", "pop-priority-"+string(p), m.say.A11yMenuPriority(name), m.priorityText(p), task.Priority == p, keyboard,
				func() {
					m.closeMenu()
					_ = m.app.Update(task.ID, data.Patch{Priority: data.PriorityPtr(p)})
				},
				func() { theme.Text(c, theme.CompactStrong, ink, string(p)) })
		}
	})
}
