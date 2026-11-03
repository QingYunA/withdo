package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The official model (M15, v44): Settings > Model's card, the composer's chip and the quiet lines
// where it does not answer.

var officialReady = data.OfficialInfo{Phase: data.OfficialReady, Left: 86, Limit: 100, ResetsAt: fixedNow().Add(14 * time.Hour)}

func TestTheOfficialCardIsTheFirstCardWhileNoProviderOfOneIsInUse(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.Info = officialReady
	r.assistant(t)
	has(t, r.tt, "Withdo model", "Included with your account", "Available", "· 86 left today", "Test", "Use your own model")
	hasNot(t, r.tt, "Switch to", "Use this", "Not set up", "Sign in to use it")
	if err := r.tt.Click("Test"); err != nil {
		t.Fatal(err)
	}
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Connected · 120 ms")
}

func TestTheOfficialCardSaysWhenTheDayIsUsedUp(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.Info = data.OfficialInfo{Phase: data.OfficialUsedUp, Left: 0, Limit: 100, ResetsAt: fixedNow().Add(14 * time.Hour)}
	r.assistant(t)
	has(t, r.tt, "Withdo model", "Used up for today · Back tomorrow", "Test")
	hasNot(t, r.tt, "Available", "left today")
}

func TestSignedOutTheOfficialCardOffersSignIn(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.Info = data.OfficialInfo{Phase: data.OfficialSignedOut, Left: -1}
	r.assistant(t)
	has(t, r.tt, "Withdo model", "Sign in to use it. There is a free allowance every day.", "Use your own model")
	hasNot(t, r.tt, "Test", "Available")
	if err := r.tt.Click("Sign in"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.s.pane != PaneAccount {
		t.Fatalf("Sign in opened pane %v, want Account", r.s.pane)
	}
}

func TestAProviderInUseMakesTheOfficialModelAOneRowGroupWithUseThis(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "deepseek")
	r.fm.Info = officialReady
	r.assistant(t)
	has(t, r.tt, "DeepSeek", "Withdo model", "Use this", "Use your own model")
	hasNot(t, r.tt, "Switch to", "Available") // the row is the quiet one: no big card, and no test of it
	if err := r.tt.Click("Use this"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if active := currentProviderOf(r.app); active.ID != "" {
		t.Fatalf("Use this left %q in use", active.ID)
	}
	has(t, r.tt, "Withdo model", "Test")
	hasNot(t, r.tt, "Use this")
	if !r.fm.KeySaved("deepseek") {
		t.Fatal("Use this forgot DeepSeek's key")
	}
}

func TestWithoutTheOfficialModelThePaneIsAsItWas(t *testing.T) {
	r := newSettingsRig(t) // a server that does not offer it
	r.assistant(t)
	has(t, r.tt, "Not set up: dates and reminders are still read on this computer.", "Switch to")
	hasNot(t, r.tt, "Withdo model", "Use this", "Test")
}

func TestTheChipNamesTheOfficialModelAndItsPopoverSwitchesBack(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	fm := r.app.Models.(*data.FakeModels)
	fm.Info = officialReady
	mustNil(t, fm.Save("deepseek", data.ProviderEdit{Model: "deepseek-chat", Key: "sk-x"}))
	r.onProject() // Home has no chip
	has(t, r.tt, "Withdo")
	hasNot(t, r.tt, "DeepSeek · deepseek-chat")
	// The popover lists the official model, checked, and the provider that is set up.
	if err := r.tt.Click(r.m.say.ComposerModelLabel("Withdo")); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	has(t, r.tt, "Withdo", "DeepSeek", "deepseek-chat")
	if err := r.tt.Click(r.m.say.ComposerModelLabel("DeepSeek · deepseek-chat")); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	has(t, r.tt, "DeepSeek · deepseek-chat")
	// And the other way: Withdo again.
	if err := r.tt.Click(r.m.say.ComposerModelLabel("DeepSeek · deepseek-chat")); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if err := r.tt.Click(r.m.say.ComposerModelLabel("Withdo")); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if fm.Official().InUse != true {
		t.Fatal("choosing Withdo in the chip did not make the official model the one in use")
	}
}

func TestSignedOutTheQuietLineOffersSignInAndNeverADialog(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.app.Models.(*data.FakeModels).Info = data.OfficialInfo{Phase: data.OfficialSignedOut, Left: -1}
	r.tt.Frame()
	hasNot(t, r.tt, "Signed out, the input only reads dates on this computer.")
	r.tt.Key(ui.Cmd, ui.KeyK)
	has(t, r.tt, "Signed out, the input only reads dates on this computer.", "Sign in")
	hasNot(t, r.tt, "Set up the assistant")
	link, _ := r.tt.Find("Sign in")
	lx, ly := center(link)
	r.tt.Move(lx, ly)
	if err := r.tt.Click("Sign in"); err != nil {
		t.Fatal(err)
	}
	if len(r.rec.panes) != 1 || r.rec.panes[0] != PaneAccount {
		t.Errorf("asked to show %v, want the Account pane", r.rec.panes)
	}
}

func TestUsedUpTheQuietLineSaysSoWithoutALink(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.app.Models.(*data.FakeModels).Info = data.OfficialInfo{Phase: data.OfficialUsedUp, Left: 0, Limit: 100}
	r.tt.Key(ui.Cmd, ui.KeyK)
	has(t, r.tt, "Today's allowance of the Withdo model is used up: dates and reminders are read on this computer. Back tomorrow.")
	hasNot(t, r.tt, "Sign in", "Set up the assistant")
}
