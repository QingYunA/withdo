package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// moving lets the tester play animations, which it does not by default (see still).
func moving(tt *ui.Tester) { tt.SetPreferences(ui.Preferences{}) }

// at moves the app clock to d after the fixed now and draws the frame, twice: a fold measures the
// row it closes in the frame before.
func at(tt *ui.Tester, app *App, d time.Duration) {
	app.Now = func() time.Time { return fixedNow().Add(d) }
	tt.Frame()
	tt.Frame()
}

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// yOf is where the text is drawn from the top of the window.
func yOf(t *testing.T, tt *ui.Tester, text string) float32 {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q in %q", text, tt.Texts())
	}
	return r.Y
}

func near(a, b float32) bool { return a-b < 1.01 && b-a < 1.01 }

// phaseAt is the phase of task at d after the click, as a row that shows it as it was draws it.
func phaseAt(app *App, task data.Task, d time.Duration, reduce bool) phase {
	app.Now = func() time.Time { return fixedNow().Add(d) }
	var ph phase
	tt := ui.NewTester(func(c *ui.Context) { ph = app.completionPhase(c, task) }, 10, 10)
	tt.SetPreferences(ui.Preferences{ReduceMotion: reduce})
	tt.Frame()
	return ph
}

func TestAcceptStampsInSevenFramesHoldsThenFolds(t *testing.T) {
	app, _, _ := newTestApp(t)
	was, _ := app.Task(2)
	app.Accept(was)
	if got, _ := app.Task(2); got.Status != data.Done {
		t.Fatalf("status %v: the store says done at once", got.Status)
	}
	for _, c := range []struct {
		ms, stamp int
		whole     bool
	}{{0, 0, true}, {17, 1, true}, {50, 2, true}, {119, 6, true}, {190, 6, true}} {
		ph := phaseAt(app, was, ms(c.ms), false)
		if ph.stamp != c.stamp || (ph.height == 1 && ph.opacity == 1) != c.whole || !ph.moving {
			t.Errorf("at %d ms: stamp %d, height %v, opacity %v; want stamp %d, whole %v", c.ms, ph.stamp, ph.height, ph.opacity, c.stamp, c.whole)
		}
	}
	if ph := phaseAt(app, was, ms(199), false); ph.height != 1 || ph.opacity != 1 {
		t.Errorf("the fold begins at 199 ms: %v, %v", ph.height, ph.opacity)
	}
	if ph := phaseAt(app, was, ms(309), false); ph.height < 0.12 || ph.height > 0.13 || ph.opacity != 0.0 && ph.opacity > 0.1 {
		t.Errorf("110 ms into the fold: height %v (ease-out cubic: 1/8), opacity %v", ph.height, ph.opacity)
	}
	if ph := phaseAt(app, was, ms(419), false); ph.moving || ph.height != 1 {
		t.Errorf("at 420 ms the animation is over: %+v", ph)
	}
	if ph := phaseAt(app, was, ms(10), true); ph.stamp != theme.StampFrames-1 {
		t.Errorf("Reduce Motion: stamp frame %d at once, want the last", ph.stamp)
	}
}

func TestAcceptedRowStaysThroughTheStampThenFoldsAwaySmoothly(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	moving(tt)
	y0, done0 := yOf(t, tt, "Today"), yOf(t, tt, "Port the MCP server")
	if err := tt.Click("Accept: Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 0 {
		t.Error("Accept opened the inspector")
	}
	for _, d := range []int{0, 17, 50, 119, 150, 198} {
		at(tt, app, ms(d))
		if !tt.HasText("Native UI with mygo") || yOf(t, tt, "Today") != y0 || yOf(t, tt, "Port the MCP server") != done0 {
			t.Errorf("at %d ms the row should hold still: Today %v, was %v", d, yOf(t, tt, "Today"), y0)
		}
	}
	prev := y0
	for _, d := range []int{230, 260, 300, 340, 380, 410} {
		at(tt, app, ms(d))
		if y := yOf(t, tt, "Today"); y >= prev {
			t.Errorf("at %d ms Today is at %v, not above %v: the rows below slide up as the row folds", d, y, prev)
		} else {
			prev = y
		}
	}
	at(tt, app, ms(419))
	last := yOf(t, tt, "Today")
	at(tt, app, ms(440))
	if got := yOf(t, tt, "Today"); !near(got, last) {
		t.Errorf("Today jumps %v pt when the animation ends", got-last)
	}
	if !above(t, tt, "Done today", "Native UI with mygo") || tt.HasText("PR #12 is ready for your review · adhd-todo") {
		t.Errorf("the task did not end up in Done today: %q", tt.Texts())
	}
}

