package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// logbookPage is how many finished tasks the Logbook loads at a time.
const logbookPage = 100

// logState is the Logbook: how much of it is loaded, and what was loaded for which snapshot.
type logState struct {
	limit int // 0 until the screen is first drawn
	rev   int
	for_  int // the limit the tasks were loaded for
	tasks []data.Task
	more  bool // older tasks exist than are loaded
}

// logDay is the tasks finished on one calendar day.
type logDay struct {
	label string
	tasks []data.Task
}

// logbookTasks is the finished tasks, newest first, as many as are loaded. They are read again
// when the data changes or more is asked for.
func (m *Main) logbookTasks() []data.Task {
	l := &m.manage.log
	if l.limit == 0 {
		l.limit = logbookPage
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

// logbookDays groups the tasks by the day they were finished, in the order they come.
func (m *Main) logbookDays() []logDay {
	now := m.app.Now()
	var days []logDay
	last := ""
	for _, t := range m.logbookTasks() {
		at := doneAt(t).In(now.Location())
		if key := at.Format("2006-01-02"); key != last {
			days, last = append(days, logDay{label: m.clock.Day(at, now)}), key
		}
		days[len(days)-1].tasks = append(days[len(days)-1].tasks, t)
	}
	return days
}

// logbookIDs is the task ids of the rows, for the arrow keys.
func (m *Main) logbookIDs() []int {
	var ids []int
	for _, t := range m.logbookTasks() {
		ids = append(ids, t.ID)
	}
	return ids
}

// logbookScreen is the finished tasks, newest first, a header for each day.
func (m *Main) logbookScreen(c *ui.Context, t *theme.Tokens) {
	m.pageHeader(c, t, m.say.ManageNavLogbook(), m.say.ManageLogbookSubtitle(), nil)
	if !m.app.snap.loaded {
		return
	}
	days := m.logbookDays()
	if len(days) == 0 {
		ui.Column(c).Padding(32, 0, 0).Gap(4).Shrink(0).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, m.say.ManageLogbookEmptyTitle())
			theme.Paragraph(c, theme.Body, t.TextSecondary, m.say.ManageLogbookEmptyBody(), 0)
		})
		return
	}
	for _, d := range days {
		ui.Row(c.Key("log-day-"+d.label)).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, d.label)
			theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(d.tasks)))
		})
		for _, task := range d.tasks {
			m.logbookRow(c, t, task)
		}
	}
	if m.manage.log.more && link(c, t, m.say.ManageLogbookOlder()).Margin(16, 0, 0).Clicked() {
		m.manage.log.limit += logbookPage
	}
}

// logbookRow is a finished task: its glyph (a way to reopen it), its title and project, and the
// time of day it was finished.
func (m *Main) logbookRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	fill, opacity, _ := m.app.completionPhase(c, task.ID)
	row := m.rowShell(c, t, task.ID, plainRow, opacity).AlignItems(ui.Center)
	row.Children(func() {
		m.app.glyphSlot(c, t, task, fill)
		body(c, func() { m.titleLine(c, t, task, m.app.projectName(task.ProjectID), true) })
		right(c, func() { plainText(c, t, m.clock.Time(doneAt(task))) })
	})
	if row.Clicked() {
		m.openRow(task.ID)
	}
}
