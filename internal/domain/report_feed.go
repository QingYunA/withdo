package domain

import (
	"context"
	"fmt"
)

// ReportEvent is a report with what a notification about it needs of its task.
type ReportEvent struct {
	Report
	TaskTitle string
	// ProjectName is the task's project; nil for the Inbox.
	ProjectName *string
	// PRNumber is the task's pull request; nil when it has none.
	PRNumber *int64
}

// LatestReportAt is the created_at of the newest report; "" when there is none.
func (c *Core) LatestReportAt(ctx context.Context) (string, error) {
	var at string
	err := c.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(created_at), '') FROM reports").Scan(&at)
	if err != nil {
		return "", fmt.Errorf("latest report time: %w", err)
	}
	return at, nil
}

// ReportsSince returns the reports filed at or after the time at (a created_at, as ReportEvent
// carries it), oldest first, with their task's title, project and pull request. Reading by time,
// not by id, keeps it right when a deleted report's id is reused. A report of a task that was
// deleted since is not among them.
func (c *Core) ReportsSince(ctx context.Context, at string) ([]ReportEvent, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT r.id, r.task_id, r.agent, r.status, r.summary, r.next_step,
		r.created_at, r.cwd, r.kind, r.links, t.title, p.name, t.pr_number
		FROM reports r JOIN tasks t ON t.id = r.task_id LEFT JOIN projects p ON p.id = t.project_id
		WHERE r.created_at >= ? ORDER BY r.created_at, r.id`, at)
	if err != nil {
		return nil, fmt.Errorf("list reports since %q: %w", at, err)
	}
	return collect(rows, func(s scanner) (ReportEvent, error) {
		var e ReportEvent
		var links string
		err := s.Scan(&e.ID, &e.TaskID, &e.Agent, &e.Status, &e.Summary, &e.NextStep,
			&e.CreatedAt, &e.Cwd, &e.Kind, &links, &e.TaskTitle, &e.ProjectName, &e.PRNumber)
		e.Links = decodeLinks(links)
		return e, err
	})
}
