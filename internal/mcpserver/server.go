// Package mcpserver exposes adhd-todo's core as an MCP server with ten tools.
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// Instructions is sent to every client at initialize; it tells agents when to
// call report_progress.
var Instructions = strings.Join([]string{
	"adhd-todo tracks what coding agents are doing across repos, branches and pull requests, so the user can see everything in one place.",
	"The user watches these reports while you work, so report often. Call report_progress:",
	"- when you start a task (summary = the goal; status \"doing\"),",
	"- after each step: you understood the problem, finished a part, ran the tests, changed your plan,",
	"- when you get blocked (status \"blocked\"; say in next_step what you need),",
	"- when you open a PR (pr_url, status \"review\"),",
	"- when you finish (status \"done\").",
	"Write summary, next_step and title in the language the user writes to you in (Chinese if they write Chinese), however the code and the repo are written.",
	"Always pass cwd: your current working directory. The server process may run in a different directory, and in a git worktree cwd decides the branch.",
	"Calls from the same repo and branch update one task, so keep each report to one or two short sentences. Pass task_id if you have it.",
	"When the user asks you to do an adhd-todo task by its number (\"do #44\"), read it with get_task and report on it with that task_id.",
	"get_task lists the pictures attached to a task with their file paths; get_task_image shows one by its number.",
	"When one piece of work covers several adhd-todo tasks (\"do #69 and #70\"), report on each of them with its own task_id, at least when you start and when you finish.",
	"Please pass title on your first report: a short name for the task. Without it the title is the first sentence of the summary.",
	"Reports match a task by repo and branch; with no branch (outside git, or a detached HEAD) each report without task_id creates a new task, so pass the task_id of your first report on the rest.",
	"When you start a subagent, give it your task_id and tell it to pass parent_task_id on its reports, so its work shows under your task.",
	"A subagent reports status \"done\" when it hands its work back: its parent reviews it, not the user.",
	"When you made something that can be opened (a report page, a design draft), attach it to the report with links.",
	"Do not report trivial questions or read-only exploration.",
}, "\n")

// Name and Version identify the server to clients.
const (
	Name    = "adhd-todo"
	Version = "0.8.1"
)

// Options customise the server.
type Options struct {
	// DefaultCwd is used by tools called without cwd; the process's working
	// directory when empty.
	DefaultCwd string
	// ImagesDir is the folder holding the pictures of tasks (paths.ImagesDir).
	ImagesDir string
	// Logger receives diagnostics. It must not write to stdout, which belongs
	// to the stdio transport. Defaults to a text logger on stderr.
	Logger *slog.Logger
}

// tools carries what every tool handler needs.
type tools struct {
	core       *domain.Core
	defaultCwd string
	imagesDir  string
	log        *slog.Logger
}

// New builds the MCP server over core.
func New(core *domain.Core, opts Options) *mcp.Server {
	t := &tools{core: core, defaultCwd: opts.DefaultCwd, imagesDir: opts.ImagesDir, log: opts.Logger}
	if t.log == nil {
		t.log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	if t.defaultCwd == "" {
		if wd, err := os.Getwd(); err == nil {
			t.defaultCwd = wd
		}
	}
	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version},
		&mcp.ServerOptions{Instructions: Instructions, Logger: t.log})
	server.AddReceivingMiddleware(rewriteSchemaErrors)
	t.registerReportProgress(server)
	t.registerCreateTask(server)
	t.registerUpdateTask(server)
	t.registerGetTask(server)
	t.registerGetTaskImage(server)
	t.registerListTasks(server)
	t.registerListCategories(server)
	t.registerGetOverview(server)
	t.registerUpsertProject(server)
	t.registerCreateMilestone(server)
	return server
}

// Run serves the MCP protocol on stdio until the client closes stdin or ctx ends.
func Run(ctx context.Context, server *mcp.Server) error {
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		return fmt.Errorf("serve MCP on stdio: %w", err)
	}
	return nil
}

// ok builds a successful result: human-readable text plus structured content.
func ok(text string, data any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: text}},
		StructuredContent: data,
	}, nil, nil
}

// fail turns a domain error into an isError result whose text names the bad
// field. Unexpected errors are logged and reported as "internal error".
func (t *tools) fail(err error) (*mcp.CallToolResult, any, error) {
	msg := err.Error()
	var fe *domain.FieldError
	if !errors.As(err, &fe) {
		t.log.Error("tool error", "error", err)
		msg = "internal error: " + msg
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: msg}}, IsError: true}, nil, nil
}

// agentName is the connected client's name, e.g. "claude-code".
func agentName(req *mcp.CallToolRequest) string {
	if info := req.ClientInfo(); info != nil && info.Name != "" {
		return info.Name
	}
	return "agent"
}

// scopeConflict reports the error for tools that take either cwd or project_id.
func scopeConflict(cwd string, projectID *int64) error {
	if cwd != "" && projectID != nil {
		return &domain.FieldError{Field: "cwd", Msg: "provide either cwd or project_id, not both"}
	}
	return nil
}
