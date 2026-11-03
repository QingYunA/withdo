package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// TestCalendarIsTheDesktopsMonth pins Calendar against data.BuildCalendar: the same days, the
// same items in the same order, for the demo data.
func TestCalendarIsTheDesktopsMonth(t *testing.T) {
	phone := demoPhone(t)
	desktop, err := data.OpenReal(context.Background(), phone.dbPath, data.RealOptions{
		Now: func() time.Time { return demoNow },
	})
	if err != nil {
		t.Fatalf("open the desktop store: %v", err)
	}
	defer func() { _ = desktop.Close() }()
	snap, err := desktop.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var milestones []data.Milestone
	for _, list := range snap.Milestones {
		milestones = append(milestones, list...)
	}

	for _, start := range []time.Weekday{time.Monday, time.Sunday} {
		raw, err := phone.Calendar(2026, 10, int64(start))
		if err != nil {
			t.Fatal(err)
		}
		var got calendarJSON
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Calendar: %v", err)
		}
		want := data.BuildCalendar(data.CalendarInput{
			Month: time.Date(2026, 10, 1, 0, 0, 0, 0, demoNow.Location()), Now: demoNow, WeekStart: start,
			Tasks: snap.Tasks, Milestones: milestones,
			Agent: func(id int) string {
				if r := snap.LatestReport(id); r != nil {
					return r.Agent
				}
				return ""
			},
		})
		if got.Year != 2026 || got.Month != 10 || got.Weeks*7 != len(want) || len(got.Days) != len(want) {
			t.Fatalf("start %v: grid is %d-%d with %d weeks and %d days, want %d days", start, got.Year, got.Month, got.Weeks, len(got.Days), len(want))
		}
		items := 0
		for i, d := range got.Days {
			w := want[i]
			if d.Date != w.Date.Format("2006-01-02") || d.InMonth != w.InMonth || d.Weekend != w.Weekend || d.Today != w.Today {
				t.Errorf("start %v: day %d = %+v, want %+v", start, i, d, w)
			}
			if len(d.Items) != len(w.Items) {
				t.Errorf("start %v: %s has %d items, want %d", start, d.Date, len(d.Items), len(w.Items))
				continue
			}
			for j, it := range d.Items {
				wi := w.Items[j]
				items++
				if it.Kind != calKindWord(wi.Kind) || it.Overdue != wi.Overdue || it.Agent != wi.Agent {
					t.Errorf("start %v: %s item %d = %+v, want %+v", start, d.Date, j, it, wi)
				}
				if (wi.Kind == data.CalMilestone) != (it.Milestone != nil) || (wi.Kind != data.CalMilestone) != (it.Task != nil) {
					t.Errorf("%s item %d carries the wrong subject: %+v", d.Date, j, it)
				}
				if it.Task != nil && it.Task.ID != wi.Task.ID || it.Milestone != nil && it.Milestone.ID != wi.Milestone.ID {
					t.Errorf("%s item %d is another row than the desktop's", d.Date, j)
				}
			}
		}
		if items == 0 {
			t.Errorf("start %v: the demo month holds no items", start)
		}
		if start == time.Monday {
			checkGolden(t, "calendar.golden.json", raw)
		}
	}
}

// TestCalendarWeeksFollowTheFirstWeekday: the grid starts on the week start on or before the 1st,
// a month the phone has never seen is empty, and a bad month is an error.
func TestCalendarWeeksFollowTheFirstWeekday(t *testing.T) {
	phone, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = phone.Close() }()
	cases := []struct {
		year, month, start int64
		first              string
		weeks              int
	}{
		{2026, 11, 1, "2026-10-26", 6}, // Nov 1 is a Sunday: six Monday weeks
		{2026, 11, 0, "2026-11-01", 5},
		{2026, 2, 1, "2026-01-26", 5},
		{2027, 2, 1, "2027-02-01", 4}, // Feb 1 a Monday, 28 days: four weeks
	}
	for _, c := range cases {
		raw, err := phone.Calendar(c.year, c.month, c.start)
		if err != nil {
			t.Fatal(err)
		}
		var got calendarJSON
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatal(err)
		}
		if got.Weeks != c.weeks || len(got.Days) != c.weeks*7 || got.Days[0].Date != c.first {
			t.Errorf("%d-%d start %d: %d weeks from %s, want %d from %s", c.year, c.month, c.start, got.Weeks, got.Days[0].Date, c.weeks, c.first)
		}
		if findNull(decodeAny(t, raw), "Calendar") != "" {
			t.Errorf("an empty month has a null: %s", raw)
		}
	}
	for _, bad := range [][3]int64{{2026, 0, 1}, {2026, 13, 1}, {2026, 10, 7}, {2026, 10, -1}} {
		if _, err := phone.Calendar(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("Calendar%v did not fail", bad)
		}
	}
}

func decodeAny(t *testing.T, raw string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
