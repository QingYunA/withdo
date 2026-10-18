package views

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// fakeVoices is the CuePlayer that keeps what it was asked.
type fakeVoices struct {
	mu     sync.Mutex
	played []theme.Cue
	stops  int
}

func (f *fakeVoices) Play(c theme.Cue) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.played = append(f.played, c)
}

func (f *fakeVoices) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
}

// fakeTimer is a timer that fires when the test says.
type fakeTimer struct {
	d       time.Duration
	f       func()
	stopped bool
}

func (t *fakeTimer) Stop() bool { t.stopped = true; return true }

type soundRig struct {
	tt     *ui.Tester
	m      *Main
	app    *App
	voices *fakeVoices
	timers []*fakeTimer
}

// soundMain is the Summary page on a busy day or week, with the cues' timers in the test's hands. The
// scene moves unless reduce is set.
func soundMain(t *testing.T, week, reduce bool) *soundRig {
	t.Helper()
	app, _, _ := summaryApp(t)
	r := &soundRig{app: app, voices: &fakeVoices{}}
	app.Sound = r.voices
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 900)
	tt.SetPreferences(ui.Preferences{ReduceMotion: reduce})
	m.sum.track.async = func(f func()) { f() }
	m.sum.track.after = func(d time.Duration, f func()) stopper {
		ft := &fakeTimer{d: d, f: f}
		r.timers = append(r.timers, ft)
		return ft
	}
	m.sum.week = week
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	r.tt, r.m = tt, m
	return r
}

// fire runs the timers that were not stopped.
func (r *soundRig) fire() {
	for _, ft := range r.timers {
		if !ft.stopped {
			ft.f()
		}
	}
}

func (r *soundRig) pending() int {
	n := 0
	for _, ft := range r.timers {
		if !ft.stopped {
			n++
		}
	}
	return n
}

func sounds(cues []theme.Cue) []theme.Sound {
	var out []theme.Sound
	for _, c := range cues {
		out = append(out, c.Sound)
	}
	return out
}

func TestTheSceneSchedulesItsCuesFromThePlanOnceWhenItStartsPlaying(t *testing.T) {
	r := soundMain(t, false, false)
	want := r.m.summaryFor(r.app.Now()).day.SoundCues(false)
	if len(r.timers) != len(want) || len(want) < 4 {
		t.Fatalf("%d timers for %d cues", len(r.timers), len(want))
	}
	for i, ft := range r.timers {
		if ft.d > want[i].At || ft.d < want[i].At-time.Second {
			t.Errorf("cue %d (%+v) is timed %v", i, want[i], ft.d)
		}
	}
	r.tt.Frame()
	r.tt.Frame()
	if len(r.timers) != len(want) {
		t.Errorf("%d timers after more frames: the cues are set once per play", len(r.timers))
	}
	r.fire()
	if !slices.Equal(r.voices.played, want) {
		t.Errorf("played %+v, want %+v", r.voices.played, want)
	}
	if got := sounds(r.voices.played); got[0] != theme.SoundLid || got[len(got)-1] != theme.SoundDone {
		t.Errorf("sounds %v: the lid first, the done last", got)
	}
}

func TestTheWeekTicksAndEndsOnDone(t *testing.T) {
	r := soundMain(t, true, false)
	r.fire()
	got := sounds(r.voices.played)
	if len(got) < 2 || got[0] != theme.SoundTick || got[len(got)-1] != theme.SoundDone {
		t.Errorf("sounds %v", got)
	}
}

func TestWithReduceMotionOnlyTheDoneSounds(t *testing.T) {
	for _, week := range []bool{false, true} {
		r := soundMain(t, week, true)
		r.fire()
		if got := sounds(r.voices.played); !slices.Equal(got, []theme.Sound{theme.SoundDone}) {
			t.Errorf("week %v: sounds %v", week, got)
		}
	}
}

func TestNoCuesWhileTheSoundEffectsAreOffOrMacOSsAre(t *testing.T) {
	app, _, _ := summaryApp(t)
	app.SetSoundEffects(false)
	r := &soundRig{app: app, voices: &fakeVoices{}}
	app.Sound = r.voices
	tt, m, _ := newTesterFor(app, 1100)
	moving(tt)
	m.sum.track.async = func(f func()) { f() }
	m.sum.track.after = func(d time.Duration, f func()) stopper {
		ft := &fakeTimer{d: d, f: f}
		r.timers = append(r.timers, ft)
		return ft
	}
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	if len(r.timers) != 0 {
		t.Errorf("%d cues with the Sound effects switch off", len(r.timers))
	}

	asked := 0
	app.SetSoundEffects(true)
	app.SystemSounds = func() bool { asked++; return false }
	m.replayScene()
	tt.Frame()
	tt.Frame()
	if len(r.timers) != 0 || asked != 1 {
		t.Errorf("%d cues, macOS asked %d times: it says no, once per play", len(r.timers), asked)
	}
}

func TestACueDoesNotSoundWhenTheWindowIsNotKey(t *testing.T) {
	r := soundMain(t, false, false)
	key := false
	r.app.WindowKey = func() bool { return key }
	r.fire()
	if len(r.voices.played) != 0 {
		t.Errorf("played %+v in a window that is not key", r.voices.played)
	}
	key = true
	r.fire()
	if len(r.voices.played) == 0 {
		t.Error("nothing played once the window was key")
	}
}

func TestReplayingStopsWhatIsPendingAndSchedulesAgain(t *testing.T) {
	r := soundMain(t, false, false)
	first := len(r.timers)
	r.tt.ClickAt(r.m.sum.scene.X+20, r.m.sum.scene.Y+10) // the sky
	r.tt.Frame()
	if len(r.timers) != 2*first {
		t.Fatalf("%d timers, want the day's cues again", len(r.timers))
	}
	for i, ft := range r.timers {
		if ft.stopped != (i < first) {
			t.Errorf("timer %d stopped %v", i, ft.stopped)
		}
	}
	if r.voices.stops == 0 {
		t.Error("what was sounding was not stopped")
	}
}

func TestLeavingThePageStopsTheCuesToCome(t *testing.T) {
	r := soundMain(t, false, false)
	r.m.goTo(nav{})
	r.tt.Frame()
	if r.pending() != 0 || r.voices.stops != 1 {
		t.Errorf("%d cues pending, %d stops", r.pending(), r.voices.stops)
	}
	r.tt.Frame()
	if r.voices.stops != 1 {
		t.Errorf("%d stops: the page asks every frame, the sound once", r.voices.stops)
	}
	r.m.goTo(nav{screen: screenSummary})
	r.tt.Frame()
	if r.pending() == 0 {
		t.Error("coming back did not play the scene's sounds")
	}
}

func TestAnotherPeriodPlaysItsOwnCues(t *testing.T) {
	r := soundMain(t, false, false)
	first := len(r.timers)
	if err := r.tt.Click("This week"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if len(r.timers) == first {
		t.Error("the week set no cues")
	}
	if r.pending() != len(r.timers)-first {
		t.Errorf("%d pending: the old day's cues should be stopped", r.pending())
	}
}

func TestWithoutAPlayerTheSceneIsSilent(t *testing.T) {
	tt, m := summaryMain(t, mustSummaryApp(t), false)
	moving(tt)
	m.replayScene()
	tt.Frame()
	if m.app.Sound != nil || len(m.sum.track.timers) != 0 {
		t.Error("cues were set with no player")
	}
}

func mustSummaryApp(t *testing.T) *App {
	app, _, _ := summaryApp(t)
	return app
}
