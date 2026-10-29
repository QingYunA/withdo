package views

import (
	"testing"
	"time"
)

func TestGeneralPaneHasTheAutomaticUpdatesSwitchOnByDefault(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "Software updates", "Withdo 0.2.0 · not checked yet", "Check now", "Automatically check for updates")
	if !r.rec.AutomaticUpdates() {
		t.Error("updates are not checked until the user says so")
	}
}

// The quiet line says when the updater last checked, and the link asks it to check now.
func TestTheUpdatesRowShowsTheLastCheckAndChecksNow(t *testing.T) {
	r := newSettingsRig(t)
	r.rec.lastCheck = fixedNow().Add(-2 * time.Minute)
	r.tt.Frame()
	has(t, r.tt, "Withdo 0.2.0 · checked 2 minutes ago")
	if err := r.tt.Click("Check now"); err != nil {
		t.Fatal(err)
	}
	if r.rec.checks != 1 {
		t.Errorf("Check now asked the updater %d times", r.rec.checks)
	}
}

func TestTheAutomaticUpdatesSwitchGoesToTheUpdater(t *testing.T) {
	r := newSettingsRig(t)
	if err := r.tt.Click("Automatically check for updates"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.rec.AutomaticUpdates() {
		t.Fatal("clicking the switch left the checks on")
	}
	if err := r.tt.Click("Automatically check for updates"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if !r.rec.AutomaticUpdates() {
		t.Error("the switch does not turn the checks on again")
	}
}

func TestABuildWithoutUpdatesHasNoUpdatesRow(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.Act.(*recorder).noUpdates = true })
	if r.tt.HasText("Software updates") || r.tt.HasText("Automatically check for updates") {
		t.Errorf("a build that cannot update offers to: %q", r.tt.Texts())
	}
}
