package mcpserver_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// get_task tells an agent that a handed task waits for one, and changes nothing.
func TestGetTaskOfAHandedTaskCarriesTheNoteAndChangesNothing(t *testing.T) {
	r := newRig(t)
	p := must[*domain.Project](t)(r.core.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "acme/widgets"}))
	task := must[*domain.Task](t)(r.core.CreateTask(ctx, domain.CreateTaskInput{Title: "Write the plan", ProjectID: &p.ID}))
	handed := must[*domain.Task](t)(r.core.HandOff(ctx, task.ID))

	res := r.ok("get_task", map[string]any{"task_id": task.ID})
	note, _ := res.Data["note"].(string)
	if !strings.Contains(note, "task_id") || !strings.Contains(note, "Waiting for an agent") {
		t.Errorf("note = %q", note)
	}
	if !strings.Contains(res.Text, note) {
		t.Errorf("text lacks the note: %q", res.Text)
	}

	after := must[*domain.Task](t)(r.core.GetTask(ctx, task.ID))
	if !reflect.DeepEqual(after, handed) {
		t.Errorf("reading changed the task: %+v -> %+v", handed, after)
	}
	if reports := must[[]domain.Report](t)(r.core.ListReports(ctx, task.ID)); len(reports) != 0 {
		t.Errorf("reading filed reports: %+v", reports)
	}
}

func TestGetTaskOfOtherTasksHasNoNote(t *testing.T) {
	r := newRig(t)
	plain := must[*domain.Task](t)(r.core.CreateTask(ctx, domain.CreateTaskInput{Title: "plain", Source: domain.SourceAgent}))
	if res := r.ok("get_task", map[string]any{"task_id": plain.ID}); res.Data["note"] != nil {
		t.Errorf("plain task note = %v", res.Data["note"])
	}

	// A finished task carries no hint.
	old := must[*domain.Task](t)(r.core.CreateTask(ctx, domain.CreateTaskInput{Title: "old"}))
	must[*domain.Task](t)(r.core.HandOff(ctx, old.ID))
	done := domain.StatusDone
	must[*domain.Task](t)(r.core.UpdateTask(ctx, old.ID, domain.TaskPatch{Status: &done}))
	if res := r.ok("get_task", map[string]any{"task_id": old.ID}); res.Data["note"] != nil {
		t.Errorf("finished task note = %v", res.Data["note"])
	}
}

// One piece of work that covers several tasks reports on each: the instructions and the tool say so.
func TestAgentsAreToldToReportOnEveryTask(t *testing.T) {
	r := newRig(t)
	if got := r.session.InitializeResult().Instructions; !strings.Contains(got, "#69 and #70") || !strings.Contains(got, "each of them with its own task_id") {
		t.Errorf("instructions lack the several-tasks sentence: %q", got)
	}
	tools, err := r.session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var desc string
	for _, tool := range tools.Tools {
		if tool.Name == "report_progress" {
			desc = tool.Description
		}
	}
	if !strings.Contains(desc, "several adhd-todo tasks") {
		t.Errorf("report_progress description = %q", desc)
	}
}
