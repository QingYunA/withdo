package domain

import (
	"context"
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// AnyAgent is what HandedTo holds for a task handed over: not one agent but any agent, the word
// reports use for a client with no name. Rows written by an earlier version may hold an agent's id.
const AnyAgent = "agent"

// HandOff gives a task to any agent: it becomes the agents' (source agent), goes back to To do and
// open, and waits there with HandedTo (AnyAgent) and HandedAt set until an agent reports on it.
func (c *Core) HandOff(ctx context.Context, taskID int64) (*Task, error) {
	ts := c.stamp()
	return c.setHandOff(ctx, taskID, "hand off", func(q store.Querier) error {
		_, err := q.ExecContext(ctx, `UPDATE tasks SET source = ?, status = ?, completed_at = NULL, handed_to = ?,
			handed_at = ?, updated_at = ? WHERE id = ?`,
			SourceAgent, StatusTodo, AnyAgent, ts, ts, taskID)
		return err
	})
}

// TakeBack makes a task the user's again: source user, HandedTo and HandedAt cleared. Its status
// stays.
func (c *Core) TakeBack(ctx context.Context, taskID int64) (*Task, error) {
	return c.setHandOff(ctx, taskID, "take back", func(q store.Querier) error {
		_, err := q.ExecContext(ctx, `UPDATE tasks SET source = ?, handed_to = NULL, handed_at = NULL, updated_at = ?
			WHERE id = ?`, SourceUser, c.stamp(), taskID)
		return err
	})
}

// RestoreHandOff puts a task's source and hand-off back as they were, for the Undo of TakeBack. A
// blank agentID and a nil at mean "not waiting". It starts nothing and leaves the status alone.
func (c *Core) RestoreHandOff(ctx context.Context, taskID int64, agentID string, at *string, source Source) (*Task, error) {
	if source != SourceUser && source != SourceAgent {
		return nil, invalid("source", "must be user or agent")
	}
	return c.setHandOff(ctx, taskID, "restore hand-off", func(q store.Querier) error {
		_, err := q.ExecContext(ctx, "UPDATE tasks SET source = ?, handed_to = ?, handed_at = ?, updated_at = ? WHERE id = ?",
			source, strPtr(agentID), at, c.stamp(), taskID)
		return err
	})
}

// setHandOff runs change on an existing task in one write and returns the task afterwards.
func (c *Core) setHandOff(ctx context.Context, taskID int64, what string, change func(q store.Querier) error) (*Task, error) {
	var out *Task
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getTask(ctx, q, taskID)
		if err != nil {
			return err
		}
		if cur == nil {
			return notFound("task_id", "task", taskID)
		}
		if err := change(q); err != nil {
			return err
		}
		out, err = getTask(ctx, q, taskID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("%s task %d: %w", what, taskID, err)
	}
	return out, nil
}
