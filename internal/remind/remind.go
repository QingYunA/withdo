// Package remind computes the ready-made reminders of the task menu. The desktop's menu and the
// phone's bottom sheet offer the same four, at the same moments, so the rule lives here instead of
// in either UI.
package remind

import "time"

// Kind names a ready-made reminder of the task menu.
type Kind int

const (
	Later    Kind = iota // today at 18:00
	Tomorrow             // tomorrow at 9:00
	Monday               // next Monday at 9:00
	Weekend              // Saturday at 10:00
)

// String is the JSON name of the kind: later, tomorrow, monday or weekend.
func (k Kind) String() string {
	switch k {
	case Later:
		return "later"
	case Tomorrow:
		return "tomorrow"
	case Monday:
		return "monday"
	case Weekend:
		return "weekend"
	}
	return ""
}

// Preset is a reminder the menu offers: which, and the moment it means.
type Preset struct {
	Kind Kind
	At   time.Time
}

// Presets are the ready-made reminders for now, in the order the menu lists them. Later today is
// offered while 18:00 is over half an hour away. Tomorrow is always there. Next Monday is the
// Monday after today (a week ahead on a Monday) and is left out when that is tomorrow. This weekend
// is the coming Saturday, 10:00, or today's before then, and is left out from Saturday 9:30 on and
// on Sunday.
func Presets(now time.Time) []Preset {
	at := func(days, hour int) time.Time {
		y, m, d := now.Date()
		return time.Date(y, m, d+days, hour, 0, 0, 0, now.Location())
	}
	var out []Preset
	if evening := at(0, 18); evening.Sub(now) > 30*time.Minute {
		out = append(out, Preset{Later, evening})
	}
	out = append(out, Preset{Tomorrow, at(1, 9)})
	toMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
	if toMonday == 0 {
		toMonday = 7
	}
	if toMonday > 1 {
		out = append(out, Preset{Monday, at(toMonday, 9)})
	}
	switch toSaturday := (int(time.Saturday) - int(now.Weekday()) + 7) % 7; {
	case now.Weekday() == time.Sunday:
	case toSaturday == 0 && at(0, 10).Sub(now) > 30*time.Minute:
		out = append(out, Preset{Weekend, at(0, 10)})
	case toSaturday > 0:
		out = append(out, Preset{Weekend, at(toSaturday, 10)})
	}
	return out
}
