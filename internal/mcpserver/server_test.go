package mcpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/mcpserver"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

var ctx = context.Background()

// rig is a connected client/server pair over an in-memory transport.
type rig struct {
	t       *testing.T
	session *mcp.ClientSession
	core    *domain.Core
	dir     string // a GitHub-backed repo on branch feat/login, the server's default cwd
}

type result struct {
	IsError bool
	Text    string
	Data    map[string]any
}

func newRig(t *testing.T) *rig {
	t.Helper()
	testutil.IsolateGit(t)
	db, _ := testutil.OpenDB(t)
	core := domain.New(db, domain.Options{})
	dir := testutil.NewRepo(t, t.TempDir(), "widgets", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Branch: "feat/login"})
	server := mcpserver.New(core, mcpserver.Options{DefaultCwd: dir, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	serverT, clientT := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-agent", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close(); _ = serverSession.Close() })
	return &rig{t: t, session: session, core: core, dir: dir}
}

func (r *rig) call(name string, args map[string]any) result {
	r.t.Helper()
	res, err := r.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		r.t.Fatalf("%s: protocol error: %v", name, err)
	}
	out := result{IsError: res.IsError}
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out.Text += tc.Text
		}
	}
	if res.StructuredContent != nil {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, &out.Data); err != nil {
			r.t.Fatalf("structured content: %v", err)
		}
	}
	return out
}

func (r *rig) ok(name string, args map[string]any) result {
	r.t.Helper()
	res := r.call(name, args)
	if res.IsError {
		r.t.Fatalf("%s failed: %s", name, res.Text)
	}
	return res
}

func (res result) obj(key string) map[string]any {
	m, _ := res.Data[key].(map[string]any)
	return m
}

func (res result) list(key string) []any {
	l, _ := res.Data[key].([]any)
	return l
}

func TestListsTheEightTools(t *testing.T) {
	r := newRig(t)
	list, err := r.session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
		if len(tool.Description) <= 10 {
			t.Errorf("%s has no useful description", tool.Name)
		}
		schema, _ := json.Marshal(tool.InputSchema)
		if !strings.Contains(string(schema), `"type":"object"`) {
			t.Errorf("%s input schema is not an object: %s", tool.Name, schema)
		}
	}
	sort.Strings(names)
	want := []string{"create_milestone", "create_task", "get_overview", "get_task", "list_tasks", "report_progress", "update_task", "upsert_project"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("tools = %v, want %v", names, want)
	}
}

func TestInitializeAdvertisesNameAndInstructions(t *testing.T) {
	r := newRig(t)
	init := r.session.InitializeResult()
	if init.ServerInfo.Name != "adhd-todo" {
		t.Errorf("server name = %q", init.ServerInfo.Name)
	}
	for _, w := range []string{"report_progress", "start", "blocked", "PR", "finish", "Always pass cwd",
		"after each step", "in the language the user writes to you in"} {
		if !strings.Contains(init.Instructions, w) {
			t.Errorf("instructions lack %q", w)
		}
	}
}

func TestReportProgressCreatesThenUpdates(t *testing.T) {
	r := newRig(t)
	a := r.ok("report_progress", map[string]any{"summary": "starting login"})
	if a.Data["created"] != true || a.Data["status"] != "doing" {
		t.Fatalf("first report: %+v", a.Data)
	}
	if a.obj("task")["branch"] != "feat/login" {
		t.Errorf("branch = %v", a.obj("task")["branch"])
	}
	if !strings.Contains(a.Text, "#1") || !strings.HasPrefix(a.Text, "Created") {
		t.Errorf("text = %q", a.Text)
	}

	b := r.ok("report_progress", map[string]any{
		"summary": "PR up", "status": "review", "next_step": "wait for review",
		"pr_url": "https://github.com/acme/widgets/pull/3",
	})
	if b.Data["created"] != false || b.Data["status"] != "review" || b.Data["task_id"] != a.Data["task_id"] {
		t.Errorf("second report: %+v", b.Data)
	}
	if b.obj("task")["pr_number"] != float64(3) || !strings.HasPrefix(b.Text, "Updated") {
		t.Errorf("pr/task: %+v / %q", b.obj("task"), b.Text)
	}
}

func TestReportProgressRecordsClientNameAsAgent(t *testing.T) {
	r := newRig(t)
	a := r.ok("report_progress", map[string]any{"summary": "hi"})
	got := r.ok("get_task", map[string]any{"task_id": a.Data["task_id"]})
	reports := got.list("reports")
	if len(reports) != 1 || reports[0].(map[string]any)["agent"] != "test-agent" {
		t.Errorf("reports = %+v", reports)
	}
}

