package mcpserver_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// An agent that was handed a task reports on it by task_id: the task stops waiting and moves to doing.
func TestReportProgressByTaskIDTakesAHandedOffTask(t *testing.T) {
	r := newRig(t)
	p := must[*domain.Project](t)(r.core.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "acme/widgets"}))
	task := must[*domain.Task](t)(r.core.CreateTask(ctx, domain.CreateTaskInput{Title: "Write the plan", ProjectID: &p.ID}))
	must[*domain.Task](t)(r.core.HandOff(ctx, task.ID))

	waiting := r.ok("get_task", map[string]any{"task_id": task.ID}).obj("task")
	if waiting["handed_to"] != domain.AnyAgent || waiting["status"] != "todo" || waiting["source"] != "agent" {
		t.Fatalf("waiting task = %+v", waiting)
	}

	res := r.ok("report_progress", map[string]any{"task_id": task.ID, "status": "doing", "summary": "on it"})
	got := res.obj("task")
	if res.Data["created"] != false || got["status"] != "doing" || got["handed_to"] != nil || got["handed_at"] != nil {
		t.Errorf("after the report: %+v", res.Data)
	}
	stored := r.ok("get_task", map[string]any{"task_id": task.ID}).obj("task")
	if stored["handed_to"] != nil || stored["handed_at"] != nil || stored["status"] != "doing" {
		t.Errorf("stored task = %+v", stored)
	}
}

func must[T any](t *testing.T) func(T, error) T {
	t.Helper()
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}
