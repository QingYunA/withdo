package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// pastGrace is how far behind the clock a reminder may be and still count as "now".
const pastGrace = time.Minute

// parseWhen reads an absolute time the model wrote: RFC 3339 with a UTC offset or Z. A time
// without a zone is rejected rather than guessed, so the model has to be explicit about it. It
// returns the stored form, and rejects a reminder that is already in the past.
func parseWhen(field, s string, now time.Time, future bool) (string, error) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("%s %q must be RFC 3339 with a UTC offset, e.g. %s",
			field, s, now.Format(time.RFC3339))
	}
	if future && t.Before(now.Add(-pastGrace)) {
		return "", fmt.Errorf("%s %s is in the past (it is now %s); pick a later time or ask the user",
			field, s, now.Format(time.RFC3339))
	}
	return domain.FormatTime(t), nil
}

// localRFC3339 renders a stored timestamp in the user's zone, the form the model writes times in.
func localRFC3339(stored *string, loc *time.Location) string {
	if stored == nil {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, *stored)
	if err != nil {
		return *stored
	}
	return t.In(loc).Format(time.RFC3339)
}

// whenLabel is a moment the way a receipt says it: "Today 4:00 PM", "Tomorrow 9:00 AM", "Fri
// 9:00 AM", "Oct 20 9:00 AM".
func whenLabel(stored string, now time.Time) string {
	t, err := time.Parse(time.RFC3339Nano, stored)
	if err != nil {
		return stored
	}
	t, now = t.In(now.Location()), now.In(now.Location())
	days := dayIndex(t) - dayIndex(now)
	var day string
	switch {
	case days == 0:
		day = "Today"
	case days == 1:
		day = "Tomorrow"
	case days == -1:
		day = "Yesterday"
	case days > 1 && days < 7:
		day = t.Format("Mon")
	case t.Year() != now.Year():
		day = t.Format("Jan 2, 2006")
	default:
		day = t.Format("Jan 2")
	}
	return day + " " + t.Format("3:04 PM")
}

func dayIndex(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}
