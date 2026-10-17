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

// LatestReportID is the id of the newest report; 0 when there is none. A reader that starts here
// sees only the reports that come after.
func (c *Core) LatestReportID(ctx context.Context) (int64, error) {
	var id int64
	err := c.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM reports").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("latest report id: %w", err)
	}
	return id, nil
}

// ReportsAfter returns the reports with an id above after, oldest first, with their task's title,
// project and pull request. A report of a task that was deleted since is not among them.
func (c *Core) ReportsAfter(ctx context.Context, after int64) ([]ReportEvent, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT r.id, r.task_id, r.agent, r.status, r.summary, r.next_step,
		r.created_at, r.cwd, r.kind, t.title, p.name, t.pr_number
		FROM reports r JOIN tasks t ON t.id = r.task_id LEFT JOIN projects p ON p.id = t.project_id
		WHERE r.id > ? ORDER BY r.id`, after)
	if err != nil {
		return nil, fmt.Errorf("list reports after %d: %w", after, err)
	}
	return collect(rows, func(s scanner) (ReportEvent, error) {
		var e ReportEvent
		err := s.Scan(&e.ID, &e.TaskID, &e.Agent, &e.Status, &e.Summary, &e.NextStep,
			&e.CreatedAt, &e.Cwd, &e.Kind, &e.TaskTitle, &e.ProjectName, &e.PRNumber)
		return e, err
	})
}
