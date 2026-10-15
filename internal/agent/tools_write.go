package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

type createArgs struct {
	Title     string  `json:"title" jsonschema:"description=Short task title. For a reminder the thing to remember."`
	ProjectID *int64  `json:"project_id,omitempty" jsonschema:"description=Project id from list_projects. Omit for the Inbox."`
	Notes     *string `json:"notes,omitempty"`
	Priority  string  `json:"priority,omitempty" jsonschema:"enum=P0,enum=P1,enum=P2,enum=P3,description=P0 is urgent and P2 the default."`
	DueAt     string  `json:"due_at,omitempty" jsonschema:"description=Deadline as RFC 3339 with a UTC offset."`
	RemindAt  string  `json:"remind_at,omitempty" jsonschema:"description=When to remind the user as RFC 3339 with a UTC offset."`
}

func (tb *toolbox) create(ctx context.Context, a createArgs) (reply, error) {
	now := tb.now()
	in := domain.CreateTaskInput{
		Title: strings.TrimSpace(a.Title), Source: domain.SourceUser, ProjectID: a.ProjectID,
		Notes: a.Notes, Priority: domain.Priority(a.Priority),
	}
	var err error
	if in.DueAt, err = optWhen("due_at", a.DueAt, now, false); err != nil {
		return fail(ctx, err)
	}
	if in.RemindAt, err = optWhen("remind_at", a.RemindAt, now, true); err != nil {
		return fail(ctx, err)
	}
	t, err := tb.core.CreateTask(ctx, in)
	if err != nil {
		return fail(ctx, err)
	}
	tb.addReceipt(Receipt{
		TaskID: t.ID, Title: t.Title, Status: string(t.Status), Bell: t.RemindAt != nil,
		Detail: tb.createDetail(*t),
	}, undoCreate(tb.core, t.ID))
	v := tb.view(*t)
	return reply{OK: true, Task: &v}, nil
}

func optWhen(field, s string, now time.Time, future bool) (*string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	stored, err := parseWhen(field, s, now, future)
	if err != nil {
		return nil, err
	}
	return &stored, nil
}

func (tb *toolbox) createDetail(t domain.Task) Detail {
	now := tb.now()
	switch {
	case t.RemindAt != nil:
		return Detail{Kind: DetailReminder, At: storedTime(*t.RemindAt, now)}
	case t.DueAt != nil:
		return Detail{Kind: DetailDue, At: storedTime(*t.DueAt, now)}
	}
	return Detail{Kind: DetailAdded, Project: tb.projectLabel(t.ProjectID)}
}

type updateArgs struct {
	TaskID    int64    `json:"task_id" jsonschema:"description=Id from find_tasks."`
	Title     *string  `json:"title,omitempty"`
	Status    *string  `json:"status,omitempty" jsonschema:"enum=backlog,enum=todo,enum=doing,enum=blocked,enum=review,enum=done,enum=cancelled"`
	Priority  *string  `json:"priority,omitempty" jsonschema:"enum=P0,enum=P1,enum=P2,enum=P3"`
	Notes     *string  `json:"notes,omitempty"`
	ProjectID *int64   `json:"project_id,omitempty" jsonschema:"description=Move the task to this project."`
	DueAt     *string  `json:"due_at,omitempty" jsonschema:"description=RFC 3339 with a UTC offset."`
	RemindAt  *string  `json:"remind_at,omitempty" jsonschema:"description=RFC 3339 with a UTC offset."`
	Clear     []string `json:"clear,omitempty" jsonschema:"description=Fields to empty: notes or project or due_at or remind_at."`
}

func (tb *toolbox) update(ctx context.Context, a updateArgs) (reply, error) {
	before, err := tb.core.GetTask(ctx, a.TaskID)
	if err != nil || before == nil {
		if err == nil {
			err = fmt.Errorf("task %d not found", a.TaskID)
		}
		return fail(ctx, err)
	}
	patch, err := tb.patchFrom(a, *before)
	if err != nil {
		return fail(ctx, err)
	}
	after, err := tb.core.UpdateTask(ctx, a.TaskID, patch)
	if err != nil {
		return fail(ctx, err)
	}
	tb.addReceipt(Receipt{
		TaskID: after.ID, Title: after.Title, Status: string(after.Status), Bell: patch.RemindAt.Present && after.RemindAt != nil,
		Detail: tb.changeDetail(patch, *after),
	}, undoPatch(tb.core, *before, patch))
	v := tb.view(*after)
	return reply{OK: true, Task: &v}, nil
}

