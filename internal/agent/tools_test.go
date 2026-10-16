package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/cloudwego/eino/schema"
)

func TestUndoCreate(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, call("create_task", m{"title": "water plants", "priority": "P1"}), say("ok"))
	res := r.ask("water plants")
	id := res.Receipts[0]
	if err := r.agent.Undo(context.Background(), id.ID); err != nil {
		t.Fatal(err)
	}
	if tk := r.task(id.TaskID); tk != nil {
		t.Errorf("task survived Undo: %+v", tk)
	}
	if err := r.agent.Undo(context.Background(), id.ID); !errors.Is(err, ErrNothingToUndo) {
		t.Errorf("second Undo = %v, want ErrNothingToUndo", err)
	}
}

func TestUndoCreateOfATaskThatIsAlreadyGone(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, call("create_task", m{"title": "x"}), say("ok"))
	res := r.ask("x")
	if err := r.core.DeleteTask(context.Background(), res.Receipts[0].TaskID); err != nil {
		t.Fatal(err)
	}
	if err := r.agent.Undo(context.Background(), res.Receipts[0].ID); err != nil {
		t.Errorf("Undo of a vanished task = %v, want nil", err)
	}
}

func TestUndoUpdateRestoresOnlyWhatChanged(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	old := "2026-10-20T01:00:00.000Z"
	notes := "keep me"
	tk := r.mustTask(domain.CreateTaskInput{Title: "old title", Notes: &notes, RemindAt: &old, Priority: domain.P3})
	r.model.steps = []step{
		call("update_task", m{"task_id": tk.ID, "title": "new title", "remind_at": "2026-10-15T09:00:00+08:00", "priority": "P0"}),
		say("ok"),
	}
	res := r.ask("rename and move the reminder")
	d := res.Receipts[0].Detail
	if d.Kind != DetailUpdated || len(d.Changes) != 3 || d.Changes[0].Kind != ChangeReminder ||
		d.Changes[1].Kind != ChangePriority || d.Changes[1].Value != "P0" || d.Changes[2].Kind != ChangeRenamed {
		t.Errorf("detail = %+v", d)
	}
	wantReminder := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	if !d.Changes[0].At.Equal(wantReminder) {
		t.Errorf("reminder at %v, want %v", d.Changes[0].At, wantReminder)
	}
	got := r.task(tk.ID)
	eq(t, "title", got.Title, "new title")
	eq(t, "priority", string(got.Priority), "P0")

	// Someone else edits the notes after the agent: Undo must not touch them.
	changed := "edited by hand"
	if _, err := r.core.UpdateTask(context.Background(), tk.ID, domain.TaskPatch{Notes: domain.Set(changed)}); err != nil {
		t.Fatal(err)
	}
	if err := r.agent.Undo(context.Background(), res.Receipts[0].ID); err != nil {
		t.Fatal(err)
	}
	got = r.task(tk.ID)
	eq(t, "title", got.Title, "old title")
	eq(t, "priority", string(got.Priority), "P3")
	eq(t, "remind_at", str(got.RemindAt), old)
	eq(t, "notes", str(got.Notes), changed)
}

func TestUndoUpdateThatClearedAField(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	due := "2026-10-20T01:00:00.000Z"
	tk := r.mustTask(domain.CreateTaskInput{Title: "t", DueAt: &due})
	r.model.steps = []step{call("update_task", m{"task_id": tk.ID, "clear": []string{"due_at"}}), say("ok")}
	res := r.ask("no deadline")
	if d := res.Receipts[0].Detail; d.Kind != DetailUpdated || len(d.Changes) != 1 || d.Changes[0].Kind != ChangeDue || !d.Changes[0].At.IsZero() {
		t.Errorf("detail = %+v, want a cleared due time", d)
	}
	if r.task(tk.ID).DueAt != nil {
		t.Fatal("due_at not cleared")
	}
	if err := r.agent.Undo(context.Background(), res.Receipts[0].ID); err != nil {
		t.Fatal(err)
	}
	eq(t, "due_at", str(r.task(tk.ID).DueAt), due)
}

func TestUndoMoveBetweenProjectsDropsAndRestoresMilestone(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	ctx := context.Background()
	p1, _ := r.core.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "me/one"})
	p2, _ := r.core.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "me/two"})
	ms, err := r.core.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p1.ID, Title: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	tk := r.mustTask(domain.CreateTaskInput{Title: "t", ProjectID: &p1.ID, MilestoneID: &ms.ID})
	r.model.steps = []step{call("update_task", m{"task_id": tk.ID, "project_id": p2.ID}), say("moved")}
	res := r.ask("move it to two")
	got := r.task(tk.ID)
	if got.ProjectID == nil || *got.ProjectID != p2.ID || got.MilestoneID != nil {
		t.Fatalf("after move: project %v milestone %v", got.ProjectID, got.MilestoneID)
	}
	if err := r.agent.Undo(ctx, res.Receipts[0].ID); err != nil {
		t.Fatal(err)
	}
	got = r.task(tk.ID)
	if got.ProjectID == nil || *got.ProjectID != p1.ID || got.MilestoneID == nil || *got.MilestoneID != ms.ID {
		t.Errorf("after undo: project %v milestone %v", got.ProjectID, got.MilestoneID)
	}
}

