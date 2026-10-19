package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The end-of-day reminder counts from the store's snapshot and the page from the app's: for any day
// or week they say the same.
func TestTheReminderCountsWhatTheSummaryPageShows(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	answered(t, tt, rec)
	snap, err := app.Store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	now := app.Now()
	sawTasks := false
	for _, week := range []bool{false, true} {
		for back := 0; back < 4; back++ {
			m.sum.week, m.sum.back = week, back
			p := m.periodOf(now)
			page := m.summaryFor(now).sum.Counts()
			if reminder := data.CountSummary(snap, p, nil); reminder != page {
				t.Errorf("week %v, %d back: reminder %+v, page %+v", week, back, reminder, page)
			}
			sawTasks = sawTasks || page.Finished > 0
		}
	}
	if !sawTasks {
		t.Error("the fixture finished nothing in any period")
	}
	m.sum.week, m.sum.back = false, 0
	if got := m.summaryFor(now).sum.Counts(); got != (data.SummaryCounts{Finished: 8, Merged: 3}) {
		t.Errorf("today = %+v, want 8 finished and 3 merged", got)
	}
}
