package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// handRig is the window on project 1 with a clock the test moves.
type handRig struct {
	reduce bool
	tt     *ui.Tester
	m      *Main
	app    *App
	store  *data.Fake
	now    time.Time
}

func newHandRig(t *testing.T, reduceMotion bool) *handRig {
	t.Helper()
	r := &handRig{now: fixedNow(), reduce: reduceMotion}
	r.store = data.NewFake(func() time.Time { return r.now })
	r.app = NewApp(r.store, &recorder{}, testConfig())
	r.app.Now = func() time.Time { return r.now }
	r.m = NewMain(r.app)
	r.tt = ui.NewTester(r.m.View, 1100, 720)
	r.tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	r.tt.SetPreferences(ui.Preferences{ReduceMotion: reduceMotion})
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.tt.Frame()
	r.tt.Frame()
	return r
}

func (r *handRig) after(d time.Duration) {
	r.now = r.now.Add(d)
	r.tt.Frame()
}

// grab presses the row showing text and starts dragging it; it returns where the row was grabbed.
func (r *handRig) grab(t *testing.T, text string) (x, y float32) {
	t.Helper()
	from, ok := r.tt.Find(text)
	if !ok {
		t.Fatalf("no row %q in %q", text, r.tt.Texts())
	}
	x, y = center(from)
	r.tt.Press(x, y)
	r.tt.Move(x+8, y+8)
	return x + 8, y + 8
}

// dragOnto drags the row showing text to the middle of the row showing onto, and lets go unless hold.
func (r *handRig) dragOnto(t *testing.T, text, onto string, hold bool) {
	t.Helper()
	r.grab(t, text)
	to, ok := r.tt.Find(onto)
	if !ok {
		t.Fatalf("no %q in %q", onto, r.tt.Texts())
	}
	tx, ty := center(to)
	r.tt.Move(tx, ty)
	if !hold {
		r.tt.Release(tx, ty)
		r.tt.Frame()
	}
}

func TestDroppingMyRowOnTheAgentsTrayHandsItOverAndStartsNothing(t *testing.T) {
	r := newHandRig(t, true)
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	task, _ := r.app.Task(5)
	if task.Source != data.SourceAgent || task.HandedTo != data.AnyAgent || task.Status != data.Todo {
		t.Fatalf("task 5 after the drop: %+v", task)
	}
	has(t, r.tt, "Waiting for an agent", "Handed to an agent", "Tell any agent “do #5” to pick it up", "Undo")
	hasNot(t, r.tt, "opened in Terminal", "prompt copied", "Prompt copied")
	if !above(t, r.tt, "Waiting for an agent", "Fix tray race on launch") {
		t.Error("Waiting for an agent is the first group of the Agents column")
	}
	if got := r.tt.Clipboard(); got != "" {
		t.Errorf("the clipboard holds %q: nothing is copied", got)
	}
}

func TestTheWaitingRowAndTheToastAreInChinese(t *testing.T) {
	r := newHandRig(t, true)
	inChinese(r.tt)
	r.dragOnto(t, "Decide on the activation policy", "Port core to Go", false)
	has(t, r.tt, "已交给 Agent", "等 Agent 接手", "在任意 Agent 里说「做 #6」就会接手", "撤销")
}

func TestTheTrayInvitesTheDrop(t *testing.T) {
	r := newHandRig(t, true)
	hasNot(t, r.tt, "Drop to hand to")
	r.dragOnto(t, "Write the release notes", "Port core to Go", true)
	r.tt.Frame()
	has(t, r.tt, "Drop to hand to an agent")
	r.tt.Release(400, 600)
	r.tt.Frame()
	hasNot(t, r.tt, "Drop to hand to")
}

func TestOnlyTasksOfMineDragToTheTrayAndOnlyUnfinishedOnes(t *testing.T) {
	r := newHandRig(t, true)
	r.dragOnto(t, "Draft the UI spec", "Port core to Go", false) // done
	if task, _ := r.app.Task(13); task.Source != data.SourceUser {
		t.Errorf("a finished task was handed over: %+v", task)
	}
	r.dragOnto(t, "Port core to Go", "Port core to Go", false) // an agent's row on its own column
	if task, _ := r.app.Task(1); task.HandedTo != "" || task.Source != data.SourceAgent {
		t.Errorf("an agent's task changed: %+v", task)
	}
}

func TestUndoTakesTheTaskBackWithItsStatus(t *testing.T) {
	r := newHandRig(t, true)
	r.dragOnto(t, "Decide on the activation policy", "Port core to Go", false) // Doing, P1
	if task, _ := r.app.Task(6); task.Status != data.Todo || task.Source != data.SourceAgent {
		t.Fatalf("after the hand-off %+v", task)
	}
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	if task, _ := r.app.Task(6); task.Source != data.SourceUser || task.Status != data.Doing || task.HandedTo != "" {
		t.Errorf("after Undo %+v", task)
	}
}

