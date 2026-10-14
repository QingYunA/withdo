package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

const maxListLimit = 200

type listTasksArgs struct {
	ProjectID   *int64          `json:"project_id"`
	Cwd         string          `json:"cwd"`
	Status      []domain.Status `json:"status"`
	MilestoneID *int64          `json:"milestone_id"`
	Limit       *int            `json:"limit"`
}

func (t *tools) registerListTasks(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tasks",
		Title:       "List tasks",
		Description: "List tasks, newest first, optionally filtered by project, status or milestone.",
		InputSchema: object(map[string]*jsonschema.Schema{
			"project_id": id("Project id"),
			"cwd":        withDesc(cwdProp, "List tasks of the project that owns this directory"),
			"status": {Type: "array", Description: "Only tasks in these statuses",
				Items: statusEnum("")},
			"milestone_id": id("Milestone id"),
			"limit": {Type: "integer", Minimum: ptr(1.0), Maximum: ptr(float64(maxListLimit)),
				Description: fmt.Sprintf("Default %d", domain.DefaultTaskLimit)},
		}),
	}, t.listTasks)
}

func (t *tools) listTasks(ctx context.Context, _ *mcp.CallToolRequest, a listTasksArgs) (*mcp.CallToolResult, any, error) {
	if err := scopeConflict(a.Cwd, a.ProjectID); err != nil {
		return t.fail(err)
	}
	scope := a.ProjectID
	if a.Cwd != "" {
		resolved, err := t.core.ResolveProjectFromCwd(ctx, a.Cwd)
		if err != nil {
			return t.fail(err)
		}
		if resolved == nil {
			return ok("No tasks (cwd is not in a git repository).", map[string]any{"tasks": []domain.Task{}})
		}
		scope = &resolved.Project.ID
	}
	filter := domain.TaskFilter{ProjectID: scope, Statuses: a.Status, MilestoneID: a.MilestoneID}
	if a.Limit != nil {
		filter.Limit = *a.Limit
	}
	tasks, err := t.core.ListTasks(ctx, filter)
	if err != nil {
		return t.fail(err)
	}
	text := "No tasks."
	if len(tasks) > 0 {
		lines := make([]string, len(tasks))
		for i, k := range tasks {
			lines[i] = domain.FormatTaskLine(k)
		}
		text = strings.Join(lines, "\n")
	}
	return ok(text, map[string]any{"tasks": tasks})
}

type getOverviewArgs struct {
	StaleHours *float64 `json:"stale_hours"`
}

func (t *tools) registerGetOverview(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "get_overview",
		Title: "Get overview",
		Description: "Per project: tasks in doing, blocked or review, each with its latest report and a stale flag " +
			"when nothing was reported for stale_hours.",
		InputSchema: object(map[string]*jsonschema.Schema{
			"stale_hours": {Type: "number", ExclusiveMinimum: ptr(0.0),
				Description: fmt.Sprintf("Hours without a report before a task is stale; default %d", domain.DefaultStaleHours)},
		}),
	}, t.getOverview)
}

func (t *tools) getOverview(ctx context.Context, _ *mcp.CallToolRequest, a getOverviewArgs) (*mcp.CallToolResult, any, error) {
	opts := domain.OverviewOptions{}
	if a.StaleHours != nil {
		if *a.StaleHours <= 0 {
			return t.fail(fieldErr("stale_hours", "must be greater than 0"))
		}
		opts.StaleHours = *a.StaleHours
	}
	overview, err := t.core.GetOverview(ctx, opts)
	if err != nil {
		return t.fail(err)
	}
	return ok(domain.FormatOverview(overview), overview)
}

type upsertProjectArgs struct {
	Cwd      string `json:"cwd"`
	RepoSlug string `json:"repo_slug"`
	Name     string `json:"name"`
}

func (t *tools) registerUpsertProject(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "upsert_project",
		Title:       "Create or rename project",
		Description: "Create the project for a repo (by cwd or repo_slug), or rename it if it already exists.",
		InputSchema: object(map[string]*jsonschema.Schema{
			"cwd":       withDesc(cwdProp, "A directory inside the git repo (use this or repo_slug)"),
			"repo_slug": text("GitHub owner/repo (use this or cwd)"),
			"name":      text("Display name; renames an existing project"),
		}),
	}, t.upsertProject)
}

func (t *tools) upsertProject(ctx context.Context, _ *mcp.CallToolRequest, a upsertProjectArgs) (*mcp.CallToolResult, any, error) {
	p, err := t.core.UpsertProject(ctx, domain.UpsertProjectInput{Cwd: a.Cwd, RepoSlug: a.RepoSlug, Name: a.Name})
	if err != nil {
		return t.fail(err)
	}
	slug := ""
	if p.RepoSlug != nil {
		slug = fmt.Sprintf(" (%s)", *p.RepoSlug)
	}
	return ok(fmt.Sprintf("Project #%d %s%s", p.ID, p.Name, slug), map[string]any{"project": p})
}

type createMilestoneArgs struct {
	Title     string `json:"title"`
	Cwd       string `json:"cwd"`
	ProjectID *int64 `json:"project_id"`
	DueAt     string `json:"due_at"`
}

func (t *tools) registerCreateMilestone(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_milestone",
		Title:       "Create milestone",
		Description: "Create a milestone under a project (resolved from cwd or project_id).",
		InputSchema: object(map[string]*jsonschema.Schema{
			"title":      text(""),
			"cwd":        withDesc(cwdProp, "Directory whose repo owns the milestone; defaults to the server's cwd"),
			"project_id": id("Project id"),
			"due_at":     timestamp("ISO 8601 date or datetime"),
		}, "title"),
	}, t.createMilestone)
}

func (t *tools) createMilestone(ctx context.Context, _ *mcp.CallToolRequest, a createMilestoneArgs) (*mcp.CallToolResult, any, error) {
	if err := scopeConflict(a.Cwd, a.ProjectID); err != nil {
		return t.fail(err)
	}
	due, err := timestampArg("due_at", a.DueAt)
	if err != nil {
		return t.fail(err)
	}
	cwd := ""
	if a.ProjectID == nil {
		cwd = firstNonEmpty(a.Cwd, t.defaultCwd)
	}
	projectID, err := t.core.ResolveProjectID(ctx, a.ProjectID, cwd)
	if err != nil {
		return t.fail(err)
	}
	if projectID == nil {
		return t.fail(fieldErr("cwd", "not inside a git repository; pass project_id"))
	}
	m, err := t.core.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: *projectID, Title: a.Title, DueAt: due})
	if err != nil {
		return t.fail(err)
	}
	return ok(fmt.Sprintf("Milestone #%d %s", m.ID, m.Title), map[string]any{"milestone": m})
}
