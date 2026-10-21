package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The menu bar panel: the composer of Mine, focused, today's first rows, and one line about the agents.

// popoverRig is the panel over an app, in a tester as wide as the panel and as tall as it asks to be.
type popoverRig struct {
	tt     *ui.Tester
	p      *Popover
	app    *App
	closed int
}

func newPopoverRig(t *testing.T, app *App) *popoverRig {
	t.Helper()
	r := &popoverRig{app: app, p: NewPopover(app)}
	height := 600
	r.p.Close = func() { r.closed++ }
	r.p.Resize = func(h int) { height = h }
	r.tt = ui.NewTester(r.p.View, int(PopoverWidth), height)
	still(r.tt)
	r.p.Show()
	// The composer takes the keyboard on the third frame; the panel's height is known after the first.
	for i := 0; i < 4; i++ {
		r.tt.Frame()
		r.tt.SetSize(int(PopoverWidth), height)
	}
	return r
}

func TestPopoverRowsAreTheFirstFiveOfToday(t *testing.T) {
	var today []data.Item
	for i := 1; i <= 7; i++ {
		today = append(today, data.Item{Task: data.Task{ID: i}})
	}
	got := popoverRows(today)
	if len(got) != 5 || got[0].ID != 1 || got[4].ID != 5 {
		t.Errorf("rows = %v, want the first five in order", got)
	}
	if got := popoverRows(today[:2]); len(got) != 2 {
		t.Errorf("two rows of today gave %d", len(got))
	}
	if got := popoverRows(nil); len(got) != 0 {
		t.Errorf("no rows of today gave %d", len(got))
	}
}

func TestAgentCountsSplitWorkingFromStuck(t *testing.T) {
	row := func(s data.Status) data.AgentRow { return data.AgentRow{Item: data.Item{Task: data.Task{Status: s}}} }
	tests := []struct {
		name           string
		rows           []data.AgentRow
		working, stuck int
	}{
		{"none", nil, 0, 0},
		{"working only", []data.AgentRow{row(data.Doing), row(data.Doing)}, 2, 0},
		{"stuck only", []data.AgentRow{row(data.Blocked)}, 0, 1},
		{"both", []data.AgentRow{row(data.Doing), row(data.Blocked), row(data.Doing)}, 2, 1},
	}
	for _, tc := range tests {
		working, stuck := agentCounts(data.AgentsScreen{Working: tc.rows})
		if working != tc.working || stuck != tc.stuck {
			t.Errorf("%s: working %d, stuck %d, want %d, %d", tc.name, working, stuck, tc.working, tc.stuck)
		}
	}
}

func TestPopoverHeightGrowsWithRowsAndWhatIsUnderTheComposer(t *testing.T) {
	none, one, five := popoverHeight(0, false, false), popoverHeight(1, false, false), popoverHeight(5, false, false)
	if !(none < one && one < five) {
		t.Errorf("heights %v, %v, %v do not grow with the rows", none, one, five)
	}
	if line, reply := popoverHeight(3, false, true), popoverHeight(3, true, true); !(popoverHeight(3, false, false) < line && line < reply) {
		t.Errorf("a quiet line (%v) and a reply (%v) take room", line, reply)
	}
}

func TestPopoverShowsTodayAndTheFooter(t *testing.T) {
	app, _, _ := newTestApp(t)
	r := newPopoverRig(t, app)
	for _, want := range []string{"Today", "Reply to design feedback", "Open ADHD Todo"} {
		if !r.tt.HasText(want) {
			t.Errorf("missing %q in %q", want, r.tt.Texts())
		}
	}
	for _, gone := range []string{"Later", "Done today", "From your agents"} {
		if r.tt.HasText(gone) {
			t.Errorf("%q is on the panel: it lists today only", gone)
		}
	}
	working, stuck := agentCounts(app.buildAgents())
	if working == 0 || stuck == 0 {
		t.Fatalf("the demo data has %d working and %d stuck agents; the test needs both", working, stuck)
	}
	if !r.tt.HasText("·") {
		t.Errorf("no separator in the footer: %q", r.tt.Texts())
	}
}

func TestPopoverFooterSaysNothingWithoutAgents(t *testing.T) {
	app, _ := newEmptyApp(t)
	r := newPopoverRig(t, app)
	for _, gone := range []string{"working", "stuck", "·"} {
		for _, text := range r.tt.Texts() {
			if text == gone {
				t.Errorf("the footer says %q with no agents", gone)
			}
		}
	}
	if !r.tt.HasText("Open ADHD Todo") {
		t.Errorf("no way to the main window: %q", r.tt.Texts())
	}
}

