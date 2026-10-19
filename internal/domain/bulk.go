package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// Changing several tasks at once, for the window's bulk actions. Each runs in one transaction, so
// either every task changes or none does, and each hands back what the caller needs for one Undo:
// the tasks as they were (BulkUndo's input), or snapshots of the deleted ones.

// TaskState is what a bulk change can touch on a task: where it is, how it stands, whose it is and
// whether it waits for an agent. RestoreStates puts it back exactly.
type TaskState struct {
	ID          int64
	ProjectID   *int64
	MilestoneID *int64
	Status      Status
	Priority    Priority
	Source      Source
	HandedTo    *string
	HandedAt    *string
	CompletedAt *string
}

// StateOf is the TaskState of t.
func StateOf(t Task) TaskState {
	return TaskState{
		ID: t.ID, ProjectID: t.ProjectID, MilestoneID: t.MilestoneID, Status: t.Status, Priority: t.Priority,
		Source: t.Source, HandedTo: t.HandedTo, HandedAt: t.HandedAt, CompletedAt: t.CompletedAt,
	}
}

// UpdateTasks applies one patch to every task in ids, in one transaction. It returns the tasks as
// they were before, and with them the open subtasks that closed along with a task that became
// done or cancelled, in the order they were changed. A missing task, or a patch one of them
// refuses, changes nothing. Only the fields of TaskState come back with RestoreStates; the app
// narrows its patch to those (data.BulkPatch), so a caller that passes more loses that part of its Undo.
func (c *Core) UpdateTasks(ctx context.Context, ids []int64, patch TaskPatch) ([]Task, error) {
	var before []Task
	err := c.write(ctx, func(q store.Querier) error {
		before = nil
		for _, id := range ids {
			cur, err := getTask(ctx, q, id)
			if err != nil {
				return err
			}
			if cur == nil {
				return notFound("task_id", "task", id)
			}
			before = append(before, *cur)
			if patch.Status != nil && patch.Status.Terminal() && !cur.Status.Terminal() {
				subs, err := openSubtasks(ctx, q, id)
				if err != nil {
					return err
				}
				before = append(before, subs...)
			}
			if _, err := c.updateTask(ctx, q, id, patch); err != nil {
				return fmt.Errorf("task %d: %w", id, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("update %d tasks: %w", len(ids), err)
	}
	return before, nil
}

// openSubtasks are the subtasks of id, and theirs, that are not finished.
func openSubtasks(ctx context.Context, q store.Querier, id int64) ([]Task, error) {
	rows, err := q.QueryContext(ctx,
		"SELECT "+taskCols+" FROM tasks WHERE parent_id = ? AND status NOT IN ('done','cancelled') ORDER BY id", id)
	if err != nil {
		return nil, fmt.Errorf("find subtasks of %d: %w", id, err)
	}
	subs, err := collect(rows, scanTask)
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, s := range subs {
		out = append(out, s)
		deeper, err := openSubtasks(ctx, q, s.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, deeper...)
	}
	return out, nil
}

// HandOffTasks hands every task in ids to any agent, as HandOff does, in one transaction. It returns
// the tasks as they were before. Waiting for an agent lists the newest first, so each task is
// stamped a millisecond after the one that follows it in ids: the list then reads in the order the
// tasks were given.
func (c *Core) HandOffTasks(ctx context.Context, ids []int64) ([]Task, error) {
	now := c.now()
	var before []Task
	err := c.write(ctx, func(q store.Querier) error {
		before = nil
		for i, id := range ids {
			cur, err := getTask(ctx, q, id)
			if err != nil {
				return err
			}
			if cur == nil {
				return notFound("task_id", "task", id)
			}
			before = append(before, *cur)
			ts := FormatTime(now.Add(time.Duration(len(ids)-1-i) * time.Millisecond))
			if err := handOff(ctx, q, id, ts); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hand off %d tasks: %w", len(ids), err)
	}
	return before, nil
}

// RestoreStates puts tasks back as they were, in one transaction: the Undo of UpdateTasks and
// HandOffTasks. A task that is gone since is skipped.
func (c *Core) RestoreStates(ctx context.Context, states []TaskState) error {
	ts := c.stamp()
	err := c.write(ctx, func(q store.Querier) error {
		for _, s := range states {
			cur, err := getTask(ctx, q, s.ID)
			if err != nil {
				return err
			}
			if cur == nil {
				continue
			}
			projectID := s.ProjectID
			if projectID, s.MilestoneID, err = liveRefs(ctx, q, projectID, s.MilestoneID); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE tasks SET project_id = ?, milestone_id = ?, status = ?, priority = ?,
				source = ?, handed_to = ?, handed_at = ?, completed_at = ?, updated_at = ? WHERE id = ?`,
				projectID, s.MilestoneID, s.Status, s.Priority, s.Source, s.HandedTo, s.HandedAt, s.CompletedAt, ts, s.ID); err != nil {
				return fmt.Errorf("restore task %d: %w", s.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("restore %d tasks: %w", len(states), err)
	}
	return nil
}

// liveRefs drops a project or milestone that is gone since, leaving the task in the Inbox.
func liveRefs(ctx context.Context, q store.Querier, project, milestone *int64) (*int64, *int64, error) {
	t := Task{ProjectID: project, MilestoneID: milestone}
	if err := dropGoneRefs(ctx, q, &t); err != nil {
		return nil, nil, err
	}
	return t.ProjectID, t.MilestoneID, nil
}

// DeleteTasks removes every task in ids with its reports, in one transaction, and returns what
// RestoreTasks needs to put them back.
func (c *Core) DeleteTasks(ctx context.Context, ids []int64) ([]TaskSnapshot, error) {
	snaps := make([]TaskSnapshot, 0, len(ids))
	for _, id := range ids {
		s, err := c.SnapshotTask(ctx, id)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, *s)
	}
	err := c.write(ctx, func(q store.Querier) error {
		for _, id := range ids {
			if _, err := q.ExecContext(ctx, "DELETE FROM reports WHERE task_id = ?", id); err != nil {
				return fmt.Errorf("delete reports: %w", err)
			}
			if _, err := q.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id); err != nil {
				return fmt.Errorf("delete task %d: %w", id, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("delete %d tasks: %w", len(ids), err)
	}
	return snaps, nil
}

// RestoreTasks puts snapshotted tasks back with their ids, in one transaction: the Undo of
// DeleteTasks.
func (c *Core) RestoreTasks(ctx context.Context, snaps []TaskSnapshot) error {
	err := c.write(ctx, func(q store.Querier) error {
		for _, s := range snaps {
			if _, err := restoreTask(ctx, q, s); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("restore %d tasks: %w", len(snaps), err)
	}
	return nil
}
