package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// hopRig is the window over the seeded workspace with a clock the test moves and a Seen store it
// can look into. The first frame is on Mine, which marks what was handed over as seen.
type hopRig struct {
	tt    *ui.Tester
	m     *Main
	app   *App
	store *data.Fake
	seen  *data.MemorySeen
	now   time.Time
}

func newHopRig(t *testing.T, reduceMotion bool) *hopRig {
	t.Helper()
	r := &hopRig{now: fixedNow(), seen: &data.MemorySeen{}}
	r.store = data.NewFake(func() time.Time { return r.now })
	r.app = NewApp(r.store, &recorder{}, testConfig())
	r.app.Now = func() time.Time { return r.now }
	r.app.Seen = r.seen
	r.start(reduceMotion)
	return r
}

// start opens the window, as a launch does.
func (r *hopRig) start(reduceMotion bool) {
	r.m = NewMain(r.app)
	r.tt = ui.NewTester(r.m.View, 1100, 720)
	r.tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	r.tt.SetPreferences(ui.Preferences{ReduceMotion: reduceMotion})
}

// handOver makes an agent hand something over: a task of its own in review.
func (r *hopRig) handOver(title string) data.Task {
	task, err := r.store.CreateTask(data.CreateTaskInput{Title: title, Source: data.SourceAgent, Status: data.Review})
	if err != nil {
		panic(err)
	}
	r.app.Refresh()
	return task
}

// leave goes to Agents and draws the frames the hop needs to find its rows and take off.
func (r *hopRig) leave() {
	r.m.goTo(nav{screen: screenAgents})
	r.tt.Frame()
	r.tt.Frame()
}

// after moves the clock and draws a frame.
func (r *hopRig) after(d time.Duration) {
	r.now = r.now.Add(d)
	r.tt.Frame()
}

func TestMineMarksWhatIsHandedOverAsSeenAndPersistsIt(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	seen, _ := r.seen.LoadSeen()
	if len(seen) != 2 || seen[3].IsZero() || seen[2].IsZero() {
		t.Fatalf("seen %v: the blocked task 3 and the review task 2 are handed over and Mine is shown", seen)
	}
	if r.m.hop.shown != 0 || len(r.m.hop.flights) != 0 {
		t.Errorf("badge %d, flights %d: nothing is new while Mine is shown", r.m.hop.shown, len(r.m.hop.flights))
	}
	// A new launch reads the record: leaving Mine at once hops nothing.
	r2 := &hopRig{now: r.now, seen: r.seen, store: r.store}
	r2.app = NewApp(r.store, &recorder{}, testConfig())
	r2.app.Now = func() time.Time { return r2.now }
	r2.app.Seen = r.seen
	r2.start(false)
	r2.leave()
	if len(r2.m.hop.flights) != 0 || r2.m.hop.shown != 0 {
		t.Errorf("after a restart: flights %d, badge %d for rows that were seen", len(r2.m.hop.flights), r2.m.hop.shown)
	}
}

func TestSeenForgetsTasksThatAreGone(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	if err := r.store.DeleteTask(3); err != nil {
		t.Fatal(err)
	}
	r.app.Refresh()
	r.tt.Frame()
	seen, _ := r.seen.LoadSeen()
	if _, ok := seen[3]; ok || len(seen) != 1 {
		t.Errorf("seen %v still remembers the deleted task", seen)
	}
}

func TestANewHandoverHopsFromAgentsToMineAndTheBadgeCountsUpAsItLands(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	task := r.handOver("Port the watcher")
	r.leave()
	if len(r.m.hop.flights) != 1 || r.m.hop.flights[0].id != task.ID {
		t.Fatalf("flights %+v, want one for task %d", r.m.hop.flights, task.ID)
	}
	if r.m.hop.shown != 0 {
		t.Errorf("badge %d before the sprite lands", r.m.hop.shown)
	}
	r.after(theme.HopLanding - time.Millisecond)
	if r.m.hop.shown != 0 {
		t.Errorf("badge %d one millisecond before landing", r.m.hop.shown)
	}
	r.after(2 * time.Millisecond)
	if r.m.hop.shown != 1 || !r.tt.HasText("1 new from your agents") {
		t.Errorf("badge %d after landing; texts %q", r.m.hop.shown, r.tt.Texts())
	}
	r.after(theme.HopDuration)
	if len(r.m.hop.flights) != 0 || r.m.hop.shown != 1 {
		t.Errorf("after the hop: flights %d, badge %d: the badge stays until Mine is shown", len(r.m.hop.flights), r.m.hop.shown)
	}
}

