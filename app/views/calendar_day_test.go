package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const (
	cardLabel    = "Items of Wed, Oct 21"
	cardInput    = "New task for this day"
	showAll21    = "Show all 5 on Oct 21"
	addTo20      = "Add to Oct 20"
	tuesday20    = "Tue, Oct 20"
	crowdedTitle = "Five"
)

// crowdedApp has five items on Wed Oct 21, more than a cell lists, and a window tall enough that the
// cell says "2 more".
func crowdedApp(t *testing.T) (*App, *data.Real, *ui.Tester, *Main) {
	t.Helper()
	app, f := calendarApp(t)
	for _, title := range []string{"One", "Two", "Three", "Four", crowdedTitle} {
		task, _ := f.CreateTask(data.CreateTaskInput{Title: title, Source: data.SourceUser})
		_, _ = f.UpdateTask(task.ID, data.Patch{DueAt: oct(21, 18, 0)})
	}
	app.Refresh()
	tt, m := calendarTester(t, app, 1100)
	tt.SetSize(1100, 1000)
	tt.Frame()
	return app, f, tt, m
}

// openCrowdedCard clicks "N more" of the 21st.
func openCrowdedCard(t *testing.T, tt *ui.Tester) {
	t.Helper()
	if err := tt.Click(showAll21); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(cardLabel) {
		t.Fatalf("no card in %q", tt.Texts())
	}
}

// doubleClickBottomOf double clicks the bottom of a cell, below its items.
func doubleClickBottomOf(t *testing.T, tt *ui.Tester, day string) {
	t.Helper()
	r, ok := tt.Find(day)
	if !ok {
		t.Fatalf("no %q in %q", day, tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H-6)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H-6)
}

func TestMoreIsALinkThatOpensTheCardWithEveryItemOfTheDay(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	has(t, tt, "2 more")
	hasNot(t, tt, cardLabel, crowdedTitle)
	if err := tt.Click(showAll21); err != nil {
		t.Fatal(err)
	}
	has(t, tt, cardLabel, "Wednesday, Oct 21", "5 items", "One", "Two", "Three", "Four", crowdedTitle, cardInput)
	if d := m.calendarDay(); d.Day() != 21 {
		t.Errorf("the day is %v: the click on \"more\" selects its day, and no other", d)
	}
}

func TestMoreIsLinkStyled(t *testing.T) {
	_, _, tt, _ := crowdedApp(t)
	r, _ := tt.Find(showAll21)
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if tt.Cursor() != ui.CursorPointer {
		t.Errorf("cursor %v over \"more\"", tt.Cursor())
	}
}

func TestTypingInTheCardAndEnterAddsATaskDueThatDayAndKeepsTheCardOpen(t *testing.T) {
	app, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	if err := tt.Click(cardInput); err != nil {
		t.Fatal(err)
	}
	tt.Type("Water the plants")
	tt.Key(0, ui.KeyEnter)
	task := taskByTitle(t, app, "Water the plants")
	if want := data.EndOfDay(time.Date(2026, 10, 21, 0, 0, 0, 0, time.Local)); task.DueAt == nil || !task.DueAt.Equal(want) {
		t.Errorf("due %v, want %v", task.DueAt, want)
	}
	if !m.cal.card.isOpen() || m.cal.card.buf != "" {
		t.Errorf("card open %v, input %q: it stays open and the input is cleared", m.cal.card.isOpen(), m.cal.card.buf)
	}
	has(t, tt, "Water the plants", "6 items")
}

func TestEnterOnNothingTypedAddsNothing(t *testing.T) {
	app, _, tt, _ := crowdedApp(t)
	openCrowdedCard(t, tt)
	if err := tt.Click(cardInput); err != nil {
		t.Fatal(err)
	}
	before := len(app.snap.tasks)
	tt.Type("   ")
	tt.Key(0, ui.KeyEnter)
	if got := len(app.snap.tasks); got != before {
		t.Errorf("%d tasks, were %d", got, before)
	}
}

func TestEscClosesTheCardWithTheInputFocusedOrNot(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	tt.Key(0, ui.KeyEscape)
	if m.cal.card.isOpen() || tt.HasText(cardLabel) {
		t.Error("Esc did not close the card")
	}
	openCrowdedCard(t, tt)
	if err := tt.Click(cardInput); err != nil {
		t.Fatal(err)
	}
	tt.Type("half")
	tt.Key(0, ui.KeyEscape)
	if m.cal.card.isOpen() {
		t.Error("Esc in the input did not close the card")
	}
	openCrowdedCard(t, tt)
	if m.cal.card.buf != "" {
		t.Errorf("what was typed in the closed card came back: %q", m.cal.card.buf)
	}
}

func TestAClickOutsideTheCardClosesItAndDoesNothingElse(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	r, _ := tt.Find(tuesday20)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H-6)
	if m.cal.card.isOpen() {
		t.Error("the card stays after a click outside")
	}
	if d := m.calendarDay(); d.Day() != 21 {
		t.Errorf("the click outside also selected %v", d)
	}
}