func TestCmdZUndoesTheHandOffWhileItsToastShows(t *testing.T) {
	r := newHandRig(t, true)
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	r.tt.Key(ui.Cmd, ui.KeyZ)
	r.tt.Frame()
	if task, _ := r.app.Task(5); task.Source != data.SourceUser || task.HandedTo != "" {
		t.Errorf("after ⌘Z %+v", task)
	}
	hasNot(t, r.tt, "Waiting for an agent")
}

func TestDraggingAnAgentsRowOntoMineTakesItBackAndUndoRestoresIt(t *testing.T) {
	r := newHandRig(t, true)
	r.dragOnto(t, "Write the release notes", "Port core to Go", false) // now waiting
	r.after(10 * time.Second)
	r.dragOnto(t, "Write the release notes", "Decide on the activation policy", false)
	task, _ := r.app.Task(5)
	if task.Source != data.SourceUser || task.HandedTo != "" {
		t.Fatalf("after the take-back %+v", task)
	}
	has(t, r.tt, "Yours now")
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	if task, _ = r.app.Task(5); task.Source != data.SourceAgent || task.HandedTo != data.AnyAgent {
		t.Errorf("after Undo %+v: the hand-off was not restored", task)
	}
}

// userPixels counts the bluish pixels (the shirt of the user, seen through the copy's 75 % opacity) in a
// window of the last frame: the sprite under a carried row.
func (r *handRig) userPixels(x, y, w, h float32) int {
	img := r.tt.Image()
	n := 0
	for py := int(y); py < int(y+h); py++ {
		for px := int(x); px < int(x+w); px++ {
			if c := img.RGBAAt(px, py); int(c.B)-int(c.R) > 60 && int(c.B)-int(c.G) > 60 {
				n++
			}
		}
	}
	return n
}

func TestALiftedRowIsCarriedByTheUsersSprite(t *testing.T) {
	r := newHandRig(t, true)
	if r.m.ho.carry.seen {
		t.Fatal("a sprite before anything is lifted")
	}
	r.tt.SetScale(1)
	x, y := r.grab(t, "Write the release notes")
	r.tt.Move(x+40, y+60)
	r.tt.Frame()
	if !r.m.ho.carry.has {
		t.Fatal("the row is lifted and the sprite is not carrying it")
	}
	// The sprite stands under the left end of the card, which follows the pointer.
	if n := r.userPixels(x-400, y-20, 700, 140); n == 0 {
		t.Error("no pixel of the sprite's shirt near the carried row")
	}
	r.tt.Release(x+40, y+60)
	r.tt.Frame()
	r.tt.Frame()
	if r.m.ho.carry.has {
		t.Error("the sprite still carries a row that was dropped")
	}
}

func TestTheCarrierWalksWhileThePointerMovesAndStandsStillWhenItStops(t *testing.T) {
	r := newHandRig(t, false)
	x, y := r.grab(t, "Write the release notes")
	r.after(10 * time.Millisecond)
	base := r.m.ho.asked
	for i := 0; i < 5; i++ {
		r.tt.Move(x+float32(10*i), y+float32(5*i))
		r.after(30 * time.Millisecond)
	}
	if r.m.ho.asked <= base {
		t.Fatal("the pointer moves and the sprite asks for no frame")
	}
	if bob := theme.CarryBob(r.now.Sub(r.m.ho.carry.since)); bob < 0 || bob > theme.CarryBobHeight {
		t.Errorf("bob %v", bob)
	}
	// It stops after the pointer does.
	r.after(time.Second)
	r.after(time.Second)
	asked := r.m.ho.asked
	for i := 0; i < 5; i++ {
		r.after(time.Second)
	}
	if r.m.ho.asked != asked {
		t.Errorf("a sprite that stands still asked for %d frames", r.m.ho.asked-asked)
	}
}

func TestReduceMotionKeepsTheCarrierButItDoesNotWalk(t *testing.T) {
	r := newHandRig(t, true)
	x, y := r.grab(t, "Write the release notes")
	for i := 0; i < 5; i++ {
		r.tt.Move(x+float32(10*i), y+float32(5*i))
		r.after(30 * time.Millisecond)
	}
	if !r.m.ho.carry.has {
		t.Fatal("the sprite does not carry the row with Reduce Motion")
	}
	if r.m.ho.asked != 0 {
		t.Errorf("with Reduce Motion %d frames were asked for", r.m.ho.asked)
	}
}

// agentBox is where the robot of the Agents header rests.
func (r *handRig) agentBox() ui.Rect {
	e, _ := r.tt.Find("Agents")
	return ui.Rect{X: e.X - 9 - theme.ColumnAvatar, Y: e.Y + (e.H-theme.ColumnAvatar)/2, W: theme.ColumnAvatar, H: theme.ColumnAvatar}
}