func TestReportProgressExplicitCwdBranchAndTitle(t *testing.T) {
	r := newRig(t)
	other := testutil.NewRepo(t, t.TempDir(), "other", testutil.RepoOptions{Remote: "git@github.com:acme/other.git"})
	a := r.ok("report_progress", map[string]any{"summary": "s", "cwd": other, "branch": "x", "title": "Custom"})
	task := a.obj("task")
	if task["title"] != "Custom" || task["branch"] != "x" {
		t.Errorf("task = %+v", task)
	}
}

func TestReportProgressStoresOnlyTheCwdTheAgentPassed(t *testing.T) {
	r := newRig(t)
	a := r.ok("report_progress", map[string]any{"summary": "no cwd"})
	b := r.ok("report_progress", map[string]any{"summary": "with cwd", "cwd": r.dir})
	if got := a.obj("task")["id"]; got != b.obj("task")["id"] {
		t.Fatalf("expected one task, got %v and %v", got, b.obj("task")["id"])
	}
	reports := r.ok("get_task", map[string]any{"task_id": a.Data["task_id"]}).list("reports")
	if len(reports) != 2 {
		t.Fatalf("reports = %+v", reports)
	}
	if got := reports[0].(map[string]any)["cwd"]; got != "" {
		t.Errorf("report without cwd stored %v, want empty", got)
	}
	if got := reports[1].(map[string]any)["cwd"]; got != r.dir {
		t.Errorf("report with cwd stored %v, want %v", got, r.dir)
	}
}

func TestReportProgressByTaskIDIgnoresDefaultCwd(t *testing.T) {
	r := newRig(t)
	a := r.ok("report_progress", map[string]any{"summary": "one"})
	b := r.ok("report_progress", map[string]any{"summary": "two", "task_id": a.Data["task_id"], "status": "blocked"})
	if b.Data["created"] != false || b.Data["status"] != "blocked" {
		t.Errorf("got %+v", b.Data)
	}
}

func TestToolErrorsAreIsErrorResultsNamingTheField(t *testing.T) {
	r := newRig(t)
	tests := []struct {
		name  string
		tool  string
		args  map[string]any
		field string
	}{
		{"report: missing summary", "report_progress", map[string]any{}, "summary"},
		{"report: empty summary", "report_progress", map[string]any{"summary": ""}, "summary"},
		{"report: bad status", "report_progress", map[string]any{"summary": "s", "status": "weird"}, "status"},
		{"report: bad pr_url", "report_progress", map[string]any{"summary": "s", "pr_url": "not-a-url"}, "pr_url"},
		{"report: zero task_id", "report_progress", map[string]any{"summary": "s", "task_id": 0}, "task_id"},
		{"report: unknown task_id", "report_progress", map[string]any{"summary": "s", "task_id": 999}, "task_id"},
		{"create: missing title", "create_task", map[string]any{}, "title"},
		{"create: bad priority", "create_task", map[string]any{"title": "x", "priority": "P9"}, "priority"},
		{"create: bad due_at", "create_task", map[string]any{"title": "x", "due_at": "tomorrow"}, "due_at"},
		{"create: bad remind_at", "create_task", map[string]any{"title": "x", "remind_at": "soon"}, "remind_at"},
		{"create: unknown project", "create_task", map[string]any{"title": "x", "project_id": 999}, "project_id"},
		{"create: unknown milestone", "create_task", map[string]any{"title": "x", "milestone_id": 999}, "milestone_id"},
		{"create: bad pr_url", "create_task", map[string]any{"title": "x", "pr_url": "nope"}, "pr_url"},
		{"update: unknown task", "update_task", map[string]any{"task_id": 999}, "task_id"},
		{"update: bad status", "update_task", map[string]any{"task_id": 1, "status": "nope"}, "status"},
		{"update: bad due_at", "update_task", map[string]any{"task_id": 1, "due_at": "x"}, "due_at"},
		{"update: bad remind_at", "update_task", map[string]any{"task_id": 1, "remind_at": "x"}, "remind_at"},
		{"update: bad pr_url", "update_task", map[string]any{"task_id": 1, "pr_url": "x"}, "pr_url"},
		{"get: unknown task", "get_task", map[string]any{"task_id": 999}, "task_id"},
		{"list: limit 0", "list_tasks", map[string]any{"limit": 0}, "limit"},
		{"list: limit too big", "list_tasks", map[string]any{"limit": 201}, "limit"},
		{"list: bad status", "list_tasks", map[string]any{"status": []string{"nope"}}, "status"},
		{"overview: negative stale_hours", "get_overview", map[string]any{"stale_hours": -1}, "stale_hours"},
		{"upsert: nothing given", "upsert_project", map[string]any{}, "cwd"},
		{"upsert: bad slug", "upsert_project", map[string]any{"repo_slug": "nope"}, "repo_slug"},
		{"milestone: non-git cwd", "create_milestone", map[string]any{"title": "v1", "cwd": t.TempDir()}, "cwd"},
		{"milestone: unknown project", "create_milestone", map[string]any{"title": "v1", "project_id": 99}, "project_id"},
		{"milestone: bad due_at", "create_milestone", map[string]any{"title": "v1", "due_at": "x"}, "due_at"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tool == "update_task" {
				r.ok("create_task", map[string]any{"title": "seed"})
			}
			res := r.call(tt.tool, tt.args)
			if !res.IsError {
				t.Fatalf("expected an error result, got %q", res.Text)
			}
			if !strings.Contains(res.Text, tt.field) {
				t.Errorf("error %q does not name %q", res.Text, tt.field)
			}
		})
	}
}

