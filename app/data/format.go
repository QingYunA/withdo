package data

import (
	"fmt"
	"time"
)

// Age writes how long ago t was, in the short form the UI uses: "now", "2m", "3h", "2d", then
// a date such as "Oct 12" once it is a week or more.
func Age(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	}
	return t.Format("Jan 2")
}

// When writes a future (or just past) moment for a due or reminder time: "in 1h", "in 20m",
// "today", "tomorrow", a weekday within the week, otherwise a date. Moments in the past read
// "1h ago" or "yesterday".
func When(t, now time.Time) string {
	d := t.Sub(now)
	if d < 0 {
		return whenPast(t, now)
	}
	if d < time.Hour {
		return fmt.Sprintf("in %dm", max(int(d/time.Minute), 1))
	}
	if d < 6*time.Hour {
		return fmt.Sprintf("in %dh", int(d/time.Hour))
	}
	return DayWord(t, now)
}

func whenPast(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 6*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	}
	return DayWord(t, now)
}

// DayWord names the calendar day of t relative to now: "today", "tomorrow", "yesterday", the
// weekday inside the next six days, otherwise "Oct 12" (with the year when it is not this one).
func DayWord(t, now time.Time) string {
	t, now = t.In(now.Location()), now.In(now.Location())
	days := dayIndex(t) - dayIndex(now)
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	case days == -1:
		return "yesterday"
	case days > 1 && days < 7:
		return t.Format("Monday")
	}
	if t.Year() != now.Year() {
		return t.Format("Jan 2, 2006")
	}
	return t.Format("Jan 2")
}

func dayIndex(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// Overdue reports whether a due time has passed.
func Overdue(t, now time.Time) bool { return t.Before(now) }
