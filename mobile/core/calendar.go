package core

import (
	"fmt"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// calendarItemJSON is one thing on a day of the Calendar tab: data.CalItem with its kind in words,
// the project's name, and the task or the milestone that put it there (the other is left out).
type calendarItemJSON struct {
	// Kind is "milestone", "reminder", "due" or "done" (data.CalKind).
	Kind string
	// At is the moment that put the item on the day, RFC 3339 with the device's offset.
	At string
	// Overdue is an open task whose due time has passed, however its item is shown.
	Overdue bool
	// Agent is the agent that wrote the task by its latest report ("agent" with none), "" for a
	// task of mine and for a milestone.
	Agent string
	// Project is the name of the task's or milestone's project, "" for the Inbox.
	Project   string
	Task      *data.Task      `json:",omitempty"`
	Milestone *data.Milestone `json:",omitempty"`
}

// calendarDayJSON is one cell of the month grid.
type calendarDayJSON struct {
	Date    string // 2026-11-03, the device's calendar day
	InMonth bool
	Weekend bool
	Today   bool
	Items   []calendarItemJSON
}

// calendarJSON is the JSON of Calendar.
type calendarJSON struct {
	Year  int
	Month int
	Weeks int
	Days  []calendarDayJSON
}

// calKindWord names a data.CalKind in the JSON.
func calKindWord(k data.CalKind) string {
	switch k {
	case data.CalMilestone:
		return "milestone"
	case data.CalReminder:
		return "reminder"
	case data.CalDone:
		return "done"
	}
	return "due"
}

// Calendar returns the month grid of the Calendar tab as JSON: what the desktop's Calendar page
// draws (data.MonthGrid and data.BuildCalendar over the same snapshot). month is 1 to 12;
// weekStart is the day the weeks start on, 0 Sunday to 6 Saturday (time.Weekday), which the phone
// takes from the system. The grid holds the weeks the month needs (four to six), 7 days each.
//
//	{"Year":2026,"Month":11,"Weeks":6,"Days":[{"Date":"2026-10-26","InMonth":false,"Weekend":false,
//	  "Today":false,"Items":[Item...]}, ...]}
//
// An Item has Kind ("milestone", "reminder", "due" or "done"), At, Overdue, Agent, Project and
// either Task (a data.Task) or Milestone (a data.Milestone), the other absent. A day's items come
// in the desktop's order: milestones, open items by time, finished items. Cancelled tasks, closed
// milestones and subtasks are not there; an open task due and reminded the same day shows once,
// as the reminder.
func (c *Core) Calendar(year, month, weekStart int64) (string, error) {
	if month < 1 || month > 12 || year < 1 || year > 9999 {
		return "", fmt.Errorf("month %d-%d is out of range", year, month)
	}
	if weekStart < 0 || weekStart > 6 {
		return "", fmt.Errorf("weekStart %d is out of range", weekStart)
	}
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	now := c.nowTime()
	var milestones []data.Milestone
	for _, list := range snap.Milestones {
		milestones = append(milestones, list...)
	}
	days := data.BuildCalendar(data.CalendarInput{
		Month:      time.Date(int(year), time.Month(month), 1, 0, 0, 0, 0, now.Location()),
		Now:        now,
		WeekStart:  time.Weekday(weekStart),
		Tasks:      snap.Tasks,
		Milestones: milestones,
		Agent: func(id int) string {
			if r := snap.LatestReport(id); r != nil {
				return r.Agent
			}
			return ""
		},
	})
	out := calendarJSON{Year: int(year), Month: int(month), Weeks: len(days) / 7, Days: make([]calendarDayJSON, 0, len(days))}
	for _, d := range days {
		day := calendarDayJSON{
			Date: d.Date.Format("2006-01-02"), InMonth: d.InMonth, Weekend: d.Weekend, Today: d.Today,
			Items: make([]calendarItemJSON, 0, len(d.Items)),
		}
		for _, it := range d.Items {
			item := calendarItemJSON{
				Kind: calKindWord(it.Kind), At: it.At.Format(time.RFC3339), Overdue: it.Overdue, Agent: it.Agent,
			}
			if it.Kind == data.CalMilestone {
				m := it.Milestone
				pid := m.ProjectID
				item.Milestone, item.Project = &m, snap.ProjectName(&pid)
			} else {
				t := it.Task
				item.Task, item.Project = &t, snap.ProjectName(t.ProjectID)
			}
			day.Items = append(day.Items, item)
		}
		out.Days = append(out.Days, day)
	}
	return marshal(out)
}