func TestScopeConflictsNameCwd(t *testing.T) {
	r := newRig(t)
	p := r.ok("upsert_project", map[string]any{"repo_slug": "acme/api"})
	pid := p.obj("project")["id"]
	for _, tool := range []string{"create_task", "list_tasks", "create_milestone"} {
		args := map[string]any{"project_id": pid, "cwd": r.dir}
		if tool != "list_tasks" {
			args["title"] = "x"
		}
		res := r.call(tool, args)
		if !res.IsError || !strings.Contains(res.Text, "cwd") || !strings.Contains(res.Text, "not both") {
			t.Errorf("%s: %+v", tool, res)
		}
	}
}

func TestTaskLifecycle(t *testing.T) {
	r := newRig(t)
	c := r.ok("create_task", map[string]any{
		"title": "Write docs", "priority": "P1", "notes": "n",
		"due_at": "2026-12-01", "remind_at": "2026-11-30T09:00:00+08:00",
	})
	task := c.obj("task")
	if task["title"] != "Write docs" || task["priority"] != "P1" || task["source"] != "agent" || task["status"] != "todo" {
		t.Errorf("task = %+v", task)
	}
	if task["project_id"] == nil {
		t.Error("task should be filed under the default cwd's project")
	}
	if task["due_at"] != "2026-12-01T00:00:00.000Z" || task["remind_at"] != "2026-11-30T01:00:00.000Z" {
		t.Errorf("timestamps not normalised: %v %v", task["due_at"], task["remind_at"])
	}

	id := task["id"]
	u := r.ok("update_task", map[string]any{"task_id": id, "status": "done", "notes": nil})
	got := u.obj("task")
	if got["status"] != "done" || got["notes"] != nil || got["completed_at"] == nil {
		t.Errorf("updated = %+v", got)
	}
	u = r.ok("update_task", map[string]any{"task_id": id, "status": "doing", "remind_at": nil, "due_at": "2027-01-01T00:00:00Z", "pr_url": "https://github.com/a/b/pull/9"})
	got = u.obj("task")
	if got["completed_at"] != nil || got["remind_at"] != nil || got["pr_number"] != float64(9) || got["due_at"] != "2027-01-01T00:00:00.000Z" {
		t.Errorf("reopened = %+v", got)
	}
	g := r.ok("get_task", map[string]any{"task_id": id})
	if g.obj("task")["id"] != id || len(g.list("reports")) != 0 {
		t.Errorf("get_task = %+v", g.Data)
	}
}

func TestUpdateTaskNullsDetachProjectAndMilestone(t *testing.T) {
	r := newRig(t)
	p := r.ok("upsert_project", map[string]any{"repo_slug": "acme/api"}).obj("project")["id"]
	m := r.ok("create_milestone", map[string]any{"title": "v1", "project_id": p}).obj("milestone")["id"]
	task := r.ok("create_task", map[string]any{"title": "x", "project_id": p, "milestone_id": m}).obj("task")
	if task["project_id"] != p || task["milestone_id"] != m {
		t.Fatalf("task = %+v", task)
	}
	u := r.ok("update_task", map[string]any{"task_id": task["id"], "milestone_id": nil, "project_id": nil}).obj("task")
	if u["project_id"] != nil || u["milestone_id"] != nil {
		t.Errorf("updated = %+v", u)
	}
}

func TestGetTaskReturnsReportTimeline(t *testing.T) {
	r := newRig(t)
	r.ok("report_progress", map[string]any{"summary": "one"})
	b := r.ok("report_progress", map[string]any{"summary": "two"})
	g := r.ok("get_task", map[string]any{"task_id": b.Data["task_id"]})
	var summaries []string
	for _, rep := range g.list("reports") {
		summaries = append(summaries, rep.(map[string]any)["summary"].(string))
	}
	if strings.Join(summaries, ",") != "one,two" || !strings.Contains(g.Text, "two") {
		t.Errorf("summaries = %v, text = %q", summaries, g.Text)
	}
}