func TestCreateTaskTakesACategory(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, call("create_task", m{"title": "pay rent", "category": "life"}), say("ok"))
	res := r.ask("pay rent")
	d := res.Receipts[0].Detail
	eq(t, "receipt category", d.Category, "life")
	got := r.task(res.Receipts[0].TaskID)
	if got.Category == nil || *got.Category != domain.CategoryLife {
		t.Errorf("category = %v, want life", got.Category)
	}
}

func TestCreateTaskWithoutACategoryLeavesItOut(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, call("create_task", m{"title": "x"}), say("ok"))
	res := r.ask("x")
	eq(t, "receipt category", res.Receipts[0].Detail.Category, "")
	if got := r.task(res.Receipts[0].TaskID); got.Category != nil {
		t.Errorf("category = %v, want none", *got.Category)
	}
}

func TestCreateTaskRejectsAnUnknownCategory(t *testing.T) {
	var saw string
	r := newRig(t, "Asia/Shanghai", oct14, call("create_task", m{"title": "x", "category": "play"}),
		func(_ context.Context, in []*schema.Message) (*schema.Message, error) {
			saw = lastToolText(in)
			return schema.AssistantMessage("sorry", nil), nil
		})
	r.ask("x")
	contains(t, "tool result", saw, "category")
}

