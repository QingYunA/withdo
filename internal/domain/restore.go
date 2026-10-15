package domain

import (
	"context"
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// TaskSnapshot is a task with its reports, taken before it is deleted so it can be put back.
type TaskSnapshot struct {
	Task    Task
	Reports []Report // oldest first
}

// SnapshotTask reads a task and its reports as they are now; a missing task is a not-found error.
func (c *Core) SnapshotTask(ctx context.Context, id int64) (*TaskSnapshot, error) {
	t, err := c.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, notFound("task_id", "task", id)
	}
	rs, err := c.ListReports(ctx, id)
	if err != nil {
		return nil, err
	}
	return &TaskSnapshot{Task: *t, Reports: rs}, nil
}

// RestoreTask puts a snapshotted task back with its id, every field and its reports, in one
// transaction. It is the Undo of DeleteTask. A task that is already there (its id was taken
// meanwhile) is a conflict. A project or milestone that is gone since leaves the task in the Inbox
// with no milestone.
func (c *Core) RestoreTask(ctx context.Context, s TaskSnapshot) (*Task, error) {
	t := s.Task
	var out *Task
	err := c.write(ctx, func(q store.Querier) error {
		if cur, err := getTask(ctx, q, t.ID); err != nil {
			return err
		} else if cur != nil {
			return conflict("task_id", "task %d already exists", t.ID)
		}
		if err := dropGoneRefs(ctx, q, &t); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `INSERT INTO tasks (id, project_id, milestone_id, title, notes, status,
			priority, due_at, remind_at, branch, pr_url, pr_number, source, created_at, updated_at, completed_at,
			reminded_at, pr_state, pr_checks, pr_review, pr_checked_at, pr_closed_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.ProjectID, t.MilestoneID, t.Title, t.Notes, t.Status, t.Priority, t.DueAt, t.RemindAt,
			t.Branch, t.PRURL, t.PRNumber, t.Source, t.CreatedAt, t.UpdatedAt, t.CompletedAt, t.RemindedAt,
			t.PRState, t.PRChecks, t.PRReview, t.PRCheckedAt, t.PRClosedSeenAt); err != nil {
			return fmt.Errorf("insert task: %w", err)
		}
		for _, r := range s.Reports {
			if _, err := q.ExecContext(ctx,
				"INSERT INTO reports (id, task_id, agent, status, summary, next_step, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
				r.ID, t.ID, r.Agent, r.Status, r.Summary, r.NextStep, r.CreatedAt); err != nil {
				return fmt.Errorf("insert report: %w", err)
			}
		}
		var err error
		out, err = getTask(ctx, q, t.ID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("restore task %d: %w", t.ID, err)
	}
	return out, nil
}

// dropGoneRefs clears the project and milestone of t that no longer exist.
func dropGoneRefs(ctx context.Context, q store.Querier, t *Task) error {
	if t.MilestoneID != nil {
		m, err := getMilestone(ctx, q, *t.MilestoneID)
		if err != nil {
			return err
		}
		if m == nil {
			t.MilestoneID = nil
		}
	}
	if t.ProjectID != nil {
		p, err := getProject(ctx, q, *t.ProjectID)
		if err != nil {
			return err
		}
		if p == nil {
			t.ProjectID, t.MilestoneID = nil, nil
		}
	}
	return nil
}