func TestListTasks(t *testing.T) {
	r := newRig(t)
	r.ok("report_progress", map[string]any{"summary": "a"})
	r.ok("create_task", map[string]any{"title": "b"})
	if n := len(r.ok("list_tasks", map[string]any{}).list("tasks")); n != 2 {
		t.Errorf("all = %d", n)
	}
	if n := len(r.ok("list_tasks", map[string]any{"status": []string{"doing"}}).list("tasks")); n != 1 {
		t.Errorf("doing = %d", n)
	}
	if n := len(r.ok("list_tasks", map[string]any{"cwd": r.dir, "limit": 1}).list("tasks")); n != 1 {
		t.Errorf("by cwd with limit = %d", n)
	}
	empty := r.ok("list_tasks", map[string]any{"status": []string{"cancelled"}})
	if empty.Text != "No tasks." || len(empty.list("tasks")) != 0 || empty.Data["tasks"] == nil {
		t.Errorf("empty = %+v", empty)
	}
}

func TestListTasksWithNonGitCwdReturnsNothing(t *testing.T) {
	r := newRig(t)
	r.ok("report_progress", map[string]any{"summary": "a"})
	res := r.ok("list_tasks", map[string]any{"cwd": t.TempDir()})
	if len(res.list("tasks")) != 0 || res.Data["tasks"] == nil || !strings.Contains(res.Text, "not in a git repository") {
		t.Errorf("res = %+v", res)
	}
}

func TestProjectAndMilestoneTools(t *testing.T) {
	r := newRig(t)
	a := r.ok("upsert_project", map[string]any{"cwd": r.dir})
	b := r.ok("upsert_project", map[string]any{"cwd": r.dir, "name": "Widgets"})
	if b.obj("project")["id"] != a.obj("project")["id"] || b.obj("project")["name"] != "Widgets" {
		t.Errorf("rename: %+v", b.Data)
	}
	if !strings.Contains(b.Text, "(acme/widgets)") {
		t.Errorf("text = %q", b.Text)
	}
	local := testutil.NewRepo(t, t.TempDir(), "local", testutil.RepoOptions{})
	if txt := r.ok("upsert_project", map[string]any{"cwd": local}).Text; strings.Contains(txt, "(") {
		t.Errorf("no slug expected: %q", txt)
	}

	m := r.ok("create_milestone", map[string]any{"title": "v1", "cwd": r.dir, "due_at": "2026-12-31T00:00:00Z"})
	ms := m.obj("milestone")
	if ms["title"] != "v1" || ms["status"] != "open" || ms["due_at"] != "2026-12-31T00:00:00.000Z" {
		t.Errorf("milestone = %+v", ms)
	}
	byDefault := r.ok("create_milestone", map[string]any{"title": "v2"})
	if byDefault.obj("milestone")["project_id"] != a.obj("project")["id"] {
		t.Errorf("default cwd should own the milestone: %+v", byDefault.Data)
	}
}

func TestGetOverview(t *testing.T) {
	r := newRig(t)
	empty := r.ok("get_overview", map[string]any{})
	if !strings.Contains(empty.Text, "No active tasks") || len(empty.list("projects")) != 0 || empty.Data["projects"] == nil {
		t.Errorf("empty = %+v", empty)
	}
	r.ok("report_progress", map[string]any{"summary": "working"})
	o := r.ok("get_overview", map[string]any{})
	groups := o.list("projects")
	if len(groups) != 1 {
		t.Fatalf("groups = %+v", groups)
	}
	task := groups[0].(map[string]any)["tasks"].([]any)[0].(map[string]any)
	if task["stale"] != false || task["status"] != "doing" || task["latest_report"] == nil {
		t.Errorf("task = %+v", task)
	}
	if !strings.Contains(o.Text, "widgets") {
		t.Errorf("text = %q", o.Text)
	}
	custom := r.ok("get_overview", map[string]any{"stale_hours": 0.0001})
	if custom.Data["stale_hours"] != 0.0001 {
		t.Errorf("stale_hours = %v", custom.Data["stale_hours"])
	}
}

func TestDefaultsWithoutOptions(t *testing.T) {
	db, _ := testutil.OpenDB(t)
	server := mcpserver.New(domain.New(db, domain.Options{}), mcpserver.Options{})
	if server == nil {
		t.Fatal("New returned nil")
	}
}

func TestInternalErrorsAreReportedAsInternal(t *testing.T) {
	r := newRig(t)
	// Break the schema under the server's feet to force a non-field error.
	if _, err := r.core.ListProjects(ctx); err != nil {
		t.Fatal(err)
	}
	db, _ := testutil.OpenDB(t)
	_ = db.Close()
	server := mcpserver.New(domain.New(db, domain.Options{}), mcpserver.Options{DefaultCwd: r.dir, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ss.Close() }()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "create_task", Arguments: map[string]any{"title": "x", "project_id": 1}})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !res.IsError || !strings.HasPrefix(text, "internal error:") {
		t.Errorf("got isError=%v %q", res.IsError, text)
	}
}
