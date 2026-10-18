package views

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSelectButtonTurnsOnSelectModeWhereAClickTicksARow(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Select"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"0 selected", "Select all", "Done"} {
		if !tt.HasText(want) {
			t.Errorf("select mode lacks %q in %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Book dentist"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.openID != 0 {
		t.Error("a click in select mode opened the inspector")
	}
	wantPicked(t, m, "Book dentist")
	if !tt.HasText("1 selected") {
		t.Errorf("the count did not follow: %q", tt.Texts())
	}
	if err := tt.Click("Select all"); err != nil {
		t.Fatal(err)
	}
	if m.multi.count() != len(m.rowIDs()) {
		t.Errorf("Select all ticked %d of %d rows", m.multi.count(), len(m.rowIDs()))
	}
	if err := tt.Click("Done"); err != nil {
		t.Fatal(err)
	}
	if m.multi.mode || m.multi.count() != 0 {
		t.Error("Done did not leave select mode and clear the selection")
	}
}

func TestEscLeavesSelectMode(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Select"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if m.multi.mode {
		t.Error("Esc left the page in select mode")
	}
}

func TestSpaceTicksTheKeyboardRowInSelectMode(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Select"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeySpace)
	if m.multi.count() != 1 {
		t.Errorf("Space ticked %d rows", m.multi.count())
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeySpace)
	if m.multi.count() != 2 {
		t.Errorf("the first tick was lost moving on: %d ticked", m.multi.count())
	}
}

func TestCalendarAndSummaryHaveNoSelectButton(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	for _, n := range []nav{{screen: screenCalendar}, {screen: screenSummary}} {
		m.goTo(n)
		tt.Frame()
		if tt.HasText("Select") {
			t.Errorf("screen %v has a Select button", n.screen)
		}
	}
	for _, n := range []nav{{screen: screenAgents}, {screen: screenInbox}, {screen: screenCompleted}, {screen: screenProject, projectID: 1}} {
		m.goTo(n)
		tt.Frame()
		if !tt.HasText("Select") {
			t.Errorf("screen %v has no Select button", n.screen)
		}
	}
}

func TestTheSelectButtonFlashesOnTheFirstCmdClickOfAList(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	now := fixedNow()
	m.app.Now = func() time.Time { return now }
	if err := tt.ClickWith(ui.Cmd, "Book dentist"); err != nil {
		t.Fatal(err)
	}
	if !m.hint.seen[m.pageKey()] || !m.hint.at.Equal(now) {
		t.Fatal("the first ⌘-click did not flash the Select button")
	}
	later := now.Add(time.Minute)
	m.app.Now = func() time.Time { return later }
	if err := tt.ClickWith(ui.Cmd, "Reply to design feedback"); err != nil {
		t.Fatal(err)
	}
	if !m.hint.at.Equal(now) {
		t.Error("the button flashed again on the second ⌘-click")
	}
}