func TestTheDoneRowOpensAtTheTopOfDoneTodayWhileTheOtherFolds(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	moving(tt)
	if err := tt.Click("Accept: Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	gap := func() float32 { return yOf(t, tt, "Port the MCP server") - yOf(t, tt, "Done today") }
	at(tt, app, ms(198))
	before := gap()
	prev := before
	for _, d := range []int{230, 270, 320, 380, 419} {
		at(tt, app, ms(d))
		if g := gap(); g < prev || g > before+40 {
			t.Errorf("at %d ms the opening row has made room for %v pt, after %v", d, g-before, prev-before)
		} else {
			prev = g
		}
	}
	at(tt, app, ms(430))
	if g := gap(); g-before < 35 || g-before > 45 {
		t.Errorf("the Done row ends %v pt tall, want about one row", g-before)
	}
}

func TestAcceptWithReducedMotionFadesThenIsGoneWithNoFoldOrOpening(t *testing.T) {
	tt, _, app := newMainTester(t, 1100) // Reduce Motion
	y0, gap0 := yOf(t, tt, "Today"), yOf(t, tt, "Port the MCP server")-yOf(t, tt, "Done today")
	if err := tt.Click("Accept: Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	for _, d := range []int{0, 100, 250, 300} {
		at(tt, app, ms(d))
		if yOf(t, tt, "Today") != y0 {
			t.Errorf("at %d ms Today moved (%v, was %v): the row only fades", d, yOf(t, tt, "Today"), y0)
		}
	}
	at(tt, app, ms(199))
	if g := yOf(t, tt, "Port the MCP server") - yOf(t, tt, "Done today"); g-gap0 < 35 {
		t.Errorf("the Done row should be there at once when the fade begins: %v pt more", g-gap0)
	}
	at(tt, app, ms(199+180))
	if yOf(t, tt, "Today") >= y0 {
		t.Errorf("after the fade the row is gone and Today is at %v, was %v", yOf(t, tt, "Today"), y0)
	}
}

func TestTheSectionFoldsWithItsLastRowAndLeavesNoJumpAtTheEnd(t *testing.T) {
	f := demoStore(t, fixedNow, false)
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "Write the notes"})
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "Tidy the docs", Source: data.SourceAgent, Status: data.Review})
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, _, _ := newTesterFor(app, 1100)
	moving(tt)
	has(t, tt, "From your agents", "Tidy the docs")
	y0 := yOf(t, tt, "Later")
	if err := tt.Click("Accept: Tidy the docs"); err != nil {
		t.Fatal(err)
	}
	at(tt, app, ms(309)) // 110 ms into the fold: an eighth of the section is left
	mid := yOf(t, tt, "Later")
	at(tt, app, ms(415))
	almost := yOf(t, tt, "Later")
	at(tt, app, ms(440))
	if tt.HasText("From your agents") {
		t.Error("the header of an emptied section is still drawn")
	}
	end := yOf(t, tt, "Later")
	if want := end + (y0-end)/8; !near(mid, want) {
		t.Errorf("mid-fold Later is at %v, want %v: the header and the box close with the row (from %v to %v)", mid, want, y0, end)
	}
	if !near(almost, end) {
		t.Errorf("the page jumps %v pt when the section is dropped: its header and box did not fold", end-almost)
	}
}

func TestAHairlineFoldsWithItsRowAndNoneStandsAtTheTopOfTheBox(t *testing.T) {
	phs := []phase{
		{moving: true, height: 0.5}, {moving: true, height: 0.25}, {height: 1}, {height: 1},
	}
	got := ruleShares(phs)
	want := []float32{0.5, 0.25, 0.25, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rule %d: %v, want %v (all %v)", i, got[i], want[i], got)
		}
	}
}

func TestOtherCompletionsFoldTheRowAfterTheirGlyphFill(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	moving(tt)
	y0 := yOf(t, tt, "Later")
	click(t, tt, "Mark done: Reply to design feedback")
	for _, d := range []int{100, 220, 290} {
		at(tt, app, ms(d))
		if yOf(t, tt, "Later") != y0 {
			t.Errorf("at %d ms the glyph is still filling and holding: Later at %v, was %v", d, yOf(t, tt, "Later"), y0)
		}
	}
	at(tt, app, ms(400))
	if y := yOf(t, tt, "Later"); y >= y0 {
		t.Errorf("at 400 ms the row should be folding: Later at %v, was %v", y, y0)
	}
	at(tt, app, ms(520))
	end := yOf(t, tt, "Later")
	at(tt, app, ms(560))
	if got := yOf(t, tt, "Later"); !near(got, end) {
		t.Errorf("the page jumps %v pt when the fold ends", got-end)
	}
}

func TestAcceptingTwiceDoesNotRestartTheStamp(t *testing.T) {
	app, _, _ := newTestApp(t)
	was, _ := app.Task(2)
	app.Accept(was)
	app.Now = func() time.Time { return fixedNow().Add(100 * time.Millisecond) }
	app.Accept(was)
	if got := app.completing[2].at; !got.Equal(fixedNow()) {
		t.Errorf("the stamp restarted at %v", got)
	}
}
