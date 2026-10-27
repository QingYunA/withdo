package domain

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/gitctx"
	"github.com/QingYunA/adhd-todo/internal/store"
)

// Reports the core files itself when it closes a subtask.
const (
	// HousekeepingAgent is the agent name on reports the core files by itself.
	HousekeepingAgent = "withdo"
	// LegacyHousekeepingAgent is what HousekeepingAgent was called before the rename; reports filed
	// under it are still in people's databases.
	LegacyHousekeepingAgent = "adhd-todo"
	// ClosedWithParentSummary is the report of a subtask closed because its parent finished.
	ClosedWithParentSummary = "Closed with its parent task"
	// BranchFinishedSummary is the report of a subtask whose branch was merged or deleted.
	BranchFinishedSummary = "Its branch was merged or deleted"
)

const (
	// subagentParentWindow is how recently the parent must have reported to adopt a worktree's task.
	subagentParentWindow = 12 * time.Hour
	// subagentBranchPrefix marks a branch a coding agent made for its subagent.
	subagentBranchPrefix = "worktree-agent-"
	// maxAncestorWalk bounds the walk up a chain of parents.
	maxAncestorWalk = 100
	// finishedCheckTimeout and maxFinishedChecks bound one CloseFinishedSubtasks run.
	finishedCheckTimeout = 30 * time.Second
	maxFinishedChecks    = 200
)

