package views

import (
	"cmp"
	"strings"

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
	colored          bool     // the icon is a category's: its palette pixels keep their colors at rest
	checked, danger  bool
	more             bool // a chevron after the hint: the choice opens a page
	disabled         bool // drawn quiet; the pointer and the arrow keys pass over it
	run              func()
}

// popRow is a line of the menu. The pointer or the keyboard highlights it with the accent, the way
// the system's menus do.
func (m *Main) popRow(c *ui.Context, t *theme.Tokens, s popSpec) {
	pm := &m.pop
	idx := -1
	if !s.disabled {
		idx = len(pm.items)
		pm.items = append(pm.items, s.run)
	}
	row := ui.ButtonBase(c.Key("pop-"+s.key)).Height(popRowH).Shrink(0).Padding(0, 8).Gap(10).Radius(7).
		AlignItems(ui.Center).Label(cmp.Or(s.a11y, s.label)).Cursor(ui.CursorPointer)
	ink, quiet := t.Text, t.TextTertiary
	if s.danger {
		ink = t.Blocked()
	}
	if s.disabled {
		ink = t.TextTertiary
	} else if row.Hovered() || pm.hl == idx {
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
					var glyph ui.Element
					if s.colored && ink == t.Text {
						glyph = ui.Image(c, s.icon) // a category's palette pixels keep their colors
					} else {
						glyph = ui.Icon(c, s.icon) // one color
					}
					glyph.Size(theme.IconMenu, theme.IconMenu).TextColor(color)
				} else {
					theme.Text(c, theme.Compact, quiet, s.mark)
				}
			})
		}
		theme.Text(c, theme.Compact, ink, s.label).Grow(1).MinWidth(0).Ellipsis("…")
		switch {
		case s.checked:
			ui.Icon(c, theme.IconCheck).Size(theme.IconSmall, theme.IconSmall).TextColor(ink).Shrink(0)
		case s.hint != "":
			theme.Text(c, theme.Meta, quiet, s.hint).Shrink(0)
		}
		if s.more {
			ui.Icon(c, theme.IconForward).Size(theme.IconSmall, theme.IconSmall).TextColor(quiet).Shrink(0).Margin(0, 0, 0, -4)
		}
	})
	if row.Clicked() && !s.disabled {
		s.run()
	}
}

// menuRuleH is the height of the hairline between groups of lines with its margins.
const menuRuleH float32 = 11

// menuRule is the hairline between groups of lines.
func menuRule(c *ui.Context, t *theme.Tokens) {
	ui.Box(c).Height(1).Shrink(0).Margin(5, 8).Background(t.Border)
}

// mainPage is the whole menu: the strips, then what else can be done with the task.
func (m *Main) mainPage(c *ui.Context, t *theme.Tokens, task data.Task) {
	pm := &m.pop
	m.stripLabel(c, t, "status", m.say.PropStatus(), statusLabel(m.say, task.Status))
	m.statusStrip(c, t, task, false)
	one := []data.Task{task}
	if mine(task) { // the user's own task has the star, one line, not the four priorities
		m.popRow(c, t, m.starPopRow(t, task))
	} else {
		m.stripLabel(c, t, "priority", m.say.PropPriority(), m.priorityText(task.Priority))
		m.priorityStrip(c, t, task, false)
	}
	menuRule(c, t)
	m.popRow(c, t, m.categoryPopRow(task))
	m.popRow(c, t, actionSpec(actionOf(one, actMove), "", m.say.MenuMoveTo(), func() { pm.openPage(pageMove) }))
	m.popRow(c, t, popSpec{key: "remind", icon: theme.IconClock, label: m.say.MenuRemindMe(), hint: keyRemind.hint(), run: func() {
		pm.page, pm.back, pm.hl = pageRemind, true, -1
	}})
	if task.PRURL != "" {
		m.popRow(c, t, popSpec{key: "pr", icon: theme.IconArrowOut, label: m.say.ActionOpenPR(), run: func() {
			m.closeMenu()
			m.app.Act.OpenURL(task.PRURL)
		}})
	}
	if task.Source == data.SourceAgent {
		m.popRow(c, t, popSpec{key: "open-in", icon: theme.IconAgents, label: m.say.MenuOpenIn(), run: func() {
			pm.page, pm.back, pm.hl = pageOpenIn, true, 0
		}})
	}
	m.popRow(c, t, popSpec{key: "ask", icon: theme.IconSparkle, label: m.say.MenuAsk(), hint: keyAsk.hint(), run: func() {
		m.closeMenu()
		m.askAboutTask(task.ID)
	}})
	menuRule(c, t)
	m.popRow(c, t, actionSpec(actionOf(one, actDelete), "", m.say.MenuDelete(), func() {
		m.closeMenu()
		m.deleteTask(c, task.ID)
	}))
}

// starPopRow is the line of the menu of a task of the user's own that toggles its important star:
// Mark important, or Not important while it is set (then the star is drawn in its color).
func (m *Main) starPopRow(t *theme.Tokens, task data.Task) popSpec {
	spec := popSpec{key: "important", icon: theme.IconImportantOff, label: m.say.MenuMarkImportant(), hint: keyPriority.hint(), run: func() {
		m.closeMenu()
		m.toggleImportant(task)
	}}
	if starred(task) {
		spec.icon, spec.label, spec.tint = theme.IconImportant, m.say.MenuNotImportant(), t.Important()
	}
	return spec
}

