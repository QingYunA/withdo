package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// completedPage is how many finished tasks Completed loads at a time.
const completedPage = 100

// completedState is the history: how much of it is loaded, and what was loaded for which snapshot
// and scope.
type completedState struct {
	limit int // 0 until the screen is first drawn
	rev   int
	for_  int // the limit the tasks were loaded for
	scope nav // the scope the tasks were loaded for
	tasks []data.Task
	more  bool // older tasks exist than are loaded
}

// openHistory opens the full history of what was finished: of everything (project 0), of a project,
// or of what agents wrote. It is reached from a list's "Show all" link only; its back control returns
// to the page the link was on.
func (m *Main) openHistory(project int, agents bool) {
	m.histBack = m.nav
	m.manage.completed.limit = 0
	m.goTo(nav{screen: screenCompleted, projectID: project, agents: agents})
}

// completedTasks is the finished tasks, newest first, as many as are loaded. They are read again
// when the data changes, the scope changes or more is asked for.
func (m *Main) completedTasks() []data.Task {
	l := &m.manage.completed
	if l.limit == 0 {
		l.limit = completedPage
	}
	if l.rev == m.app.rev && l.for_ == l.limit && l.scope == m.nav {
		return l.tasks
	}
	l.rev, l.for_, l.scope = m.app.rev, l.limit, m.nav
	tasks, err := m.app.Store.LogbookOf(m.nav.projectID, m.nav.agents, l.limit+1)
	if err != nil {
		m.app.snap.err = err
		return l.tasks
	}
	l.more = len(tasks) > l.limit
	if l.more {
		tasks = tasks[:l.limit]
	}
	l.tasks = topLevel(tasks) // a finished subtask is part of its parent's story
	return l.tasks
}

// completedScreen is the history: the finished tasks, newest first, a header for each day, under a
// back control to the list it was opened from.
func (m *Main) completedScreen(c *ui.Context, t *theme.Tokens) {
	m.backControl(c, t)
	m.pageHeader(c, t, m.say.ManageNavCompleted(), m.historyScope(), nil)
	if !m.app.snap.loaded {
		return
	}
	days, now := m.pg.Sections, m.app.Now()
	if len(days) == 0 {
		emptyState(c, t, m.say.ManageCompletedEmptyTitle(), m.say.ManageCompletedEmptyBody(), 56, nil)
		return
	}
	for _, d := range days {
		label := m.clock.Day(d.Day, now)
		ui.Row(c.Key("completed-day-"+label)).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, label)
			theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(d.Tasks)))
		})
		for _, task := range d.Tasks {
			m.completedRow(c, t, task)
		}
	}
	if m.manage.completed.more && link(c, t, m.say.ManageCompletedOlder()).Margin(16, 0, 0).Clicked() {
		m.manage.completed.limit += completedPage
	}
}

// completedRow is a finished task: its glyph (a way to reopen it), its title and project, and the
// time of day it was finished.
func (m *Main) completedRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	m.finishedRow(c, t, task, finishedOpts{glyph: true, project: m.app.projectName(task.ProjectID), when: m.clock.Time(doneAt(task))})
}

// historyScope is the quiet line beside the history's title: what it covers.
func (m *Main) historyScope() string {
	switch {
	case m.nav.projectID != 0 && m.nav.agents:
		return m.say.HistoryOf(m.app.projectName(&m.nav.projectID) + " · " + m.say.NavAgents())
	case m.nav.projectID != 0:
		return m.say.HistoryOf(m.app.projectName(&m.nav.projectID))
	case m.nav.agents:
		return m.say.HistoryOf(m.say.NavAgents())
	}
	return m.say.ManageCompletedSubtitle()
}

// backControl is the line above the history's title: a chevron and the name of the page the link
// came from. A history opened in some other way falls back to Today.
func (m *Main) backControl(c *ui.Context, t *theme.Tokens) {
	to := m.histBack
	if to.screen == screenCompleted {
		to = nav{}
	}
	name := m.navName(to)
	btn := ui.ButtonBase(c.Key("history-back")).Shrink(0).AlignSelf(ui.Start).Padding(2, 6, 2, 0).Radius(theme.RadiusControl).
		Label(m.say.A11yBackTo(name)).Cursor(ui.CursorPointer)
	ink := t.Accent
	if btn.Hovered() {
		ink = t.Accent.Mix(t.Text, 0.2)
	}
	btn.Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, theme.IconBack).Size(theme.IconSmall, theme.IconSmall).TextColor(ink).Shrink(0)
			theme.Text(c, theme.Meta, ink, name)
		})
	})
	if btn.Clicked() {
		m.goTo(to)
	}
}

// navName is the name of the page n shows, as the sidebar and the title say it.
func (m *Main) navName(n nav) string {
	switch n.screen {
	case screenAgents:
		return m.say.NavAgents()
	case screenCalendar:
		return m.say.NavCalendar()
	case screenProject:
		return m.app.projectName(&n.projectID)
	}
	return m.say.NavMine()
}