func sameInt(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// checkParent validates parentID as the parent of task self (0 for a task not yet created) in
// projectID and returns the project the task belongs to: the parent's when projectID is nil.
// field names the input in errors.
func checkParent(ctx context.Context, q store.Querier, field string, self, parentID int64, projectID *int64) (*int64, error) {
	parent, err := getTask(ctx, q, parentID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, notFound(field, "task", parentID)
	}
	if projectID == nil {
		projectID = parent.ProjectID
	} else if !sameInt(projectID, parent.ProjectID) {
		return nil, invalid(field, "task %d belongs to another project", parentID)
	}
	if self == 0 {
		return projectID, nil
	}
	// A task is not its own ancestor: walk up from the parent looking for self.
	for at, steps := parent, 0; at != nil; steps++ {
		if at.ID == self || steps > maxAncestorWalk {
			return nil, invalid(field, "task %d is the task itself or one of its subtasks", parentID)
		}
		if at.ParentID == nil {
			break
		}
		if at, err = getTask(ctx, q, *at.ParentID); err != nil {
			return nil, err
		}
	}
	return projectID, nil
}

// ListSubtasks returns the tasks whose parent is parentID, oldest first.
func (c *Core) ListSubtasks(ctx context.Context, parentID int64) ([]Task, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT "+taskCols+" FROM tasks WHERE parent_id = ? ORDER BY id", parentID)
	if err != nil {
		return nil, fmt.Errorf("list subtasks of %d: %w", parentID, err)
	}
	return collect(rows, scanTask)
}

// closeSubtasks marks the open subtasks of parentID (and theirs) done, filing summary against each
// as HousekeepingAgent.
func (c *Core) closeSubtasks(ctx context.Context, q store.Querier, parentID int64, summary string) error {
	rows, err := q.QueryContext(ctx,
		"SELECT "+taskCols+" FROM tasks WHERE parent_id = ? AND status NOT IN ('done','cancelled') ORDER BY id", parentID)
	if err != nil {
		return fmt.Errorf("find subtasks of %d: %w", parentID, err)
	}
	open, err := collect(rows, scanTask)
	if err != nil {
		return err
	}
	for _, t := range open {
		if err := c.finishSubtask(ctx, q, t.ID, summary); err != nil {
			return err
		}
	}
	return nil
}

// finishSubtask makes one open task done with a report; its own subtasks follow through updateTask.
func (c *Core) finishSubtask(ctx context.Context, q store.Querier, id int64, summary string) error {
	done := StatusDone
	if _, err := c.updateTask(ctx, q, id, TaskPatch{Status: &done}); err != nil {
		return err
	}
	_, err := c.addReport(ctx, q, AddReportInput{TaskID: id, Agent: HousekeepingAgent, Summary: summary, Status: &done})
	return err
}

// subagentMainBranch says whether info is a worktree of a subagent, and when it is, returns the
// branch checked out in the repository's main worktree: where the parent task lives. A subagent
// works in a linked worktree under <repo>/.claude/worktrees/, or on a worktree-agent-* branch.
func (c *Core) subagentMainBranch(ctx context.Context, info *gitctx.Info) string {
	if info == nil || info.Toplevel == info.RepoRoot {
		return ""
	}
	under := filepath.Join(info.RepoRoot, ".claude", "worktrees") + string(filepath.Separator)
	inClaudeWorktrees := strings.HasPrefix(filepath.Clean(info.Toplevel)+string(filepath.Separator), under)
	if !inClaudeWorktrees && !strings.HasPrefix(info.Branch, subagentBranchPrefix) {
		return ""
	}
	main, err := c.git(ctx, info.RepoRoot)
	if err != nil {
		return ""
	}
	return main.Branch
}

// findSubagentParent returns the most recently reported open task of agent on branch within the
// project, reported since the cutoff; nil when there is none.
func findSubagentParent(ctx context.Context, q store.Querier, projectID *int64, branch, agent, since string) (*Task, error) {
	return one(q.QueryRowContext(ctx, `SELECT `+prefixCols(taskCols)+` FROM tasks t JOIN reports r ON r.task_id = t.id
		WHERE t.project_id IS ? AND t.branch = ? AND t.status NOT IN ('done','cancelled')
		AND r.agent = ? AND r.created_at >= ? ORDER BY r.created_at DESC, r.id DESC LIMIT 1`,
		projectID, branch, agent, since), scanTask)
}

// CloseFinishedSubtasks marks done the open subtasks whose branch no longer exists in the project's
// repository, or was merged into the main worktree's HEAD after the subtask began, filing a report
// from HousekeepingAgent. It returns how many it closed. It is cheap and bounded: it looks only at
// subtasks with a branch in a project with a repository path, at most maxFinishedChecks of them,
// within finishedCheckTimeout; a repository git cannot read is skipped.
func (c *Core) CloseFinishedSubtasks(ctx context.Context) (int, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT `+prefixCols(taskCols)+`, p.repo_path FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE t.parent_id IS NOT NULL AND t.status NOT IN ('done','cancelled') AND t.branch IS NOT NULL AND t.branch <> ''
		AND p.repo_path IS NOT NULL AND p.repo_path <> '' ORDER BY t.id LIMIT ?`, maxFinishedChecks)
	if err != nil {
		return 0, fmt.Errorf("find subtasks to check: %w", err)
	}
	type candidate struct {
		task Task
		repo string
	}
	var candidates []candidate
	err = func() error {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var cand candidate
			if err := rows.Scan(append(taskDest(&cand.task), &cand.repo)...); err != nil {
				return err
			}
			candidates = append(candidates, cand)
		}
		return rows.Err()
	}()
	if err != nil {
		return 0, fmt.Errorf("read subtasks to check: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, finishedCheckTimeout)
	defer cancel()
	closed := 0
	for _, cand := range candidates {
		began := parseStamp(cand.task.CreatedAt)
		finished, gitErr := gitctx.BranchFinished(ctx, cand.repo, *cand.task.Branch, began)
		if gitErr != nil || !finished {
			if ctx.Err() != nil {
				break
			}
			continue
		}
		did, err := c.closeIfOpen(ctx, cand.task.ID)
		if err != nil {
			return closed, err
		}
		if did {
			closed++
		}
	}
	return closed, nil
}

// closeIfOpen finishes a subtask that is still open, with the branch summary.
func (c *Core) closeIfOpen(ctx context.Context, id int64) (bool, error) {
	did := false
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getTask(ctx, q, id)
		if err != nil || cur == nil || cur.Status.Terminal() {
			return err
		}
		did = true
		return c.finishSubtask(ctx, q, id, BranchFinishedSummary)
	})
	if err != nil {
		return false, fmt.Errorf("close subtask %d: %w", id, err)
	}
	return did, nil
}

func parseStamp(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
