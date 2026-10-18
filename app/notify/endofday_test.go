package notify

import (
	"errors"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/egoist/mygo"
)

func TestTheEndOfDayNotificationIsWordedPerTheSpec(t *testing.T) {
	day, week := data.SummaryCounts{Finished: 7, Merged: 3}, data.SummaryCounts{Finished: 25, Merged: 9}
	for _, c := range []struct {
		name   string
		friday bool
		day    data.SummaryCounts
		week   data.SummaryCounts
		want   mygo.NotificationOptions
		zh     mygo.NotificationOptions
	}{
		{"a day", false, day, week,
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "Day's done", Body: "7 finished today, 3 PRs merged. See today's shipping?"},
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "收工了", Body: "今天做完 7 件，合并了 3 个 PR。来看看今天的出货单？"}},
		{"a day with one PR", false, data.SummaryCounts{Finished: 2, Merged: 1}, week,
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "Day's done", Body: "2 finished today, 1 PR merged. See today's shipping?"},
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "收工了", Body: "今天做完 2 件，合并了 1 个 PR。来看看今天的出货单？"}},
		{"a day without a merge", false, data.SummaryCounts{Finished: 4}, week,
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "Day's done", Body: "4 finished today. See today's shipping?"},
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "收工了", Body: "今天做完 4 件。来看看今天的出货单？"}},
		{"Friday is the week", true, day, week,
			mygo.NotificationOptions{ID: "end-of-day/week", Title: "Day's done", Body: "This week: 25 finished. The weekly summary is ready."},
			mygo.NotificationOptions{ID: "end-of-day/week", Title: "收工了，这周辛苦", Body: "这周做完 25 件。周报写好了，来看看？"}},
		{"a day with nothing", false, data.SummaryCounts{}, week,
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "Day's done", Body: "That's it for today. See you tomorrow."},
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "收工了", Body: "今天先到这里，明天见。"}},
		{"a Friday with nothing all week", true, data.SummaryCounts{}, data.SummaryCounts{},
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "Day's done", Body: "That's it for today. See you tomorrow."},
			mygo.NotificationOptions{ID: "end-of-day/day", Title: "收工了", Body: "今天先到这里，明天见。"}},
	} {
		if got := EndOfDayOptions(en, c.friday, c.day, c.week); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
		if got := EndOfDayOptions(zh, c.friday, c.day, c.week); got != c.zh {
			t.Errorf("%s (zh): %+v, want %+v", c.name, got, c.zh)
		}
	}
}

func TestEndOfDayIDsOpenTheirPeriod(t *testing.T) {
	for _, week := range []bool{false, true} {
		if got, ok := ParseEndOfDayID(EndOfDayID(week)); !ok || got != week {
			t.Errorf("week %v: %v %v", week, got, ok)
		}
	}
	for _, id := range []string{"summary", "task/3", "end-of-day"} {
		if _, ok := ParseEndOfDayID(id); ok {
			t.Errorf("%q is not an end-of-day id", id)
		}
	}
}

// reminder is an EndOfDay on a fake clock with fake counts, recording what it showed.
type reminder struct {
	*EndOfDay
	now     time.Time
	on      bool
	at      string
	last    string
	viewing bool
	failing bool
	shown   []mygo.NotificationOptions
	counted int
}

func newReminder() *reminder {
	r := &reminder{now: time.Date(2026, 10, 14, 18, 29, 0, 0, time.Local), on: true} // Wednesday
	r.EndOfDay = &EndOfDay{
		Now: func() time.Time { return r.now }, Enabled: func() bool { return r.on }, At: func() string { return r.at },
		Last: func() string { return r.last }, SetLast: func(d string) { r.last = d }, Viewing: func() bool { return r.viewing },
		Count: func(time.Time) (data.SummaryCounts, data.SummaryCounts, error) {
			r.counted++
			if r.failing {
				return data.SummaryCounts{}, data.SummaryCounts{}, errors.New("no store")
			}
			return data.SummaryCounts{Finished: 7, Merged: 3}, data.SummaryCounts{Finished: 25}, nil
		},
		Say:  func() locales.Messages { return en },
		Show: func(o mygo.NotificationOptions) { r.shown = append(r.shown, o) },
	}
	return r
}

