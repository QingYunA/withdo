package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// Reminder is a task whose remind_at has arrived.
type Reminder struct {
	TaskID int64  `json:"task_id"`
	Title  string `json:"title"`
	// ProjectName is nil for inbox tasks.
	ProjectName *string `json:"project_name"`
}

const dueReminders = `FROM tasks t LEFT JOIN projects p ON p.id = t.project_id
	WHERE t.remind_at IS NOT NULL AND t.remind_at <= ? AND t.reminded_at IS NULL
	AND t.status NOT IN ('done','cancelled')`

// HasDueReminders is a cheap read-only check for ClaimDueReminders.
func (c *Core) HasDueReminders(ctx context.Context, now time.Time) (bool, error) {
	var one int
	err := c.db.QueryRowContext(ctx, "SELECT 1 "+dueReminders+" LIMIT 1", FormatTime(now)).Scan(&one)
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, fmt.Errorf("check due reminders: %w", err)
	}
	return true, nil
}

// ClaimDueReminders returns tasks whose reminder is due (remind_at <= now, not
// yet reminded, not done or cancelled) and marks each as reminded in the same
// BEGIN IMMEDIATE transaction, so a reminder is handed out exactly once even
// with several processes polling.
func (c *Core) ClaimDueReminders(ctx context.Context, now time.Time) ([]Reminder, error) {
	ts := FormatTime(now)
	var claimed []Reminder
	err := c.write(ctx, func(q store.Querier) error {
		rows, err := q.QueryContext(ctx,
			"SELECT t.id, t.title, p.name "+dueReminders+" ORDER BY t.remind_at, t.id", ts)
		if err != nil {
			return err
		}
		due, err := collect(rows, func(s scanner) (Reminder, error) {
			var r Reminder
			return r, s.Scan(&r.TaskID, &r.Title, &r.ProjectName)
		})
		if err != nil {
			return err
		}
		for _, r := range due {
			if _, err := q.ExecContext(ctx, "UPDATE tasks SET reminded_at = ? WHERE id = ? AND reminded_at IS NULL", ts, r.TaskID); err != nil {
				return err
			}
		}
		claimed = due
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("claim due reminders: %w", err)
	}
	return claimed, nil
}
