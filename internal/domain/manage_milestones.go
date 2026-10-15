package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// Milestone statuses.
const (
	MilestoneOpen   = "open"
	MilestoneClosed = "closed"
)

// MilestonePatch changes a milestone. Title and Status are left alone when nil, DueAt when omitted;
// Clear() removes the due date.
type MilestonePatch struct {
	Title  *string
	DueAt  Nullable[string] // stored form, see ParseTimestamp
	Status *string
}

// UpdateMilestone renames a milestone, moves or clears its due date, or closes or reopens it.
func (c *Core) UpdateMilestone(ctx context.Context, id int64, patch MilestonePatch) (*Milestone, error) {
	var out *Milestone
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getMilestone(ctx, q, id)
		if err != nil {
			return err
		}
		if cur == nil {
			return notFound("milestone_id", "milestone", id)
		}
		next := *cur
		if patch.Title != nil {
			title := strings.TrimSpace(*patch.Title)
			if title == "" {
				return invalid("title", "must not be empty")
			}
			next.Title = title
		}
		if patch.Status != nil {
			if *patch.Status != MilestoneOpen && *patch.Status != MilestoneClosed {
				return invalid("status", "must be open or closed")
			}
			next.Status = *patch.Status
		}
		next.DueAt = patch.DueAt.apply(cur.DueAt)
		if _, err := q.ExecContext(ctx, "UPDATE milestones SET title = ?, due_at = ?, status = ? WHERE id = ?",
			next.Title, next.DueAt, next.Status, id); err != nil {
			return fmt.Errorf("update milestone: %w", err)
		}
		out, err = getMilestone(ctx, q, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("update milestone %d: %w", id, err)
	}
	return out, nil
}
