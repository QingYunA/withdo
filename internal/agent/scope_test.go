package agent

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// system is the system prompt of the first call the model was sent.
func (r *rig) system() string { return r.model.sent(0)[0].Content }

func (r *rig) askIn(sc Scope, text string) Result {
	r.t.Helper()
	res, err := r.agent.Ask(context.Background(), text, sc)
	if err != nil {
		r.t.Fatalf("Ask(%q): %v", text, err)
	}
	return res
}

func (r *rig) project(name string) *domain.Project {
	r.t.Helper()
	p, err := r.core.CreateProject(context.Background(), domain.CreateProjectInput{Name: name})
	if err != nil {
		r.t.Fatal(err)
	}
	return p
}

func TestPromptTellsTheModelWhereTheRequestComesFrom(t *testing.T) {
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC)
	projects := []domain.Project{{ID: 3, Name: "adhd-todo"}}
	notes := "see the redirect"
	due := "2026-10-16T09:00:00.000Z"
	focus := &domain.Task{ID: 21, Title: "Book dentist", Status: domain.StatusTodo, Priority: domain.P2,
		ProjectID: ptr(int64(3)), DueAt: &due, Notes: &notes, Source: domain.SourceUser}
	tests := []struct {
		name  string
		scope Scope
		focus *domain.Task
		want  []string
		not   []string
	}{
		{"now", Scope{}, nil, []string{"Where the request comes from: the Now page", "Inbox unless"}, []string{"Focused task"}},
		{"project", Scope{Kind: ScopeProject, ProjectID: 3}, nil,
			[]string{"the page of the project “adhd-todo” (id 3)", "goes in that project unless the words name another"}, []string{"Focused task"}},
		{"inbox", Scope{Kind: ScopeInbox}, nil, []string{"the Inbox page", "Inbox"}, []string{"Focused task"}},
		{"task", Scope{Kind: ScopeTask, TaskID: 21}, focus, []string{
			"the inspector of one task", "“this”, “it”", "Focused task: id 21", "Book dentist", "status todo",
			"priority P2", "project adhd-todo", "due 2026-10-16", "see the redirect", "update_task",
		}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := BuildPrompt(now, projects, tc.scope, tc.focus)
			for _, w := range tc.want {
				if !strings.Contains(p, w) {
					t.Errorf("prompt lacks %q:\n%s", w, p)
				}
			}
			for _, w := range tc.not {
				if strings.Contains(p, w) {
					t.Errorf("prompt has %q:\n%s", w, p)
				}
			}
		})
	}
}

func TestPromptHasTheInferenceRules(t *testing.T) {
	p := BuildPrompt(time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC), nil, Scope{}, nil)
	for _, want := range []string{
		`"urgent", "asap" or "blocker"`, "P1",
		`"someday"`, "P3",
		`"drop everything"`, `"production is down"`, "P0",
		`"p1"`, // a priority the user writes is taken as written
		`"remind me"`, "remind_at",
		`"by Friday"`, "due_at",
		"morning is 09:00", "afternoon 14:00", "evening 19:00", `"tonight" 20:00`,
		"bare day", "end of day",
		"what you inferred",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}

func TestProjectScopedCreateDefaultsToTheProject(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14,
		call("create_task", m{"title": "Fix the login redirect loop", "priority": "P1", "due_at": "2026-10-16T18:00:00+08:00"}),
		say("Added it to adhd-todo, due Friday, P1."))
	p := r.project("adhd-todo")
	res := r.askIn(Scope{Kind: ScopeProject, ProjectID: p.ID}, "fix the login redirect loop, p1, friday")
	if len(res.Receipts) != 1 {
		t.Fatalf("receipts = %+v", res.Receipts)
	}
	rc := res.Receipts[0]
	tk := r.task(rc.TaskID)
	if tk.ProjectID == nil || *tk.ProjectID != p.ID {
		t.Fatalf("task went to project %v, want %d", tk.ProjectID, p.ID)
	}
	eq(t, "priority", string(tk.Priority), "P1")
	wantMoment(t, rc.Detail, DetailDue, "2026-10-16T18:00:00+08:00")
	eq(t, "receipt priority", rc.Detail.Priority, "P1")
	contains(t, "system prompt", r.system(), "“adhd-todo” (id")
}

