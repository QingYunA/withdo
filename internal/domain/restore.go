package domain

import (
	"context"
	"fmt"
	"math"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// TaskSnapshot is a task with its reports, taken before it is deleted so it can be put back.
type TaskSnapshot struct {
	Task     Task
	Reports  []Report // oldest first
	Images   []TaskImage
	Subtasks []int64 // the ids of its subtasks, which the delete leaves top-level
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
	imgs, err := c.ListTaskImages(ctx, id)
	if err != nil {
		return nil, err
	}
	subs, err := c.ListTasks(ctx, TaskFilter{ParentID: &id, Limit: math.MaxInt32})
	if err != nil {
		return nil, err
	}
	snap := &TaskSnapshot{Task: *t, Reports: rs, Images: imgs}
	for _, sub := range subs {
		snap.Subtasks = append(snap.Subtasks, sub.ID)
	}
	return snap, nil
}

// RestoreTask puts a snapshotted task back with its id, every field and its reports, in one
// transaction. It is the Undo of DeleteTask. A task that is already there (its id was taken
// meanwhile) is a conflict. A project or milestone that is gone since leaves the task in the Inbox
// with no milestone; a parent that is gone, or is in another project now, leaves it top-level. Its
// subtasks that are still top-level in its project go back under it.
func (c *Core) RestoreTask(ctx context.Context, s TaskSnapshot) (*Task, error) {
	var out *Task
	err := c.write(ctx, func(q store.Querier) error {
		var err error
		out, err = restoreTask(ctx, q, s)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("restore task %d: %w", s.Task.ID, err)
	}
	return out, nil
}

// restoreTask is RestoreTask inside the caller's transaction.
func restoreTask(ctx context.Context, q store.Querier, s TaskSnapshot) (*Task, error) {
	t := s.Task
	if cur, err := getTask(ctx, q, t.ID); err != nil {
		return nil, err
	} else if cur != nil {
		return nil, conflict("task_id", "task %d already exists", t.ID)
	}
	if err := dropGoneRefs(ctx, q, &t); err != nil {
		return nil, err
	}
	if _, err := q.ExecContext(ctx, `INSERT INTO tasks (id, project_id, milestone_id, title, notes, status,
		priority, due_at, remind_at, branch, pr_url, pr_number, source, created_at, updated_at, completed_at,
		reminded_at, pr_state, pr_checks, pr_review, pr_checked_at, pr_closed_seen_at, parent_id, handed_to, handed_at, category_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.MilestoneID, t.Title, t.Notes, t.Status, t.Priority, t.DueAt, t.RemindAt,
		t.Branch, t.PRURL, t.PRNumber, t.Source, t.CreatedAt, t.UpdatedAt, t.CompletedAt, t.RemindedAt,
		t.PRState, t.PRChecks, t.PRReview, t.PRCheckedAt, t.PRClosedSeenAt, t.ParentID, t.HandedTo, t.HandedAt, t.CategoryID); err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}
	for _, sub := range s.Subtasks {
		if _, err := q.ExecContext(ctx, "UPDATE tasks SET parent_id = ? WHERE id = ? AND parent_id IS NULL AND project_id IS ?",
			t.ID, sub, t.ProjectID); err != nil {
			return nil, fmt.Errorf("relink subtask %d: %w", sub, err)
		}
	}
	for _, r := range s.Reports {
		links, err := encodeLinks(r.Links)
		if err != nil {
			return nil, err
		}
		if _, err := q.ExecContext(ctx,
			"INSERT INTO reports (id, task_id, agent, status, summary, next_step, created_at, cwd, kind, links) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			r.ID, t.ID, r.Agent, r.Status, r.Summary, r.NextStep, r.CreatedAt, r.Cwd, r.Kind, links); err != nil {
			return nil, fmt.Errorf("insert report: %w", err)
		}
	}
	for _, im := range s.Images {
		if err := relinkImage(ctx, q, t.ID, im); err != nil {
			return nil, err
		}
	}
	return getTask(ctx, q, t.ID)
}

// relinkImage puts a snapshotted link back; the picture's row is made again if it was collected
// meanwhile (its files are the caller's concern).
func relinkImage(ctx context.Context, q store.Querier, taskID int64, im TaskImage) error {
	if _, err := q.ExecContext(ctx, `INSERT OR IGNORE INTO images (hash, ext, mime, bytes, created_at) VALUES (?, ?, ?, ?, ?)`,
		im.Hash, im.Ext, im.MIME, im.Bytes, im.CreatedAt); err != nil {
		return fmt.Errorf("insert image: %w", err)
	}
	if _, err := q.ExecContext(ctx, `INSERT OR IGNORE INTO task_images (task_id, image_hash, name, width, height, added_at)
		VALUES (?, ?, ?, ?, ?, ?)`, taskID, im.Hash, im.Name, im.Width, im.Height, im.AddedAt); err != nil {
		return fmt.Errorf("link image: %w", err)
	}
	return nil
}

// dropGoneRefs clears the project, milestone, category and parent of t that no longer exist, and a
// parent that is in another project than t now.
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
	if t.CategoryID != nil {
		cat, err := getCategory(ctx, q, *t.CategoryID)
		if err != nil {
			return err
		}
		if cat == nil {
			t.CategoryID = nil
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
	if t.ParentID != nil {
		parent, err := getTask(ctx, q, *t.ParentID)
		if err != nil {
			return err
		}
		if parent == nil || !sameInt(parent.ProjectID, t.ProjectID) {
			t.ParentID = nil
		}
	}
	return nil
}
