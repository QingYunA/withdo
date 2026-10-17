package domain

import (
	"context"
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// PRState is where a pull request stands.
type PRState string

// Pull request states.
const (
	PROpen   PRState = "open"
	PRDraft  PRState = "draft"
	PRMerged PRState = "merged"
	PRClosed PRState = "closed" // closed without merging
)

// PRChecks sums up a pull request's CI checks.
type PRChecks string

// Check summaries.
const (
	PRChecksPassing PRChecks = "passing"
	PRChecksFailing PRChecks = "failing"
	PRChecksPending PRChecks = "pending"
	PRChecksNone    PRChecks = "none"
)

// PRReview is a pull request's review decision.
type PRReview string

// Review decisions.
const (
	PRReviewApproved         PRReview = "approved"
	PRReviewChangesRequested PRReview = "changes_requested"
	PRReviewRequired         PRReview = "review_required"
	PRReviewNone             PRReview = "none"
)

// GitHubAgent is the source of the reports the poller files.
const GitHubAgent = "github"

// ReportKind says what a report of the poller is about, so that a reader need not parse its text.
type ReportKind string

// Report kinds; every other report has none.
const (
	KindPRMerged         ReportKind = "pr_merged"
	KindPRClosed         ReportKind = "pr_closed"
	KindChecksFailing    ReportKind = "checks_failing"
	KindChangesRequested ReportKind = "changes_requested"
)

// PRReport is a report the poller files: its text and what it is about.
type PRReport struct {
	Kind    ReportKind
	Summary string
}

// PRSnapshot is what was seen of a pull request. The zero value means it was never checked.
type PRSnapshot struct {
	State  PRState
	Checks PRChecks
	Review PRReview
}

// PullRequest is the snapshot stored on the task; the zero value before the first check.
func (t Task) PullRequest() PRSnapshot {
	var s PRSnapshot
	if t.PRState != nil {
		s.State = *t.PRState
	}
	if t.PRChecks != nil {
		s.Checks = *t.PRChecks
	}
	if t.PRReview != nil {
		s.Review = *t.PRReview
	}
	return s
}

// ListPRTasks returns the tasks the poller follows: not done or cancelled, with a pr_url.
func (c *Core) ListPRTasks(ctx context.Context) ([]Task, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT "+taskCols+
		" FROM tasks WHERE pr_url IS NOT NULL AND status NOT IN ('done','cancelled') ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list pull request tasks: %w", err)
	}
	return collect(rows, scanTask)
}

// PRUpdate is the outcome of checking one pull request. URL is the pr_url that was checked.
type PRUpdate struct {
	URL      string
	Snapshot PRSnapshot
	// Done marks the task done: its pull request was merged.
	Done bool
	// Reports are summaries to file as reports from GitHubAgent, oldest first.
	Reports []PRReport
}

// RecordPR stores what a check saw, in one transaction with its effects: the task becomes done and
// reports are filed. It reports false, having written nothing, when the task is gone, finished, or
// points at another pull request now: the answer is stale. Leaving the closed state forgets that
// the user dismissed it. The task's updated_at moves only when it becomes done.
func (c *Core) RecordPR(ctx context.Context, taskID int64, u PRUpdate) (bool, error) {
	applied := false
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getTask(ctx, q, taskID)
		if err != nil {
			return err
		}
		if cur == nil || cur.Status.Terminal() || cur.PRURL == nil || *cur.PRURL != u.URL {
			return nil
		}
		ts := c.stamp()
		_, err = q.ExecContext(ctx, `UPDATE tasks SET pr_state = ?, pr_checks = ?, pr_review = ?, pr_checked_at = ?,
			pr_closed_seen_at = CASE WHEN ? = 'closed' THEN pr_closed_seen_at END WHERE id = ?`,
			nullIfEmpty(string(u.Snapshot.State)), nullIfEmpty(string(u.Snapshot.Checks)), nullIfEmpty(string(u.Snapshot.Review)),
			ts, string(u.Snapshot.State), taskID)
		if err != nil {
			return fmt.Errorf("store pull request state: %w", err)
		}
		done := StatusDone
		if u.Done {
			if _, err := c.updateTask(ctx, q, taskID, TaskPatch{Status: &done}); err != nil {
				return err
			}
		}
		for _, r := range u.Reports {
			in := AddReportInput{TaskID: taskID, Agent: GitHubAgent, Summary: r.Summary, Kind: r.Kind}
			if u.Done {
				in.Status = &done
			}
			if _, err := c.addReport(ctx, q, in); err != nil {
				return err
			}
		}
		applied = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("record pull request of task %d: %w", taskID, err)
	}
	return applied, nil
}

// DismissClosedPR records that the user has seen their task's pull request closed and wants to keep
// the task: the Needs you row goes away. It does nothing for a pull request that is not closed.
func (c *Core) DismissClosedPR(ctx context.Context, taskID int64) error {
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getTask(ctx, q, taskID)
		if err != nil {
			return err
		}
		if cur == nil {
			return notFound("task_id", "task", taskID)
		}
		_, err = q.ExecContext(ctx, "UPDATE tasks SET pr_closed_seen_at = ? WHERE id = ? AND pr_state = 'closed'", c.stamp(), taskID)
		return err
	})
	if err != nil {
		return fmt.Errorf("dismiss closed pull request of task %d: %w", taskID, err)
	}
	return nil
}

// forgetPullRequest clears what was learned about a pull request when the task now points at another.
func forgetPullRequest(ctx context.Context, q store.Querier, before, after Task) error {
	if samePtr(before.PRURL, after.PRURL) {
		return nil
	}
	_, err := q.ExecContext(ctx, `UPDATE tasks SET pr_state = NULL, pr_checks = NULL, pr_review = NULL,
		pr_checked_at = NULL, pr_closed_seen_at = NULL WHERE id = ?`, before.ID)
	if err != nil {
		return fmt.Errorf("forget pull request state: %w", err)
	}
	return nil
}

func samePtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
