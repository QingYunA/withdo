package domain

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/store"
)

const (
	taskCols = "id, project_id, milestone_id, title, notes, status, priority, due_at, remind_at, " +
		"branch, pr_url, pr_number, source, created_at, updated_at, completed_at, reminded_at"
	// DefaultTaskLimit is how many tasks ListTasks returns when no limit is given.
	DefaultTaskLimit = 50
)

var prURLPattern = regexp.MustCompile(`^https?://github\.com/[^/\s]+/[^/\s]+/pull/(\d+)(?:[/?#].*)?$`)

// ParsePRNumber extracts the number from a GitHub pull request URL.
func ParsePRNumber(url string) (int64, bool) {
	m := prURLPattern.FindStringSubmatch(strings.TrimSpace(url))
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	return n, err == nil
}

func prNumber(url *string) *int64 {
	if url == nil {
		return nil
	}
	if n, ok := ParsePRNumber(*url); ok {
		return &n
	}
	return nil
}

func scanTask(s scanner) (Task, error) {
	var t Task
	err := s.Scan(&t.ID, &t.ProjectID, &t.MilestoneID, &t.Title, &t.Notes, &t.Status, &t.Priority,
		&t.DueAt, &t.RemindAt, &t.Branch, &t.PRURL, &t.PRNumber, &t.Source,
		&t.CreatedAt, &t.UpdatedAt, &t.CompletedAt, &t.RemindedAt)
	return t, err
}

func getTask(ctx context.Context, q store.Querier, id int64) (*Task, error) {
	return one(q.QueryRowContext(ctx, "SELECT "+taskCols+" FROM tasks WHERE id = ?", id), scanTask)
}

// GetTask returns the task with the given id, or nil.
func (c *Core) GetTask(ctx context.Context, id int64) (*Task, error) {
	t, err := getTask(ctx, c.db, id)
	if err != nil {
		return nil, fmt.Errorf("get task %d: %w", id, err)
	}
	return t, nil
}

// CreateTaskInput describes a new task. Zero values mean "default": status
// todo, priority P2, no project (the inbox). DueAt and RemindAt are in stored
// form (see ParseTimestamp).
type CreateTaskInput struct {
	Title       string
	Source      Source
	ProjectID   *int64
	MilestoneID *int64
	Notes       *string
	Status      Status
	Priority    Priority
	DueAt       *string
	RemindAt    *string
	Branch      *string
	PRURL       *string
}