func (r *reminder) at1830(h, m int) { r.now = time.Date(2026, 10, r.now.Day(), h, m, 0, 0, time.Local) }

func TestTheReminderComesOnceAt1830OnAWeekday(t *testing.T) {
	r := newReminder()
	r.Tick()
	if len(r.shown) != 0 {
		t.Fatal("it came a minute early")
	}
	r.at1830(18, 30)
	r.Tick()
	r.at1830(18, 31)
	r.Tick()
	r.at1830(19, 5)
	r.Tick()
	if len(r.shown) != 1 || r.shown[0].Body != "7 finished today, 3 PRs merged. See today's shipping?" {
		t.Errorf("shown %+v", r.shown)
	}
	if r.last != "2026-10-14" {
		t.Errorf("last %q", r.last)
	}
}

func TestTheReminderFollowsTheTimeTheUserPicks(t *testing.T) {
	r := newReminder()
	r.at = "17:00"
	r.at1830(17, 0)
	r.Tick()
	if len(r.shown) != 1 {
		t.Errorf("shown %d at 17:00", len(r.shown))
	}
	r2 := newReminder()
	r2.at = "garbage" // the default stands
	r2.at1830(18, 30)
	r2.Tick()
	if len(r2.shown) != 1 {
		t.Errorf("shown %d with an unreadable time", len(r2.shown))
	}
}

func TestTheReminderDoesNotComeAtTheWeekendOrLongAfterItsTime(t *testing.T) {
	r := newReminder()
	r.now = time.Date(2026, 10, 17, 18, 30, 0, 0, time.Local) // Saturday
	r.Tick()
	r.now = time.Date(2026, 10, 18, 18, 30, 0, 0, time.Local) // Sunday
	r.Tick()
	r.now = time.Date(2026, 10, 15, 20, 31, 0, 0, time.Local) // Thursday, two hours late
	r.Tick()
	r.now = time.Date(2026, 10, 15, 22, 0, 0, 0, time.Local) // the app opened at ten at night
	r.Tick()
	if len(r.shown) != 0 {
		t.Errorf("shown %+v", r.shown)
	}
	r.now = time.Date(2026, 10, 15, 19, 0, 0, 0, time.Local) // woke at seven: still today's
	r.Tick()
	if len(r.shown) != 1 {
		t.Errorf("a Mac that woke half an hour late: %d", len(r.shown))
	}
}

func TestTheReminderIsOffWithItsSwitch(t *testing.T) {
	r := newReminder()
	r.on = false
	r.at1830(18, 30)
	r.Tick()
	if len(r.shown) != 0 || r.counted != 0 || r.last != "" {
		t.Errorf("shown %d, counted %d, last %q", len(r.shown), r.counted, r.last)
	}
}

func TestTheReminderIsLeftOutWhileSummaryShowsAndNotSentLater(t *testing.T) {
	r := newReminder()
	r.viewing = true
	r.at1830(18, 30)
	r.Tick()
	r.viewing = false
	r.at1830(18, 40)
	r.Tick()
	if len(r.shown) != 0 {
		t.Errorf("shown %+v", r.shown)
	}
	if r.last != "2026-10-14" {
		t.Errorf("last %q: the day is spent", r.last)
	}
}

func TestOnFridayTheReminderIsTheWeeks(t *testing.T) {
	r := newReminder()
	r.now = time.Date(2026, 10, 16, 18, 30, 0, 0, time.Local)
	r.Tick()
	if len(r.shown) != 1 || r.shown[0].ID != "end-of-day/week" || r.shown[0].Body != "This week: 25 finished. The weekly summary is ready." {
		t.Errorf("shown %+v", r.shown)
	}
}

func TestWhenTheStoreCannotBeReadTheReminderSaysNothing(t *testing.T) {
	r := newReminder()
	r.failing = true
	r.at1830(18, 30)
	r.Tick()
	if len(r.shown) != 0 {
		t.Errorf("shown %+v", r.shown)
	}
}
