package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const (
	heroPlaceholder = "Call the landlord tomorrow at 9, and have Claude fix the login bug"
	panelTitle      = "Agent"
	panelDrop       = "Hand to an agent"
)

// Home opens with the date as the title and one big input over the lists; the composer is not at the bottom.
func TestHomeOpensWithTheBigInputAtTheTopAndNoBottomComposer(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Frame()
	has(t, r.tt, "Wednesday, October 14", heroPlaceholder, "Add images", "⌥ Space", "Send")
	hasNot(t, r.tt, "What do you want to do today?", "Return to add", "⌘K")
	input, _ := r.tt.Find(heroPlaceholder)
	today, ok := r.tt.Find("Today 2")
	if !ok {
		today, ok = r.tt.Find("Next")
	}
	if !ok {
		t.Fatalf("no section in %q", r.tt.Texts())
	}
	if input.Y > 300 || input.Y > today.Y {
		t.Errorf("the input is at %v and the lists at %v: the input belongs at the top, over the lists", input.Y, today.Y)
	}
	if r.m.dockHeight != 0 {
		t.Errorf("a composer dock of %v pt is pinned at the bottom of Home", r.m.dockHeight)
	}
	if !above(t, r.tt, "Wednesday, October 14", heroPlaceholder) {
		t.Error("the date and the input are out of order")
	}
	hasNot(t, r.tt, "DeepSeek") // the model is chosen in Settings, not in the box
}

func TestTheHomeComposerSpansBothColumnsAndHasTwoRows(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Frame()
	input, _ := r.tt.Find(heroPlaceholder)
	send, _ := r.tt.Find("Send")
	agent, _ := r.tt.Find("Fix tray race on launch")
	if send.X < agent.X { // send is at the right end of the box, over the tray's column
		t.Errorf("send at x=%v is not over the Agents tray at x=%v: the box spans both columns", send.X, agent.X)
	}
	if send.Y <= input.Y {
		t.Errorf("the controls (y=%v) are not on a row under the text (y=%v)", send.Y, input.Y)
	}
}

func TestHomeTitleAndPlaceholderInChinese(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	inChinese(r.tt)
	r.tt.Frame()
	has(t, r.tt, "10月14日 星期三", "明天 9 点给房东打电话，再让 Claude 修一下登录的 bug")
	hasNot(t, r.tt, "今天想做点什么？", "回车添加")
}

// Agents and the Calendar have no composer; a project keeps its small one.
func TestOnlyProjectsKeepASmallInput(t *testing.T) {
	for name, n := range map[string]nav{
		"agents": {screen: screenAgents}, "calendar": {screen: screenCalendar},
	} {
		t.Run(name, func(t *testing.T) {
			r := newAssistRig(t, &data.FakeAssistant{Ready: true})
			r.m.goTo(n)
			r.tt.Frame()
			hasNot(t, r.tt, heroPlaceholder, "Add a task, or ask me anything", "⌘K")
			if r.m.pg.Composer || r.m.dockHeight != 0 {
				t.Errorf("composer %v, dock %v pt", r.m.pg.Composer, r.m.dockHeight)
			}
		})
	}
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.onProject()
	has(t, r.tt, "Add a task, or ask me anything")
	if r.m.dockHeight == 0 {
		t.Error("a project has no composer dock")
	}
}

// Cmd+K on a page with no composer goes to Home and puts the cursor in its big input.
func TestCmdKFromAgentsGoesHomeAndFocusesTheBigInput(t *testing.T) {
	for _, n := range []nav{{screen: screenAgents}, {screen: screenCalendar}} {
		r := newAssistRig(t, &data.FakeAssistant{Ready: true})
		r.m.goTo(n)
		r.tt.Frame()
		r.tt.Key(ui.Cmd, ui.KeyK)
		for i := 0; i < 5; i++ {
			r.tt.Frame()
		}
		r.tt.Type("hello")
		if r.m.nav != (nav{}) {
			t.Errorf("from %+v Cmd+K went to %+v, want Home", n, r.m.nav)
		}
		if got := r.m.cmp[surfacePage].buf; got != "hello" {
			t.Errorf("from %+v the big input holds %q", n, got)
		}
	}
}

func TestCmdKOnHomeFocusesTheBigInput(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Frame()
	r.tt.Type("hello")
	if got := r.m.cmp[surfacePage].buf; got != "hello" {
		t.Errorf("typed into the big input: %q", got)
	}
}