// patchFrom turns the model's arguments into a domain patch, validating times and enums.
func (tb *toolbox) patchFrom(a updateArgs, before domain.Task) (domain.TaskPatch, error) {
	now := tb.now()
	p := domain.TaskPatch{Title: a.Title}
	if a.Status != nil {
		s := domain.Status(*a.Status)
		p.Status = &s
	}
	if a.Priority != nil {
		pr := domain.Priority(*a.Priority)
		p.Priority = &pr
	}
	if a.Notes != nil {
		p.Notes = domain.Set(*a.Notes)
	}
	if a.ProjectID != nil {
		p.ProjectID = domain.Set(*a.ProjectID)
		if before.MilestoneID != nil {
			p.MilestoneID = domain.Clear[int64]() // a milestone belongs to one project
		}
	}
	var err error
	if p.DueAt, err = nullableWhen("due_at", a.DueAt, now, false); err != nil {
		return p, err
	}
	if p.RemindAt, err = nullableWhen("remind_at", a.RemindAt, now, true); err != nil {
		return p, err
	}
	for _, f := range a.Clear {
		switch f {
		case "notes":
			p.Notes = domain.Clear[string]()
		case "project":
			p.ProjectID, p.MilestoneID = domain.Clear[int64](), domain.Clear[int64]()
		case "due_at":
			p.DueAt = domain.Clear[string]()
		case "remind_at":
			p.RemindAt = domain.Clear[string]()
		default:
			return p, fmt.Errorf("cannot clear %q: use notes, project, due_at or remind_at", f)
		}
	}
	if isEmptyPatch(p) {
		return p, fmt.Errorf("nothing to change: pass at least one field")
	}
	return p, nil
}

func nullableWhen(field string, s *string, now time.Time, future bool) (domain.Nullable[string], error) {
	if s == nil {
		return domain.Nullable[string]{}, nil
	}
	stored, err := parseWhen(field, *s, now, future)
	if err != nil {
		return domain.Nullable[string]{}, err
	}
	return domain.Set(stored), nil
}

func isEmptyPatch(p domain.TaskPatch) bool {
	return p.Title == nil && p.Status == nil && p.Priority == nil && !p.Notes.Present && !p.ProjectID.Present &&
		!p.DueAt.Present && !p.RemindAt.Present
}

// changeDetail lists what an update changed, in the order the receipt says it.
func (tb *toolbox) changeDetail(p domain.TaskPatch, after domain.Task) Detail {
	now := tb.now()
	var changes []Change
	if p.RemindAt.Present {
		changes = append(changes, Change{Kind: ChangeReminder, At: storedOrZero(after.RemindAt, now)})
	}
	if p.DueAt.Present {
		changes = append(changes, Change{Kind: ChangeDue, At: storedOrZero(after.DueAt, now)})
	}
	if p.Status != nil {
		changes = append(changes, Change{Kind: ChangeStatus, Value: string(after.Status)})
	}
	if p.Priority != nil {
		changes = append(changes, Change{Kind: ChangePriority, Value: string(after.Priority)})
	}
	if p.ProjectID.Present {
		changes = append(changes, Change{Kind: ChangeProject, Value: tb.projectLabel(after.ProjectID)})
	}
	if p.Title != nil {
		changes = append(changes, Change{Kind: ChangeRenamed})
	}
	if p.Notes.Present {
		changes = append(changes, Change{Kind: ChangeNotes})
	}
	return Detail{Kind: DetailUpdated, Changes: changes}
}

// storedOrZero is a stored timestamp as a time, or the zero time for a cleared field.
func storedOrZero(stored *string, now time.Time) time.Time {
	if stored == nil {
		return time.Time{}
	}
	return storedTime(*stored, now)
}

type completeArgs struct {
	TaskID int64 `json:"task_id" jsonschema:"description=Id from find_tasks."`
}

func (tb *toolbox) complete(ctx context.Context, a completeArgs) (reply, error) {
	before, err := tb.core.GetTask(ctx, a.TaskID)
	if err != nil || before == nil {
		if err == nil {
			err = fmt.Errorf("task %d not found", a.TaskID)
		}
		return fail(ctx, err)
	}
	if before.Status.Terminal() {
		v := tb.view(*before)
		return reply{OK: true, Task: &v, Note: "already " + string(before.Status)}, nil
	}
	done := domain.StatusDone
	patch := domain.TaskPatch{Status: &done}
	after, err := tb.core.UpdateTask(ctx, a.TaskID, patch)
	if err != nil {
		return fail(ctx, err)
	}
	tb.addReceipt(Receipt{TaskID: after.ID, Title: after.Title, Status: string(after.Status), Detail: Detail{Kind: DetailDone}},
		undoPatch(tb.core, *before, patch))
	v := tb.view(*after)
	return reply{OK: true, Task: &v}, nil
}
