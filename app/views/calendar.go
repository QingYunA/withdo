package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/words"
	"github.com/egoist/mygo/ui"
)

// calendarState is what the Calendar page shows: the month, and the day the composer adds to. Both
// are set the first time the page is drawn (to today) and kept while I am on other pages.
type calendarState struct {
	month time.Time // the first of the month, local midnight
	day   time.Time // the selected day, local midnight
	card  dayCard   // the card of one day, over the grid
}

// calDrag is the value an item of the calendar carries while it is dragged to another day.
type calDrag struct {
	id   int
	kind data.CalKind
}

// ensureCalendar sets the month and the day to today's the first time they are asked for.
func (m *Main) ensureCalendar() {
	if !m.cal.day.IsZero() {
		return
	}
	m.calendarToday()
}

// calendarDay is the selected day.
func (m *Main) calendarDay() time.Time {
	m.ensureCalendar()
	return m.cal.day
}

// calendarMonth is the first day of the month shown.
func (m *Main) calendarMonth() time.Time {
	m.ensureCalendar()
	return m.cal.month
}

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// calendarToday returns to the current month and selects today.
func (m *Main) calendarToday() {
	now := m.app.Now()
	m.cal.day, m.cal.month = data.Midnight(now), firstOfMonth(now)
	m.cal.card.close()
}

// selectDay selects a day. The month stays when the day is on the grid it shows (a click on a
// grey day of the next month does not turn the page); a day beyond it turns the page.
func (m *Main) selectDay(d time.Time) {
	m.ensureCalendar()
	d = data.Midnight(d.In(m.app.Now().Location()))
	m.cal.day = d
	days := m.calendarDays()
	if d.Before(days[0].Date) || !d.Before(days[len(days)-1].Date.AddDate(0, 0, 1)) {
		m.cal.month = firstOfMonth(d)
	}
}

// shiftMonth shows the month n months from the one shown, and moves the selection along to the
// same day of that month (the last day of a shorter one).
func (m *Main) shiftMonth(n int) {
	m.ensureCalendar()
	month := m.cal.month.AddDate(0, n, 0)
	last := month.AddDate(0, 1, -1).Day()
	m.cal.month = month
	m.cal.card.close()
	m.cal.day = time.Date(month.Year(), month.Month(), min(m.cal.day.Day(), last), 0, 0, 0, 0, month.Location())
}

// stepDay moves the selection by days (a week is 7), turning the page when it leaves the grid.
func (m *Main) stepDay(days int) { m.selectDay(m.calendarDay().AddDate(0, 0, days)) }

// calendarDays is the month grid with what is on each day.
func (m *Main) calendarDays() []data.CalDay {
	var ms []data.Milestone
	for _, list := range m.app.snap.milestones {
		ms = append(ms, list...)
	}
	return data.BuildCalendar(data.CalendarInput{
		Month: m.calendarMonth(), Now: m.app.Now(), WeekStart: m.weekStart,
		Tasks: m.app.snap.tasks, Milestones: ms, Agent: m.calendarAgent,
	})
}

// calendarAgent is the agent a task's latest report came from.
func (m *Main) calendarAgent(id int) string {
	if r := m.app.latestReport(id); r != nil {
		return r.Agent
	}
	return ""
}

// firstWeekday is the day the calendar's weeks start on, for the language and region of a frame.
func firstWeekday(c *ui.Context) time.Weekday { return words.FirstWeekday(c.Locale(), c.Region()) }

// calAvatar is the avatar before a task's title: me for what I wrote (the assistant writing it
// for me counts as me), the agent's own for what an agent reported.
func (m *Main) calAvatar(it data.CalItem) theme.Who {
	if it.Task.Source == data.SourceAgent {
		return theme.AgentAvatar(it.Agent)
	}
	return theme.You
}

// calInk is the color of an item's title: red for an open task past its due time, quiet for a
// finished one.
func (m *Main) calInk(t *theme.Tokens, it data.CalItem) ui.Color {
	switch {
	case it.Kind == data.CalDone:
		return t.TextTertiary
	case it.Overdue:
		return t.Blocked()
	}
	return t.Text
}

// visibleItems says how many of n items a cell with that many lines lists, and how many it leaves
// to "N more": at most CalendarMaxItems, and when some are left the "more" line takes one line.
func visibleItems(n, lines int) (shown, more int) {
	limit := min(theme.CalendarMaxItems, lines)
	if n <= limit {
		return n, 0
	}
	shown = max(0, min(theme.CalendarMaxItems, lines-1))
	return shown, n - shown
}

// moveCalItem is a drop of an item on a day: its due date, or for a reminder item its reminder,
// goes to that day and keeps its time. A finished task stays where it was finished.
func (m *Main) moveCalItem(d calDrag, day time.Time) {
	task, ok := m.app.Task(d.id)
	if !ok || d.kind == data.CalDone || task.Status.Terminal() {
		return
	}
	switch {
	case d.kind == data.CalReminder && task.RemindAt != nil:
		if at := data.OnDay(*task.RemindAt, day); !at.Equal(*task.RemindAt) {
			_ = m.app.Update(task.ID, data.Patch{RemindAt: &at})
		}
	case d.kind == data.CalDue && task.DueAt != nil:
		if at := data.OnDay(*task.DueAt, day); !at.Equal(*task.DueAt) {
			_ = m.app.Update(task.ID, data.Patch{DueAt: &at})
		}
	}
}

// calendarKey is the keyboard of the Calendar page when no text input has it: the arrows walk the
// days (a week up and down), and Cmd+arrows turn the month.
func (m *Main) calendarKey(c *ui.Context) bool {
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyLeft):
		m.shiftMonth(-1)
	case c.Shortcut(ui.Cmd, ui.KeyRight):
		m.shiftMonth(1)
	case c.Shortcut(0, ui.KeyLeft):
		m.stepDay(-1)
	case c.Shortcut(0, ui.KeyRight):
		m.stepDay(1)
	case c.Shortcut(0, ui.KeyUp):
		m.stepDay(-7)
	case c.Shortcut(0, ui.KeyDown):
		m.stepDay(7)
	default:
		return false
	}
	return true
}

// calendarPlace is where the composer's requests on the Calendar page come from.
func (m *Main) calendarPlace() data.Place {
	return data.Place{Kind: data.PlaceCalendar, Day: m.calendarDay().Format("2006-01-02")}
}
