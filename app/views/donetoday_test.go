package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestMineShowsDoneTodayLastWithWhoFinishedItAndWhen(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	for _, want := range []string{"Done today", "Port the MCP server", "Claude Code · PR #14 merged", "1:50 PM",
		"Send the packaging answer", "You", "11:30 AM"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if !above(t, tt, "Later", "Done today") || !above(t, tt, "Done today", "Port the MCP server") ||
		!above(t, tt, "Port the MCP server", "Send the packaging answer") {
		t.Error("Done today comes last, newest first")
	}
	if tt.HasText("Draft the UI spec") {
		t.Error("yesterday's task is in Done today")
	}
	if !tt.HasText("4") { // the sidebar's Mine count is what needs attention, not what is done
		t.Errorf("count missing in %q", tt.Texts())
	}
}

func TestDoneTodayStopsAtFiveAndLinksToCompleted(t *testing.T) {
	f := demoStore(t, fixedNow, false)
	for i := 0; i < 7; i++ {
		task, _ := f.CreateTask(data.CreateTaskInput{Title: "Finished " + itoa(i)})
		_, _ = f.UpdateTask(task.ID, data.Patch{Status: data.StatusPtr(data.Done)})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1400)
	tt.Frame()
	rows := 0
	for _, s := range tt.Texts() {
		if len(s) > 9 && s[:9] == "Finished " {
			rows++
		}
	}
	if rows != 5 || !tt.HasText("Done today") || !tt.HasText("7") {
		t.Errorf("%d rows, texts %q", rows, tt.Texts())
	}
	if tt.HasText("Nothing needs you right now") != true {
		t.Error("Done today alone leaves Mine clear: nothing needs me")
	}
	if err := tt.Click("Show all in Completed"); err != nil {
		t.Fatal(err)
	}
	if m.nav.screen != screenCompleted {
		t.Errorf("went to %+v", m.nav)
	}
}

func TestDoneTodayIsAbsentWithNothingFinished(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	if tt.HasText("Done today") {
		t.Error("an empty section is drawn")
	}
}

func TestDoneRowsNameWhoFinishedThemOnlyInCompleted(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	for _, want := range []string{"Port the MCP server", "Today", "Draft the UI spec", "Yesterday"} {
		if !tt.HasText(want) {
			t.Errorf("list: missing %q in %q", want, tt.Texts())
		}
	}
	onlyInSidebar(t, tt, "Claude Code") // the columns say who finished a row
	m.goTo(nav{screen: screenCompleted})
	tt.Frame()
	for _, want := range []string{"Claude Code", "You", "1:50 PM"} {
		if !tt.HasText(want) {
			t.Errorf("Completed: missing %q in %q", want, tt.Texts())
		}
	}
}

func TestClickingMyOwnGlyphCompletesInOneClickAndItJoinsDoneToday(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	click(t, tt, "Mark done: Book dentist")
	if got, _ := app.Task(8); got.Status != data.Done || got.CompletedAt == nil {
		t.Fatalf("status %v", got.Status)
	}
	if _, playing := app.completing[8]; !playing {
		t.Error("no completion animation was started")
	}
	app.completing = map[int]completion{} // the animation is over
	tt.Frame()
	if !above(t, tt, "Done today", "Book dentist") {
		t.Errorf("the finished task did not move to Done today: %q", tt.Texts())
	}
	if m.pop.open {
		t.Error("a menu opened too")
	}
}

func TestCompletionFillsTheGlyphUnlessMotionIsReduced(t *testing.T) {
	app, _, _ := newTestApp(t)
	task, _ := app.Task(8)
	app.SetStatus(task, data.Done)
	app.Now = func() time.Time { return fixedNow().Add(100 * time.Millisecond) } // partway through the 220 ms
	var fill float32
	for _, c := range []struct {
		reduce bool
		check  func(float32) bool
		want   string
	}{
		{false, func(f float32) bool { return f > 0 && f < 1 }, "between 0 and 1: the fill is animated"},
		{true, func(f float32) bool { return f == 1 }, "1: the plain, full fill"},
	} {
		tt := ui.NewTester(func(c *ui.Context) { fill = app.completionPhase(c, data.Task{ID: 8, Status: data.Todo}).fill }, 10, 10)
		tt.SetPreferences(ui.Preferences{ReduceMotion: c.reduce})
		tt.Frame()
		if !c.check(fill) {
			t.Errorf("reduce motion %v: fill %v, want %s", c.reduce, fill, c.want)
		}
	}
}

func TestSidebarRowsAre32PointsAndNamesStayOnOneLine(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	for _, name := range []string{"adhd-todo", "answer-me-with-html", "picpocket"} {
		r, ok := tt.Find(name)
		if !ok || r.H > 20.5 || r.X+r.W > 216 {
			t.Errorf("%s at %+v: one 20 pt line inside the 216 pt sidebar", name, r)
		}
	}
	a, _ := tt.Find("adhd-todo")
	b, _ := tt.Find("answer-me-with-html")
	if d := b.Y - a.Y; d < 31.5 || d > 32.5 {
		t.Errorf("rows are %v pt apart, want 32", d)
	}
}

func TestDoneTodayFitsBesideTheInspectorAtTheMinimumWindow(t *testing.T) {
	tt, m, _ := newMainTester(t, 860)
	tt.SetSize(860, 1400)
	m.OpenTask(21)
	tt.Frame()
	if !tt.HasText("Done today") || !tt.HasText("Send the packaging answer") {
		t.Errorf("narrow: %q", tt.Texts())
	}
	r, _ := tt.Find("Send the packaging answer")
	if r.W <= 0 || r.X+r.W > 860-361 {
		t.Errorf("the title is at %+v: it must stay left of the inspector", r)
	}
}
