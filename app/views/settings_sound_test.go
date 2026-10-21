package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

func TestGeneralPaneHasTheSoundEffectsSwitchOnByDefault(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Advanced")
	r.tt.Scroll(340, 300, 0, 3000)
	r.tt.Frame()
	has(t, r.tt, "Sound effects", "Quiet sounds while the day is shipped in Summary")
	if !r.app.SoundEffects() {
		t.Error("the sounds are off until the user says so")
	}
}

func TestTheSoundEffectsSwitchKeepsTheChoiceAndTheOtherPrefs(t *testing.T) {
	prefs := &data.MemoryPrefs{}
	r := newSettingsRig(t, func(a *App) { a.Prefs = prefs })
	r.app.SetEndOfDay(false)
	r.pane(t, "Advanced")
	r.tt.Scroll(340, 300, 0, 3000)
	r.tt.Frame()
	if err := r.tt.Click("Play sound effects"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.app.SoundEffects() {
		t.Fatal("clicking the switch left the sounds on")
	}
	stored, _ := prefs.LoadPrefs()
	if !stored.SoundOff || !stored.EndOfDayOff {
		t.Errorf("stored %+v", stored)
	}
	if again := (&App{Prefs: prefs}); again.SoundEffects() {
		t.Error("the choice was not kept")
	}
	if err := r.tt.Click("Play sound effects"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if !r.app.SoundEffects() {
		t.Error("the switch does not turn the sounds on again")
	}
}
