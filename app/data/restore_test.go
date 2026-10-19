package data

import (
	"testing"
	"time"
)

// reportOn files a report from an agent against a task.
func reportOn(t *testing.T, s *Real, id int, summary string) {
	t.Helper()
	s.SeedReport(Report{TaskID: id, Agent: "claude-code", Status: Doing, Summary: summary})
}

func TestRestoreTaskPutsBackWhatDeleteTook(t *testing.T) {
	withStore(t, func(t *testing.T, s *Real, _ *time.Time) {
		p := must[Project](t)(s.CreateProject(NewProject{Name: "work"}))
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan", ProjectID: &p.ID, Priority: P1}))
		must[Task](t)(s.UpdateTask(task.ID, Patch{Notes: StrPtr("two pages")}))
		reportOn(t, s, task.ID, "outlined")
		before := must[TaskDetail](t)(s.GetTask(task.ID))

		snap := must[TaskSnapshot](t)(s.SnapshotTask(task.ID))
		if err := s.DeleteTask(task.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetTask(task.ID); err == nil {
			t.Fatal("the task survived its deletion")
		}
		back := must[Task](t)(s.RestoreTask(snap))
		if back.ID != task.ID {
			t.Errorf("restored id %d, want %d", back.ID, task.ID)
		}
		after := must[TaskDetail](t)(s.GetTask(task.ID))
		if after.Task.Title != before.Task.Title || after.Task.Notes != "two pages" || after.Task.Priority != P1 ||
			after.Task.ProjectID == nil || *after.Task.ProjectID != p.ID || !after.Task.UpdatedAt.Equal(before.Task.UpdatedAt) {
			t.Errorf("restored %+v, want %+v", after.Task, before.Task)
		}
		if len(after.Reports) != 1 || after.Reports[0].Summary != "outlined" || after.Reports[0].ID != before.Reports[0].ID {
			t.Errorf("reports %+v, want %+v", after.Reports, before.Reports)
		}
	})
}

func TestRestoreTaskRefusesATaskThatIsBack(t *testing.T) {
	withStore(t, func(t *testing.T, s *Real, _ *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "here"}))
		snap := must[TaskSnapshot](t)(s.SnapshotTask(task.ID))
		if _, err := s.RestoreTask(snap); err == nil {
			t.Error("restored a task that was never deleted")
		}
		if _, err := s.SnapshotTask(9999); err == nil {
			t.Error("snapshotted a task that does not exist")
		}
	})
}
