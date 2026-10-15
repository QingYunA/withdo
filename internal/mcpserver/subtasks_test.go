package mcpserver_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReportProgressParentTaskID(t *testing.T) {
	r := newRig(t)
	parent := r.ok("report_progress", map[string]any{"summary": "main work"})
	kid := r.ok("report_progress", map[string]any{
		"summary": "sub work", "branch": "worktree-agent-1", "parent_task_id": parent.Data["task_id"],
	})
	if kid.Data["created"] != true || kid.obj("task")["parent_id"] != parent.Data["task_id"] {
		t.Fatalf("subtask report: %+v", kid.Data)
	}
	if parent.obj("task")["parent_id"] != nil {
		t.Errorf("a top-level task has no parent_id: %+v", parent.obj("task"))
	}
}

func TestReportProgressParentTaskIDErrors(t *testing.T) {
	r := newRig(t)
	for name, args := range map[string]map[string]any{
		"unknown parent": {"summary": "s", "parent_task_id": 999},
		"zero parent":    {"summary": "s", "parent_task_id": 0},
	} {
		res := r.call("report_progress", args)
		if !res.IsError || !strings.Contains(res.Text, "parent_task_id") {
			t.Errorf("%s: %+v", name, res)
		}
	}
}

func TestGetTaskListsSubtasks(t *testing.T) {
	r := newRig(t)
	parent := r.ok("report_progress", map[string]any{"summary": "main work"})
	pid := parent.Data["task_id"]
	a := r.ok("report_progress", map[string]any{"summary": "first sub", "branch": "sub-a", "parent_task_id": pid})
	r.ok("report_progress", map[string]any{"summary": "second sub", "branch": "sub-b", "parent_task_id": pid, "status": "review"})

	got := r.ok("get_task", map[string]any{"task_id": pid})
	subs := got.list("subtasks")
	if len(subs) != 2 {
		t.Fatalf("subtasks = %+v", subs)
	}
	first := subs[0].(map[string]any)
	if first["id"] != a.Data["task_id"] || first["title"] != "first sub" || first["status"] != "doing" {
		t.Errorf("first subtask = %+v", first)
	}
	if second := subs[1].(map[string]any); second["status"] != "review" {
		t.Errorf("second subtask = %+v", second)
	}
	if !strings.Contains(got.Text, "Subtasks") || !strings.Contains(got.Text, "first sub") {
		t.Errorf("text lacks the subtasks: %q", got.Text)
	}

	none := r.ok("get_task", map[string]any{"task_id": a.Data["task_id"]})
	if l := none.list("subtasks"); l == nil || len(l) != 0 {
		t.Errorf("a task without subtasks lists an empty array, got %+v", none.Data["subtasks"])
	}
}

func TestListTasksByParentTaskID(t *testing.T) {
	r := newRig(t)
	parent := r.ok("report_progress", map[string]any{"summary": "main work"})
	pid := parent.Data["task_id"]
	kid := r.ok("report_progress", map[string]any{"summary": "sub", "branch": "sub-a", "parent_task_id": pid})
	r.ok("create_task", map[string]any{"title": "unrelated"})

	got := r.ok("list_tasks", map[string]any{"parent_task_id": pid})
	tasks := got.list("tasks")
	if len(tasks) != 1 || tasks[0].(map[string]any)["id"] != kid.Data["task_id"] {
		t.Fatalf("tasks = %+v", tasks)
	}
	if res := r.call("list_tasks", map[string]any{"parent_task_id": 0}); !res.IsError {
		t.Error("parent_task_id 0 should be rejected")
	}
}

func TestInstructionsExplainSubagents(t *testing.T) {
	r := newRig(t)
	got := r.session.InitializeResult().Instructions
	for _, w := range []string{
		"pass title on your first report",
		"give it your task_id",
		"parent_task_id",
		"reports status \"done\" when it hands its work back",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("instructions lack %q", w)
		}
	}
	if n := len(strings.Split(got, "\n")); n > 20 {
		t.Errorf("instructions have %d lines; keep them short", n)
	}
}

func TestToolSchemasAcceptParentTaskID(t *testing.T) {
	r := newRig(t)
	list, err := r.session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, tool := range list.Tools {
		schema, _ := json.Marshal(tool.InputSchema)
		found[tool.Name] = strings.Contains(string(schema), `"parent_task_id"`)
	}
	if !found["report_progress"] || !found["list_tasks"] {
		t.Errorf("parent_task_id in schemas: %v", found)
	}
}
