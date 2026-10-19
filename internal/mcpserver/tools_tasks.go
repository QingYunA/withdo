package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// categoryDesc tells an agent when to fill category in.
const categoryDesc = "Name of one of the user's categories, in any case (list_categories); Work and Life always exist. " +
	"Leave it out for a task in a repository's project, which counts as Work"

type createTaskArgs struct {
	Title       string           `json:"title"`
	Cwd         string           `json:"cwd"`
	ProjectID   *int64           `json:"project_id"`
	Notes       *string          `json:"notes"`
	Priority    *domain.Priority `json:"priority"`
	Category    *string          `json:"category"`
	MilestoneID *int64           `json:"milestone_id"`
	DueAt       string           `json:"due_at"`
	RemindAt    string           `json:"remind_at"`
	Branch      string           `json:"branch"`
	PRURL       string           `json:"pr_url"`
}

func (t *tools) registerCreateTask(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_task",
		Title:       "Create task",
		Description: "Create a task under a project (resolved from cwd or project_id).",
		InputSchema: object(map[string]*jsonschema.Schema{
			"title":        text("Short task title"),
			"cwd":          withDesc(cwdProp, "Working directory whose repo owns the task; defaults to the server's cwd"),
			"project_id":   id("Project to file the task under (instead of cwd)"),
			"notes":        anyText(""),
			"priority":     priorityEnum("P0 (urgent) to P3 (low); default P2"),
			"category":     text(categoryDesc),
			"milestone_id": id(""),
			"due_at":       timestamp(timestampDesc),
			"remind_at":    timestamp(timestampDesc),
			"branch":       text(""),
			"pr_url":       prURL,
		}, "title"),
	}, t.createTask)
}

func (t *tools) createTask(ctx context.Context, _ *mcp.CallToolRequest, a createTaskArgs) (*mcp.CallToolResult, any, error) {
	in, err := t.createTaskInput(ctx, a)
	if err != nil {
		return t.fail(err)
	}
	task, err := t.core.CreateTask(ctx, in)
	if err != nil {
		return t.fail(err)
	}
	return ok("Created "+domain.FormatTaskLine(*task), map[string]any{"task": task})
}

func (t *tools) createTaskInput(ctx context.Context, a createTaskArgs) (domain.CreateTaskInput, error) {
	in := domain.CreateTaskInput{
		Title: a.Title, Source: domain.SourceAgent, MilestoneID: a.MilestoneID, Notes: a.Notes,
		Branch: strPtr(a.Branch), PRURL: strPtr(a.PRURL),
	}
	if a.Priority != nil {
		in.Priority = *a.Priority
	}
	if err := scopeConflict(a.Cwd, a.ProjectID); err != nil {
		return in, err
	}
	if err := domain.CheckCwd(a.Cwd); err != nil {
		return in, err
	}
	var err error
	if a.Category != nil {
		id, err := t.core.ResolveCategory(ctx, *a.Category)
		if err != nil {
			return in, err
		}
		in.CategoryID = &id
	}
	if in.DueAt, err = timestampArg("due_at", a.DueAt); err != nil {
		return in, err
	}
	if in.RemindAt, err = timestampArg("remind_at", a.RemindAt); err != nil {
		return in, err
	}
	if err = urlArg("pr_url", a.PRURL); err != nil {
		return in, err
	}
	cwd := ""
	if a.ProjectID == nil {
		cwd = firstNonEmpty(a.Cwd, t.defaultCwd)
	}
	in.ProjectID, err = t.core.ResolveProjectID(ctx, a.ProjectID, cwd)
	return in, err
}

type updateTaskArgs struct {
	TaskID      int64                   `json:"task_id"`
	Title       *string                 `json:"title"`
	Notes       domain.Nullable[string] `json:"notes"`
	Status      *domain.Status          `json:"status"`
	Priority    *domain.Priority        `json:"priority"`
	Category    domain.Nullable[string] `json:"category"`
	ProjectID   domain.Nullable[int64]  `json:"project_id"`
	MilestoneID domain.Nullable[int64]  `json:"milestone_id"`
	DueAt       domain.Nullable[string] `json:"due_at"`
	RemindAt    domain.Nullable[string] `json:"remind_at"`
	Branch      domain.Nullable[string] `json:"branch"`
	PRURL       domain.Nullable[string] `json:"pr_url"`
}