// CreateTask inserts a task. A milestone with no project assigns the
// milestone's project; a milestone from a different project is rejected.
func (c *Core) CreateTask(ctx context.Context, in CreateTaskInput) (*Task, error) {
	in, err := withDefaults(in)
	if err != nil {
		return nil, err
	}
	var out *Task
	err = c.write(ctx, func(q store.Querier) error {
		out, err = c.insertTask(ctx, q, in)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return out, nil
}

func withDefaults(in CreateTaskInput) (CreateTaskInput, error) {
	if in.Title == "" {
		return in, invalid("title", "must not be empty")
	}
	if in.Status == "" {
		in.Status = StatusTodo
	}
	if in.Priority == "" {
		in.Priority = P2
	}
	if in.Source == "" {
		in.Source = SourceUser
	}
	switch {
	case !in.Status.Valid():
		return in, invalid("status", "must be one of %s", joinStatuses())
	case !in.Priority.Valid():
		return in, invalid("priority", "must be one of P0, P1, P2, P3")
	}
	return in, nil
}

func (c *Core) insertTask(ctx context.Context, q store.Querier, in CreateTaskInput) (*Task, error) {
	projectID, err := checkRefs(ctx, q, in.ProjectID, in.MilestoneID)
	if err != nil {
		return nil, err
	}
	ts := c.stamp()
	var completed *string
	if in.Status.Terminal() {
		completed = &ts
	}
	res, err := q.ExecContext(ctx, `INSERT INTO tasks (project_id, milestone_id, title, notes, status, priority,
		due_at, remind_at, branch, pr_url, pr_number, source, created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		projectID, in.MilestoneID, in.Title, in.Notes, in.Status, in.Priority,
		in.DueAt, in.RemindAt, in.Branch, in.PRURL, prNumber(in.PRURL), in.Source, ts, ts, completed)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}
	return getTask(ctx, q, id)
}

// checkRefs validates the project and milestone references and returns the
// project to store, inferred from the milestone when none is given.
func checkRefs(ctx context.Context, q store.Querier, projectID, milestoneID *int64) (*int64, error) {
	if projectID != nil {
		p, err := getProject(ctx, q, *projectID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, notFound("project_id", "project", *projectID)
		}
	}
	if milestoneID == nil {
		return projectID, nil
	}
	m, err := getMilestone(ctx, q, *milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, notFound("milestone_id", "milestone", *milestoneID)
	}
	if projectID != nil && m.ProjectID != *projectID {
		return nil, invalid("milestone_id", "milestone %d belongs to another project", *milestoneID)
	}
	if projectID == nil {
		return &m.ProjectID, nil
	}
	return projectID, nil
}

func joinStatuses() string {
	names := make([]string, len(Statuses))
	for i, s := range Statuses {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}

// TaskPatch changes a task. Pointer fields are left alone when nil; Nullable
// fields are left alone when omitted and cleared with Clear().
type TaskPatch struct {
	Title       *string
	Status      *Status
	Priority    *Priority
	Notes       Nullable[string]
	ProjectID   Nullable[int64]
	MilestoneID Nullable[int64]
	DueAt       Nullable[string]
	RemindAt    Nullable[string]
	Branch      Nullable[string]
	PRURL       Nullable[string]
}

// UpdateTask patches a task in place. Moving to done/cancelled stamps
// completed_at once; moving back to an open status clears it. Writing
// remind_at, even to the same value, re-arms its reminder. pr_number always
// follows pr_url.
func (c *Core) UpdateTask(ctx context.Context, id int64, patch TaskPatch) (*Task, error) {
	var out *Task
	err := c.write(ctx, func(q store.Querier) error {
		var err error
		out, err = c.updateTask(ctx, q, id, patch)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("update task %d: %w", id, err)
	}
	return out, nil
}

func (c *Core) updateTask(ctx context.Context, q store.Querier, id int64, patch TaskPatch) (*Task, error) {
	cur, err := getTask(ctx, q, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, notFound("task_id", "task", id)
	}
	next, err := applyPatch(*cur, patch)
	if err != nil {
		return nil, err
	}
	projectID, err := checkRefs(ctx, q, next.ProjectID, next.MilestoneID)
	if err != nil {
		return nil, err
	}
	ts := c.stamp()
	next.UpdatedAt = ts
	next.CompletedAt = completedAt(*cur, next.Status, ts)
	if patch.RemindAt.Present {
		next.RemindedAt = nil
	}
	_, err = q.ExecContext(ctx, `UPDATE tasks SET project_id = ?, milestone_id = ?, title = ?, notes = ?, status = ?,
		priority = ?, due_at = ?, remind_at = ?, reminded_at = ?, branch = ?, pr_url = ?, pr_number = ?,
		updated_at = ?, completed_at = ? WHERE id = ?`,
		projectID, next.MilestoneID, next.Title, next.Notes, next.Status, next.Priority, next.DueAt,
		next.RemindAt, next.RemindedAt, next.Branch, next.PRURL, prNumber(next.PRURL), ts, next.CompletedAt, id)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return getTask(ctx, q, id)
}

// applyPatch returns cur with patch applied and validated; cur is not modified.
func applyPatch(cur Task, p TaskPatch) (Task, error) {
	next := cur
	if p.Title != nil {
		if *p.Title == "" {
			return cur, invalid("title", "must not be empty")
		}
		next.Title = *p.Title
	}
	if p.Status != nil {
		if !p.Status.Valid() {
			return cur, invalid("status", "must be one of %s", joinStatuses())
		}
		next.Status = *p.Status
	}
	if p.Priority != nil {
		if !p.Priority.Valid() {
			return cur, invalid("priority", "must be one of P0, P1, P2, P3")
		}
		next.Priority = *p.Priority
	}
	next.Notes = p.Notes.apply(cur.Notes)
	next.ProjectID = p.ProjectID.apply(cur.ProjectID)
	next.MilestoneID = p.MilestoneID.apply(cur.MilestoneID)
	next.DueAt = p.DueAt.apply(cur.DueAt)
	next.RemindAt = p.RemindAt.apply(cur.RemindAt)
	next.Branch = p.Branch.apply(cur.Branch)
	next.PRURL = p.PRURL.apply(cur.PRURL)
	return next, nil
}

// completedAt keeps the original completion time while a task stays closed.
func completedAt(cur Task, status Status, ts string) *string {
	switch {
	case !status.Terminal():
		return nil
	case cur.Status.Terminal():
		return cur.CompletedAt
	default:
		return &ts
	}
}

// TaskFilter narrows ListTasks. Zero values mean "no filter".
type TaskFilter struct {
	ProjectID   *int64
	Inbox       bool // only tasks without a project
	Statuses    []Status
	MilestoneID *int64
	Limit       int // DefaultTaskLimit when 0
}

// ListTasks returns matching tasks, newest first.
func (c *Core) ListTasks(ctx context.Context, f TaskFilter) ([]Task, error) {
	var where []string
	var args []any
	if f.ProjectID != nil {
		where, args = append(where, "project_id = ?"), append(args, *f.ProjectID)
	}
	if f.Inbox {
		where = append(where, "project_id IS NULL")
	}
	if f.MilestoneID != nil {
		where, args = append(where, "milestone_id = ?"), append(args, *f.MilestoneID)
	}
	if len(f.Statuses) > 0 {
		where = append(where, "status IN ("+strings.TrimSuffix(strings.Repeat("?,", len(f.Statuses)), ",")+")")
		for _, s := range f.Statuses {
			args = append(args, string(s))
		}
	}
	query := "SELECT " + taskCols + " FROM tasks"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	limit := f.Limit
	if limit <= 0 {
		limit = DefaultTaskLimit
	}
	rows, err := c.db.QueryContext(ctx, query+" ORDER BY id DESC LIMIT ?", append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return collect(rows, scanTask)
}

// findOpenTask returns the newest unfinished task on branch within a project
// (nil project = inbox).
func findOpenTask(ctx context.Context, q store.Querier, projectID *int64, branch string) (*Task, error) {
	return one(q.QueryRowContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE project_id IS ? AND branch = ?
		AND status NOT IN ('done','cancelled') ORDER BY id DESC LIMIT 1`, projectID, branch), scanTask)
}

// DeleteTask removes a task and its progress reports. It exists for Undo of a
// task the built-in agent just created; a missing task is a not-found error.
func (c *Core) DeleteTask(ctx context.Context, id int64) error {
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getTask(ctx, q, id)
		if err != nil {
			return err
		}
		if cur == nil {
			return notFound("task_id", "task", id)
		}
		if _, err := q.ExecContext(ctx, "DELETE FROM reports WHERE task_id = ?", id); err != nil {
			return fmt.Errorf("delete reports: %w", err)
		}
		if _, err := q.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id); err != nil {
			return fmt.Errorf("delete task: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	return nil
}
