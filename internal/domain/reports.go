package domain

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/QingYunA/adhd-todo/internal/store"
)

const (
	reportCols = "id, task_id, agent, status, summary, next_step, created_at, cwd"
	// maxTitleRunes bounds a title derived from a summary.
	maxTitleRunes = 60
)

func scanReport(s scanner) (Report, error) {
	var r Report
	err := s.Scan(&r.ID, &r.TaskID, &r.Agent, &r.Status, &r.Summary, &r.NextStep, &r.CreatedAt, &r.Cwd)
	return r, err
}

// ListReports returns a task's reports, oldest first.
func (c *Core) ListReports(ctx context.Context, taskID int64) ([]Report, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT "+reportCols+" FROM reports WHERE task_id = ? ORDER BY created_at, id", taskID)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	return collect(rows, scanReport)
}

func latestReport(ctx context.Context, q store.Querier, taskID int64) (*Report, error) {
	return one(q.QueryRowContext(ctx,
		"SELECT "+reportCols+" FROM reports WHERE task_id = ? ORDER BY created_at DESC, id DESC LIMIT 1", taskID),
		scanReport)
}

// AddReportInput is a progress note. CreatedAt defaults to now.
type AddReportInput struct {
	TaskID    int64
	Agent     string
	Summary   string
	Status    *Status
	NextStep  *string
	CreatedAt string
	// Cwd is the working directory the agent reported from; "" when it did not say.
	Cwd string
}