// A sentence sent from the big input answers under it: the receipt names where it went, the fold
// opens, and the new row is tinted with accent.soft until it fades.
func TestTheReceiptIsUnderTheBigInputAndTheNewRowIsTinted(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Assistant = nil
	tt, m, _ := newTesterFor(app, 1100)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Water the plants")
	tt.Key(0, ui.KeyEnter)
	has(t, tt, "Undo", "Added to No date", "Water the plants")
	undo, _ := tt.Find("Undo")
	if h, _ := tt.Find("No date"); undo.Y > h.Y {
		t.Errorf("the receipt (%v) is under the lists (%v), not under the input", undo.Y, h.Y)
	}
	row, _ := tt.Find("Water the plants")
	tint := theme.For(false, ui.Transparent).AccentSoft
	if !paintedIn(tt.Image(), 1100, ui.Rect{X: row.X + 300, Y: row.Y, W: 20, H: row.H}, tint, 3) {
		t.Error("the row the input just made has no accent.soft fill")
	}
	app.Now = func() time.Time { return fixedNow().Add(freshFor + freshFade + 50*time.Millisecond) }
	tt.Frame()
	if _, ok := m.fresh[taskByTitle(t, app, "Water the plants").ID]; ok && paintedIn(tt.Image(), 1100, ui.Rect{X: row.X + 300, Y: row.Y, W: 20, H: row.H}, tint, 3) {
		t.Error("the tint did not fade after 2 s")
	}
}

func TestTheAgentsPanelSitsBesideTheListsInAWideWindow(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	has(t, tt, panelTitle, "Fix tray race on launch", "Reply", "Mark done", "Today", "Next")
	hasNot(t, tt, "From your agents", panelDrop) // the drop target shows only while a row is dragged
	has(t, tt, "3d", "25m")                      // the ages, at the right of the title lines
	title, _ := tt.Find(panelTitle)
	next, _ := tt.Find("Next")
	if title.X < next.X+300 {
		t.Errorf("the panel title at x=%v is not to the right of the lists at x=%v", title.X, next.X)
	}
}

func TestTheAgentsPanelGivesWayToTheInlineSectionInANarrowWindow(t *testing.T) {
	tt, m0, _ := newMainTester(t, inlineWidth)
	has(t, tt, "From your agents", "Fix tray race on launch")
	if m0.mineSplit() {
		t.Error("the tray is beside the lists in a narrow window")
	}
	// The inspector takes room from a wide window too.
	tt2, m, _ := newMainTester(t, 1100)
	m.OpenTask(5)
	tt2.Frame()
	has(t, tt2, "From your agents")
	if m.mineSplit() {
		t.Error("the tray is beside the lists with the inspector open")
	}
}

func TestTheAgentsPanelInChinese(t *testing.T) {
	tt, _, _ := newChineseTester(t, 1100)
	has(t, tt, "Agent", "回复", "标为完成", "等你回复：", "PR #12 待审查")
}

func TestAnEmptyAgentsPanelIsTheIdleCard(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, panelTitle, "Idle", "All resting", "Drag a task here", " to hand it off")
	hasNot(t, tt, "Claude Code") // no agent ever reported
}

// Dropping one of my rows on the panel hands the task over, as dragging it onto the tray of a project.
func TestDroppingMyRowOnTheAgentsPanelHandsItOver(t *testing.T) {
	r := newHandRig(t, true)
	r.m.goTo(nav{})
	r.tt.Frame()
	r.tt.Frame()
	task := taskByTitle(t, r.app, "Book dentist")
	hasNot(t, r.tt, panelDrop)
	r.dragOnto(t, "Book dentist", "Native UI with mygo", true)
	r.tt.Frame()
	has(t, r.tt, panelDrop) // the well appears once a row is dragged
	to, _ := r.tt.Find(panelDrop)
	r.tt.Release(center(to))
	r.tt.Frame()
	got, _ := r.app.Task(task.ID)
	if got.Source != data.SourceAgent || got.HandedTo != data.AnyAgent {
		t.Fatalf("after the drop on the panel: %+v", got)
	}
	has(t, r.tt, "Handed to an agent", "Undo")
}

// Only my unfinished rows hand over; an agent's row dropped on the panel changes nothing.
func TestDroppingARowOfAnAgentOnTheAgentsPanelDoesNothing(t *testing.T) {
	r := newHandRig(t, true)
	r.m.goTo(nav{})
	r.tt.Frame()
	r.tt.Frame()
	r.dragOnto(t, "Fix tray race on launch", "Native UI with mygo", false)
	if task, _ := r.app.Task(taskByTitle(t, r.app, "Fix tray race on launch").ID); task.HandedTo != "" {
		t.Errorf("an agent's task changed: %+v", task)
	}
}