func (t *tools) registerUpdateTask(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_task",
		Title:       "Update task",
		Description: "Patch fields of a task. Omitted fields are unchanged; null clears nullable fields.",
		InputSchema: object(map[string]*jsonschema.Schema{
			"task_id":      id("Task id, as returned by report_progress or create_task"),
			"title":        text(""),
			"notes":        nullable(anyText(""), "null clears the notes"),
			"status":       statusEnum(""),
			"priority":     priorityEnum(""),
			"category":     nullable(text(""), categoryDesc+"; null clears it"),
			"project_id":   nullable(id(""), "null moves the task to the inbox"),
			"milestone_id": nullable(id(""), "null detaches the milestone"),
			"due_at":       nullable(timestamp(""), "null clears the due date"),
			"remind_at":    nullable(timestamp(""), "null clears the reminder"),
			"branch":       nullable(text(""), ""),
			"pr_url":       nullable(prURL, "null clears the PR; pr_number is derived from the URL"),
		}, "task_id"),
	}, t.updateTask)
}

func (t *tools) updateTask(ctx context.Context, _ *mcp.CallToolRequest, a updateTaskArgs) (*mcp.CallToolResult, any, error) {
	due, err := nullableTimestampArg("due_at", a.DueAt)
	if err != nil {
		return t.fail(err)
	}
	remind, err := nullableTimestampArg("remind_at", a.RemindAt)
	if err != nil {
		return t.fail(err)
	}
	if err := nullableURLArg("pr_url", a.PRURL); err != nil {
		return t.fail(err)
	}
	category, err := t.categoryPatch(ctx, a.Category)
	if err != nil {
		return t.fail(err)
	}
	task, err := t.core.UpdateTask(ctx, a.TaskID, domain.TaskPatch{
		Title: a.Title, Status: a.Status, Priority: a.Priority, CategoryID: category, Notes: a.Notes,
		ProjectID: a.ProjectID, MilestoneID: a.MilestoneID, DueAt: due, RemindAt: remind,
		Branch: a.Branch, PRURL: a.PRURL,
	})
	if err != nil {
		return t.fail(err)
	}
	return ok("Updated "+domain.FormatTaskLine(*task), map[string]any{"task": task})
}

// subtaskBrief is a subtask as get_task lists it.
type subtaskBrief struct {
	ID     int64         `json:"id"`
	Title  string        `json:"title"`
	Status domain.Status `json:"status"`
}

// waitingNote is what get_task adds to a task the user handed to an agent that no agent has taken yet.
const waitingNote = "This task was handed to an agent and waits for one. If you take it, report on it with this task_id: " +
	"that takes it off \"Waiting for an agent\"."

type getTaskArgs struct {
	TaskID int64 `json:"task_id"`
}

func (t *tools) registerGetTask(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_task",
		Title:       "Get task",
		Description: "Get a task together with its progress report timeline (oldest first) and its subtasks.",
		InputSchema: object(map[string]*jsonschema.Schema{
			"task_id": id("Task id, as returned by report_progress or create_task"),
		}, "task_id"),
	}, t.getTask)
}

func (t *tools) getTask(ctx context.Context, _ *mcp.CallToolRequest, a getTaskArgs) (*mcp.CallToolResult, any, error) {
	task, err := t.core.GetTask(ctx, a.TaskID)
	if err != nil {
		return t.fail(err)
	}
	if task == nil {
		return t.fail(&domain.FieldError{Field: "task_id", Msg: fmt.Sprintf("task %d not found", a.TaskID), NotFound: true})
	}
	reports, err := t.core.ListReports(ctx, a.TaskID)
	if err != nil {
		return t.fail(err)
	}
	subtasks, err := t.core.ListSubtasks(ctx, a.TaskID)
	if err != nil {
		return t.fail(err)
	}
	lines := []string{domain.FormatTaskLine(*task)}
	for _, r := range reports {
		lines = append(lines, fmt.Sprintf("- %s %s: %s", r.CreatedAt, r.Agent, r.Summary))
	}
	brief := make([]subtaskBrief, len(subtasks))
	if len(subtasks) > 0 {
		lines = append(lines, "Subtasks:")
	}
	for i, s := range subtasks {
		brief[i] = subtaskBrief{ID: s.ID, Title: s.Title, Status: s.Status}
		lines = append(lines, fmt.Sprintf("- #%d [%s] %s", s.ID, s.Status, s.Title))
	}
	data := map[string]any{"task": task, "reports": reports, "subtasks": brief}
	if task.HandedTo != nil && !task.Status.Terminal() {
		lines = append(lines, waitingNote)
		data["note"] = waitingNote
	}
	return ok(strings.Join(lines, "\n"), data)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
