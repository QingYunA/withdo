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

// completedDay is the tasks finished on one calendar day.
type completedDay struct {
	label string
	tasks []data.Task
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
	l.tasks = tasks
	return tasks
}

// completedDays groups the tasks by the day they were finished, in the order they come.
func (m *Main) completedDays() []completedDay {
	now := m.app.Now()
	var days []completedDay
	last := ""
	for _, t := range m.completedTasks() {
		at := doneAt(t).In(now.Location())
		if key := at.Format("2006-01-02"); key != last {
			days, last = append(days, completedDay{label: m.clock.Day(at, now)}), key
		}
		days[len(days)-1].tasks = append(days[len(days)-1].tasks, t)
	}
	return days
}

// completedIDs is the task ids of the rows, for the arrow keys.
func (m *Main) completedIDs() []int {
	var ids []int
	for _, t := range m.completedTasks() {
		ids = append(ids, t.ID)
	}
	return ids
}

// completedScreen is the finished tasks, newest first, a header for each day.
func (m *Main) completedScreen(c *ui.Context, t *theme.Tokens) {
	m.pageHeader(c, t, m.say.ManageNavCompleted(), m.say.ManageCompletedSubtitle(), nil)
	if !m.app.snap.loaded {
		return
	}
	days := m.completedDays()
	if len(days) == 0 {
		ui.Column(c).Padding(32, 0, 0).Gap(4).Shrink(0).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, m.say.ManageCompletedEmptyTitle())
			theme.Paragraph(c, theme.Body, t.TextSecondary, m.say.ManageCompletedEmptyBody(), 0)
		})
		return
	}
	for _, d := range days {
		ui.Row(c.Key("completed-day-"+d.label)).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, d.label)
			theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(d.tasks)))
		})
		for _, task := range d.tasks {
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