// AddReport files a report against an existing task without touching the task.
func (c *Core) AddReport(ctx context.Context, in AddReportInput) (*Report, error) {
	var out *Report
	err := c.write(ctx, func(q store.Querier) error {
		var err error
		out, err = c.addReport(ctx, q, in)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("add report: %w", err)
	}
	return out, nil
}

func (c *Core) addReport(ctx context.Context, q store.Querier, in AddReportInput) (*Report, error) {
	if in.CreatedAt == "" {
		in.CreatedAt = c.stamp()
	}
	res, err := q.ExecContext(ctx,
		"INSERT INTO reports (task_id, agent, status, summary, next_step, created_at, cwd) VALUES (?, ?, ?, ?, ?, ?, ?)",
		in.TaskID, in.Agent, in.Status, in.Summary, in.NextStep, in.CreatedAt, cleanCwd(in.Cwd))
	if err != nil {
		return nil, fmt.Errorf("insert report: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("insert report: %w", err)
	}
	// The first report on a task handed to an agent means the agent took it.
	if _, err := q.ExecContext(ctx, "UPDATE tasks SET handed_to = NULL, handed_at = NULL WHERE id = ? AND handed_to IS NOT NULL", in.TaskID); err != nil {
		return nil, fmt.Errorf("clear hand-off: %w", err)
	}
	return one(q.QueryRowContext(ctx, "SELECT "+reportCols+" FROM reports WHERE id = ?", id), scanReport)
}

// ProgressInput is what an agent tells us about its work. Empty strings and
// nil pointers mean "not given".
type ProgressInput struct {
	Summary string
	Agent   string
	Status  *Status
	// NextStep is what happens next, or what the agent is waiting on.
	NextStep string
	// TaskID targets a task directly; otherwise the open task for Cwd's
	// project and Branch is used, or created.
	TaskID *int64
	Title  string
	// Cwd is where the agent works: it finds the project and is stored on the report.
	Cwd string
	// FallbackCwd finds the project when neither TaskID nor Cwd is given; it is not stored.
	FallbackCwd string
	Branch      string
	PRURL       string
	// ParentTaskID makes the task a subtask of that task: given by an agent that was started by
	// another one. Without it, a report from a subagent's worktree finds its parent by itself.
	ParentTaskID *int64
}

// ProgressResult is the outcome of ReportProgress.
type ProgressResult struct {
	Task    Task
	Report  Report
	Created bool
}

// ReportProgress records an agent's progress: it finds the task (by TaskID, or
// the open task on the branch of Cwd's repo), creates it as an agent task when
// none exists, applies status and PR, and appends a report. Everything after
// resolving the project happens in one transaction.
func (c *Core) ReportProgress(ctx context.Context, in ProgressInput) (*ProgressResult, error) {
	if strings.TrimSpace(in.Summary) == "" {
		return nil, invalid("summary", "must not be blank")
	}
	if in.Status != nil && !in.Status.Valid() {
		return nil, invalid("status", "must be one of %s", joinStatuses())
	}
	var projectID *int64
	branch := in.Branch
	// mainBranch is where a subagent's parent lives; empty unless cwd is a subagent's worktree.
	mainBranch := ""
	lookup := firstNonEmpty(in.Cwd, in.FallbackCwd)
	if in.TaskID == nil && lookup != "" {
		resolved, err := c.ResolveProjectFromCwd(ctx, lookup)
		if err != nil {
			return nil, err
		}
		if resolved != nil {
			projectID = &resolved.Project.ID
			if branch == "" {
				branch = resolved.Git.Branch
			}
			if in.ParentTaskID == nil {
				mainBranch = c.subagentMainBranch(ctx, resolved.Git)
			}
		}
	}
	var out *ProgressResult
	err := c.write(ctx, func(q store.Querier) error {
		var err error
		out, err = c.recordProgress(ctx, q, in, projectID, branch, mainBranch)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("report progress: %w", err)
	}
	return out, nil
}

func (c *Core) recordProgress(ctx context.Context, q store.Querier, in ProgressInput, projectID *int64, branch, mainBranch string) (*ProgressResult, error) {
	task, created, err := c.locateOrCreate(ctx, q, in, projectID, branch, mainBranch)
	if err != nil {
		return nil, err
	}
	// An agent reporting into an archived project is working there again.
	if err := c.unarchiveByID(ctx, q, task.ProjectID); err != nil {
		return nil, err
	}
	// Always touch the task so updated_at reflects the report. A task found by
	// branch only takes status and PR; one targeted by id also takes the branch.
	patch := TaskPatch{Status: in.Status}
	if in.PRURL != "" {
		patch.PRURL = Set(in.PRURL)
	}
	if in.TaskID != nil && in.Branch != "" {
		patch.Branch = Set(in.Branch)
	}
	if created {
		patch = TaskPatch{}
	} else if in.ParentTaskID != nil && !sameInt(task.ParentID, in.ParentTaskID) {
		// An existing task reported with a parent becomes that parent's subtask.
		if _, err := checkParent(ctx, q, "parent_task_id", task.ID, *in.ParentTaskID, task.ProjectID); err != nil {
			return nil, err
		}
		patch.ParentID = Set(*in.ParentTaskID)
	}
	updated, err := c.updateTask(ctx, q, task.ID, patch)
	if err != nil {
		return nil, err
	}
	report, err := c.addReport(ctx, q, AddReportInput{
		TaskID: updated.ID, Agent: in.Agent, Summary: in.Summary, Status: in.Status, NextStep: strPtr(in.NextStep), Cwd: in.Cwd,
	})
	if err != nil {
		return nil, err
	}
	// addReport cleared a hand-off after updated was read.
	updated.HandedTo, updated.HandedAt = nil, nil
	return &ProgressResult{Task: *updated, Report: *report, Created: created}, nil
}

func (c *Core) locateOrCreate(ctx context.Context, q store.Querier, in ProgressInput, projectID *int64, branch, mainBranch string) (*Task, bool, error) {
	if in.TaskID != nil {
		t, err := getTask(ctx, q, *in.TaskID)
		if err != nil {
			return nil, false, err
		}
		if t == nil {
			return nil, false, notFound("task_id", "task", *in.TaskID)
		}
		return t, false, nil
	}
	if branch != "" {
		existing, err := findOpenTask(ctx, q, projectID, branch)
		if err != nil || existing != nil {
			return existing, false, err
		}
	}
	status := StatusDoing
	if in.Status != nil {
		status = *in.Status
	}
	parentID, projectID, err := c.parentForNewTask(ctx, q, in, projectID, mainBranch)
	if err != nil {
		return nil, false, err
	}
	task, err := c.insertTask(ctx, q, CreateTaskInput{
		Title:     firstNonEmpty(in.Title, firstSentence(in.Summary), branch, truncateTitle(in.Summary)),
		Source:    SourceAgent,
		Status:    status,
		Priority:  P2,
		ProjectID: projectID,
		Branch:    strPtr(branch),
		PRURL:     strPtr(in.PRURL),
		ParentID:  parentID,
	})
	return task, true, err
}

// parentForNewTask picks the parent of a task a report is about to create, and the project it goes
// in: the given ParentTaskID (validated), else the subagent heuristic's, else none.
func (c *Core) parentForNewTask(ctx context.Context, q store.Querier, in ProgressInput, projectID *int64, mainBranch string) (*int64, *int64, error) {
	if in.ParentTaskID != nil {
		project, err := checkParent(ctx, q, "parent_task_id", 0, *in.ParentTaskID, projectID)
		return in.ParentTaskID, project, err
	}
	if mainBranch == "" {
		return nil, projectID, nil
	}
	since := FormatTime(c.now().Add(-subagentParentWindow))
	parent, err := findSubagentParent(ctx, q, projectID, mainBranch, in.Agent, since)
	if err != nil || parent == nil {
		return nil, projectID, err
	}
	return &parent.ID, projectID, nil
}

// cleanCwd normalises a reported directory; "" stays "".
func cleanCwd(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Clean(cwd)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstSentence is the default title of a task: the first sentence of text, without its closing
// full stop, cut to maxTitleRunes with an ellipsis. A sentence ends at a line break, at the CJK
// 。！？, and at a ".", "!" or "?" followed by a space or the end ("foo.go" does not end one).
func firstSentence(text string) string {
	runes := []rune(strings.TrimSpace(text))
	end := len(runes)
	for i, r := range runes {
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		stop := r == '\n' || r == '。' || r == '！' || r == '？' ||
			(r == '.' || r == '!' || r == '?') && (next == 0 || unicode.IsSpace(next))
		if stop {
			end = i
			if r != '.' && r != '。' && r != '\n' {
				end = i + 1 // ! and ? stay in the title
			}
			break
		}
	}
	return truncateRunes(strings.TrimSpace(string(runes[:end])))
}

// truncateTitle takes the first line of text, cut to maxTitleRunes with an ellipsis.
func truncateTitle(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return truncateRunes(line)
}

func truncateRunes(line string) string {
	runes := []rune(line)
	if len(runes) <= maxTitleRunes {
		return line
	}
	return string(runes[:maxTitleRunes-1]) + "…"
}
