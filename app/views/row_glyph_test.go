package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// A group says the status, so its rows have no glyph at rest. While the pointer is over a row, its
// own glyph button shows under the header's glyph, so a task can be completed from the list.

func TestAGroupedRowHasNoGlyphAtRestAndMarksMyTaskDoneWhenHovered(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	hasNot(t, tt, "Mark done: Write the release notes")
	hover(t, tt, "Write the release notes")
	has(t, tt, "Mark done: Write the release notes")
	if err := tt.Click("Mark done: Write the release notes"); err != nil {
		t.Fatal(err)
	}
	if got := taskTitled(t, app, "Write the release notes").Status; got != data.Done {
		t.Fatalf("status %v, want done", got)
	}
	if m.pop.open || m.openID != 0 {
		t.Error("the click also opened the menu or the inspector")
	}
}

func TestTheHoverGlyphGoesWhenThePointerLeavesTheRow(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	hover(t, tt, "Write the release notes")
	has(t, tt, "Mark done: Write the release notes")
	tt.Move(2, 700)
	tt.Frame()
	tt.Frame()
	hasNot(t, tt, "Mark done: Write the release notes")
}

func TestTheHoverGlyphOfAnAgentsTaskOpensTheMenuInsteadOfCompleting(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	hasNot(t, tt, "Status: To do")
	hover(t, tt, "Port the reminder watcher")
	if err := tt.Click("Status: To do"); err != nil {
		t.Fatal(err)
	}
	if got := taskTitled(t, app, "Port the reminder watcher").Status; got != data.Todo {
		t.Fatalf("status %v, an agent's task was completed", got)
	}
	if !m.pop.open || m.pop.task != 10 || !tt.HasText("Move to…") {
		t.Errorf("menu open %v for task %d", m.pop.open, m.pop.task)
	}
}

func TestTheHoverGlyphSitsUnderTheHeadersGlyphAndShiftsNothing(t *testing.T) {
	for _, c := range []struct{ title, glyph string }{
		{"Write the release notes", "Mark done: Write the release notes"},
		{"Port the reminder watcher", "Status: To do"},
	} {
		tt, _, _ := projectTester(t, 1)
		r0 := rectOf(t, tt, c.title)
		hover(t, tt, c.title)
		if r := rectOf(t, tt, c.title); !near(r.X, r0.X) || !near(r.Y, r0.Y) {
			t.Errorf("%q moved to (%v, %v) when hovered, was (%v, %v)", c.title, r.X, r.Y, r0.X, r0.Y)
		}
		g := rectOf(t, tt, c.glyph)
		// A title sits two indents in from the header's glyph, which is 18 pt wide.
		if gx, want := g.X+g.W/2, r0.X-2*listIndent+theme.GlyphSize/2; !near(gx, want) {
			t.Errorf("%q: glyph centered at x %v, want %v under the header's glyph", c.title, gx, want)
		}
		if g.X+g.W > r0.X {
			t.Errorf("%q: the glyph button reaches %v, over the row's content at %v", c.title, g.X+g.W, r0.X)
		}
	}
}

func TestTheHoverGlyphStaysWhileTheRowCompletes(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	moving(tt)
	hover(t, tt, "Write the release notes")
	click(t, tt, "Mark done: Write the release notes")
	tt.Move(2, 700)
	at(tt, app, ms(100))
	has(t, tt, "Mark done: Write the release notes") // the fill plays with the pointer away
}

func TestSelectModeShowsNoHoverGlyph(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	m.multi.EnterMode()
	tt.Frame()
	hover(t, tt, "Write the release notes")
	hasNot(t, tt, "Mark done: Write the release notes")
}

func rectOf(t *testing.T, tt *ui.Tester, text string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q in %q", text, tt.Texts())
	}
	return r
}

// gutter moves the pointer into the glyph column beside the row of title: under the header's glyph,
// left of the row, level with its title.
func gutter(t *testing.T, tt *ui.Tester, title string) {
	t.Helper()
	r := rectOf(t, tt, title)
	tt.Move(r.X-2*listIndent+theme.GlyphSize/2, r.Y+r.H/2)
	tt.Frame()
	tt.Frame()
}

func TestAPointerInTheGlyphColumnBesideARowShowsItsGlyphAndCompletesIt(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	gutter(t, tt, "Write the release notes")
	has(t, tt, "Mark done: Write the release notes")
	if err := tt.Click("Mark done: Write the release notes"); err != nil {
		t.Fatal(err)
	}
	if got := taskTitled(t, app, "Write the release notes").Status; got != data.Done {
		t.Fatalf("status %v, want done", got)
	}
	if m.openID != 0 {
		t.Error("the click opened the inspector")
	}
}

func TestAPointerInTheGlyphColumnShowsTheGlyphOfThatRowAlone(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	gutter(t, tt, "Write the release notes")
	has(t, tt, "Mark done: Write the release notes")
	hasNot(t, tt, "Mark done: Decide on the activation policy")
	gutter(t, tt, "Decide on the activation policy")
	has(t, tt, "Mark done: Decide on the activation policy")
	hasNot(t, tt, "Mark done: Write the release notes")
}
