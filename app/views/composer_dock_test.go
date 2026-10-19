package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

const dockPlaceholder = "⌘K" // the composer's hint, on every page that has one

// The composer is pinned at the bottom edge of the content column, under the page.
func TestTheComposerIsPinnedAtTheBottomUnderThePage(t *testing.T) {
	for name, n := range map[string]nav{
		"now":      {},
		"agents":   {screen: screenAgents},
		"inbox":    {screen: screenInbox},
		"calendar": {screen: screenCalendar},
		"project":  {screen: screenProject, projectID: 1},
	} {
		t.Run(name, func(t *testing.T) {
			r := newAssistRig(t, &data.FakeAssistant{Ready: true})
			r.m.goTo(n)
			r.tt.Frame()
			box, ok := r.tt.Find(dockPlaceholder)
			if !ok {
				t.Fatalf("no composer in %q", r.tt.Texts())
			}
			if box.Y < 720-120 {
				t.Errorf("the composer is at %v of 720; it belongs at the bottom edge", box.Y)
			}
			if top := 720 - r.m.dockHeight; box.Y < top || box.Y > 720 {
				t.Errorf("the composer at %v is outside its dock, which starts at %v", box.Y, top)
			}
		})
	}
}

// The last row of a long page stops above the composer when scrolled to the end.
func TestTheLastRowIsNotBehindTheComposer(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Frame()
	r.tt.Scroll(500, 300, 0, 3000)
	r.tt.Frame()
	box, ok := r.tt.Find(dockPlaceholder)
	if !ok {
		t.Fatal("no composer")
	}
	row, ok := r.tt.Find("Done today")
	if !ok {
		t.Fatalf("no last section in %q", r.tt.Texts())
	}
	if row.Y+row.H > box.Y {
		t.Errorf("the last section ends at %v, under the composer at %v", row.Y+row.H, box.Y)
	}
}

// Cmd+K still puts the cursor in the composer, now at the bottom.
func TestCmdKFocusesTheBottomComposer(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Frame()
	r.tt.Type("hello")
	if got := r.m.cmp[surfacePage].buf; got != "hello" {
		t.Errorf("typed into the composer: %q", got)
	}
}
