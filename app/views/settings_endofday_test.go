package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

func TestGeneralPaneHasTheEndOfDayReminderOnAt1830ByDefault(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "System notifications")
	hasNot(t, r.tt, "End-of-day reminder")
	r.pane(t, "Advanced")
	r.tt.Scroll(340, 300, 0, 3000)
	r.tt.Frame()
	has(t, r.tt, "End-of-day reminder", "Weekdays: today's summary, and the weekly one on Friday", "Reminder time", "6:30 PM")
	if !r.app.EndOfDayOn() || r.app.EndOfDayAt() != "18:30" || !r.app.EndOfDayEnabled() {
		t.Errorf("on %v at %q enabled %v: on at 18:30 until the user says otherwise", r.app.EndOfDayOn(), r.app.EndOfDayAt(), r.app.EndOfDayEnabled())
	}
}

func TestTheEndOfDaySwitchKeepsTheChoiceAndTheOtherPrefs(t *testing.T) {
	prefs := &data.MemoryPrefs{}
	r := newSettingsRig(t, func(a *App) { a.Prefs = prefs })
	r.app.SetNotifications(false)
	r.app.SetNotifications(true)
	r.pane(t, "Advanced")
	r.tt.Scroll(340, 300, 0, 3000)
	r.tt.Frame()
	if err := r.tt.Click("Remind me at the end of the day"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.app.EndOfDayOn() || r.app.EndOfDayEnabled() {
		t.Fatal("clicking the switch left the reminder on")
	}
	stored, _ := prefs.LoadPrefs()
	if !stored.EndOfDayOff || stored.NotificationsOff {
		t.Errorf("stored %+v", stored)
	}
	if again := (&App{Prefs: prefs}); again.EndOfDayOn() {
		t.Error("the choice was not kept")
	}
	if err := r.tt.Click("Remind me at the end of the day"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if !r.app.EndOfDayOn() {
		t.Error("the switch does not turn it on again")
	}
}

func TestTheReminderIsOffWithTheNotificationsSwitchToo(t *testing.T) {
	r := newSettingsRig(t)
	r.app.SetNotifications(false)
	if !r.app.EndOfDayOn() || r.app.EndOfDayEnabled() {
		t.Errorf("on %v enabled %v: the reminder's own switch stays, the notifications' off wins", r.app.EndOfDayOn(), r.app.EndOfDayEnabled())
	}
}

func TestThePickedTimeIsKeptAndAnUnreadableOneIsTheDefault(t *testing.T) {
	prefs := &data.MemoryPrefs{}
	app, _, _ := newTestApp(t)
	app.Prefs = prefs
	app.SetEndOfDayAt("17:30")
	if app.EndOfDayAt() != "17:30" {
		t.Errorf("at %q", app.EndOfDayAt())
	}
	app.SetEndOfDayAt("half past five") // not a time: left alone
	if app.EndOfDayAt() != "17:30" {
		t.Errorf("at %q after an unreadable time", app.EndOfDayAt())
	}
	if stored, _ := prefs.LoadPrefs(); stored.EndOfDayAt != "17:30" {
		t.Errorf("stored %+v", stored)
	}
	app.SetEndOfDayLast("2026-10-14")
	if app.EndOfDayLast() != "2026-10-14" || app.EndOfDayAt() != "17:30" {
		t.Error("keeping the last day lost another pref")
	}
	broken := &App{Prefs: &data.MemoryPrefs{}}
	_ = broken.Prefs.SavePrefs(data.Prefs{EndOfDayAt: "soon"})
	if broken.EndOfDayAt() != "18:30" {
		t.Errorf("an unreadable stored time gave %q", broken.EndOfDayAt())
	}
}

func TestThePopUpOffersTheEveningInHalfHoursAndTheTimeAlreadyPicked(t *testing.T) {
	r := newSettingsRig(t)
	opts := r.s.endOfDayTimes()
	if len(opts) != 11 || opts[0].value != "16:00" || opts[len(opts)-1].value != "21:00" || opts[5].value != "18:30" {
		t.Errorf("%+v", opts)
	}
	r.app.SetEndOfDayAt("22:15")
	opts = r.s.endOfDayTimes()
	if len(opts) != 12 || opts[11].value != "22:15" {
		t.Errorf("a time outside the list is added: %+v", opts)
	}
}

func TestTheEndOfDayRowsInChinese(t *testing.T) {
	r := newSettingsRig(t)
	r.tt.SetLocale("zh-Hans")
	r.tt.Frame()
	r.pane(t, "高级")
	r.tt.Scroll(340, 300, 0, 3000)
	r.tt.Frame()
	has(t, r.tt, "收工提醒", "工作日到点提醒看日报，周五提醒看周报", "提醒时间")
}
