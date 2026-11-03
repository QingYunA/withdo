package core

import (
	"encoding/json"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

// TestCalendarShowsAMilestoneOnItsDay: an open milestone is an item of its due day, with its
// project's name and no task, ahead of the day's tasks.
func TestCalendarShowsAMilestoneOnItsDay(t *testing.T) {
	phone := demoPhone(t)
	projects, err := phone.Projects()
	if err != nil {
		t.Fatal(err)
	}
	var list []data.Project
	if err := json.Unmarshal([]byte(projects), &list); err != nil || len(list) == 0 {
		t.Fatalf("projects: %v %s", err, projects)
	}
	if _, err := phone.CreateMilestone(int64(list[0].ID), "v0.13", "2026-10-22T09:00:00Z"); err != nil {
		t.Fatal(err)
	}
	raw, err := phone.Calendar(2026, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	var got calendarJSON
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range got.Days {
		if d.Date != "2026-10-22" {
			continue
		}
		if len(d.Items) == 0 || d.Items[0].Kind != "milestone" {
			t.Fatalf("the 22nd starts with %+v, want the milestone", d.Items)
		}
		m := d.Items[0]
		found = m.Milestone != nil && m.Milestone.Title == "v0.13" && m.Task == nil && m.Project == list[0].Name
	}
	if !found {
		t.Errorf("the milestone is not on the 22nd of %s", raw)
	}
}
