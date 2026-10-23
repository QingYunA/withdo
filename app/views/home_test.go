package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

const (
	heroPlaceholder = "Tell me anything — I'll turn it into tasks"
	heroHint        = "Return to add · ⌥ Space from any app · One sentence can hold several things"
	panelTitle      = "What your agents are doing"
	panelDrop       = "Drag a to-do here to hand it to an agent"
)

// Home opens with the day, the greeting and one big input; the composer is not at the bottom.
func TestHomeOpensWithTheBigInputAtTheTopAndNoBottomComposer(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Frame()
	has(t, r.tt, "Wednesday, October 14", "What do you want to do today?", heroPlaceholder, heroHint)
	input, _ := r.tt.Find(heroPlaceholder)
	later, ok := r.tt.Find("Later")
	if !ok {
		t.Fatalf("no Later in %q", r.tt.Texts())
	}
	if input.Y > 300 || input.Y > later.Y {
		t.Errorf("the input is at %v and Later at %v: the input belongs at the top, over the lists", input.Y, later.Y)
	}
	if r.m.dockHeight != 0 {
		t.Errorf("a composer dock of %v pt is pinned at the bottom of Home", r.m.dockHeight)
	}
	if !above(t, r.tt, "What do you want to do today?", heroPlaceholder) || !above(t, r.tt, "Wednesday, October 14", "What do you want to do today?") {
		t.Error("the date, the greeting and the input are out of order")
	}
	hasNot(t, r.tt, "⌘K") // the hint line says the keys; the box does not repeat them
}

func TestHomeGreetingAndHintInChinese(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	inChinese(r.tt)
	r.tt.Frame()
	has(t, r.tt, "今天想做点什么？", "回车添加 · 在任何 App 里按 ⌥ Space · 一句话里可以有好几件事", "告诉我任何事，我来变成任务")
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

// A sentence sent from the big input answers under it, where the hint is; the hint comes back after.
func TestTheReceiptReplacesTheHintUnderTheBigInput(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Assistant = nil
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, heroHint)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Water the plants")
	tt.Key(0, ui.KeyEnter)
	has(t, tt, "Undo")
	hasNot(t, tt, heroHint)
	input, _ := tt.Find("Undo")
	if h, _ := tt.Find("Later"); input.Y > h.Y {
		t.Errorf("the receipt (%v) is under the lists (%v), not under the input", input.Y, h.Y)
	}
}

func TestTheAgentsPanelSitsBesideTheListsInAWideWindow(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	has(t, tt, panelTitle, panelDrop, "Fix tray race on launch", "Answer", "Today", "Later")
	hasNot(t, tt, "From your agents")
	has(t, tt, "3d", "25m") // the ages, at the right of the title lines
	title, _ := tt.Find(panelTitle)
	later, _ := tt.Find("Later")
	if title.X < later.X+300 {
		t.Errorf("the panel title at x=%v is not to the right of the lists at x=%v", title.X, later.X)
	}
}

func TestTheAgentsPanelGivesWayToTheInlineSectionInANarrowWindow(t *testing.T) {
	tt, _, _ := newMainTester(t, inlineWidth)
	has(t, tt, "From your agents", "Fix tray race on launch")
	hasNot(t, tt, panelTitle, panelDrop)
	// The inspector takes room from a wide window too.
	tt2, m, _ := newMainTester(t, 1100)
	m.OpenTask(5)
	tt2.Frame()
	has(t, tt2, "From your agents")
	hasNot(t, tt2, panelTitle)
}

func TestTheAgentsPanelInChinese(t *testing.T) {
	tt, _, _ := newChineseTester(t, 1100)
	has(t, tt, "Agent 在做什么", "把左边的待办拖到这里，交给 Agent")
}

func TestAnEmptyAgentsPanelIsQuietAndKeepsTheDropTarget(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, panelTitle, "No agent is working on anything", panelDrop)
}

// Dropping one of my rows on the panel hands the task over, as dragging it onto the tray of a project.
func TestDroppingMyRowOnTheAgentsPanelHandsItOver(t *testing.T) {
	r := newHandRig(t, true)
	r.m.goTo(nav{})
	r.tt.Frame()
	r.tt.Frame()
	task := taskByTitle(t, r.app, "Book dentist")
	r.dragOnto(t, "Book dentist", panelDrop, true)
	r.tt.Frame()
	has(t, r.tt, panelDrop)
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
	r.dragOnto(t, "Fix tray race on launch", panelDrop, false)
	if task, _ := r.app.Task(taskByTitle(t, r.app, "Fix tray race on launch").ID); task.HandedTo != "" {
		t.Errorf("an agent's task changed: %+v", task)
	}
}
