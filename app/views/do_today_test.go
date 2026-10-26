package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestDoTodayIsInTheMenuOfATaskOfMineAboveMarkImportant(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	before := taskTitled(t, app, "Book dentist")
	rightClick(t, tt, "Book dentist")
	has(t, tt, "Do today", "Mark important", "T")
	if !above(t, tt, "Do today", "Mark important") {
		t.Error("Do today sits above Mark important")
	}
	click(t, tt, "Do today")
	got := taskTitled(t, app, "Book dentist")
	if got.DueAt == nil || data.DaysFrom(*got.DueAt, app.Now()) != 0 {
		t.Fatalf("due %v, want today", got.DueAt)
	}
	if b, a := before.DueAt.Hour(), got.DueAt.In(app.Now().Location()).Hour(); b != before.DueAt.In(app.Now().Location()).Hour() || a != b {
		t.Errorf("the time of day moved: %d to %d", b, a)
	}
	has(t, tt, "Moved to today")
	tt.Key(ui.Cmd, ui.KeyZ)
	if again := taskTitled(t, app, "Book dentist"); again.DueAt == nil || !again.DueAt.Equal(*before.DueAt) {
		t.Errorf("due %v after Undo, want %v", again.DueAt, before.DueAt)
	}
}

func TestDoTodayGivesAnUndatedTaskTheEndOfTheDayAndUndoClearsIt(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	task := taskTitled(t, app, "Look into Liquid Glass for the sidebar")
	if task.DueAt != nil {
		t.Fatalf("fixture task is dated %v", task.DueAt)
	}
	m.sel = task.ID
	tt.Frame()
	tt.Key(0, ui.KeyT)
	tt.Frame()
	got := taskTitled(t, app, "Look into Liquid Glass for the sidebar")
	if got.DueAt == nil || data.DaysFrom(*got.DueAt, app.Now()) != 0 || got.DueAt.In(app.Now().Location()).Hour() != defaultDueHour {
		t.Fatalf("due %v, want today at the end of the working day", got.DueAt)
	}
	click(t, tt, "Undo")
	if again := taskTitled(t, app, "Look into Liquid Glass for the sidebar"); again.DueAt != nil {
		t.Errorf("due %v after Undo, want none", again.DueAt)
	}
}

func TestDoTodayIsHiddenWhenTheTaskIsDueTodayOrIsNotMine(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	rightClick(t, tt, "Decide on the activation policy") // due in five hours
	hasNot(t, tt, "Do today")
	tt.Key(0, ui.KeyEscape)
	rightClick(t, tt, "Fix tray race on launch") // an agent's
	hasNot(t, tt, "Do today")
}
