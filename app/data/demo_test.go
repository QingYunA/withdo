package data

import (
	"strconv"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local) }

// openDemo is OpenDemo closed with the test.
func openDemo(t *testing.T, now func() time.Time, seeded bool) *Real {
	t.Helper()
	r, err := OpenDemo(now, seeded)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

// seededDemo is a store with the seed workspace at fixedNow.
func seededDemo(t *testing.T) *Real { t.Helper(); return openDemo(t, fixedNow, true) }

// emptyDemo is a store with nothing in it, at fixedNow.
func emptyDemo(t *testing.T) *Real { t.Helper(); return openDemo(t, fixedNow, false) }

func TestOverviewGroupsActiveTasksByAttention(t *testing.T) {
	f := seededDemo(t)
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
	f := seededDemo(t)
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

func TestSeedProjectIsEmptyAndListed(t *testing.T) {
	f := emptyDemo(t)
	p := f.SeedProject("scratch", "me/scratch")
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

func itoa(n int) string { return strconv.Itoa(n) }
