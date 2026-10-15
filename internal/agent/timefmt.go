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

// storedTime reads a stored timestamp in the zone of now. A malformed one is the zero time.
func storedTime(stored string, now time.Time) time.Time {
	t, err := time.Parse(time.RFC3339Nano, stored)
	if err != nil {
		return time.Time{}
	}
	return t.In(now.Location())
}
