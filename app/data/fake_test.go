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
	if len(ov.Groups) != 2 || ov.Groups[0].Project.Name != "adhd-todo" || ov.Groups[1].Project.Name != "answer-me-with-html" {
		t.Fatalf("groups %+v", ov.Groups)
	}
	first := ov.Groups[0].Tasks
	if first[0].Status != Blocked || first[1].Status != Review || first[2].Status != Doing {
		t.Errorf("order %v %v %v, want blocked, review, doing", first[0].Status, first[1].Status, first[2].Status)
	}
	if !first[0].Stale || first[2].Stale {
		t.Errorf("stale flags: blocked %v, doing %v", first[0].Stale, first[2].Stale)
	}
	if g := ov.Groups[1].Tasks[0]; g.Stale || g.LatestReport == nil || g.LatestReport.Summary != "Writing tests for the reply parser" {
		t.Errorf("second project's task %+v", g)
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

	f.CheckReminders() // task 7 came due at the seed's "now"
	f.CheckReminders()
	if len(got) != 1 || got[0].TaskID != 7 || got[0].ProjectName != "" {
		t.Errorf("reminders %+v, want one for task 7", got)
	}
}

func TestAddProjectIsEmptyAndListed(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	p := f.AddProject("scratch", "me/scratch")
	ps, _ := f.ListProjects()
	if len(ps) != 1 || ps[0].ID != p.ID || ps[0].Name != "scratch" {
		t.Errorf("projects %+v", ps)
	}
	if ts, _ := f.ListTasks(Filter{}); len(ts) != 0 {
		t.Errorf("tasks %v", ts)
	}
}

func TestDaysFromCountsCalendarDays(t *testing.T) {
	now := fixedNow() // 2:30 PM
	for d, want := range map[time.Duration]int{
		-30 * time.Minute:    0,
		9 * time.Hour:        0, // 11:30 PM
		10 * time.Hour:       1, // 12:30 AM
		-15 * time.Hour:      -1,
		3 * 24 * time.Hour:   3,
		-10 * 24 * time.Hour: -10,
	} {
		if got := DaysFrom(now.Add(d), now); got != want {
			t.Errorf("DaysFrom(now%+v) = %d, want %d", d, got, want)
		}
	}
}
