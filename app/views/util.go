package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// overviewTask describes a task the way Store.Overview does: with its latest report and whether
// it has gone stale.
func (a *App) overviewTask(t data.Task) data.OverviewTask {
	ot := data.OverviewTask{Task: t}
	since := t.UpdatedAt
	if r := a.latestReport(t.ID); r != nil {
		ot.LatestReport = r
		since = r.CreatedAt
	}
	ot.HoursSinceActivity = int(a.Now().Sub(since) / time.Hour)
	stale := a.snap.overview.StaleHours
	if stale == 0 {
		stale = data.DefaultStaleHours
	}
	ot.Stale = ot.HoursSinceActivity >= stale
	return ot
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