func TestUpdateCategoryAndUndo(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	tk := r.mustTask(domain.CreateTaskInput{Title: "t", Category: ptr(domain.CategoryWork)})
	r.model.steps = []step{call("update_task", m{"task_id": tk.ID, "category": "life"}), say("ok")}
	res := r.ask("that is a life thing")
	d := res.Receipts[0].Detail
	if d.Kind != DetailUpdated || len(d.Changes) != 1 || d.Changes[0].Kind != ChangeCategory || d.Changes[0].Value != "life" {
		t.Errorf("detail = %+v", d)
	}
	if got := r.task(tk.ID).Category; got == nil || *got != domain.CategoryLife {
		t.Fatalf("category = %v, want life", got)
	}
	if err := r.agent.Undo(context.Background(), res.Receipts[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := r.task(tk.ID).Category; got == nil || *got != domain.CategoryWork {
		t.Errorf("after undo category = %v, want work", got)
	}
}

func TestClearCategoryAndUndo(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	tk := r.mustTask(domain.CreateTaskInput{Title: "t", Category: ptr(domain.CategoryLife)})
	r.model.steps = []step{call("update_task", m{"task_id": tk.ID, "clear": []string{"category"}}), say("ok")}
	res := r.ask("unsort it")
	if d := res.Receipts[0].Detail; len(d.Changes) != 1 || d.Changes[0].Kind != ChangeCategory || d.Changes[0].Value != "" {
		t.Errorf("detail = %+v, want a cleared category", d)
	}
	if r.task(tk.ID).Category != nil {
		t.Fatal("category not cleared")
	}
	if err := r.agent.Undo(context.Background(), res.Receipts[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := r.task(tk.ID).Category; got == nil || *got != domain.CategoryLife {
		t.Errorf("after undo category = %v, want life", got)
	}
}

func TestTaskViewShowsTheCategory(t *testing.T) {
	tb := &toolbox{now: func() time.Time { return oct14 }, journal: newJournal()}
	life := domain.CategoryLife
	eq(t, "category", tb.view(domain.Task{Category: &life}).Category, "life")
	eq(t, "none", tb.view(domain.Task{}).Category, "")
}

func TestUndoComplete(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	tk := r.mustTask(domain.CreateTaskInput{Title: "t", Status: domain.StatusDoing})
	r.model.steps = []step{call("complete_task", m{"task_id": tk.ID}), say("done")}
	res := r.ask("finish it")
	if res.Receipts[0].Detail.Kind != DetailDone {
		t.Errorf("detail = %+v", res.Receipts[0].Detail)
	}
	if got := r.task(tk.ID); got.Status != domain.StatusDone || got.CompletedAt == nil {
		t.Fatalf("not completed: %+v", got)
	}
	if err := r.agent.Undo(context.Background(), res.Receipts[0].ID); err != nil {
		t.Fatal(err)
	}
	got := r.task(tk.ID)
	eq(t, "status", string(got.Status), "doing")
	if got.CompletedAt != nil {
		t.Errorf("completed_at = %v after undo", *got.CompletedAt)
	}
}

func TestCompleteAnAlreadyDoneTaskMakesNoReceipt(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	tk := r.mustTask(domain.CreateTaskInput{Title: "t", Status: domain.StatusDone})
	r.model.steps = []step{call("complete_task", m{"task_id": tk.ID}), say("it was done")}
	if res := r.ask("finish it"); len(res.Receipts) != 0 {
		t.Errorf("receipts = %+v", res.Receipts)
	}
}

func TestToolFailuresReachTheModel(t *testing.T) {
	cases := []struct {
		name string
		step step
		want string
	}{
		{"unknown task", call("update_task", m{"task_id": 999, "title": "x"}), "not found"},
		{"nothing to change", call("update_task", m{"task_id": 1}), "nothing to change"},
		{"bad clear", call("update_task", m{"task_id": 1, "clear": []string{"title"}}), "cannot clear"},
		{"bad priority", call("create_task", m{"title": "x", "priority": "P9"}), "priority"},
		{"empty title", call("create_task", m{"title": "  "}), "title"},
		{"unknown project", call("create_task", m{"title": "x", "project_id": 77}), "not found"},
		{"complete unknown", call("complete_task", m{"task_id": 999}), "not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var saw string
			r := newRig(t, "Asia/Shanghai", oct14, tc.step, func(_ context.Context, in []*schema.Message) (*schema.Message, error) {
				saw = lastToolText(in)
				return schema.AssistantMessage("sorry", nil), nil
			})
			r.mustTask(domain.CreateTaskInput{Title: "exists"})
			res := r.ask("go")
			contains(t, "tool result", saw, `"ok":false`)
			contains(t, "tool result", saw, tc.want)
			if len(res.Receipts) != 0 {
				t.Errorf("receipts = %+v", res.Receipts)
			}
		})
	}
}

func TestFindTasks(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	ctx := context.Background()
	p, _ := r.core.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "me/app"})
	r.mustTask(domain.CreateTaskInput{Title: "Fix login bug", ProjectID: &p.ID})
	r.mustTask(domain.CreateTaskInput{Title: "Fix signup bug", Source: domain.SourceAgent})
	r.mustTask(domain.CreateTaskInput{Title: "Fix old bug", Status: domain.StatusDone})
	tb := &toolbox{core: r.core, now: r.clock.now, journal: newJournal(), projects: []domain.Project{*p}}

	ids := func(a findArgs) []string {
		out, err := tb.find(ctx, a)
		if err != nil || !out.OK {
			t.Fatalf("find: %v %+v", err, out)
		}
		var titles []string
		for _, v := range out.Tasks {
			titles = append(titles, v.Title)
		}
		return titles
	}
	eq(t, "open bugs", strings.Join(ids(findArgs{Query: "fix BUG"}), "|"), "Fix signup bug|Fix login bug")
	eq(t, "by source", strings.Join(ids(findArgs{Source: "agent"}), "|"), "Fix signup bug")
	eq(t, "by project", strings.Join(ids(findArgs{ProjectID: &p.ID}), "|"), "Fix login bug")
	eq(t, "done on request", strings.Join(ids(findArgs{Statuses: []string{"done"}}), "|"), "Fix old bug")
	eq(t, "no match", strings.Join(ids(findArgs{Query: "zzz"}), "|"), "")
}

func TestFindCapsTheList(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	for i := 0; i < maxFound+5; i++ {
		r.mustTask(domain.CreateTaskInput{Title: "bulk"})
	}
	tb := &toolbox{core: r.core, now: r.clock.now, journal: newJournal()}
	out, _ := tb.find(context.Background(), findArgs{})
	if len(out.Tasks) != maxFound || out.Note == "" {
		t.Errorf("got %d tasks, note %q", len(out.Tasks), out.Note)
	}
}

func TestAskUserNormalizesOptions(t *testing.T) {
	tb := &toolbox{}
	_, _ = tb.ask(context.Background(), askArgs{Question: " Which? ", Options: []string{"a", " a ", "", "b", "c", "d", "e"}})
	q := tb.asked()
	eq(t, "question", q.Text, "Which?")
	eq(t, "options", strings.Join(q.Options, ","), "a,b,c,d")
}

func TestListProjectsSkipsArchived(t *testing.T) {
	arch := "2026-01-01T00:00:00.000Z"
	tb := &toolbox{projects: []domain.Project{{ID: 1, Name: "a"}, {ID: 2, Name: "b", ArchivedAt: &arch}}}
	out, _ := tb.listProjects(context.Background(), projectsArgs{})
	if len(out.Projects) != 1 || out.Projects[0].Name != "a" {
		t.Errorf("projects = %+v", out.Projects)
	}
	eq(t, "inbox name", tb.projectName(nil), "Inbox")
	eq(t, "missing project", tb.projectName(ptr(int64(9))), "project 9")
}

func ptr[T any](v T) *T { return &v }

func TestRevertPatchTouchesOnlyAppliedFields(t *testing.T) {
	notes := "n"
	before := domain.Task{ID: 1, Title: "t", Status: domain.StatusTodo, Priority: domain.P2, Notes: &notes}
	done := domain.StatusDone
	applied := domain.TaskPatch{Status: &done, DueAt: domain.Set("2026-10-20T00:00:00.000Z")}
	got := revertPatch(before, applied)
	if got.Status == nil || *got.Status != domain.StatusTodo {
		t.Errorf("status = %v", got.Status)
	}
	if !got.DueAt.Present || got.DueAt.Value != nil {
		t.Errorf("due_at should be cleared: %+v", got.DueAt)
	}
	if got.Title != nil || got.Priority != nil || got.Notes.Present || got.RemindAt.Present {
		t.Errorf("untouched fields in the revert: %+v", got)
	}
}
