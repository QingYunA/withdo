package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// catUnsorted is the nav.cat of the Unsorted page of Mine: the tasks with no category.
const catUnsorted = -1

// mineSlices are the parts of Mine the sidebar lists, each built from its own tasks so its counts
// and its "Show all" link are right: one per category, by id, and the unsorted ones.
type mineSlices struct {
	byCat    map[int]data.Mine
	unsorted data.Mine
}

// of is the slice a nav.cat names; a category that is not there gives the zero Mine.
func (s mineSlices) of(cat int) data.Mine {
	if cat == catUnsorted {
		return s.unsorted
	}
	return s.byCat[cat]
}

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

// inMineSlice accepts the tasks that count under the category cat, or under none for catUnsorted.
func (a *App) inMineSlice(cat int) func(data.Task) bool {
	want := max(cat, 0)
	return func(t data.Task) bool { return a.categoryOf(t) == want }
}

// inAgentSlice accepts the tasks of the agent named name, the way the sidebar and the page title
// spell it.
func (a *App) inAgentSlice(name string) func(data.Task) bool {
	return func(t data.Task) bool { return agentName(a.taskAgent(t)) == name }
}

// buildPages builds what the frame shows: all of Mine and Agents, the slices the sidebar lists, and
// the page that is open, which is one of those or a slice of its own.
func (m *Main) buildPages() {
	a := m.app
	m.mine, m.agents = a.buildMine(), a.buildAgents()
	m.slices = mineSlices{byCat: make(map[int]data.Mine, len(a.snap.categories)), unsorted: a.buildMineOf(a.inMineSlice(catUnsorted))}
	for _, c := range a.snap.categories {
		m.slices.byCat[c.ID] = a.buildMineOf(a.inMineSlice(c.ID))
	}
	m.agentRows = m.agents.Tally(func(t data.Task) string { return agentName(a.taskAgent(t)) })
	m.minePage, m.agentsPage = m.mine, m.agents
	switch {
	case m.nav.screen == screenMine && m.nav.cat != 0:
		m.minePage = m.slices.of(m.nav.cat)
	case m.nav.screen == screenAgents && m.nav.sub != "":
		m.agentsPage = a.buildAgentsOf(a.inAgentSlice(m.nav.sub))
	}
}

// mineTitle is the title of the Mine page: Mine, or the name of its category.
func (m *Main) mineTitle() string {
	if m.nav.cat == catUnsorted {
		return m.say.CategoryUnsorted()
	}
	if c, ok := m.app.Category(m.nav.cat); ok {
		return categoryTitle(m.say, c)
	}
	return m.say.NavMine()
}

// agentsTitle is the title of the Agents page: Agents, or the agent's name.
func (m *Main) agentsTitle() string {
	if m.nav.sub != "" {
		return m.nav.sub
	}
	return m.say.NavAgents()
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

// mineSubRows are the rows under Mine: every category by position, empty ones too, then Unsorted
// while it has rows (or is the open page, so its highlight does not vanish under the pointer).
func (m *Main) mineSubRows(c *ui.Context, t *theme.Tokens) {
	for _, cat := range m.app.snap.categories {
		m.subRow(c, t, "cat-"+itoa(cat.ID), nav{screen: screenMine, cat: cat.ID}, categoryTitle(m.say, cat), m.slices.byCat[cat.ID].Count(),
			func() { categoryGlyph(c, cat, theme.AvatarSmall) }, func(mn *ui.Menu) { m.categoryMenu(c, mn, cat) })
	}
	to := nav{screen: screenMine, cat: catUnsorted}
	if !m.slices.unsorted.Empty() || m.nav == to {
		m.subRow(c, t, "mine-unsorted", to, m.say.CategoryUnsorted(), m.slices.unsorted.Count(), nil, m.mineMenu)
	}
}

// agentSubRows are the rows under Agents: one per agent that has a task at work or finished one
// today, in the order of the Agents screen.
func (m *Main) agentSubRows(c *ui.Context, t *theme.Tokens) {
	rows, open := m.agentRows, m.nav.sub
	if m.nav.screen == screenAgents && open != "" && !hasAgentRow(rows, open) {
		rows = append(append([]data.AgentTally(nil), rows...), data.AgentTally{Name: open})
	}
	for _, r := range rows {
		who := theme.AgentAvatar(strings.ToLower(r.Name))
		m.subRow(c, t, "agent:"+r.Name, nav{screen: screenAgents, sub: r.Name}, r.Name, r.Working, func() {
			theme.AvatarIcon(c.Key("avatar"), who, theme.AvatarSmall)
		}, nil)
	}
}

func hasAgentRow(rows []data.AgentTally, name string) bool {
	for _, r := range rows {
		if r.Name == name {
			return true
		}
	}
	return false
}

// categoryGlyph draws a category's icon, size pt square, in its color.
func categoryGlyph(c *ui.Context, cat data.Category, size float32) ui.Element {
	return ui.Image(c, theme.CategoryIcon(cat.Icon)).Size(size, size).TextColor(theme.CategoryColor(cat.Color)).Shrink(0)
}

// subRowIndent is how far a sub-row's content sits in from its parent's: one 18 pt icon and the row's
// gap, so its slot starts where the parent's label does.
const subRowIndent = theme.IconRow + sidebarRowGap

// subRow is a sidebar row under Mine or Agents, one step in from its parent. Its 12 pt slot, flush
// with the parent's label, holds what lead draws (the category's icon, the agent's avatar) or stays
// empty when lead is nil, so the sub-rows' labels line up. menu is the row's context menu, if it has one.
func (m *Main) subRow(c *ui.Context, t *theme.Tokens, key any, to nav, label string, count int, lead func(), menu func(*ui.Menu)) {
	selected := m.nav == to && !m.searching()
	row := sidebarRow(c, t, key, selected).Padding(0, sidebarRowPad, 0, sidebarRowPad+subRowIndent)
	if menu != nil {
		row.ContextMenu(menu)
	}
	row.Children(func() {
		ui.Row(c.Key("lead")).Width(theme.AvatarSmall).Height(theme.AvatarSmall).Shrink(0).Children(func() {
			if lead != nil {
				lead()
			}
		})
		sidebarLabel(c, t, label, selected)
		sidebarCount(c, t, count)
	})
	if row.Clicked() {
		m.goTo(to)
	}
}
