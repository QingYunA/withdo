package views

import (
	"slices"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestDraggingOneOfTheSelectedRowsHandsOverAllOfThemAtOnce(t *testing.T) {
	r := newHandRig(t, true)
	cmdClick(t, r.tt, "Decide on the activation policy", "Write the release notes")
	r.tt.Frame()
	r.dragOnto(t, "Write the release notes", "Port core to Go", true)
	r.tt.Frame()
	has(t, r.tt, "Drop to hand to an agent (2)")
	to, _ := r.tt.Find("Port core to Go")
	x, y := center(to)
	r.tt.Release(x, y)
	r.tt.Frame()
	for _, id := range []int{5, 6} {
		if task, _ := r.app.Task(id); task.Source != data.SourceAgent || task.HandedTo != data.AnyAgent {
			t.Errorf("task %d = %+v, want handed over", id, task)
		}
	}
	has(t, r.tt, "Handed 2 tasks to an agent")
	if got := r.m.ho.landed.ids; !slices.Equal(got, []int{6, 5}) && !slices.Equal(got, []int{5, 6}) {
		t.Errorf("landed %v, want both rows", got)
	}
	if m := r.m.multi; m.Active() {
		t.Error("the selection outlived the hand-off")
	}
}

func TestADroppedStackIsUndoneAsOne(t *testing.T) {
	r := newHandRig(t, true)
	cmdClick(t, r.tt, "Decide on the activation policy", "Write the release notes")
	r.tt.Frame()
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	r.tt.Key(ui.Cmd, ui.KeyZ)
	r.tt.Frame()
	for _, id := range []int{5, 6} {
		if task, _ := r.app.Task(id); task.Source != data.SourceUser || task.HandedTo != "" {
			t.Errorf("task %d = %+v after ⌘Z, want the user's again", id, task)
		}
	}
}

func TestDraggingARowThatIsNotSelectedTakesOnlyThatRowAndClearsTheSelection(t *testing.T) {
	r := newHandRig(t, true)
	cmdClick(t, r.tt, "Decide on the activation policy", "Draft the UI spec")
	r.tt.Frame()
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	if task, _ := r.app.Task(5); task.HandedTo == "" {
		t.Error("the dragged row was not handed over")
	}
	if task, _ := r.app.Task(6); task.HandedTo != "" {
		t.Error("a selected row that was not dragged went along")
	}
	if r.m.multi.Active() {
		t.Error("the selection was kept")
	}
	has(t, r.tt, "Handed to an agent")
	hasNot(t, r.tt, "Handed 2 tasks")
}

func TestTheOtherRowsOfAStackFadeAndTheDroppedRowsEachKeepTheWash(t *testing.T) {
	r := newHandRig(t, false)
	cmdClick(t, r.tt, "Decide on the activation policy", "Write the release notes")
	r.tt.Frame()
	r.grab(t, "Write the release notes")
	r.after(40 * time.Millisecond)
	if !r.m.fadesInStack(6) || r.m.fadesInStack(0) {
		t.Errorf("stack %v: the other selected row does not fade", r.m.ho.stack)
	}
	r.tt.Release(300, 300)
	r.tt.Frame()

	r = newHandRig(t, false)
	cmdClick(t, r.tt, "Decide on the activation policy", "Write the release notes")
	r.tt.Frame()
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	for _, id := range []int{5, 6} {
		if r.m.ho.rowWash(id, r.now) <= 0 {
			t.Errorf("row %d has no wash after landing", id)
		}
	}
	// The robot squashes once per task, catchEvery apart: at the start of the second squash it is
	// squashing for the second task.
	if _, ok := r.m.ho.catching(r.now.Add(catchEvery + time.Millisecond)); !ok {
		t.Error("no squash at the second task's beat")
	}
}

func TestDraggingSelectedCardsToALaneChangesAllTheirStatuses(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	m.board[1] = true
	tt.Frame()
	cmdClick(t, tt, "Write the release notes", "Port the reminder watcher")
	tt.Frame()
	dragTo(t, tt, "Write the release notes", "Doing")
	tt.Frame()
	for _, title := range []string{"Write the release notes", "Port the reminder watcher"} {
		if got := taskTitled(t, app, title).Status; got != data.Doing {
			t.Errorf("%q is %s, want doing", title, got)
		}
	}
	if !tt.HasText("Moved 2 tasks to Doing") {
		t.Errorf("no toast in %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := taskTitled(t, app, "Port the reminder watcher").Status; got != data.Todo {
		t.Errorf("⌘Z left it %s", got)
	}
}
