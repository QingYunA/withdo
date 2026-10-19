package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// completedPage is how many finished tasks Completed loads at a time.
const completedPage = 100

// completedState is Completed: how much of it is loaded, and what was loaded for which snapshot.
type completedState struct {
	limit int // 0 until the screen is first drawn
	rev   int
	for_  int // the limit the tasks were loaded for
	tasks []data.Task
	more  bool // older tasks exist than are loaded
}

// completedTasks is the finished tasks, newest first, as many as are loaded. They are read again
// when the data changes or more is asked for.
func (m *Main) completedTasks() []data.Task {
	l := &m.manage.completed
	if l.limit == 0 {
		l.limit = completedPage
	}
	if l.rev == m.app.rev && l.for_ == l.limit {
		return l.tasks
	}
	l.rev, l.for_ = m.app.rev, l.limit
	tasks, err := m.app.Store.Logbook(l.limit + 1)
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

// completedScreen is the finished tasks, newest first, a header for each day.
func (m *Main) completedScreen(c *ui.Context, t *theme.Tokens) {
	m.pageHeader(c, t, m.say.ManageNavCompleted(), m.say.ManageCompletedSubtitle(), nil)
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
