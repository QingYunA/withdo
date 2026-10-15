package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The inspector puts a small ✦ after a value the assistant set in the last 10 seconds.

func countMarks(r *assistRig) int {
	n := 0
	for _, s := range r.tt.Texts() {
		if s == "✦" {
			n++
		}
	}
	return n
}

func TestInspectorMarksWhatTheAssistantJustSetForTenSeconds(t *testing.T) {
	due := fixedNow().Add(48 * time.Hour)
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{
		Text: "Moved it.",
		Receipts: []data.Receipt{{ID: 1, TaskID: 5, Title: "Write the release notes", Status: data.Todo,
			Detail: data.Detail{Kind: data.DetailUpdated, Changes: []data.Change{
				{Kind: data.ChangeDue, At: due}, {Kind: data.ChangePriority, Value: "P1"},
			}}}},
	}, nil)})
	r.ask("move the release notes to friday, urgent")
	r.answer(t)
	r.m.OpenTask(5)
	r.tt.Frame()
	if got := countMarks(r); got != 2 {
		t.Fatalf("%d marks right after the change, want 2 (due and priority)", got)
	}
	r.now = r.now.Add(9 * time.Second)
	r.tt.Frame()
	if got := countMarks(r); got != 2 {
		t.Errorf("%d marks after 9 s, want them still", got)
	}
	r.now = r.now.Add(2 * time.Second)
	r.tt.Frame()
	if got := countMarks(r); got != 0 {
		t.Errorf("%d marks after 11 s, want none", got)
	}
}

func TestACreatedTaskIsMarkedForWhatWasInferred(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{
		Text: "Added.",
		Receipts: []data.Receipt{{ID: 2, TaskID: 6, Title: "Reply to Anna", Status: data.Todo, Bell: true,
			Detail: data.Detail{Kind: data.DetailReminder, At: fixedNow().Add(18 * time.Hour), Priority: "P1"}}},
	}, nil)})
	r.ask("remind me to reply to Anna tomorrow, urgent")
	r.answer(t)
	r.m.OpenTask(6)
	r.tt.Frame()
	if got := countMarks(r); got != 2 {
		t.Errorf("%d marks, want 2 (remind and priority)", got)
	}
}

func TestOnlyTheTaskTheAssistantChangedIsMarkedAndUndoClearsIt(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{
		Text: "Done.",
		Receipts: []data.Receipt{{ID: 3, TaskID: 5, Title: "Write the release notes", Status: data.Todo,
			Detail: data.Detail{Kind: data.DetailUpdated, Changes: []data.Change{{Kind: data.ChangePriority, Value: "P0"}}}}},
	}, nil)})
	r.ask("release notes are urgent")
	r.answer(t)
	r.m.OpenTask(7)
	r.tt.Frame()
	if got := countMarks(r); got != 0 {
		t.Errorf("%d marks on a task the assistant did not touch", got)
	}
	r.m.OpenTask(5)
	r.tt.Frame()
	if got := countMarks(r); got != 1 {
		t.Fatalf("%d marks, want 1", got)
	}
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if got := countMarks(r); got != 0 {
		t.Errorf("%d marks after Undo", got)
	}
}
