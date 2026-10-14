package data

import (
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local) }

func TestOverviewGroupsActiveTasksByAttention(t *testing.T) {
	f := NewFake(fixedNow)
	ov, err := f.Overview()
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Groups) != 2 || ov.Groups[0].Project.Name != "adhd-todo" || ov.Groups[1].Project.Name != "docs" {
		t.Fatalf("groups %+v", ov.Groups)
	}
	first := ov.Groups[0].Tasks
	if first[0].Status != Blocked || first[1].Status != Review || first[2].Status != Doing {
		t.Errorf("order %v %v %v, want blocked, review, doing", first[0].Status, first[1].Status, first[2].Status)
	}
	if !first[0].Stale || first[2].Stale {
		t.Errorf("stale flags: blocked %v, doing %v", first[0].Stale, first[2].Stale)
	}
	if g := ov.Groups[1].Tasks[0]; !g.Stale || g.HoursSinceActivity != 54 {
		t.Errorf("docs task stale=%v hours=%d, want true and 54", g.Stale, g.HoursSinceActivity)
	}
}

func TestUpdateTaskCompletesAndRearmsReminder(t *testing.T) {
	f := NewFake(fixedNow)
	task, err := f.UpdateTask(7, Patch{Status: StatusPtr(Done)})
	if err != nil || task.CompletedAt == nil {
		t.Fatalf("done: %+v, %v", task, err)
	}
	task, _ = f.UpdateTask(7, Patch{Status: StatusPtr(Todo)})
	if task.CompletedAt != nil {
		t.Error("completed_at survives reopening")
	}
	if _, err := f.UpdateTask(7, Patch{Title: StrPtr("")}); err == nil {
		t.Error("an empty title is accepted")
	}
}

func TestSubscribersHearChangesAndReminders(t *testing.T) {
	now := fixedNow()
	f := NewFake(func() time.Time { return now })
	changes := 0
	cancel := f.Subscribe(func() { changes++ })
	var got []Reminder
	f.SubscribeReminders(func(r Reminder) { got = append(got, r) })

	if _, err := f.CreateTask(CreateTaskInput{Title: "New"}); err != nil {
		t.Fatal(err)
	}
	cancel()
	f.UpdateTask(1, Patch{Notes: StrPtr("x")})
	if changes != 1 {
		t.Errorf("heard %d changes, want 1 (the second came after cancel)", changes)
	}

	now = now.Add(2 * time.Hour) // task 7 reminds in one hour
	f.CheckReminders()
	f.CheckReminders()
	if len(got) != 1 || got[0].TaskID != 7 || got[0].ProjectName != "adhd-todo" {
		t.Errorf("reminders %+v, want one for task 7", got)
	}
}

func TestWordsForTimes(t *testing.T) {
	now := fixedNow()
	cases := []struct {
		got, want string
	}{
		{Age(now.Add(-30*time.Second), now), "now"},
		{Age(now.Add(-2*time.Minute), now), "2m"},
		{Age(now.Add(-3*time.Hour), now), "3h"},
		{Age(now.Add(-48*time.Hour), now), "2d"},
		{Age(now.Add(-10*24*time.Hour), now), "Oct 4"},
		{When(now.Add(time.Hour), now), "in 1h"},
		{When(now.Add(20*time.Minute), now), "in 20m"},
		{When(now.Add(24*time.Hour), now), "tomorrow"},
		{When(now.Add(-2*time.Hour), now), "2h ago"},
		{DayWord(now.Add(3*24*time.Hour), now), "Saturday"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