func TestPopoverComposerIsFocusedAndFilesPlainTasksIntoMine(t *testing.T) {
	app, store, _ := newTestApp(t)
	before, _ := store.Snapshot()
	r := newPopoverRig(t, app)
	r.tt.Type("buy oat milk")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	after, _ := store.Snapshot()
	if len(after.Tasks) != len(before.Tasks)+1 {
		t.Fatalf("%d tasks after typing into the panel, want %d", len(after.Tasks), len(before.Tasks)+1)
	}
	var made data.Task
	for _, task := range after.Tasks {
		if task.Title == "buy oat milk" {
			made = task
		}
	}
	if made.ID == 0 || made.ProjectID != nil || made.Source != data.SourceUser {
		t.Errorf("the task = %+v, want one of mine with no project", made)
	}
	if r.p.m.cmp[surfacePage].buf != "" {
		t.Errorf("the field still holds %q", r.p.m.cmp[surfacePage].buf)
	}
}

func TestPopoverSendsToTheAssistantWithTheMinePlace(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	app, _, _ := assistApp(t, fa)
	r := newPopoverRig(t, app)
	r.tt.Type("call the landlord")
	r.tt.Key(0, ui.KeyEnter)
	app.Act.(*recorder).drain(t)
	if got := fa.Places(); len(got) != 1 || got[0] != (data.Place{Kind: data.PlaceNow}) {
		t.Errorf("sent from %+v, want Mine", got)
	}
}

func TestPopoverRowOpensTheTaskInTheMainWindowAndCloses(t *testing.T) {
	app, _, rec := newTestApp(t)
	r := newPopoverRig(t, app)
	if err := r.tt.Click("Reply to design feedback"); err != nil {
		t.Fatal(err)
	}
	if len(rec.opened) != 1 {
		t.Fatalf("opened %v, want the task", rec.opened)
	}
	if task, _ := app.Task(rec.opened[0]); task.Title != "Reply to design feedback" {
		t.Errorf("opened %q", task.Title)
	}
	if r.closed != 1 {
		t.Errorf("the panel closed %d times, want once", r.closed)
	}
}

func TestPopoverOpenLinkBringsTheMainWindowAndCloses(t *testing.T) {
	app, _, rec := newTestApp(t)
	r := newPopoverRig(t, app)
	if err := r.tt.Click("Open ADHD Todo"); err != nil {
		t.Fatal(err)
	}
	if rec.mains != 1 || r.closed != 1 {
		t.Errorf("main window asked %d times, panel closed %d, want once each", rec.mains, r.closed)
	}
}

func TestPopoverEscClosesItOnlyWhenNothingElseTakesIt(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	app, _, _ := assistApp(t, fa)
	r := newPopoverRig(t, app)
	r.tt.Key(0, ui.KeyEscape)
	if r.closed != 1 {
		t.Errorf("Esc closed the panel %d times, want once", r.closed)
	}
}

func TestPopoverHasItsOwnDraftAndReplyApartFromTheWindow(t *testing.T) {
	app, _, _ := newTestApp(t)
	r := newPopoverRig(t, app)
	r.tt.Type("only in the panel")
	win := NewMain(app)
	if win.cmp[surfacePage].buf != "" {
		t.Errorf("the window's composer holds %q", win.cmp[surfacePage].buf)
	}
}

func TestPopoverSpeaksChinese(t *testing.T) {
	app, _, _ := newTestApp(t)
	p := NewPopover(app)
	tt := ui.NewTester(p.View, int(PopoverWidth), 500)
	still(tt)
	inChinese(tt)
	tt.Frame()
	for _, want := range []string{"今天", "打开 ADHD Todo"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	working, stuck := agentCounts(app.buildAgents())
	if working == 0 || stuck == 0 {
		t.Skip("the demo data has no agents at work")
	}
	if !hasSuffixText(tt, "个卡住") {
		t.Errorf("no stuck count in %q", tt.Texts())
	}
}

// hasSuffixText reports whether some text on the screen ends in suffix.
func hasSuffixText(tt *ui.Tester, suffix string) bool {
	for _, s := range tt.Texts() {
		if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}