func TestProjectScopeLetsTheWordsNameAnotherProjectOrTheInbox(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	here, other := r.project("here"), r.project("other")
	r.model.steps = []step{
		call("create_task", m{"title": "in other", "project_id": other.ID}), say("ok"),
		call("create_task", m{"title": "in inbox", "project_id": 0}), say("ok"),
	}
	sc := Scope{Kind: ScopeProject, ProjectID: here.ID}
	a := r.askIn(sc, "in other: in other").Receipts[0]
	b := r.askIn(sc, "in the inbox: in inbox").Receipts[0]
	if got := r.task(a.TaskID).ProjectID; got == nil || *got != other.ID {
		t.Errorf("named project: got %v, want %d", got, other.ID)
	}
	if got := r.task(b.TaskID).ProjectID; got != nil {
		t.Errorf("project_id 0 must mean the Inbox, got %d", *got)
	}
}

func TestInboxAndNowScopesCreateInTheInbox(t *testing.T) {
	for _, sc := range []Scope{{}, {Kind: ScopeInbox}} {
		r := newRig(t, "Asia/Shanghai", oct14, call("create_task", m{"title": "loose"}), say("ok"))
		r.project("a-project")
		rc := r.askIn(sc, "loose").Receipts[0]
		if got := r.task(rc.TaskID).ProjectID; got != nil {
			t.Errorf("scope %+v: task went to project %d", sc, *got)
		}
	}
}

func TestAnArchivedOrMissingProjectScopeFallsBackToTheInbox(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, call("create_task", m{"title": "x"}), say("ok"))
	rc := r.askIn(Scope{Kind: ScopeProject, ProjectID: 999}, "x").Receipts[0]
	if got := r.task(rc.TaskID).ProjectID; got != nil {
		t.Errorf("task went to project %d", *got)
	}
	contains(t, "prompt", r.system(), "the Now page")
}

func TestTaskScopedUpdateChangesTheFocusedTask(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	tk := r.mustTask(domain.CreateTaskInput{Title: "Book dentist"})
	r.model.steps = []step{
		call("update_task", m{"task_id": tk.ID, "remind_at": "2026-10-16T14:00:00+08:00", "priority": "P1"}),
		say("Moved it to Friday 2 PM."),
	}
	res := r.askIn(Scope{Kind: ScopeTask, TaskID: tk.ID}, "move it to friday afternoon, urgent")
	got := r.task(tk.ID)
	eq(t, "remind_at", str(got.RemindAt), "2026-10-16T06:00:00.000Z")
	eq(t, "priority", string(got.Priority), "P1")
	d := res.Receipts[0].Detail
	if d.Kind != DetailUpdated || len(d.Changes) != 2 {
		t.Errorf("detail = %+v", d)
	}
	sys := r.system()
	contains(t, "system prompt", sys, "Focused task: id "+strconv.FormatInt(tk.ID, 10))
	contains(t, "system prompt", sys, "Book dentist")
}

func TestTaskScopeOfATaskThatIsGoneIsAnError(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, say("never"))
	_, err := r.agent.Ask(context.Background(), "hi", Scope{Kind: ScopeTask, TaskID: 404})
	if err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Errorf("err = %v", err)
	}
}

func TestReceiptsShowWhatWasInferred(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14,
		call("create_task", m{"title": "Review PR 12", "priority": "P1", "remind_at": "2026-10-15T09:00:00+08:00", "due_at": "2026-10-15T10:00:00+08:00"}),
		call("create_task", m{"title": "Plain"}),
		call("create_task", m{"title": "Someday", "priority": "P3"}),
		say("ok"))
	res := r.ask("three things")
	both, plain, low := res.Receipts[0].Detail, res.Receipts[1].Detail, res.Receipts[2].Detail
	wantMoment(t, both, DetailReminder, "2026-10-15T09:00:00+08:00")
	wantDue, _ := time.Parse(time.RFC3339, "2026-10-15T10:00:00+08:00")
	if !both.Due.Equal(wantDue) || both.Priority != "P1" {
		t.Errorf("reminder and due: %+v", both)
	}
	if plain.Priority != "" || !plain.Due.IsZero() {
		t.Errorf("a plain task has nothing inferred: %+v", plain)
	}
	if low.Kind != DetailAdded || low.Priority != "P3" {
		t.Errorf("low priority: %+v", low)
	}
}
