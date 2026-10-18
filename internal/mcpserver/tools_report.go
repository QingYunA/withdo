package mcpserver

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

type reportProgressArgs struct {
	Summary  string         `json:"summary"`
	Status   *domain.Status `json:"status"`
	NextStep string         `json:"next_step"`
	TaskID   *int64         `json:"task_id"`
	Title    string         `json:"title"`
	Cwd      string         `json:"cwd"`
	Branch   string         `json:"branch"`
	PRURL    string         `json:"pr_url"`
	// ParentTaskID is the task of the agent that started this one as a subagent.
	ParentTaskID *int64 `json:"parent_task_id"`
	// Links are pages to open: a report page, a design draft.
	Links []domain.ReportLink `json:"links"`
}

func (t *tools) registerReportProgress(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "report_progress",
		Title: "Report progress",
		Description: "Report what you are working on. Finds the open task for this repo and branch (or task_id), " +
			"creating it if needed, then records a progress report. Call when you start, get blocked, " +
			"after each step, open a PR, and finish. Write in the user's language. " +
			"When one piece of work covers several adhd-todo tasks, report on each of them with its own task_id.",
		InputSchema: object(map[string]*jsonschema.Schema{
			"summary":   text("One or two sentences: what you are doing or just did"),
			"status":    statusEnum("New task status; omit to leave unchanged"),
			"next_step": text("What happens next, or what you are waiting on"),
			"task_id":   id("Task to report on; omit to match by repo and branch"),
			"title":     text("Title for a newly created task; defaults to the first sentence of the summary"),
			"cwd":       withDesc(cwdProp, "Your current working directory. Always pass it: it decides the repo and branch"),
			"branch":    text("Branch name; defaults to the current git branch of cwd"),
			"pr_url":    prURL,
			"parent_task_id": id("The task_id of the agent that started you as a subagent; makes your task its subtask. " +
				"Pass it on every report if you were given one"),
			"links": linksProp,
		}, "summary"),
	}, t.reportProgress)
}

func (t *tools) reportProgress(ctx context.Context, req *mcp.CallToolRequest, a reportProgressArgs) (*mcp.CallToolResult, any, error) {
	if err := urlArg("pr_url", a.PRURL); err != nil {
		return t.fail(err)
	}
	fallback := ""
	if a.TaskID == nil && a.Cwd == "" {
		fallback = t.defaultCwd
	}
	res, err := t.core.ReportProgress(ctx, domain.ProgressInput{
		Summary: a.Summary, Agent: agentName(req), Status: a.Status, NextStep: a.NextStep,
		TaskID: a.TaskID, Title: a.Title, Cwd: a.Cwd, FallbackCwd: fallback, Branch: a.Branch, PRURL: a.PRURL, ParentTaskID: a.ParentTaskID,
		Links: a.Links,
	})
	if err != nil {
		return t.fail(err)
	}
	verb := "Updated"
	if res.Created {
		verb = "Created"
	}
	line := fmt.Sprintf("%s %s", verb, domain.FormatTaskLine(res.Task))
	if res.Created && res.Task.Branch == nil {
		// Without a branch the next report would not find this task again.
		line += fmt.Sprintf("\nNo branch here: pass task_id %d on your next reports.", res.Task.ID)
	}
	data := map[string]any{"task_id": res.Task.ID, "status": res.Task.Status, "created": res.Created, "task": res.Task}
	if len(res.Dropped) > 0 {
		data["dropped_links"] = res.Dropped
		for _, d := range res.Dropped {
			line += fmt.Sprintf("\nDropped link %s: %s", d.URL, d.Reason)
		}
	}
	return ok(line, data)
}

// linksProp is the links argument of report_progress.
var linksProp = &jsonschema.Schema{
	Type:     "array",
	MaxItems: ptr(domain.MaxReportLinks),
	Description: "Pages to open: an HTML report, a design draft, a document. " +
		"https://, http:// or file:// (absolute). A link that cannot be kept is dropped and listed in dropped_links; the report is still filed",
	Items: object(map[string]*jsonschema.Schema{
		"url":   {Type: "string"},
		"title": {Type: "string", Description: "optional; a longer title is cut to 80 characters"},
	}, "url"),
}
