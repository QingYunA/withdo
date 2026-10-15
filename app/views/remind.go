package views

import "time"

// remindKind names a ready-made reminder of the task menu.
type remindKind int

const (
	remindLater    remindKind = iota // today at 18:00
	remindTomorrow                   // tomorrow at 9:00
	remindMonday                     // next Monday at 9:00
	remindWeekend                    // Saturday at 10:00
)

// remindPreset is a reminder the menu offers: which, and the moment it means.
type remindPreset struct {
	kind remindKind
	at   time.Time
}

// remindPresets are the ready-made reminders for now, in the order the menu lists them. Later today
// is offered while 18:00 is over half an hour away. Tomorrow is always there. Next Monday is
// the Monday after today (a week ahead on a Monday) and is left out when that is tomorrow. This
// weekend is the coming Saturday, 10:00, or today's before then, and is left out from Saturday
// 9:30 on and on Sunday.
func remindPresets(now time.Time) []remindPreset {
	at := func(days, hour int) time.Time {
		y, m, d := now.Date()
		return time.Date(y, m, d+days, hour, 0, 0, 0, now.Location())
	}
	var out []remindPreset
	if evening := at(0, 18); evening.Sub(now) > 30*time.Minute {
		out = append(out, remindPreset{remindLater, evening})
	}
	out = append(out, remindPreset{remindTomorrow, at(1, 9)})
	toMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
	if toMonday == 0 {
		toMonday = 7
	}
	if toMonday > 1 {
		out = append(out, remindPreset{remindMonday, at(toMonday, 9)})
	}
	switch toSaturday := (int(time.Saturday) - int(now.Weekday()) + 7) % 7; {
	case now.Weekday() == time.Sunday:
	case toSaturday == 0 && at(0, 10).Sub(now) > 30*time.Minute:
		out = append(out, remindPreset{remindWeekend, at(0, 10)})
	case toSaturday > 0:
		out = append(out, remindPreset{remindWeekend, at(toSaturday, 10)})
	}
	return out
}
