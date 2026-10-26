package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
)

// createTaskJSON is the composer's input for CreateTask. Every key is optional but title.
//
//	{"title":"...","project_id":1,"parent_id":0,"status":"todo",
//	 "priority":"P2","category_id":0,"due_at":"2026-12-01T09:00:00+08:00"}
//
// An absent or zero project_id and parent_id mean none; an absent or zero category_id means
// unsorted. due_at is ISO 8601, as internal/domain.ParseTimestamp accepts.
type createTaskJSON struct {
	Title      string `json:"title"`
	ProjectID  *int64 `json:"project_id"`
	ParentID   *int64 `json:"parent_id"`
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	CategoryID *int64 `json:"category_id"`
	DueAt      string `json:"due_at"`
}

// CreateTask makes a plain task of the user's, in a project or the Inbox, and returns the created
// task as a data.Task JSON (the same shape Task(id) has under "Task"). It is the composer without
// a model: the source is always the user, and an empty status or priority takes the defaults
// (todo, P2).
func (c *Core) CreateTask(inputJSON string) (string, error) {
	var in createTaskJSON
	if err := json.Unmarshal([]byte(inputJSON), &in); err != nil {
		return "", errors.New("could not read the new task")
	}
	due, err := optionalTime(in.DueAt)
	if err != nil {
		return "", err
	}
	task, err := c.real.CreateTask(data.CreateTaskInput{
		Title: in.Title, Source: data.SourceUser, Status: data.Status(in.Status),
		Priority: data.Priority(in.Priority), ProjectID: intPointer(in.ProjectID),
		ParentID: intPointer(in.ParentID), Category: intValue(in.CategoryID), DueAt: due,
	})
	if err != nil {
		return "", shortError(err)
	}
	return marshal(task)
}

// optional is one field of an update JSON: Set says the key was there, and a JSON null leaves
// Value nil.
type optional[T any] struct {
	Set   bool
	Value *T
}

