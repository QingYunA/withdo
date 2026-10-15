package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// An update reads like a create: the same shapes, joined by " · ", with no "Updated" in front.
func TestUpdateReceiptsReadLikeCreates(t *testing.T) {
	at := fixedNow().Add(18*time.Hour + 30*time.Minute) // tomorrow 9:00
	cases := []struct {
		name    string
		changes []data.Change
		want    string
	}{
		{"reminder", []data.Change{{Kind: data.ChangeReminder, At: at}}, "Tomorrow 9:00 AM · reminder"},
		{"due and priority", []data.Change{{Kind: data.ChangeDue, At: at}, {Kind: data.ChangePriority, Value: "P1"}}, "Tomorrow 9:00 AM · due · P1"},
		{"project", []data.Change{{Kind: data.ChangeProject, Value: "adhd-todo"}}, "Moved to adhd-todo"},
		{"inbox", []data.Change{{Kind: data.ChangeProject}}, "Moved to Inbox"},
		{"done", []data.Change{{Kind: data.ChangeStatus, Value: "done"}}, "Marked done"},
		{"status", []data.Change{{Kind: data.ChangeStatus, Value: "review"}}, "Set to In review"},
		{"cleared", []data.Change{{Kind: data.ChangeReminder}, {Kind: data.ChangeDue}}, "Reminder cleared · Due cleared"},
		{"title and notes", []data.Change{{Kind: data.ChangeRenamed}, {Kind: data.ChangeNotes}}, "Renamed · Notes updated"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Text: "Done.", Receipts: []data.Receipt{{
				ID: 1, TaskID: 5, Title: "Write the release notes", Status: data.Todo,
				Detail: data.Detail{Kind: data.DetailUpdated, Changes: c.changes},
			}}}, nil)})
			r.ask("change it")
			r.answer(t)
			has(t, r.tt, c.want)
			hasNot(t, r.tt, "Updated")
		})
	}
}

func TestUpdateReceiptsInChinese(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Text: "好了。", Receipts: []data.Receipt{{
		ID: 1, TaskID: 5, Title: "Write the release notes", Status: data.Todo,
		Detail: data.Detail{Kind: data.DetailUpdated, Changes: []data.Change{{Kind: data.ChangeProject, Value: "adhd-todo"}, {Kind: data.ChangeStatus, Value: "doing"}}},
	}}}, nil)})
	inChinese(r.tt)
	r.ask("改一下")
	r.answer(t)
	has(t, r.tt, "移到adhd-todo · 设为进行中")
	hasNot(t, r.tt, "已更新")
}

// One word per status, wherever it is shown.
func TestEveryStatusHasOneLabel(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	want := map[data.Status]string{
		data.Backlog: "Backlog", data.Todo: "To do", data.Doing: "Doing", data.Blocked: "Blocked",
		data.Review: "In review", data.Done: "Done", data.Cancelled: "Cancelled",
	}
	for s, w := range want {
		if got := statusLabel(r.m.say, s); got != w {
			t.Errorf("%s reads %q, want %q", s, got, w)
		}
	}
}