func TestTheAgentHopsEveryEightHundredMillisecondsWhileARowIsOverTheTray(t *testing.T) {
	r := newHandRig(t, false)
	rest := r.agentBox()
	if f := r.m.agentFrame(r.reduce, rest); f != rest {
		t.Fatalf("at rest %+v, want %+v", f, rest)
	}
	r.dragOnto(t, "Write the release notes", "Port core to Go", true)
	r.tt.Frame()
	if !r.m.ho.over {
		t.Fatal("the row is over the tray and the agent does not know")
	}
	r.after(300 * time.Millisecond)
	up := r.m.agentFrame(r.reduce, rest)
	if up.Y >= rest.Y-3 {
		t.Errorf("mid hop at %v, rest %v: the agent should be up", up.Y, rest.Y)
	}
	r.after(theme.ReachEvery)
	if again := r.m.agentFrame(r.reduce, rest); again != up {
		t.Errorf("the second hop %+v differs from the first %+v", again, up)
	}
	asked := r.m.ho.asked
	r.after(10 * time.Millisecond)
	if r.m.ho.asked == asked {
		t.Error("the agent hops and no frame was asked for")
	}
	r.tt.Release(400, 600) // let go away from the tray
	r.after(10 * time.Millisecond)
	if r.m.ho.over || r.m.agentFrame(r.reduce, rest) != rest {
		t.Error("the agent keeps hopping after the row left")
	}
}

func TestTheAgentSquashesOnceOnTheDropAndTheRowKeepsAWashThatFades(t *testing.T) {
	r := newHandRig(t, false)
	rest := r.agentBox()
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	f := r.m.agentFrame(r.reduce, rest)
	if f.W <= rest.W || f.H >= rest.H {
		t.Errorf("at the drop %+v, rest %+v: wider and shorter", f, rest)
	}
	if got := r.m.ho.rowWash(5, r.now); got < theme.WashMax-0.001 {
		t.Errorf("wash %v at the drop, want %v", got, theme.WashMax)
	}
	r.after(theme.WashFor / 2)
	if got := r.m.ho.rowWash(5, r.now); got <= 0 || got >= theme.WashMax {
		t.Errorf("half way %v", got)
	}
	if r.m.agentFrame(r.reduce, rest) != rest {
		t.Error("the agent is still squashed after it settled")
	}
	if got := r.m.ho.rowWash(6, r.now); got != 0 {
		t.Errorf("another row has a wash of %v", got)
	}
	r.after(theme.WashFor)
	asked := r.m.ho.asked
	for i := 0; i < 5; i++ {
		r.after(time.Second)
	}
	if r.m.ho.asked != asked || len(r.m.ho.landed.ids) != 0 {
		t.Errorf("%d frames asked for after the wash was gone", r.m.ho.asked-asked)
	}
}

func TestReduceMotionTheAgentNeitherHopsNorSquashes(t *testing.T) {
	r := newHandRig(t, true)
	rest := r.agentBox()
	r.dragOnto(t, "Write the release notes", "Port core to Go", true)
	r.after(300 * time.Millisecond)
	if f := r.m.agentFrame(r.reduce, rest); f != rest {
		t.Errorf("hopping with Reduce Motion: %+v", f)
	}
	r.tt.Release(400, 600)
	r.tt.Frame()
}

func TestAStillColumnsWindowAsksForNoFrames(t *testing.T) {
	r := newHandRig(t, false)
	for i := 0; i < 5; i++ {
		r.after(time.Second)
	}
	if r.m.ho.asked != 0 {
		t.Errorf("a window with nothing to animate asked for %d frames", r.m.ho.asked)
	}
}

func TestAHandOffNamesNoAgentAnywhereEvenWhenCodexReportedLast(t *testing.T) {
	r := newHandRig(t, true)
	r.store.SeedReportAs(15, "codex", "Tidying up", r.now.Add(-time.Second)) // the latest report of the project, from Codex
	r.store.SeedReportAs(1, "codex", "Still at it", r.now)
	r.app.Refresh()
	r.tt.Frame()
	r.dragOnto(t, "Write the release notes", "Port core to Go", true)
	r.tt.Frame()
	has(t, r.tt, "Drop to hand to an agent")
	hasNot(t, r.tt, "Drop to hand to Codex", "Drop to hand to Claude Code")
	r.tt.Release(400, 600) // let go away from the tray: nothing was handed over
	r.tt.Frame()
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	task, _ := r.app.Task(5)
	if task.HandedTo != data.AnyAgent {
		t.Errorf("handed to %q, want %q", task.HandedTo, data.AnyAgent)
	}
	has(t, r.tt, "Handed to an agent", "Tell any agent “do #5” to pick it up")
	hasNot(t, r.tt, "Handed to Codex", "Handed to Claude Code")
	onlyInSidebar(t, r.tt, "Codex", "Claude Code") // the sidebar lists the agents at work; no row says one
	if paintedLeftOf(t, r.tt, 1100, "Write the release notes", 30, codexColor) {
		t.Error("the waiting row carries the sprite of Codex")
	}
	if !paintedLeftOf(t, r.tt, 1100, "Write the release notes", 30, accentBotColor) {
		t.Error("the waiting row does not carry the robot of the Agents header")
	}
}

var (
	codexColor     = spritePixel(theme.AgentAvatar("codex").Sprite(), 3, 2)
	accentBotColor = spritePixel(theme.AccentBot(theme.For(false, ui.Transparent).Accent).Sprite(), 2, 2)
)
