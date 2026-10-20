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
	reportCols = "id, task_id, agent, status, summary, next_step, created_at, cwd, kind, links"
	// maxTitleRunes bounds a title derived from a summary.
	maxTitleRunes = 60
	// ReplyAgent is the agent name of a reply the user writes on a task.
	ReplyAgent = "you"
	// KindReply marks a report that is the user's reply, not an agent's progress note.
	KindReply ReportKind = "reply"
)

func scanReport(s scanner) (Report, error) {
	var r Report
	var links string
	err := s.Scan(&r.ID, &r.TaskID, &r.Agent, &r.Status, &r.Summary, &r.NextStep, &r.CreatedAt, &r.Cwd, &r.Kind, &links)
	r.Links = decodeLinks(links)
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

// reportBatch is how many task ids one query of the batched report reads takes, well under SQLite's
// limit on variables.
const reportBatch = 5000

// ListReportsOf reads the reports of many tasks at once, each task's oldest first like ListReports. A
// task without reports is not in the map.
func (c *Core) ListReportsOf(ctx context.Context, taskIDs []int64) (map[int64][]Report, error) {
	out := map[int64][]Report{}
	err := inBatches(taskIDs, func(marks string, args []any) error {
		rows, err := c.db.QueryContext(ctx,
			"SELECT "+reportCols+" FROM reports WHERE task_id IN ("+marks+") ORDER BY created_at, id", args...)
		if err != nil {
			return fmt.Errorf("list reports: %w", err)
		}
		all, err := collect(rows, scanReport)
		for _, r := range all {
			out[r.TaskID] = append(out[r.TaskID], r)
		}
		return err
	})
	return out, err
}

// latestReportsOf is latestReport for many tasks at once; a task without reports is not in the map.
func (c *Core) latestReportsOf(ctx context.Context, taskIDs []int64) (map[int64]Report, error) {
	out := map[int64]Report{}
	err := inBatches(taskIDs, func(marks string, args []any) error {
		rows, err := c.db.QueryContext(ctx, `SELECT `+reportCols+` FROM (SELECT `+reportCols+`,
			ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY created_at DESC, id DESC) AS n
			FROM reports WHERE task_id IN (`+marks+`)) WHERE n = 1`, args...)
		if err != nil {
			return fmt.Errorf("latest reports: %w", err)
		}
		all, err := collect(rows, scanReport)
		for _, r := range all {
			out[r.TaskID] = r
		}
		return err
	})
	return out, err
}

// inBatches calls fn with the placeholders and arguments of each batch of ids.
func inBatches(ids []int64, fn func(marks string, args []any) error) error {
	for start := 0; start < len(ids); start += reportBatch {
		batch := ids[start:min(start+reportBatch, len(ids))]
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		if err := fn(strings.TrimSuffix(strings.Repeat("?,", len(batch)), ","), args); err != nil {
			return err
		}
	}
	return nil
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
	// Kind says what a report of the poller is about; "" for any other.
	Kind ReportKind
	// Links are stored as given; ReportProgress cleans them first (see CleanLinks).
	Links []ReportLink
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
	links, err := encodeLinks(in.Links)
	if err != nil {
		return nil, err
	}
	newID, err := c.syncID(ctx, q, "reports")
	if err != nil {
		return nil, err
	}
	res, err := q.ExecContext(ctx,
		"INSERT INTO reports (id, task_id, agent, status, summary, next_step, created_at, cwd, kind, links) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		newID, in.TaskID, in.Agent, in.Status, in.Summary, in.NextStep, in.CreatedAt, cleanCwd(in.Cwd), in.Kind, links)
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

// AddReply files the user's reply on a task as a report by ReplyAgent with KindReply. The task
// itself is left alone: its status, its timestamps and a hand-off waiting for an agent stay as they
// were. The report syncs like any other report.
func (c *Core) AddReply(ctx context.Context, taskID int64, text string) (*Report, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, invalid("text", "must not be empty")
	}
	var out *Report
	err := c.write(ctx, func(q store.Querier) error {
		task, err := getTask(ctx, q, taskID)
		if err != nil {
			return err
		}
		if task == nil {
			return notFound("task_id", "task", taskID)
		}
		out, err = c.addReply(ctx, q, taskID, text)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("add reply: %w", err)
	}
	return out, nil
}

// addReply inserts the reply report inside the caller's transaction. It deliberately does not use
// addReport: filing a report clears a hand-off, but the user answering a task did not take it.
func (c *Core) addReply(ctx context.Context, q store.Querier, taskID int64, text string) (*Report, error) {
	links, err := encodeLinks(nil)
	if err != nil {
		return nil, err
	}
	newID, err := c.syncID(ctx, q, "reports")
	if err != nil {
		return nil, err
	}
	res, err := q.ExecContext(ctx,
		"INSERT INTO reports (id, task_id, agent, status, summary, next_step, created_at, cwd, kind, links) VALUES (?, ?, ?, NULL, ?, NULL, ?, '', ?, ?)",
		newID, taskID, ReplyAgent, text, c.stamp(), KindReply, links)
	if err != nil {
		return nil, fmt.Errorf("insert reply: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("insert reply: %w", err)
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
	// Links are pages to attach to the report; the ones that cannot be kept come back in
	// ProgressResult.Dropped and never fail the report.
	Links []ReportLink
}

// ProgressResult is the outcome of ReportProgress.
type ProgressResult struct {
	Task    Task
	Report  Report
	Created bool
	// Dropped are the links the report did not keep, with the reason.
	Dropped []DroppedLink
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
	// The cwd only matters when it finds the project; with a task_id it is just noted on the report.
	if in.TaskID == nil {
		if err := CheckCwd(in.Cwd); err != nil {
			return nil, err
		}
	}
	var dropped []DroppedLink
	in.Links, dropped = CleanLinks(in.Links)
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
	out.Dropped = dropped
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
		TaskID: updated.ID, Agent: in.Agent, Summary: in.Summary, Status: in.Status, NextStep: strPtr(in.NextStep), Cwd: in.Cwd, Links: in.Links,
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
		Title:     firstNonEmpty(strings.TrimSpace(in.Title), firstSentence(in.Summary), branch, truncateTitle(in.Summary)),
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
