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
}

func (t *tools) registerReportProgress(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "report_progress",
		Title: "Report progress",
		Description: "Report what you are working on. Finds the open task for this repo and branch (or task_id), " +
			"creating it if needed, then records a progress report. Call when you start, get blocked, " +
			"after each step, open a PR, and finish. Write in the user's language.",
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
		}, "summary"),
	}, t.reportProgress)
}

func (t *tools) reportProgress(ctx context.Context, req *mcp.CallToolRequest, a reportProgressArgs) (*mcp.CallToolResult, any, error) {
	if err := urlArg("pr_url", a.PRURL); err != nil {
		return t.fail(err)
	}
	cwd := a.Cwd
	if a.TaskID == nil && cwd == "" {
		cwd = t.defaultCwd
	}
	res, err := t.core.ReportProgress(ctx, domain.ProgressInput{
		Summary: a.Summary, Agent: agentName(req), Status: a.Status, NextStep: a.NextStep,
		TaskID: a.TaskID, Title: a.Title, Cwd: cwd, Branch: a.Branch, PRURL: a.PRURL, ParentTaskID: a.ParentTaskID,
	})
	if err != nil {
		return t.fail(err)
	}
	verb := "Updated"
	if res.Created {
		verb = "Created"
	}
	return ok(fmt.Sprintf("%s %s", verb, domain.FormatTaskLine(res.Task)), map[string]any{
		"task_id": res.Task.ID, "status": res.Task.Status, "created": res.Created, "task": res.Task,
	})
}
