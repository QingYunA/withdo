package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// projectHasRepo reports whether a task's project is a repository, which makes its tasks work.
func (a *App) projectHasRepo(id *int) bool {
	if id == nil {
		return false
	}
	p, ok := a.Project(*id)
	return ok && p.HasRepo()
}

// categoryOf is the id of the category a task counts under: its own, else Work in a repository's
// project, else 0 (Unsorted).
func (a *App) categoryOf(t data.Task) int { return data.CategoryOf(t, a.projectHasRepo, a.workID()) }

// inCategory accepts the tasks that count under the category cat.
func (a *App) inCategory(cat int) func(data.Task) bool {
	return func(t data.Task) bool { return a.categoryOf(t) == cat }
}

// buildPages builds what the frame shows: all of Today and Agents, and the page that is open, which
// is Today narrowed to the category a chip picked.
func (m *Main) buildPages() {
	a := m.app
	m.mine, m.agents = a.buildMine(), groupByAgent(a.buildAgents(), func(t data.Task) string { return agentName(a.taskAgent(t)) })
	m.minePage, m.agentsPage = m.mine, m.agents
	if m.nav.screen == screenMine && m.catFilter != 0 {
		m.minePage = a.buildMineOf(a.inCategory(m.catFilter))
	}
}

// groupByAgent puts the rows at work of one agent together, the agents in the order their newest
// row comes, so the Agents page can head each group with the agent. The rows keep their order inside
// a group, and the finished ones stay as they are.
func groupByAgent(s data.AgentsScreen, name func(data.Task) string) data.AgentsScreen {
	var order []string
	by := map[string][]data.AgentRow{}
	for _, r := range s.Working {
		n := name(r.Task)
		if _, ok := by[n]; !ok {
			order = append(order, n)
		}
		by[n] = append(by[n], r)
	}
	grouped := make([]data.AgentRow, 0, len(s.Working))
	for _, n := range order {
		grouped = append(grouped, by[n]...)
	}
	s.Working = grouped
	return s
}

// categoryLabel is a category as people read it, from the label a receipt carries: the key of a
// built-in category (work, life) in the user's language, the name of any other as it is.
func categoryLabel(say locales.Messages, label string) string {
	switch label {
	case data.BuiltinWork:
		return say.CategoryWork()
	case data.BuiltinLife:
		return say.CategoryLife()
	}
	return label
}

// categoryTitle is a category as people read it: Work and Life in the user's language, the rest by
// name.
func categoryTitle(say locales.Messages, c data.Category) string {
	if c.Builtin != "" {
		return categoryLabel(say, c.Builtin)
	}
	return c.Name
}

// categoryGlyph draws a category's icon, size pt square, in its color.
func categoryGlyph(c *ui.Context, cat data.Category, size float32) ui.Element {
	return ui.Image(c, theme.CategoryIcon(cat.Icon)).Size(size, size).TextColor(theme.CategoryColor(cat.Color)).Shrink(0)
}

// subRowIndent is how far a sub-project's content sits in from its parent's: one 18 pt icon and the
// row's gap, so its folder starts where the parent's label does.
const subRowIndent = theme.IconRow + sidebarRowGap

// rowCategory is the category a row of mine shows as a chip: the one the task counts under. On a
// project's page a task of a repository counts under Work without saying so, since every row there
// would repeat it: only a category the user (or the assistant) gave shows.
func (m *Main) rowCategory(task data.Task) (data.Category, bool) {
	if task.Source != data.SourceUser || m.width < dropPillsBelow {
		return data.Category{}, false
	}
	id := m.app.categoryOf(task)
	if id == 0 || (task.CategoryID == 0 && m.nav.screen != screenMine) {
		return data.Category{}, false
	}
	return m.app.Category(id)
}

// rowChip is the small colored chip of a row's category: its swatch and name in meta, on
// bg.subtle. Clicking it narrows the list to that category.
func (m *Main) rowChip(c *ui.Context, t *theme.Tokens, cat data.Category) {
	name := categoryTitle(m.say, cat)
	label := m.say.A11yFilterCategory(name)
	btn := ui.ButtonBase(c.Key("chip")).Shrink(0).Padding(2, 7).Radius(theme.RadiusControl).Background(t.BgSubtle).
		Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			categoryGlyph(c, cat, theme.IconSmall)
			theme.Text(c, theme.Meta, t.TextSecondary, name).Shrink(0)
		})
	})
	if btn.Clicked() {
		m.catFilter = cat.ID
		m.sel = 0
	}
}

// filterBar is the line above the lists while a chip's category narrows them: "Showing Work" and an
// x. Esc clears it as well.
func (m *Main) filterBar(c *ui.Context, t *theme.Tokens) {
	if m.catFilter == 0 {
		return
	}
	cat, ok := m.app.Category(m.catFilter)
	if !ok {
		return
	}
	ui.Row(c.Key("filter-bar")).Shrink(0).AlignSelf(ui.Start).Margin(0, 0, 8).Padding(3, 4, 3, 9).Gap(6).Radius(theme.RadiusControl).
		Background(t.BgSelected).AlignItems(ui.Center).Children(func() {
		categoryGlyph(c, cat, theme.IconSmall)
		theme.Text(c, theme.Meta, t.Text, m.say.FilterShowing(categoryTitle(m.say, cat))).Shrink(0)
		clear := m.say.FilterClear()
		x := ui.ButtonBase(c.Key("filter-clear")).Size(20, 20).Shrink(0).Center().Radius(theme.RadiusControl).Label(clear).Tooltip(clear).
			Cursor(ui.CursorPointer)
		ink := t.TextTertiary
		if x.Hovered() {
			x.Background(t.BgSelected)
			ink = t.TextSecondary
		}
		x.Children(func() { ui.Icon(c, theme.IconClose).Size(theme.IconSmall, theme.IconSmall).TextColor(ink) })
		if x.Clicked() {
			m.catFilter = 0
		}
	})
}