// toggleImportant stars a task of the user's own, or takes the star off.
func (m *Main) toggleImportant(task data.Task) {
	_ = m.app.Update(task.ID, data.Patch{Priority: data.PriorityPtr(importantPriority(!starred(task)))})
}

// openInPage lists where the agent's work can be opened: its app, a terminal, an editor, Finder.
// A choice with nowhere to go (the agent never said where it works) is quiet and does nothing.
func (m *Main) openInPage(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.menuTitle(c, t, m.say.MenuOpenIn())
	icons := map[OpenWith]*ui.SVG{OpenAgentApp: theme.IconAgent, OpenTerminal: theme.IconAgents, OpenEditor: theme.IconArrowOut, OpenFinder: theme.IconFolder}
	for _, how := range openWiths {
		m.popRow(c, t, popSpec{key: "open-" + string(how), icon: icons[how], label: openWithName(m.say, how), disabled: !m.app.canOpen(how, task), run: func() {
			m.closeMenu()
			m.openWork(how, task)
		}})
	}
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
	m.priorityStripOf(c, t, func(p data.Priority) bool { return task.Priority == p }, keyboard, func(p data.Priority) {
		m.closeMenu()
		_ = m.app.Update(task.ID, data.Patch{Priority: data.PriorityPtr(p)})
	})
}

// priorityStripOf is the strip with its own idea of which choice is current and what a click does.
func (m *Main) priorityStripOf(c *ui.Context, t *theme.Tokens, current func(data.Priority) bool, keyboard bool, choose func(data.Priority)) {
	ui.Row(c.Key("pop-priority")).Gap(2).Padding(0, 2, 4).Shrink(0).Children(func() {
		for _, p := range data.Priorities {
			name := priorityName(m.say, p)
			ink := t.Text
			if p == data.P0 {
				ink = t.Blocked()
			}
			m.stripButton(c, t, "priority", "pop-priority-"+string(p), m.say.A11yMenuPriority(name), m.priorityText(p), current(p), keyboard,
				func() { choose(p) },
				func() { theme.Text(c, theme.CompactStrong, ink, string(p)) })
		}
	})
}

// categoryText is a category as the rows read it: its name, or None.
func (m *Main) categoryText(id int) string {
	if c, ok := m.app.Category(id); ok {
		return categoryTitle(m.say, c)
	}
	return m.say.ValueNone()
}

// categoryValue is a category as the inspector's value shows it: its icon in its color, when it
// has one, before its name.
func (m *Main) categoryValue(c *ui.Context, id int, ink ui.Color) {
	cat, ok := m.app.Category(id)
	ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(1).MinWidth(0).Children(func() {
		if ok {
			categoryGlyph(c, cat, theme.IconRow)
		}
		theme.Text(c, theme.Compact, ink, m.categoryText(id)).Shrink(1).MinWidth(0).Ellipsis("…")
	})
}

// categoryPopRow is the line of the whole menu that opens the Category page: the icon of the task's
// category (a list when it has none), its name at the right.
func (m *Main) categoryPopRow(task data.Task) popSpec {
	spec := popSpec{key: "category", icon: theme.IconList, label: m.say.PropCategory(), hint: m.categoryText(task.CategoryID), more: true, run: func() {
		m.pop.page, m.pop.back, m.pop.hl, m.pop.filter = pageCategory, true, 0, ""
	}}
	if cat, ok := m.app.Category(task.CategoryID); ok {
		spec.icon, spec.tint, spec.colored = theme.CategoryIcon(cat.Icon), theme.CategoryColor(cat.Color), true
	}
	return spec
}

// categoryPage is the Move to page for categories: a filter field, the categories with their icons,
// the task's own checked, and None, which unsorts.
func (m *Main) categoryPage(c *ui.Context, t *theme.Tokens, task data.Task) {
	more := popSpec{key: "cat-new", icon: theme.IconPlus, label: m.say.CategoryNew(), run: func() {
		m.closeMenu()
		m.openNewCategory(task.ID)
	}}
	m.movePageOf(c, t, finder{placeholder: m.say.MenuFindCategory(), noMatch: m.say.MenuNoCategoryMatch(), footer: &more},
		func() []popSpec { return m.categoryChoices(task) })
}

// categoryChoices are the categories and None whose name holds what is typed.
func (m *Main) categoryChoices(task data.Task) []popSpec {
	q := strings.ToLower(strings.TrimSpace(m.pop.filter))
	hit := func(name string) bool { return q == "" || strings.Contains(strings.ToLower(name), q) }
	set := func(id int) func() {
		return func() {
			m.closeMenu()
			_ = m.app.Update(task.ID, data.Patch{Category: data.IntPtr(id)})
		}
	}
	var out []popSpec
	for _, cat := range m.app.snap.categories {
		if name := categoryTitle(m.say, cat); hit(name) {
			out = append(out, popSpec{key: "cat-" + itoa(cat.ID), icon: theme.CategoryIcon(cat.Icon), tint: theme.CategoryColor(cat.Color), colored: true, label: name,
				a11y: m.say.A11yMenuCategory(name), checked: task.CategoryID == cat.ID, run: set(cat.ID)})
		}
	}
	if none := m.say.ValueNone(); hit(none) {
		out = append(out, popSpec{key: "cat-none", label: none, a11y: m.say.A11yMenuCategory(none), checked: task.CategoryID == 0, run: set(0)})
	}
	return out
}
