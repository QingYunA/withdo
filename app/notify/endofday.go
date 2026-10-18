package notify

import (
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/egoist/mygo"
)

// The end-of-day reminder (ui-spec, Notifications and Summary): Monday to Friday at a time the user
// picks, one notification that opens the Summary: today's, or on Friday the week's. It does not come
// while the window shows Summary, and a day with nothing finished says so and never "0".

// EndOfDayWindow is how long after its time the reminder may still come: a Mac that was asleep at
// 18:30 says it when it wakes at 19:00, one that is opened at ten at night says nothing.
const EndOfDayWindow = 2 * time.Hour

// EndOfDayID is the ID of the end-of-day notification: a click opens the summary of the week or of
// the day.
func EndOfDayID(week bool) string {
	if week {
		return "end-of-day/week"
	}
	return "end-of-day/day"
}

// ParseEndOfDayID reverses EndOfDayID.
func ParseEndOfDayID(id string) (week, ok bool) {
	switch id {
	case "end-of-day/day":
		return false, true
	case "end-of-day/week":
		return true, true
	}
	return false, false
}

// EndOfDay decides, once the app asks, whether the reminder is due and what it says. Everything that
// touches the system is a field, so that a test can stand in for it.
type EndOfDay struct {
	Now func() time.Time
	// Enabled says whether the reminder is on: the notifications and the reminder's own switch.
	Enabled func() bool
	// At is when it comes, as "15:04".
	At func() string
	// Last is the date the reminder last came, or was left out; SetLast keeps it.
	Last    func() string
	SetLast func(date string)
	// Viewing says whether the user is looking at Summary right now.
	Viewing func() bool
	// Count says how many tasks were finished in the day and in the week.
	Count func(now time.Time) (day, week data.SummaryCounts, err error)
	Say   func() locales.Messages
	// Show shows the notification.
	Show func(mygo.NotificationOptions)

	mu sync.Mutex
}

// Due reports whether the reminder comes at now: a weekday, within EndOfDayWindow after its time, and
// not yet today.
func (e *EndOfDay) Due(now time.Time) bool {
	if !e.Enabled() || now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	h, m, ok := data.ClockOf(e.At())
	if !ok {
		h, m, _ = data.ClockOf(data.DefaultEndOfDay)
	}
	y, mo, d := now.Date()
	at := time.Date(y, mo, d, h, m, 0, 0, now.Location())
	return !now.Before(at) && now.Before(at.Add(EndOfDayWindow)) && e.Last() != now.Format("2006-01-02")
}

// Tick shows the reminder when it is due. It remembers the day first, so a reminder left out because
// the user is looking at Summary is not sent a minute later. Call it every half minute or so; it
// blocks while the system answers.
func (e *EndOfDay) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := e.Now()
	if !e.Due(now) {
		return
	}
	e.SetLast(now.Format("2006-01-02"))
	if e.Viewing() {
		return
	}
	day, week, err := e.Count(now)
	if err != nil {
		return
	}
	e.Show(EndOfDayOptions(e.Say(), now.Weekday() == time.Friday, day, week))
}

// EndOfDayOptions words the notification: the day's, or on Friday the week's. A day with nothing
// finished says "that's it for today", whatever the weekday.
func EndOfDayOptions(say locales.Messages, friday bool, day, week data.SummaryCounts) mygo.NotificationOptions {
	switch {
	case friday && week.Finished > 0:
		return mygo.NotificationOptions{ID: EndOfDayID(true), Title: say.NotifyWeekDoneTitle(), Body: say.NotifyWeekDoneBody(week.Finished)}
	case !friday && day.Finished > 0 && day.Merged > 0:
		return mygo.NotificationOptions{ID: EndOfDayID(false), Title: say.NotifyDayDoneTitle(), Body: say.NotifyDayDoneBodyPRs(day.Finished, day.Merged)}
	case !friday && day.Finished > 0:
		return mygo.NotificationOptions{ID: EndOfDayID(false), Title: say.NotifyDayDoneTitle(), Body: say.NotifyDayDoneBody(day.Finished)}
	}
	return mygo.NotificationOptions{ID: EndOfDayID(false), Title: say.NotifyDayDoneTitle(), Body: say.NotifyDayNothingBody()}
}