func TestSeveralHandoversHopAtMostThreeAndOneTwentyMillisecondsApart(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	for i := 0; i < 5; i++ {
		r.handOver("Handed over " + itoa(i))
	}
	r.leave()
	fl := r.m.hop.flights
	if len(fl) != theme.HopMax {
		t.Fatalf("%d flights, want %d", len(fl), theme.HopMax)
	}
	for i := 1; i < len(fl); i++ {
		if got := fl[i].start.Sub(fl[i-1].start); got != theme.HopStagger {
			t.Errorf("hop %d takes off %v after the one before, want %v", i, got, theme.HopStagger)
		}
	}
	if r.m.hop.shown != 2 {
		t.Errorf("badge %d: the two that do not hop count at once", r.m.hop.shown)
	}
	r.after(theme.HopLanding + 3*theme.HopStagger)
	if r.m.hop.shown != 5 {
		t.Errorf("badge %d after all landed, want 5", r.m.hop.shown)
	}
}

func TestLookingAtMineEndsTheHopsAndClearsTheBadge(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	r.handOver("Port the watcher")
	r.leave()
	r.after(theme.HopDuration)
	if r.m.hop.shown != 1 {
		t.Fatalf("badge %d", r.m.hop.shown)
	}
	r.m.goTo(nav{})
	r.tt.Frame()
	if r.m.hop.shown != 0 || len(r.m.hop.flights) != 0 || r.tt.HasText("1 new from your agents") {
		t.Errorf("badge %d, flights %d after Mine was shown", r.m.hop.shown, len(r.m.hop.flights))
	}
	r.leave()
	if r.m.hop.shown != 0 || len(r.m.hop.flights) != 0 {
		t.Error("what was seen came back when leaving Mine")
	}
}

func TestSomethingHandedOverAgainIsNewAgain(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	r.now = r.now.Add(time.Hour)
	r.store.ReportAs(3, "claude-code", "Still stuck, now on the keychain", r.now)
	r.app.Refresh()
	r.leave()
	if len(r.m.hop.flights) != 1 || r.m.hop.flights[0].id != 3 {
		t.Errorf("flights %+v, want the blocked task 3 again", r.m.hop.flights)
	}
}

func TestReduceMotionHasNoHopAndTheBadgeFadesIn(t *testing.T) {
	r := newHopRig(t, true)
	r.tt.Frame()
	r.handOver("Port the watcher")
	r.leave()
	if len(r.m.hop.flights) != 0 {
		t.Errorf("flights %d with Reduce Motion", len(r.m.hop.flights))
	}
	if r.m.hop.shown != 1 {
		t.Fatalf("badge %d: with Reduce Motion it counts at once", r.m.hop.shown)
	}
	if !r.m.hop.shownAt.Equal(r.now) {
		t.Errorf("the fade began at %v, want %v", r.m.hop.shownAt, r.now)
	}
}

func TestTheBadgeFadesInOverTheFastMotionTime(t *testing.T) {
	at := fixedNow()
	for _, c := range []struct {
		reduce bool
		in     time.Duration
		want   float32
	}{
		{false, 0, 1}, // with motion the badge is simply there
		{true, -time.Second, 0},
		{true, 0, 0},
		{true, badgeFade / 2, 0.5},
		{true, badgeFade, 1},
		{true, 10 * badgeFade, 1},
	} {
		if got := badgeFadeOpacity(c.reduce, at.Add(c.in), at); got != c.want {
			t.Errorf("reduce %v at %v: opacity %v, want %v", c.reduce, c.in, got, c.want)
		}
	}
}

