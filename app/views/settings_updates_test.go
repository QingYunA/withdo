package views

import "testing"

func TestGeneralPaneHasTheAutomaticUpdatesSwitchOnByDefault(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "Software updates", "Checked once a day; you choose when to install", "Automatically check for updates")
	if !r.rec.AutomaticUpdates() {
		t.Error("updates are not checked until the user says so")
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
