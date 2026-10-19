package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

// sidebarHas reports whether the sidebar's rows over Inbox say text; the page beside it may say it too.
func sidebarHas(tt interface{ Texts() []string }, text string) bool {
	for _, got := range tt.Texts() {
		if got == "Inbox" {
			return false
		}
		if got == text {
			return true
		}
	}
	return false
}

// Folding: the sub-rows of Mine and Agents are folded until the user unfolds them with the chevron.

func TestSubRowsAreFoldedByDefault(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.SetMineOpen(false)
	app.SetAgentsOpen(false)
	tt, _, _ := newTesterFor(app, 1100)
	for _, gone := range []string{"Work", "Life", "Unsorted", "Claude Code"} {
		if sidebarHas(tt, gone) {
			t.Errorf("%q is drawn while its section is folded: %q", gone, tt.Texts())
		}
	}
	if !tt.HasText("Mine") || !tt.HasText("Agents") {
		t.Errorf("the parents are gone: %q", tt.Texts())
	}
}

func TestTheChevronUnfoldsASectionAndKeepsTheChoice(t *testing.T) {
	prefs := &data.MemoryPrefs{}
	app, _, _ := newTestApp(t)
	app.Prefs = prefs
	app.SetMineOpen(false)
	app.SetAgentsOpen(false)
	tt, m, _ := newTesterFor(app, 1100)
	before := m.nav
	if err := tt.Click("Expand Mine"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.nav != before {
		t.Errorf("the chevron also opened a page: %+v", m.nav)
	}
	if !sidebarHas(tt, "Work") || !sidebarHas(tt, "Life") || sidebarHas(tt, "Claude Code") {
		t.Errorf("only Mine should be unfolded: %q", tt.Texts())
	}
	if stored, _ := prefs.LoadPrefs(); !stored.MineOpen || stored.AgentsOpen {
		t.Errorf("stored %+v", stored)
	}
	if again := (&App{Prefs: prefs}); !again.MineOpen() || again.AgentsOpen() {
		t.Error("the choice was not kept")
	}
	if err := tt.Click("Collapse Mine"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if sidebarHas(tt, "Work") {
		t.Errorf("Work stays after the section was folded: %q", tt.Texts())
	}
}

func TestClickingTheRowStillOpensThePageWhileFolded(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.SetAgentsOpen(false)
	tt, m, _ := newTesterFor(app, 1100)
	if err := tt.Click("Agents"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenAgents {
		t.Errorf("nav %+v", m.nav)
	}
	if app.AgentsOpen() {
		t.Error("opening the page unfolded the section")
	}
}

func TestTheOpenSubRowStaysWhileItsSectionIsFolded(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.SetMineOpen(false)
	app.SetAgentsOpen(false)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenMine, cat: workID})
	tt.Frame()
	if !sidebarHas(tt, "Work") || sidebarHas(tt, "Life") {
		t.Errorf("only the open page's row should stay: %q", tt.Texts())
	}
	m.goTo(nav{screen: screenAgents, sub: "Claude Code"})
	tt.Frame()
	if !sidebarHas(tt, "Claude Code") || sidebarHas(tt, "Work") {
		t.Errorf("only the open page's row should stay: %q", tt.Texts())
	}
}