func TestAStillWindowAsksForNoFrames(t *testing.T) {
	r := newHopRig(t, false)
	for i := 0; i < 5; i++ {
		r.after(time.Second)
	}
	if r.m.hop.asked != 0 {
		t.Errorf("a window with nothing to animate asked for %d frames", r.m.hop.asked)
	}
	r.handOver("Port the watcher")
	r.leave()
	asked := r.m.hop.asked
	if asked == 0 {
		t.Fatal("a hop is playing and no frame was asked for")
	}
	r.after(theme.HopDuration + time.Second) // over: the sprite is gone
	asked = r.m.hop.asked
	for i := 0; i < 5; i++ {
		r.after(time.Second)
	}
	if r.m.hop.asked != asked {
		t.Errorf("frames were asked for after the hop was over: %d more", r.m.hop.asked-asked)
	}
}

func TestTheSpriteIsDrawnOnlyWhileItPlays(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	r.handOver("Port the watcher")
	r.leave()
	settle(r.tt) // the row highlight's own fade is not the hop
	r.after(theme.HopLanding / 2)
	flying := r.sidebarPixels()
	r.after(theme.HopDuration)
	settled := r.sidebarPixels()
	r.after(time.Second)
	later := r.sidebarPixels()
	if string(settled) != string(later) {
		t.Error("the window still changes after the hop is over")
	}
	if string(flying) == string(settled) {
		t.Error("the sprite in flight left no pixels")
	}
}

func TestTheSpriteInFlightDoesNotBlockTheSidebar(t *testing.T) {
	r := newHopRig(t, false)
	r.tt.Frame()
	r.handOver("Port the watcher")
	r.leave()
	r.after(theme.HopLanding / 3) // over the Mine label
	if len(r.m.hop.flights) != 1 {
		t.Fatal("no hop in flight")
	}
	if err := r.tt.Click("Mine"); err != nil {
		t.Fatal(err)
	}
	if r.m.nav.screen != screenMine {
		t.Errorf("the click went to %+v: the sprite swallowed it", r.m.nav)
	}
}

// sidebarPixels is the sidebar's top, where the hop plays, as the last frame drew it.
func (r *hopRig) sidebarPixels() []byte {
	img := r.tt.Image()
	var out []byte
	for y := 0; y < 300; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+int(theme.SidebarWidth)*4]
		out = append(out, row...)
	}
	return out
}

func TestHandedOverRowsHaveActionsThatDoTheNextThing(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Open Fix tray race on launch"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 3 {
		t.Errorf("Answer opened %d, want the blocked task 3", m.openID)
	}
	has(t, tt, "Answer", "Open PR")
	hasNot(t, tt, "Review")
}

func TestAnAgentDoneWithoutAPullRequestSaysSoAndOffersReview(t *testing.T) {
	r := newHopRig(t, false)
	r.store.ReportAs(r.handOver("Tidy the logs").ID, "codex", "Done", r.now)
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "Codex finished and is waiting for you", "Review")
}

// The hop is short and stays at the right of the rows: it never covers a row's name, and its arc
// stays below the search field above Mine.
func TestTheHopStaysByTheCountsAndBelowTheSearchField(t *testing.T) {
	h := hopper{
		mine:   ui.Rect{X: 10, Y: 96, W: 186, H: 32},
		agents: ui.Rect{X: 10, Y: 128, W: 186, H: 32},
	}
	fx, fy, tx, ty := h.ends()
	for e := time.Duration(0); e < theme.HopDuration; e += time.Millisecond {
		f, ok := theme.HopAt(e, fx, fy, tx, ty)
		if !ok {
			continue
		}
		if f.X < h.mine.X+h.mine.W/2 {
			t.Fatalf("at %v the sprite is at x %.1f, over the rows' names", e, f.X)
		}
		if f.Y < h.mine.Y-theme.HopPeak {
			t.Fatalf("at %v the sprite is at y %.1f, above the Mine row by more than %v", e, f.Y, theme.HopPeak)
		}
	}
}
