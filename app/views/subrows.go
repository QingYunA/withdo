package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The slices of Mine, named in nav.sub. An agent's slice of Agents is named by the agent's name.
const (
	subWork     = "work"
	subLife     = "life"
	subUnsorted = "unsorted"
)

// mineSlices are the parts of Mine the sidebar lists, each built from its own tasks so its counts
// and its "Show all" link are right.
type mineSlices struct{ work, life, unsorted data.Mine }

// of is the slice a nav.sub names; a sub that is none of them gives the zero Mine.
func (s mineSlices) of(sub string) data.Mine {
	switch sub {
	case subWork:
		return s.work
	case subLife:
		return s.life
	case subUnsorted:
		return s.unsorted
	}
	return data.Mine{}
}

// projectHasRepo reports whether a task's project is a repository, which makes its tasks work.
func (a *App) projectHasRepo(id *int) bool {
	if id == nil {
		return false
	}
	p, ok := a.Project(*id)
	return ok && p.HasRepo()
}

// categoryOf is the category a task counts under: its own, else work in a repository's project.
func (a *App) categoryOf(t data.Task) data.Category { return data.CategoryOf(t, a.projectHasRepo) }

// inMineSlice accepts the tasks that belong to a slice of Mine.
func (a *App) inMineSlice(sub string) func(data.Task) bool {
	want := map[string]data.Category{subWork: data.CategoryWork, subLife: data.CategoryLife}[sub]
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
	m.slices = mineSlices{
		work:     a.buildMineOf(a.inMineSlice(subWork)),
		life:     a.buildMineOf(a.inMineSlice(subLife)),
		unsorted: a.buildMineOf(a.inMineSlice(subUnsorted)),
	}
	m.agentRows = m.agents.Tally(func(t data.Task) string { return agentName(a.taskAgent(t)) })
	m.minePage, m.agentsPage = m.mine, m.agents
	switch {
	case m.nav.sub == "":
	case m.nav.screen == screenMine:
		m.minePage = m.slices.of(m.nav.sub)
	case m.nav.screen == screenAgents:
		m.agentsPage = a.buildAgentsOf(a.inAgentSlice(m.nav.sub))
	}
}

// mineTitle is the title of the Mine page: Mine, or the name of its slice.
func (m *Main) mineTitle() string {
	switch m.nav.sub {
	case subWork:
		return m.say.CategoryWork()
	case subLife:
		return m.say.CategoryLife()
	case subUnsorted:
		return m.say.CategoryUnsorted()
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

// categoryName is a category as people read it; "" is Unsorted.
func categoryName(say locales.Messages, c data.Category) string {
	switch c {
	case data.CategoryWork:
		return say.CategoryWork()
	case data.CategoryLife:
		return say.CategoryLife()
	}
	return say.CategoryUnsorted()
}

// mineSubRows are the rows under Mine: Work and Life always, Unsorted while it has rows (or is the
// open page, so its highlight does not vanish under the pointer).
func (m *Main) mineSubRows(c *ui.Context, t *theme.Tokens) {
	m.subRow(c, t, "mine-work", nav{screen: screenMine, sub: subWork}, m.say.CategoryWork(), m.slices.work.Count(), nil)
	m.subRow(c, t, "mine-life", nav{screen: screenMine, sub: subLife}, m.say.CategoryLife(), m.slices.life.Count(), nil)
	to := nav{screen: screenMine, sub: subUnsorted}
	if !m.slices.unsorted.Empty() || m.nav == to {
		m.subRow(c, t, "mine-unsorted", to, m.say.CategoryUnsorted(), m.slices.unsorted.Count(), nil)
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
		m.subRow(c, t, "agent:"+r.Name, nav{screen: screenAgents, sub: r.Name}, r.Name, r.Working, &who)
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

// subRow is a sidebar row under Mine or Agents. It has no icon of its own: the room of its parent's
// 18 pt icon is left empty, or holds the agent's 12 pt avatar, so the label lines up with the parent's.
func (m *Main) subRow(c *ui.Context, t *theme.Tokens, key any, to nav, label string, count int, avatar *theme.Who) {
	selected := m.nav == to && !m.searching()
	row := sidebarRow(c, t, key, selected)
	row.Children(func() {
		ui.Row(c.Key("lead")).Width(theme.IconRow).Height(theme.IconRow).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			if avatar != nil {
				theme.AvatarIcon(c.Key("avatar"), *avatar, theme.AvatarSmall)
			}
		})
		sidebarLabel(c, t, label, selected)
		sidebarCount(c, t, count)
	})
	if row.Clicked() {
		m.goTo(to)
	}
}
