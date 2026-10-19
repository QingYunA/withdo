package mcpserver_test

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The user's own tasks (no project, source user) are invisible to agents until the user hands them over.
func TestMCPHidesUserInbox(t *testing.T) {
	r := newRig(t)
	mine := must[*domain.Task](t)(r.core.CreateTask(ctx, domain.CreateTaskInput{Title: "secret", Status: domain.StatusDoing}))
	kid := must[*domain.Task](t)(r.core.CreateTask(ctx, domain.CreateTaskInput{Title: "secret kid", ParentID: &mine.ID}))
	notFound := func(name string, res result) {
		t.Helper()
		if !res.IsError || !strings.Contains(res.Text, "not found") {
			t.Errorf("%s: want not found, got %+v", name, res)
		}
	}
	notFound("get_task", r.call("get_task", map[string]any{"task_id": mine.ID}))
	notFound("update_task", r.call("update_task", map[string]any{"task_id": mine.ID, "title": "x"}))
	notFound("report_progress", r.call("report_progress", map[string]any{"task_id": mine.ID, "summary": "x"}))
	notFound("parent_task_id", r.call("report_progress", map[string]any{"summary": "x", "branch": "b", "parent_task_id": mine.ID}))
	notFound("list parent", r.call("list_tasks", map[string]any{"parent_task_id": mine.ID}))
	notFound("get kid", r.call("get_task", map[string]any{"task_id": kid.ID}))
	if l := r.ok("list_tasks", nil).list("tasks"); len(l) != 0 {
		t.Errorf("list_tasks leaked: %+v", l)
	}
	if strings.Contains(r.ok("get_overview", nil).Text, "secret") {
		t.Error("get_overview leaked")
	}
	got := must[*domain.Task](t)(r.core.GetTask(ctx, mine.ID))
	if got.Title != "secret" {
		t.Errorf("task was modified: %+v", got)
	}

	must[*domain.Task](t)(r.core.HandOff(ctx, mine.ID))
	r.ok("get_task", map[string]any{"task_id": mine.ID})
	if l := r.ok("list_tasks", nil).list("tasks"); len(l) != 1 {
		t.Errorf("handed task should be listed: %+v", l)
	}
	// the user's own subtask of the now-visible parent stays hidden
	if sub := r.ok("get_task", map[string]any{"task_id": mine.ID}).list("subtasks"); len(sub) != 0 {
		t.Errorf("hidden subtask leaked: %+v", sub)
	}
	r.ok("update_task", map[string]any{"task_id": mine.ID, "title": "renamed"})
}
