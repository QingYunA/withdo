// Package words says times in the user's language and regional formats: how long ago something
// happened, when a task is due, the day and hour of a reminder. The words come from the app's
// catalog and the dates and clock from package i18n, in the user's formats ("Fri" or "周五",
// "4:00 PM" or "16:00").
package words

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/egoist/mygo/i18n"
)

// Clock writes times for one reader.
type Clock struct {
	// Say is the catalog in the language the app shows.
	Say locales.Messages
	// Formats is the locale of the reader's dates and hours: the language with the region of the
	// user's formats.
	Formats string
}

// New is the clock of a reader of language, with the formats region says.
func New(language string, region i18n.Region) Clock {
	return Clock{Say: locales.In(language), Formats: region.FormatLocale(language)}
}

func (k Clock) format(t time.Time, skeleton string) string {
	return i18n.FormatDate(k.Formats, t, i18n.DateOptions{Skeleton: skeleton})
}

// Time is the time of day: "4:00 PM", "下午4:00", or "16:00" for a reader of 24 hours.
func (k Clock) Time(t time.Time) string { return k.format(t, "jm") }

// Date is a calendar date: "Oct 12", or "Oct 12, 2025" in another year than now's.
func (k Clock) Date(t, now time.Time) string {
	t = t.In(now.Location())
	if t.Year() != now.Year() {
		return k.format(t, "yMMMd")
	}
	return k.format(t, "MMMd")
}

// Heading is the day at the top of Now: "Wednesday, Oct 14".
func (k Clock) Heading(now time.Time) string { return k.format(now, "MMMEEEEd") }

// Age is how long ago t was, short: "now", "2m", "3h", "2d", then a date once it is a week or
// more.
func (k Clock) Age(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return k.Say.AgeNow()
	case d < time.Hour:
		return k.Say.AgeMinutes(int(d / time.Minute))
	case d < 24*time.Hour:
		return k.Say.AgeHours(int(d / time.Hour))
	case d < 7*24*time.Hour:
		return k.Say.AgeDays(int(d / (24 * time.Hour)))
	}
	return k.format(t.In(now.Location()), "MMMd")
}

// Span writes a span of hours as "5h" or, from two days, "3d".
func (k Clock) Span(hours int) string {
	if hours >= 48 {
		return k.Say.SpanDays(hours / 24)
	}
	return k.Say.SpanHours(max(hours, 1))
}

// Stale writes how long an agent has been quiet: "54h stale" or "3d stale".
func (k Clock) Stale(hours int) string {
	if hours >= 48 {
		return k.Say.StaleDays(hours / 24)
	}
	return k.Say.StaleHours(hours)
}

// DayInline names the calendar day of t inside a sentence: "today", "tomorrow", "yesterday", the
// weekday inside the next six days, otherwise a date.
func (k Clock) DayInline(t, now time.Time) string {
	switch days := data.DaysFrom(t, now); {
	case days == 0:
		return k.Say.DayTodayInline()
	case days == 1:
		return k.Say.DayTomorrowInline()
	case days == -1:
		return k.Say.DayYesterdayInline()
	case days > 1 && days < 7:
		return k.format(t.In(now.Location()), "EEEE")
	}
	return k.Date(t, now)
}

// day names the calendar day of t as a label: "Today", "Tomorrow", "Yesterday", the weekday
// inside the next six days (short when asked), otherwise a date.
func (k Clock) day(t, now time.Time, shortWeekday bool) string {
	switch days := data.DaysFrom(t, now); {
	case days == 0:
		return k.Say.DayToday()
	case days == 1:
		return k.Say.DayTomorrow()
	case days == -1:
		return k.Say.DayYesterday()
	case days > 1 && days < 7:
		skeleton := "EEEE"
		if shortWeekday {
			skeleton = "E"
		}
		return k.format(t.In(now.Location()), skeleton)
	}
	return k.Date(t, now)
}

// DueLabel is the right-hand label of one of my tasks: "Today", "Tomorrow", "Fri", "Oct 20". A
// reminder that is still ahead today, or came due today, reads as its time of day. It is empty
// for a task with neither a due nor a reminder time.
func (k Clock) DueLabel(t data.Task, now time.Time) string {
	at, ok := data.NextMoment(t)
	if !ok {
		return ""
	}
	if t.RemindAt != nil && at.Equal(*t.RemindAt) && data.DaysFrom(at, now) == 0 {
		return k.Time(at.In(now.Location()))
	}
	return k.day(at, now, true)
}

// When is a moment as a receipt says it: "Today 4:00 PM", "Tomorrow 9:00 AM", "Fri 9:00 AM".
func (k Clock) When(t, now time.Time) string {
	t = t.In(now.Location())
	return k.Say.DayAndTime(k.day(t, now, true), k.Time(t))
}

// Moment is a due or reminder time in the inspector: "Tomorrow" or "Tomorrow, 10:00". The hour
// shows for a reminder, and for a due time that is not at midnight.
func (k Clock) Moment(t, now time.Time, withTime bool) string {
	day := k.day(t, now, false)
	if withTime || t.Hour() != 0 && t.Minute() != 0 {
		return k.Say.DayAndTimeComma(day, k.Time(t))
	}
	return day
}

// Reminder says that a reminder came due at `at`, for the second line of a Needs you row.
func (k Clock) Reminder(at, now time.Time) string {
	at = at.In(now.Location())
	switch data.DaysFrom(at, now) {
	case 0:
		return k.Say.NeedReminderToday(k.Time(at))
	case -1:
		return k.Say.NeedReminderYesterday(k.Time(at))
	}
	return k.Say.NeedReminderOn(k.Date(at, now), k.Time(at))
}

// Day names the calendar day of t as a heading: "Today", "Yesterday", the weekday inside the next
// six days, otherwise a date.
func (k Clock) Day(t, now time.Time) string { return k.day(t, now, false) }
