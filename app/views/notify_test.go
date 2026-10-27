package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

func TestGeneralPaneHasTheNotificationsSwitchOnByDefault(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "Notifications", "System notifications",
		"Agents done or stuck, failing or merged PRs, and reminders")
	hasNot(t, r.tt, "macOS does not allow Withdo to notify", "Open System Settings")
	if !r.app.Notifications() {
		t.Error("notifications should be on until the user switches them off")
	}
}

func TestTheSwitchTurnsNotificationsOffAndOnAndKeepsTheChoice(t *testing.T) {
	prefs := &data.MemoryPrefs{}
	r := newSettingsRig(t, func(a *App) { a.Prefs = prefs })
	if err := r.tt.Click("Allow system notifications"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.app.Notifications() {
		t.Fatal("clicking the switch left the notifications on")
	}
	if stored, _ := prefs.LoadPrefs(); !stored.NotificationsOff {
		t.Errorf("stored %+v, want the switch kept off", stored)
	}
	// The next run reads it back.
	again := &App{Prefs: prefs}
	if again.Notifications() {
		t.Error("the choice was not kept")
	}
	if err := r.tt.Click("Allow system notifications"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if !r.app.Notifications() {
		t.Error("clicking the switch again left the notifications off")
	}
}

func TestSwitchingNotificationsKeepsTheOtherPrefs(t *testing.T) {
	prefs := &data.MemoryPrefs{}
	app, _, _ := newTestApp(t)
	app.Prefs = prefs
	app.SetOpenWith(OpenTerminal)
	app.SetNotifications(false)
	if stored, _ := prefs.LoadPrefs(); stored.OpenWith != "terminal" || !stored.NotificationsOff {
		t.Errorf("stored %+v", stored)
	}
}

func TestWhenMacOSRefusedTheLineWarnsAndALinkOpensSystemSettings(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.Act.(*recorder).denied = true })
	has(t, r.tt, "System notifications", "macOS does not allow Withdo to notify", "Open System Settings")
	hasNot(t, r.tt, "Agents done or stuck, failing or merged PRs, and reminders")
	if err := r.tt.Click("Open System Settings"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"x-apple.systempreferences:com.apple.Notifications-Settings.extension"}; !slices.Equal(r.rec.urls, want) {
		t.Errorf("opened %v, want %v", r.rec.urls, want)
	}
}

func TestMainSaysWhetherItShowsAllOfMine(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.Frame()
	if !m.ShowingMine() {
		t.Error("the window opens on Mine")
	}
	click(t, tt, "Agents")
	tt.Frame()
	if m.ShowingMine() {
		t.Error("the Agents page is not Mine")
	}
	m.ShowMine()
	tt.Frame()
	if !m.ShowingMine() {
		t.Error("ShowMine did not go back")
	}
	m.catFilter = workID
	tt.Frame()
	if m.ShowingMine() {
		t.Error("a category filter hides the other rows")
	}
}
