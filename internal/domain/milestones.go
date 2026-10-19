package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/store"
)

const milestoneCols = "id, project_id, title, due_at, status, created_at"

func scanMilestone(s scanner) (Milestone, error) {
	var m Milestone
	err := s.Scan(&m.ID, &m.ProjectID, &m.Title, &m.DueAt, &m.Status, &m.CreatedAt)
	return m, err
}

func getMilestone(ctx context.Context, q store.Querier, id int64) (*Milestone, error) {
	return one(q.QueryRowContext(ctx, "SELECT "+milestoneCols+" FROM milestones WHERE id = ?", id), scanMilestone)
}

// CreateMilestoneInput describes a new milestone. DueAt is in stored form
// (see ParseTimestamp) or nil.
type CreateMilestoneInput struct {
	ProjectID int64
	Title     string
	DueAt     *string
}

// CreateMilestone adds an open milestone to an existing project.
func (c *Core) CreateMilestone(ctx context.Context, in CreateMilestoneInput) (*Milestone, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, invalid("title", "must not be empty")
	}
	var out *Milestone
	err := c.write(ctx, func(q store.Querier) error {
		p, err := getProject(ctx, q, in.ProjectID)
		if err != nil {
			return err
		}
		if p == nil {
			return notFound("project_id", "project", in.ProjectID)
		}
		newID, err := c.syncID(ctx, q, "milestones")
		if err != nil {
			return err
		}
		res, err := q.ExecContext(ctx, "INSERT INTO milestones (id, project_id, title, due_at, created_at) VALUES (?, ?, ?, ?, ?)",
			newID, in.ProjectID, in.Title, in.DueAt, c.stamp())
		if err != nil {
			return fmt.Errorf("insert milestone: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("insert milestone: %w", err)
		}
		out, err = getMilestone(ctx, q, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create milestone: %w", err)
	}
	return out, nil
}

// ListMilestones returns a project's milestones, oldest first.
func (c *Core) ListMilestones(ctx context.Context, projectID int64) ([]Milestone, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT "+milestoneCols+" FROM milestones WHERE project_id = ? ORDER BY id", projectID)
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}
	return collect(rows, scanMilestone)
}

// ListAllMilestones returns every project's milestones in one query, each project's oldest first.
func (c *Core) ListAllMilestones(ctx context.Context) (map[int64][]Milestone, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT "+milestoneCols+" FROM milestones ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}
	all, err := collect(rows, scanMilestone)
	if err != nil {
		return nil, err
	}
	out := map[int64][]Milestone{}
	for _, m := range all {
		out[m.ProjectID] = append(out[m.ProjectID], m)
	}
	return out, nil
}
