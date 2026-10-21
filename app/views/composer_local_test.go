package views

import (
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The no-model path of the composer reads the sentence with local rules (internal/quickadd), adds
// each task it names with its fields, and offers one Undo for all of them.

func (r *assistRig) taskTitled(title string) (data.Task, bool) {
	for _, task := range r.app.snap.tasks {
		if task.Title == title {
			return task, true
		}
	}
	return data.Task{}, false
}

func TestWithoutAModelTheSentenceBecomesTasksWithTheirFields(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.ask("提醒我周一9点体检，还有周五前交房租 #life!")
	check, ok1 := r.taskTitled("体检")
	rent, ok2 := r.taskTitled("交房租")
	if !ok1 || !ok2 {
		t.Fatalf("tasks %v %v", ok1, ok2)
	}
	monday := time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local)
	if check.RemindAt == nil || !check.RemindAt.Equal(monday) || check.DueAt != nil || check.Priority != data.P2 {
		t.Errorf("checkup %+v, want a reminder at %v", check, monday)
	}
	friday := time.Date(2026, 10, 16, data.DefaultDueHour, 0, 0, 0, time.Local)
	if rent.DueAt == nil || !rent.DueAt.Equal(friday) || rent.RemindAt != nil || rent.Priority != data.P1 {
		t.Errorf("rent %+v, want due %v and P1", rent, friday)
	}
	if cat, ok := r.app.Category(rent.CategoryID); !ok || cat.Builtin != data.BuiltinLife {
		t.Errorf("rent is in category %+v", cat)
	}
	if check.Source != data.SourceUser || check.ProjectID != nil {
		t.Errorf("checkup %+v", check)
	}
	texts := strings.Join(r.tt.Texts(), "|")
	for _, want := range []string{"Added 2: ", "体检 (Mon 9:00 AM reminder)", "交房租 (due Fri, Life, important)", "Undo"} {
		if !strings.Contains(texts, want) {
			t.Errorf("no %q in %q", want, texts)
		}
	}
}

func TestCmdZTakesBackEveryTaskOfTheSentence(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	before := len(r.app.snap.tasks)
	r.ask("买牛奶；交房租")
	if got := len(r.app.snap.tasks); got != before+2 {
		t.Fatalf("tasks %d, want %d", got, before+2)
	}
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := len(r.app.snap.tasks); got != before {
		t.Errorf("after Cmd+Z %d tasks, want %d", got, before)
	}
}

func TestTheUndoLinkOfTheReceiptTakesBackTheTasks(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	before := len(r.app.snap.tasks)
	r.ask("买牛奶；交房租")
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	if got := len(r.app.snap.tasks); got != before {
		t.Errorf("after Undo %d tasks, want %d", got, before)
	}
	hasNot(t, r.tt, "Added 2")
}

func TestCmdZStopsOfferingTheSentenceWhenItsTimeIsOver(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.ask("买牛奶；交房租")
	before := len(r.app.snap.tasks)
	r.now = r.now.Add(undoWithin + time.Second)
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := len(r.app.snap.tasks); got != before {
		t.Errorf("Cmd+Z took back tasks after the offer ended: %d, want %d", got, before)
	}
}

func TestAPlainSentenceKeepsTheShortReceiptAndCanStillBeUndone(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	before := len(r.app.snap.tasks)
	r.ask("Water the plants")
	has(t, r.tt, "Added to Inbox")
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := len(r.app.snap.tasks); got != before {
		t.Errorf("after Cmd+Z %d tasks, want %d", got, before)
	}
}

func TestOnTheCalendarADayInTheWordsWinsOverTheDayOfThePage(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.m.goTo(nav{screen: screenCalendar})
	r.tt.Frame()
	r.ask("明天九点开会")
	task, ok := r.taskTitled("开会")
	if !ok || task.DueAt == nil {
		t.Fatalf("task %+v", task)
	}
	want := time.Date(2026, 10, 15, 9, 0, 0, 0, time.Local)
	if !task.DueAt.Equal(want) {
		t.Errorf("due %v, want %v", task.DueAt, want)
	}
}

func TestInAProjectTheSentenceGoesToTheProject(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.tt.Frame()
	r.ask("Fix the build tomorrow")
	task, ok := r.taskTitled("Fix the build")
	if !ok || task.ProjectID == nil || *task.ProjectID != 1 || task.DueAt == nil {
		t.Fatalf("task %+v", task)
	}
	has(t, r.tt, "Added 1: Fix the build (due Tomorrow)")
}
