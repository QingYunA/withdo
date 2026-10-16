package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// unlaunched is the Fake when no agent could be started: the prompt is for the clipboard.
type unlaunched struct{ *data.Fake }

func (u unlaunched) HandOff(id int, prompt string) (data.HandOffResult, error) {
	res, err := u.Fake.HandOff(id, prompt)
	res.Launched = false
	return res, err
}

// handRig is the window on project 1 with a clock the test moves.
type handRig struct {
	tt    *ui.Tester
	m     *Main
	app   *App
	store *data.Fake
	now   time.Time
}

func newHandRig(t *testing.T, reduceMotion, launched bool) *handRig {
	t.Helper()
	r := &handRig{now: fixedNow()}
	r.store = data.NewFake(func() time.Time { return r.now })
	var store data.Store = r.store
	if !launched {
		store = unlaunched{r.store}
	}
	r.app = NewApp(store, &recorder{}, testConfig())
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

// drag drops the row showing text on the row showing onto, and returns without letting go when
// hold is set.
func (r *handRig) drag(t *testing.T, text, onto string, hold bool) {
	t.Helper()
	from, ok := r.tt.Find(text)
	if !ok {
		t.Fatalf("no row %q in %q", text, r.tt.Texts())
	}
	to, ok := r.tt.Find(onto)
	if !ok {
		t.Fatalf("no %q in %q", onto, r.tt.Texts())
	}
	fx, fy := center(from)
	tx, ty := center(to)
	r.tt.Press(fx, fy)
	r.tt.Move(fx+8, fy+8)
	r.tt.Move(tx, ty)
	if !hold {
		r.tt.Release(tx, ty)
		r.tt.Frame()
	}
}

func TestDroppingMyRowOnTheAgentsTrayHandsItOver(t *testing.T) {
	r := newHandRig(t, true, true)
	r.drag(t, "Write the release notes", "Port core to Go", false)
	task, _ := r.app.Task(5)
	if task.Source != data.SourceAgent || task.HandedTo != "claude" || task.Status != data.Todo {
		t.Fatalf("task 5 after the drop: %+v", task)
	}
	has(t, r.tt, "Waiting for an agent", "Handed to Claude Code · opened in Terminal", "Handed to Claude Code — opened in Terminal", "Undo")
	if !above(t, r.tt, "Waiting for an agent", "Fix tray race on launch") {
		t.Error("Waiting for an agent is the first group of the Agents column")
	}
	if got := r.tt.Clipboard(); got != "" {
		t.Errorf("the clipboard holds %q although an agent was started", got)
	}
}

func TestAHandOffThatStartedNothingCopiesThePromptInTheAppsLanguage(t *testing.T) {
	r := newHandRig(t, true, false)
	r.drag(t, "Write the release notes", "Port core to Go", false)
	want := "Please do task #5 in adhd-todo: Write the release notes\nCover the Go core, the native UI and how to register the MCP server.\n" +
		"Start by calling report_progress (task_id: 5, status: \"doing\"), then keep reporting as adhd-todo's instructions say."
	if got := r.tt.Clipboard(); got != want {
		t.Errorf("clipboard %q, want %q", got, want)
	}
	has(t, r.tt, "Prompt copied — paste it to any agent", "Handed to Claude Code · prompt copied")
	hasNot(t, r.tt, "opened in Terminal")
}

func TestThePromptIsInChineseAndSkipsEmptyNotes(t *testing.T) {
	r := newHandRig(t, true, false)
	inChinese(r.tt)
	r.drag(t, "Decide on the activation policy", "Port core to Go", false)
	want := "请完成 adhd-todo 里的任务 #6：Decide on the activation policy\n" +
		"开始时调用 report_progress（task_id: 6，status: \"doing\"），之后按 adhd-todo 的说明继续汇报。"
	if got := r.tt.Clipboard(); got != want {
		t.Errorf("clipboard %q, want %q", got, want)
	}
	has(t, r.tt, "已复制提示词，粘贴给任意 Agent", "等 Agent 接手", "交给 Claude Code · 已复制提示词")
}

func TestTheTrayInvitesTheDropAndNamesTheAgent(t *testing.T) {
	r := newHandRig(t, true, true)
	hasNot(t, r.tt, "Drop to hand to")
	r.drag(t, "Write the release notes", "Port core to Go", true)
	r.tt.Frame()
	has(t, r.tt, "Drop to hand to Claude Code")
	r.tt.Release(400, 600)
	r.tt.Frame()
	hasNot(t, r.tt, "Drop to hand to")
}

func TestOnlyTasksOfMineDragToTheTrayAndOnlyUnfinishedOnes(t *testing.T) {
	r := newHandRig(t, true, true)
	r.drag(t, "Draft the UI spec", "Port core to Go", false) // done
	if task, _ := r.app.Task(13); task.Source != data.SourceUser {
		t.Errorf("a finished task was handed over: %+v", task)
	}
	r.drag(t, "Port core to Go", "Port core to Go", false) // an agent's row on its own column
	if task, _ := r.app.Task(1); task.HandedTo != "" || task.Source != data.SourceAgent {
		t.Errorf("an agent's task changed: %+v", task)
	}
}

func TestUndoTakesTheTaskBackWithItsStatus(t *testing.T) {
	r := newHandRig(t, true, true)
	r.drag(t, "Decide on the activation policy", "Port core to Go", false) // Doing, P1
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
	r := newHandRig(t, true, true)
	r.drag(t, "Write the release notes", "Port core to Go", false)
	r.tt.Key(ui.Cmd, ui.KeyZ)
	r.tt.Frame()
	if task, _ := r.app.Task(5); task.Source != data.SourceUser || task.HandedTo != "" {
		t.Errorf("after ⌘Z %+v", task)
	}
	hasNot(t, r.tt, "Waiting for an agent")
}

func TestDraggingAnAgentsRowOntoMineTakesItBackAndUndoRestoresIt(t *testing.T) {
	r := newHandRig(t, true, true)
	r.drag(t, "Write the release notes", "Port core to Go", false) // now waiting
	r.now = r.now.Add(10 * time.Second)
	r.tt.Frame()
	r.drag(t, "Write the release notes", "Decide on the activation policy", false)
	task, _ := r.app.Task(5)
	if task.Source != data.SourceUser || task.HandedTo != "" {
		t.Fatalf("after the take-back %+v", task)
	}
	has(t, r.tt, "Yours now")
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	if task, _ = r.app.Task(5); task.Source != data.SourceAgent || task.HandedTo != "claude" {
		t.Errorf("after Undo %+v: the hand-off was not restored", task)
	}
}

func TestAHandOffPlaysForAboutAMomentAndAsksForFramesOnlyThen(t *testing.T) {
	r := newHandRig(t, false, true)
	r.after(time.Second)
	if r.m.ho.asked != 0 || r.m.ho.anim != nil {
		t.Fatalf("a still window asked for %d frames", r.m.ho.asked)
	}
	r.drag(t, "Write the release notes", "Port core to Go", false)
	if r.m.ho.anim == nil {
		t.Fatal("no hand-off is playing")
	}
	r.after(100 * time.Millisecond)
	if r.m.ho.asked == 0 {
		t.Fatal("the hand-off plays and no frame was asked for")
	}
	r.after(theme.HandOffEnd(false))
	asked := r.m.ho.asked
	if r.m.ho.anim != nil {
		t.Fatal("the hand-off is still there when it is over")
	}
	for i := 0; i < 5; i++ {
		r.after(time.Second)
	}
	if r.m.ho.asked != asked {
		t.Errorf("%d frames asked for after the hand-off was over", r.m.ho.asked-asked)
	}
}

func TestTheNewRowOpensWhenTheParcelDoesAndKeepsAWashThatFades(t *testing.T) {
	r := newHandRig(t, false, true)
	r.drag(t, "Write the release notes", "Port core to Go", false)
	if got := r.m.ho.rowOpacity(5); got != 0 {
		t.Errorf("the row shows at once with opacity %v", got)
	}
	r.after(theme.HandOffOpenAt + theme.HandOffOpenFor)
	if got := r.m.ho.rowOpacity(5); got != 1 {
		t.Errorf("opacity %v once the parcel opened", got)
	}
	whole := r.m.ho.rowWash(5)
	r.after(theme.HandOffWash / 2)
	half := r.m.ho.rowWash(5)
	if whole < theme.HandOffWashMax-0.001 || half <= 0 || half >= whole {
		t.Errorf("wash %v then %v: it should start at %v and fade", whole, half, theme.HandOffWashMax)
	}
	if got := r.m.ho.rowOpacity(6); got != 1 {
		t.Errorf("another row has opacity %v", got)
	}
}

func TestTheSpritesLeaveTheirHeadersAndComeHome(t *testing.T) {
	r := newHandRig(t, false, true)
	r.drag(t, "Write the release notes", "Port core to Go", false)
	away := false
	for e := time.Duration(0); e < theme.HandOffEnd(false); e += 50 * time.Millisecond {
		r.after(50 * time.Millisecond)
		away = away || r.m.ho.youAway()
	}
	if !away {
		t.Error("the user's sprite never left its header")
	}
	if r.m.ho.youAway() {
		t.Error("the user's sprite is still away when the hand-off is over")
	}
}

func TestReduceMotionHasNoHopTheRowFadesOutAndIn(t *testing.T) {
	r := newHandRig(t, true, true)
	r.drag(t, "Write the release notes", "Port core to Go", false)
	f := r.m.ho.frame
	if f.YouOn || f.BotOn || f.ParcelOn {
		t.Errorf("with Reduce Motion something hops: %+v", f)
	}
	r.after(theme.HandOffReduceIn / 2)
	f = r.m.ho.frame
	if f.GhostOpacity <= 0 || f.GhostOpacity >= 1 || f.RowOpacity <= 0 || f.RowOpacity >= 1 {
		t.Errorf("half way: ghost %v, row %v, want both fading", f.GhostOpacity, f.RowOpacity)
	}
	r.after(theme.HandOffReduceIn)
	if f = r.m.ho.frame; f.GhostOpacity != 0 || f.RowOpacity != 1 {
		t.Errorf("after the fade: ghost %v, row %v", f.GhostOpacity, f.RowOpacity)
	}
}

func TestTheHandOffLeavesWhenTheScreenDoes(t *testing.T) {
	r := newHandRig(t, false, true)
	r.drag(t, "Write the release notes", "Port core to Go", false)
	r.m.goTo(nav{})
	r.after(50 * time.Millisecond)
	if r.m.ho.anim != nil {
		t.Error("the hand-off is still playing on another screen")
	}
}
