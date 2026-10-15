package data

import (
	"sort"
	"time"
)

// CalKind says why an item is on a day of the calendar. Within a day milestones come first, then
// what is still to do by its time, then what was finished.
type CalKind int

const (
	CalMilestone CalKind = iota // a milestone is due
	CalReminder                 // a reminder rings, shown with its time
	CalDue                      // a task is due
	CalDone                     // a task was finished
)

// CalItem is one thing on a day of the calendar: a task (due, reminded or finished) or a milestone.
type CalItem struct {
	Kind      CalKind
	Task      Task
	Milestone Milestone
	At        time.Time // the moment that put it on the day
	// Overdue is an open task whose due time has passed, however its item is shown.
	Overdue bool
	// Agent is the name of the agent that wrote a task, by its latest report ("agent" when it
	// has none); empty for a task of mine and for a milestone.
	Agent string
}

// CalDay is one cell of the month grid.
type CalDay struct {
	Date    time.Time // local midnight
	InMonth bool
	Weekend bool
	Today   bool
	Items   []CalItem
}

// CalendarInput is what BuildCalendar reads.
type CalendarInput struct {
	Month      time.Time // any moment of the month to show
	Now        time.Time
	WeekStart  time.Weekday
	Tasks      []Task
	Milestones []Milestone
	// Agent returns the agent named by a task's latest report, or "".
	Agent func(taskID int) string
}

// DefaultDueHour is when a bare day is a deadline: end of the working day.
const DefaultDueHour = 18

// MonthGrid returns the first day of the grid that shows month (the week start on or before the
// 1st) and how many weeks it needs.
func MonthGrid(month time.Time, weekStart time.Weekday) (first time.Time, weeks int) {
	loc := month.Location()
	y, m, _ := month.Date()
	start := time.Date(y, m, 1, 0, 0, 0, 0, loc)
	lead := (int(start.Weekday()) - int(weekStart) + 7) % 7
	first = start.AddDate(0, 0, -lead)
	days := lead + start.AddDate(0, 1, -1).Day()
	return first, (days + 6) / 7
}

// Midnight is the start of t's calendar day, in t's zone.
func Midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// OnDay moves a moment to another calendar day, keeping the time of day it has in day's zone.
func OnDay(t, day time.Time) time.Time {
	t = t.In(day.Location())
	y, m, d := day.Date()
	return time.Date(y, m, d, t.Hour(), t.Minute(), t.Second(), 0, day.Location())
}

// EndOfDay is when a task with no time words is due on a day: 18:00 local.
func EndOfDay(day time.Time) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, DefaultDueHour, 0, 0, 0, day.Location())
}

// BuildCalendar lays the tasks and milestones on the days of the month grid. Cancelled tasks are
// left out; an open task appears on the day it is due and on the day it is reminded (once, as the
// reminder, when they are the same day); a finished task appears on the day it was finished.
func BuildCalendar(in CalendarInput) []CalDay {
	first, weeks := MonthGrid(in.Month, in.WeekStart)
	loc := in.Now.Location()
	first = time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)
	days := make([]CalDay, weeks*7)
	index := map[int]int{} // day number to position in days
	for i := range days {
		d := first.AddDate(0, 0, i)
		wd := d.Weekday()
		days[i] = CalDay{
			Date: d, InMonth: d.Month() == in.Month.Month() && d.Year() == in.Month.Year(),
			Weekend: wd == time.Saturday || wd == time.Sunday, Today: DaysFrom(d, in.Now) == 0,
		}
		index[dayIndex(d)] = i
	}
	put := func(t time.Time, it CalItem) {
		if i, ok := index[dayIndex(t.In(loc))]; ok {
			it.At = t
			days[i].Items = append(days[i].Items, it)
		}
	}
	agent := func(t Task) string {
		if t.Source != SourceAgent {
			return ""
		}
		if in.Agent != nil {
			if name := in.Agent(t.ID); name != "" {
				return name
			}
		}
		return "agent"
	}
	for _, t := range in.Tasks {
		base := CalItem{Task: t, Agent: agent(t)}
		switch {
		case t.Status == Cancelled:
		case t.Status == Done:
			base.Kind = CalDone
			put(doneMoment(t), base)
		default:
			base.Overdue = t.DueAt != nil && t.DueAt.Before(in.Now)
			if t.RemindAt != nil {
				it := base
				it.Kind = CalReminder
				put(*t.RemindAt, it)
			}
			if t.DueAt != nil && !(t.RemindAt != nil && DaysFrom(*t.DueAt, *t.RemindAt) == 0) {
				it := base
				it.Kind = CalDue
				put(*t.DueAt, it)
			}
		}
	}
	for _, m := range in.Milestones {
		if m.Open && m.DueAt != nil {
			put(*m.DueAt, CalItem{Kind: CalMilestone, Milestone: m})
		}
	}
	for i := range days {
		items := days[i].Items
		sort.SliceStable(items, func(a, b int) bool {
			if ra, rb := rank(items[a].Kind), rank(items[b].Kind); ra != rb {
				return ra < rb
			}
			if !items[a].At.Equal(items[b].At) {
				return items[a].At.Before(items[b].At)
			}
			if items[a].Kind != items[b].Kind {
				return items[a].Kind < items[b].Kind
			}
			return items[a].Task.ID+items[a].Milestone.ID < items[b].Task.ID+items[b].Milestone.ID // oldest first
		})
	}
	return days
}

// rank orders a day: milestones, then what is still to do (by time), then what was finished.
func rank(k CalKind) int {
	switch k {
	case CalMilestone:
		return 0
	case CalDone:
		return 2
	}
	return 1
}

// doneMoment is when a finished task was finished: its completion time, else its last update.
func doneMoment(t Task) time.Time {
	if t.CompletedAt != nil {
		return *t.CompletedAt
	}
	return t.UpdatedAt
}
