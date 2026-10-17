package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// v9 B of the composer: the model chip is always in sight, and the box keeps its second row, with a
// stop button, while the assistant works on a request from it.

const deepseekChip = "DeepSeek · deepseek-chat"

// belowBox is the top of what sits under the composer: it moves with the height of the box and of
// the line under it.
func belowBox(t *testing.T, r *assistRig) float32 {
	t.Helper()
	rect, ok := r.tt.Find("From your agents")
	if !ok {
		t.Fatalf("no From your agents in %q", r.tt.Texts())
	}
	return rect.Y
}

func TestTheModelChipSitsInTheOneLineBoxAtRestAndFocusedAndTheBoxDoesNotGrow(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.withModels(t)
	chip, ok := r.tt.Find(deepseekChip)
	if !ok {
		t.Fatalf("no chip on the empty box at rest: %q", r.tt.Texts())
	}
	hint, _ := r.tt.Find("⌘K")
	if chip.X >= hint.X {
		t.Errorf("the chip (x=%v) is not before ⌘K (x=%v)", chip.X, hint.X)
	}
	rest := belowBox(t, r)
	r.tt.Key(ui.Cmd, ui.KeyK)
	focused, ok := r.tt.Find(deepseekChip)
	if !ok {
		t.Fatalf("no chip on the empty focused box: %q", r.tt.Texts())
	}
	if got := belowBox(t, r); got != rest {
		t.Errorf("focusing moved the page: %v -> %v", rest, got)
	}
	if focused.X+focused.W < chip.X+chip.W {
		t.Errorf("the chip moved left when ⌘K went: %v -> %v", chip, focused)
	}
}

func TestTheChipOpensItsPopoverFromTheOneLineBoxWithoutMovingThePage(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.withModels(t)
	rest := belowBox(t, r)
	rect, _ := r.tt.Find(deepseekChip)
	x, y := center(rect)
	r.tt.Move(x, y)
	r.tt.ClickAt(x, y)
	r.tt.Frame()
	has(t, r.tt, "Switch model", "Anthropic")
	if got := belowBox(t, r); got != rest {
		t.Errorf("opening the popover moved the page: %v -> %v", rest, got)
	}
	if err := r.tt.Click("Anthropic"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.activeProvider() != "anthropic" {
		t.Errorf("active %q", r.activeProvider())
	}
	has(t, r.tt, "Anthropic · claude-haiku-5-5")
}

func TestNarrowOneLineBoxesShowTheChipAsItsLogoAndKeepThePlaceholderWhole(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.withModels(t)
	has(t, r.tt, deepseekChip) // the page alone has room for the words
	r.m.OpenTask(5)
	r.tt.Frame()
	// The inspector's box, and the page's now that the inspector takes its room, keep the logo and
	// the chevron: the placeholders read whole.
	has(t, r.tt, "Change anything about this task", "Tell me anything — I'll turn it into tasks")
	if n := countText(r.tt, deepseekChip); n != 0 {
		t.Errorf("%d chips show their words, want none", n)
	}
	if n := countText(r.tt, "Model: "+deepseekChip); n != 2 {
		t.Errorf("%d chips, want the page's and the inspector's", n)
	}
	if n := countText(r.tt, "⌘K"); n != 1 {
		t.Errorf("%d ⌘K, want the page's alone", n)
	}
}

func TestWithoutAModelThereIsNoChipInTheOneLineBox(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.tt.Frame()
	hasNot(t, r.tt, deepseekChip, "Switch model")
}

// thinking sends a request that does not come back until r.fa.Block is closed.
func thinking(t *testing.T) *assistRig {
	t.Helper()
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Block: make(chan struct{}), Script: scripted(reminderReply(), nil)})
	r.withModels(t)
	return r
}

func TestWhileTheAssistantWorksTheSecondRowStaysWithAStopButton(t *testing.T) {
	r := thinking(t)
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Type("remind me")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	has(t, r.tt, "Stop", "Add images", deepseekChip, "Thinking", "· 0s", "Esc to cancel")
	if countText(r.tt, "Send") != 0 {
		t.Error("a send button beside the stop button")
	}
	working := belowBox(t, r)
	close(r.fa.Block)
	r.answer(t)
	hasNot(t, r.tt, "Thinking", "Esc to cancel")
	has(t, r.tt, "Team meeting", "Add images", deepseekChip)
	if countText(r.tt, "Stop") != 0 {
		t.Error("the stop button stays after the answer")
	}
	if got := belowBox(t, r); got != working {
		t.Errorf("the answer moved the page: %v -> %v; it takes the working line's place", working, got)
	}
}

func TestTheSecondRowGoesWhenTheAnswerDoes(t *testing.T) {
	r := thinking(t)
	rest := belowBox(t, r)
	r.ask("remind me")
	close(r.fa.Block)
	r.answer(t)
	if belowBox(t, r) == rest {
		t.Fatal("no second row while the answer shows")
	}
	r.now = r.now.Add(answerFor + time.Second)
	r.tt.Frame()
	r.tt.Frame()
	hasNot(t, r.tt, "Team meeting")
	if got := belowBox(t, r); got != rest {
		t.Errorf("the box did not shrink back with the answer: %v, rest %v", got, rest)
	}
}

func TestTheWorkingLineCountsSecondsAndSaysWhenItTakesLonger(t *testing.T) {
	r := thinking(t)
	r.ask("remind me")
	has(t, r.tt, "Thinking", "· 0s")
	r.now = r.now.Add(3 * time.Second)
	r.tt.Frame()
	has(t, r.tt, "Thinking", "· 3s", "Esc to cancel")
	r.now = r.now.Add(11*time.Second + 900*time.Millisecond) // 14.9 s
	r.tt.Frame()
	has(t, r.tt, "Thinking", "· 14s")
	hasNot(t, r.tt, "Taking longer than usual")
	r.now = r.now.Add(100 * time.Millisecond)
	r.tt.Frame()
	has(t, r.tt, "Taking longer than usual", "· 15s", "Esc to cancel")
	hasNot(t, r.tt, "Thinking")
	close(r.fa.Block)
	r.answer(t)
	hasNot(t, r.tt, "Taking longer than usual", "Esc to cancel")
}

func TestTheWorkingLineKeepsCountingWithReduceMotion(t *testing.T) {
	r := thinking(t)
	r.tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	r.ask("remind me")
	r.now = r.now.Add(2 * time.Second)
	r.tt.Frame()
	has(t, r.tt, "Thinking", "· 2s")
}

func TestClickingStopCancelsLikeEsc(t *testing.T) {
	r := thinking(t)
	r.ask("a slow one")
	if err := r.tt.Click("Stop"); err != nil {
		t.Fatal(err)
	}
	r.answer(t)
	hasNot(t, r.tt, "Thinking")
	if countText(r.tt, "Stop") != 0 {
		t.Error("the stop button stays")
	}
	if r.m.convo.thinking || r.m.convo.open {
		t.Errorf("still working or open after Stop: %+v", r.m.convo)
	}
	if r.pageBuf() != "a slow one" {
		t.Errorf("the field holds %q, want the text back", r.pageBuf())
	}
}

func TestTheWorkingLineAndStopAreOnlyInTheBoxThatSentTheRequest(t *testing.T) {
	r := thinking(t)
	r.inInspector(t, 5, "move it")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	has(t, r.tt, "Stop", "Thinking")
	if n := countText(r.tt, "Stop"); n != 1 {
		t.Errorf("%d stop buttons, want the inspector's alone", n)
	}
}