// UnmarshalJSON marks the field present, then reads its value.
func (o *optional[T]) UnmarshalJSON(raw []byte) error {
	o.Set = true
	if string(raw) == "null" {
		o.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

// updateTaskJSON is the inspector's patch. A key that is absent is left alone; a null or an empty
// string clears a nullable field. category 0 unsorts the task and project_id 0 moves it to the
// Inbox, as the desktop's data.Patch does.
//
//	{"title":"...","notes":"...","status":"doing","priority":"P1",
//	 "due_at":"2026-12-01T09:00:00+08:00","remind_at":null,"category":0,"project_id":2}
type updateTaskJSON struct {
	Title     *string          `json:"title"`
	Notes     optional[string] `json:"notes"`
	Status    *string          `json:"status"`
	Priority  *string          `json:"priority"`
	DueAt     optional[string] `json:"due_at"`
	RemindAt  optional[string] `json:"remind_at"`
	Category  optional[int64]  `json:"category"`
	ProjectID optional[int64]  `json:"project_id"`
}

// UpdateTask applies a patch to a task: title, notes, status, priority, due, remind, category and
// project. A null or empty due_at or remind_at clears it; an empty notes clears the notes; 0
// category unsorts the task and 0 project_id moves it to the Inbox. It returns only an error; read
// the task again with Task(id).
func (c *Core) UpdateTask(id int64, patchJSON string) error {
	var in updateTaskJSON
	if err := json.Unmarshal([]byte(patchJSON), &in); err != nil {
		return errors.New("could not read the change")
	}
	patch := data.Patch{Title: in.Title}
	if in.Notes.Set {
		notes := deref(in.Notes.Value)
		patch.Notes = &notes
	}
	if in.Status != nil {
		status := data.Status(*in.Status)
		patch.Status = &status
	}
	if in.Priority != nil {
		priority := data.Priority(*in.Priority)
		patch.Priority = &priority
	}
	if in.DueAt.Set {
		due, err := clearedTime(in.DueAt.Value)
		if err != nil {
			return err
		}
		patch.DueAt = due
	}
	if in.RemindAt.Set {
		remind, err := clearedTime(in.RemindAt.Value)
		if err != nil {
			return err
		}
		patch.RemindAt = remind
	}
	if in.Category.Set {
		category := intValue(in.Category.Value)
		patch.Category = &category
	}
	if in.ProjectID.Set {
		project := intValue(in.ProjectID.Value)
		patch.ProjectID = &project
	}
	if _, err := c.real.UpdateTask(int(id), patch); err != nil {
		return shortError(err)
	}
	return nil
}

// Complete marks a task done. It is the desktop's UpdateTask with status done: the completion time
// is stamped, a hand-off is cleared, and the open subtasks close with their parent.
func (c *Core) Complete(id int64) error { return c.setDone(id) }

// Accept is the Accept button of From your agents: the same done transition as Complete.
func (c *Core) Accept(id int64) error { return c.setDone(id) }

func (c *Core) setDone(id int64) error {
	done := data.Done
	if _, err := c.real.UpdateTask(int(id), data.Patch{Status: &done}); err != nil {
		return shortError(err)
	}
	return nil
}

// DeleteTask removes a task with its reports and returns the snapshot RestoreTask needs as JSON
// (internal/domain.TaskSnapshot: Task, Reports, Images and Subtasks). The JSON is opaque: pass it
// back unchanged to undo the delete.
func (c *Core) DeleteTask(id int64) (string, error) {
	snap, err := c.dom.SnapshotTask(context.Background(), id)
	if err != nil {
		return "", shortError(err)
	}
	if err := c.real.DeleteTask(int(id)); err != nil {
		return "", shortError(err)
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return "", errors.New("could not record the deleted task")
	}
	return string(raw), nil
}

// RestoreTask puts back a task DeleteTask removed, with its id, fields and reports, from the
// snapshot JSON DeleteTask returned. It is the Undo of DeleteTask.
func (c *Core) RestoreTask(snapshotJSON string) error {
	var snap domain.TaskSnapshot
	if err := json.Unmarshal([]byte(snapshotJSON), &snap); err != nil {
		return errors.New("could not read the deleted task")
	}
	if _, err := c.dom.RestoreTask(context.Background(), snap); err != nil {
		return shortError(err)
	}
	return nil
}

// Reply writes the user's reply on a task. It is filed as a report by agent "you" with kind
// "reply": Task(id) shows it in the activity, the MCP get_task tool lists it with the other
// reports, and it syncs like any report. The task's status, timestamps and a hand-off waiting for
// an agent are left alone, and the reply does not count as the agent's latest word.
func (c *Core) Reply(id int64, text string) error {
	if _, err := c.dom.AddReply(context.Background(), id, text); err != nil {
		return shortError(err)
	}
	return nil
}

// defaultDueHour is the time of day a deadline given as a bare day gets, as in app/views.
const defaultDueHour = 18

// DoToday is the task menu's Do today, the desktop's doToday: it sets the due date of a task of
// mine to today, keeping the time of day it had; a task with no due time gets 18:00, which reads
// as the day alone. It returns the old due date (RFC 3339, "" when there was none) for the
// Undo, which is UpdateTask with that due_at. A task that is not mine, is finished, or is due
// today already is refused, as the menu does not offer it.
func (c *Core) DoToday(id int64) (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	var task *data.Task
	for i := range snap.Tasks {
		if int64(snap.Tasks[i].ID) == id {
			task = &snap.Tasks[i]
			break
		}
	}
	if task == nil {
		return "", errors.New("the task was not found")
	}
	now := c.nowTime()
	if task.Source != data.SourceUser || task.Status.Terminal() ||
		(task.DueAt != nil && data.DaysFrom(*task.DueAt, now) == 0) {
		return "", errors.New("this task cannot be moved to today")
	}
	hour, minute := defaultDueHour, 0
	before := ""
	if task.DueAt != nil {
		l := task.DueAt.In(now.Location())
		hour, minute = l.Hour(), l.Minute()
		before = task.DueAt.Format(time.RFC3339Nano)
	}
	y, mo, d := now.Date()
	due := time.Date(y, mo, d, hour, minute, 0, 0, now.Location())
	if _, err := c.real.UpdateTask(int(id), data.Patch{DueAt: data.TimePtr(due)}); err != nil {
		return "", shortError(err)
	}
	return before, nil
}

// optionalTime parses an ISO 8601 date or datetime; "" and spaces mean no time.
func optionalTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	stored, err := domain.ParseTimestamp(raw)
	if err != nil {
		return nil, errors.New("the date is not an ISO 8601 date or datetime")
	}
	at, err := time.Parse(time.RFC3339Nano, stored)
	if err != nil {
		return nil, errors.New("the date is not an ISO 8601 date or datetime")
	}
	return &at, nil
}

// clearedTime is UpdateTask's time: an absent or empty value clears the field.
func clearedTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return &time.Time{}, nil
	}
	return optionalTime(*value)
}

// deref reads an optional string, with nil as "".
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// intPointer maps a JSON id to data's *int, with 0 as none.
func intPointer(v *int64) *int {
	if v == nil || *v == 0 {
		return nil
	}
	n := int(*v)
	return &n
}

// intValue maps a JSON id to data's int, with nil as 0.
func intValue(v *int64) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

// shortError turns an internal error into a sentence fit to show the user. A domain.FieldError is
// already written for a person; anything else is not allowed to carry a path, SQL or a token.
func shortError(err error) error {
	if err == nil {
		return nil
	}
	var field *domain.FieldError
	if errors.As(err, &field) {
		return errors.New(field.Error())
	}
	switch {
	case errors.Is(err, domain.ErrNoSyncIDs):
		return errors.New("no new ids left; sync first")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return errors.New("the change was cancelled")
	}
	return errors.New("something went wrong")
}
