package data

import "time"

// DaysFrom counts the calendar days from now to t: 0 for today, 1 for tomorrow, -1 for
// yesterday.
func DaysFrom(t, now time.Time) int {
	t, now = t.In(now.Location()), now.In(now.Location())
	return dayIndex(t) - dayIndex(now)
}

func dayIndex(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// Overdue reports whether a due time has passed.
func Overdue(t, now time.Time) bool { return t.Before(now) }
