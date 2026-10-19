package main

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/notify"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo/ui"
)

// A click on the end-of-day notification opens Summary: today's, or the week's.
func TestClickingTheEndOfDayNotificationOpensSummary(t *testing.T) {
	be, err := openBackend(true, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = be.close() })
	app := views.NewApp(be.store, nil, views.Config{})
	m := views.NewMain(app)
	s := &shell{app: app, main: m}
	tt := ui.NewTester(m.View, 1100, 720)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	tt.Frame()
	if m.ShowingSummary() {
		t.Fatal("the test started on Summary")
	}
	s.openFromNotification(notify.EndOfDayID(false))
	tt.Frame()
	if !m.ShowingSummary() || !tt.HasText("Today") {
		t.Errorf("the notification did not open Summary: %q", tt.Texts())
	}
	s.openFromNotification(notify.EndOfDayID(true))
	tt.Frame()
	if !m.ShowingSummary() {
		t.Error("the week's notification left Summary")
	}
}

// The reminder counts from the store the app runs on, and is wired to the app's choices.
func TestTheShellsEndOfDayCountsFromTheStoreAndFollowsThePrefs(t *testing.T) {
	be, err := openBackend(true, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = be.close() })
	app := views.NewApp(be.store, nil, views.Config{})
	app.Prefs = be.prefs
	s := &shell{app: app, main: views.NewMain(app)}
	s.notifier = s.newNotifier()
	eod := s.newEndOfDay()
	day, week, err := eod.Count(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if day.Finished < 0 || week.Finished < day.Finished {
		t.Errorf("day %+v, week %+v", day, week)
	}
	if !eod.Enabled() || eod.At() != "18:30" {
		t.Errorf("enabled %v at %q: on at 18:30 until the user says otherwise", eod.Enabled(), eod.At())
	}
	app.SetEndOfDay(false)
	app.SetEndOfDayAt("17:45")
	if eod.Enabled() || eod.At() != "17:45" {
		t.Errorf("enabled %v at %q after the user's choices", eod.Enabled(), eod.At())
	}
	eod.SetLast("2026-10-14")
	if eod.Last() != "2026-10-14" {
		t.Error("the day the reminder last came is not kept")
	}
}
