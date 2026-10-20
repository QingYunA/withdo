package mcpserver_test

import (
	"strings"
	"testing"
)

// TestGetTaskListsTheUsersReply is the MCP half of the Reply contract: an agent reading the task
// sees the user's reply among the reports, so it can answer on its next look.
func TestGetTaskListsTheUsersReply(t *testing.T) {
	r := newRig(t)
	started := r.ok("report_progress", map[string]any{"summary": "working on it", "cwd": r.dir})
	id := int64(started.Data["task_id"].(float64))
	if _, err := r.core.AddReply(ctx, id, "Please cover Android too."); err != nil {
		t.Fatalf("AddReply: %v", err)
	}

	got := r.ok("get_task", map[string]any{"task_id": id})
	reports := got.list("reports")
	if len(reports) != 2 {
		t.Fatalf("get_task reports = %+v", reports)
	}
	reply, _ := reports[len(reports)-1].(map[string]any)
	if reply["agent"] != "you" || reply["kind"] != "reply" || reply["summary"] != "Please cover Android too." {
		t.Errorf("the reply is not listed as the user's: %+v", reply)
	}
	if !strings.Contains(got.Text, "you: Please cover Android too.") {
		t.Errorf("get_task text does not show the reply:\n%s", got.Text)
	}
	if got.obj("task")["status"] != "doing" {
		t.Errorf("the reply changed the task's status: %+v", got.obj("task"))
	}
}