func TestTheCloseButtonClosesTheCard(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	if err := tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	if m.cal.card.isOpen() {
		t.Error("the × did not close the card")
	}
}

func TestClickingAnItemOfTheCardOpensItAndClosesTheCard(t *testing.T) {
	app, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	if err := tt.Click(crowdedTitle); err != nil {
		t.Fatal(err)
	}
	if want := taskByTitle(t, app, crowdedTitle).ID; m.openID != want {
		t.Errorf("open %d, want %d", m.openID, want)
	}
	if m.cal.card.isOpen() {
		t.Error("the card stays over the inspector")
	}
}

func TestADoubleClickOnACellOpensTheCardWithTheInputFocused(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	doubleClickBottomOf(t, tt, tuesday20)
	if !m.cal.card.isOpen() || m.cal.card.day.Day() != 20 {
		t.Fatalf("card %+v", m.cal.card)
	}
	if d := m.calendarDay(); d.Day() != 20 {
		t.Errorf("the double click leaves %v selected", d)
	}
	tt.Type("hello")
	if m.cal.card.buf != "hello" {
		t.Errorf("typed %q: the input has no focus", m.cal.card.buf)
	}
}

func TestASingleClickOnACellOnlySelectsItsDay(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	r, _ := tt.Find(tuesday20)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H-6)
	if m.cal.card.isOpen() || m.calendarDay().Day() != 20 {
		t.Errorf("card open %v, day %v", m.cal.card.isOpen(), m.calendarDay())
	}
}

func TestThePlusOfAHoveredCellOpensTheCardWithTheInputFocused(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	hasNot(t, tt, addTo20)
	r, _ := tt.Find(tuesday20)
	tt.Move(r.X+r.W/2, r.Y+r.H-6)
	if err := tt.Click(addTo20); err != nil {
		t.Fatalf("no + in the hovered cell: %v", err)
	}
	if !m.cal.card.isOpen() || m.cal.card.day.Day() != 20 {
		t.Fatalf("card %+v", m.cal.card)
	}
	tt.Type("x")
	if m.cal.card.buf != "x" {
		t.Errorf("typed %q: the input has no focus", m.cal.card.buf)
	}
}

func TestThePlusDoesNotMoveTheCell(t *testing.T) {
	_, _, tt, _ := crowdedApp(t)
	before, _ := tt.Find("Review PR 12")
	r, _ := tt.Find("Fri, Oct 16")
	tt.Move(r.X+r.W/2, r.Y+r.H-6)
	if !tt.HasText("Add to Oct 16") {
		t.Fatal("no + in the hovered cell")
	}
	if after, _ := tt.Find("Review PR 12"); after != before {
		t.Errorf("the item moved from %v to %v", before, after)
	}
}

func TestChangingTheMonthClosesTheCard(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	m.shiftMonth(1)
	tt.Frame()
	if m.cal.card.isOpen() || tt.HasText(cardLabel) {
		t.Error("the card stays over another month")
	}
	openCrowdedCard2 := func() {
		m.shiftMonth(-1)
		tt.Frame()
		openCrowdedCard(t, tt)
	}
	openCrowdedCard2()
	m.calendarToday()
	tt.Frame()
	if m.cal.card.isOpen() {
		t.Error("Today leaves the card open")
	}
}

func TestLeavingTheCalendarClosesTheCard(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	m.goTo(nav{})
	tt.Frame()
	if m.cal.card.isOpen() {
		t.Error("the card stays on another page")
	}
}

func TestTheArrowsDoNotMoveTheDayWhileTheCardIsOpen(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	doubleClickBottomOf(t, tt, tuesday20)
	for _, k := range []ui.Key{ui.KeyRight, ui.KeyDown, ui.KeyLeft} {
		tt.Key(0, k)
	}
	if d := m.calendarDay(); d.Day() != 20 {
		t.Errorf("the arrows moved the day to %v", d)
	}
	if !m.cal.card.isOpen() {
		t.Error("an arrow closed the card")
	}
}

func TestWithReduceMotionTheCardIsFullSizeAtOnce(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	openCrowdedCard(t, tt)
	panel, _ := tt.Find(cardLabel)
	if panel.W < theme.DayCardMinWidth || panel.H != 242 || m.cal.card.asked != 0 {
		t.Errorf("card %v, %d frames asked: want full size, 242 pt tall, and none", panel, m.cal.card.asked)
	}
}

func TestTheCardPopsOutFromItsCornerAndThenAsksForNoFrames(t *testing.T) {
	app, _, tt, m := crowdedApp(t)
	now := fixedNow()
	app.Now = func() time.Time { return now }
	tt.SetPreferences(ui.Preferences{})
	openCrowdedCard(t, tt)
	cell := m.cal.card.anchor
	// Find reports the box as drawn, so the transform shows in it: the layout stays full size.
	var widths []float32
	asked := 0
	for _, at := range []time.Duration{0, 30, 60, 110, 139, 140, 220} {
		now = fixedNow().Add(at * time.Millisecond)
		tt.Frame()
		r, _ := tt.Find(cardLabel)
		widths = append(widths, r.W)
		if r.X != cell.X || r.Y != cell.Y {
			t.Errorf("at %d ms the card's corner is %v,%v, not the cell's %v,%v", at, r.X, r.Y, cell.X, cell.Y)
		}
		if at < 140 && m.cal.card.asked <= asked {
			t.Errorf("no frame was asked for at %d ms, while the card opens", at)
		}
		asked = m.cal.card.asked
	}
	full := widths[len(widths)-1]
	if first := widths[0] / full; first < 0.5-0.01 || first > 0.95 {
		t.Errorf("the card starts at %.2f of its width, want 0.5 to 0.95", first)
	}
	for i := 1; i < len(widths); i++ {
		if widths[i] < widths[i-1] {
			t.Errorf("the card shrinks while it opens: %v", widths)
		}
	}
	if widths[5] != full {
		t.Errorf("the card is not full size at 140 ms: %v", widths)
	}
	now = now.Add(time.Second)
	tt.Frame()
	tt.Frame()
	if m.cal.card.asked != asked {
		t.Errorf("frames are still asked for once the card is done: %d, were %d", m.cal.card.asked, asked)
	}
}

func TestNothingAsksForFramesWhileTheCardIsClosed(t *testing.T) {
	_, _, tt, m := crowdedApp(t)
	tt.SetPreferences(ui.Preferences{})
	tt.Frame()
	if m.cal.card.asked != 0 {
		t.Errorf("%d frames asked with no card", m.cal.card.asked)
	}
}

func TestTheCardGrowsLeftAndUpNearTheEdgesOfTheWindow(t *testing.T) {
	anchor := ui.Rect{X: 900, Y: 600, W: 150, H: 90}
	g := dayCardPlace(anchor, 1100, 720, 150, 5)
	if !g.left || !g.up {
		t.Fatalf("box %+v: it should grow left and up", g)
	}
	if g.x < 8 || g.y < 8 || g.x+g.w > 1100-8 || g.y+g.h > 720-8 {
		t.Errorf("box %+v is outside the window", g)
	}
	if g.x+g.w != anchor.X+anchor.W || g.y+g.h != anchor.Y+anchor.H {
		t.Errorf("box %+v: its bottom right is not the cell's", g)
	}
	free := dayCardPlace(ui.Rect{X: 100, Y: 100, W: 150, H: 90}, 1100, 720, 150, 5)
	if free.left || free.up || free.x != 100 || free.y != 100 {
		t.Errorf("box %+v: it should start on the cell's top left", free)
	}
}

// A double click on an element reports a Clicked for each of its two clicks and, with the second, a
// DoubleClicked: the cell selects its day on the first and opens the card on the second. An inner
// element that asks for clicks takes them alone.
func TestADoubleClickIsTwoClicksAndOneDoubleClick(t *testing.T) {
	var outer, inner, double, innerDouble int
	tt := newTester(func(c *ui.Context) {
		box := ui.Column(c.Key("outer")).Size(200, 100).Padding(60, 0, 0)
		box.Children(func() {
			in := ui.ButtonBase(c.Key("inner")).Size(100, 30).Label("inner")
			if in.Clicked() {
				inner++
			}
			if in.DoubleClicked() {
				innerDouble++
			}
		})
		if box.Clicked() {
			outer++
		}
		if box.DoubleClicked() {
			double++
		}
	}, 300, 200)
	tt.ClickAt(10, 10)
	tt.ClickAt(10, 10)
	if outer != 2 || double != 1 {
		t.Errorf("outer: %d clicks and %d double clicks, want 2 and 1", outer, double)
	}
	outer, double = 0, 0
	_ = tt.Click("inner")
	_ = tt.Click("inner")
	if inner != 2 || innerDouble != 1 || outer != 0 || double != 0 {
		t.Errorf("inner %d/%d, outer %d/%d: the inner element takes its clicks alone", inner, innerDouble, outer, double)
	}
}

func TestTheCardIsSixTenthsWiderThanACellButNotNarrowerThan280(t *testing.T) {
	if g := dayCardPlace(ui.Rect{X: 10, Y: 10}, 1500, 900, 200, 1); g.w != 320 {
		t.Errorf("width %v for a 200 pt cell, want 320", g.w)
	}
	if g := dayCardPlace(ui.Rect{X: 10, Y: 10}, 1500, 900, 100, 1); g.w != 280 {
		t.Errorf("width %v for a 100 pt cell, want 280", g.w)
	}
	if g := dayCardPlace(ui.Rect{X: 10, Y: 10}, 1500, 900, 150, 40); g.h != theme.DayCardMaxH {
		t.Errorf("height %v for forty items, want %v: the list scrolls", g.h, theme.DayCardMaxH)
	}
}
