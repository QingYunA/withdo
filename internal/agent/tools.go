package agent

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

// Tool names, also what the prompt refers to.
const (
	toolFind     = "find_tasks"
	toolCreate   = "create_task"
	toolUpdate   = "update_task"
	toolComplete = "complete_task"
	toolProjects = "list_projects"
	toolAsk      = "ask_user"

	maxFound   = 20
	maxOptions = 4
)

// toolbox is the tools of one turn. Mutating tools record a receipt and an undo in the journal.
type toolbox struct {
	core     *domain.Core
	now      func() time.Time
	journal  *journal
	projects []domain.Project
	// categories are the user's categories as of the start of the turn; receipts label with them.
	categories []domain.Category
	// defaultProject is where create_task puts a task whose project the model did not name: the
	// project whose page the request came from.
	defaultProject *int64
	// defaultCategory is the category create_task gives a task whose category the model did not
	// name: the category page of Mine the request came from; nil elsewhere.
	defaultCategory *int64
	// images is how many pictures the request carries; the tools take their indexes.
	images int
	// memory is the assistant's notes about the user; nil has no remember and forget tools.
	memory *Memory

	mu       sync.Mutex
	receipts []Receipt
	question *Question
	keeps    []ImageKeep
}

func (tb *toolbox) projectName(id *int64) string {
	if id == nil {
		return "Inbox"
	}
	for _, p := range tb.projects {
		if p.ID == *id {
			return p.Name
		}
	}
	return fmt.Sprintf("project %d", *id)
}

// projectLabel is a project's name for a receipt: empty for the Inbox, which the UI names itself.
func (tb *toolbox) projectLabel(id *int64) string {
	if id == nil {
		return ""
	}
	return tb.projectName(id)
}

func (tb *toolbox) addReceipt(r Receipt, undo func(context.Context) error) {
	r.ID = tb.journal.add(undo)
	tb.mu.Lock()
	tb.receipts = append(tb.receipts, r)
	tb.mu.Unlock()
}

func (tb *toolbox) takeReceipts() []Receipt {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return slices.Clone(tb.receipts)
}

func (tb *toolbox) asked() *Question {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.question
}

// tools builds the tool set. ask_user returns directly: its question ends the turn.
func (tb *toolbox) tools() ([]tool.BaseTool, map[string]bool, error) {
	var out []tool.BaseTool
	var firstErr error
	build := func(t tool.InvokableTool, err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if err == nil {
			out = append(out, t)
		}
	}
	build(utils.InferTool(toolFind, "Search tasks by text, status, project or source. Returns ids. Open tasks only unless statuses are given.", tb.find))
	build(utils.InferTool(toolCreate, "Create a task for the user. For a reminder set remind_at, for a deadline due_at.", tb.create))
	build(utils.InferTool(toolUpdate, "Change fields of one task by id. Use clear to empty a field.", tb.update))
	build(utils.InferTool(toolComplete, "Mark one task done by id.", tb.complete))
	build(utils.InferTool(toolProjects, "List the projects with their ids.", tb.listProjects))
	build(utils.InferTool(toolAsk, "Ask the user a short question when you cannot act without the answer. Offer 2 to 4 options. This ends your turn.", tb.ask))
	if tb.memory != nil {
		build(utils.InferTool(toolRemember, "Save one short note about the user to your memory, which you see in every conversation.", tb.remember))
		build(utils.InferTool(toolForget, "Delete every memory note that contains the given text. Returns the notes that are left.", tb.forget))
	}
	if firstErr != nil {
		return nil, nil, fmt.Errorf("build tools: %w", firstErr)
	}
	return out, map[string]bool{toolAsk: true}, nil
}

// taskView is a task as the model sees it. Times are in the user's zone, the form it writes them in.
type taskView struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Category string `json:"category,omitempty"`
	Project  string `json:"project"`
	DueAt    string `json:"due_at,omitempty"`
	RemindAt string `json:"remind_at,omitempty"`
	Source   string `json:"source"`
}

func (tb *toolbox) view(t domain.Task) taskView {
	loc := tb.now().Location()
	return taskView{
		ID: t.ID, Title: t.Title, Status: string(t.Status), Priority: string(t.Priority),
		Category: categoryName(t), Project: tb.projectName(t.ProjectID), DueAt: localRFC3339(t.DueAt, loc),
		RemindAt: localRFC3339(t.RemindAt, loc), Source: string(t.Source),
	}
}

// categoryName is the name of a task's category as the model reads it; "" for a task nobody sorted.
func categoryName(t domain.Task) string {
	if t.Category == nil {
		return ""
	}
	return *t.Category
}

// categoryLabel is a task's category as a receipt carries it: the key (work, life) for a built-in
// one, which the app words in the user's language, and the name for one of the user's own; "" for a
// task nobody sorted.
func (tb *toolbox) categoryLabel(t domain.Task) string {
	if t.CategoryID == nil {
		return ""
	}
	for _, c := range tb.categories {
		if c.ID == *t.CategoryID && c.Builtin != "" {
			return c.Builtin
		}
	}
	return categoryName(t)
}

// reply is what every tool returns. A failure is a result, not an error, so the model can read it
// and try again; only a cancelled context aborts the turn.
type reply struct {
	OK       bool          `json:"ok"`
	Error    string        `json:"error,omitempty"`
	Note     string        `json:"note,omitempty"`
	Task     *taskView     `json:"task,omitempty"`
	Tasks    []taskView    `json:"tasks,omitempty"`
	Projects []projectView `json:"projects,omitempty"`
	// Memory is the assistant's notes after remember or forget.
	Memory string `json:"memory,omitempty"`
}

type projectView struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func fail(ctx context.Context, err error) (reply, error) {
	if ctx.Err() != nil {
		return reply{}, ctx.Err()
	}
	return reply{Error: err.Error()}, nil
}

type findArgs struct {
	Query     string   `json:"query,omitempty" jsonschema:"description=Words to look for in title and notes. Empty matches all."`
	Statuses  []string `json:"statuses,omitempty" jsonschema:"description=Only these statuses (backlog todo doing blocked review done cancelled). Default is open tasks."`
	ProjectID *int64   `json:"project_id,omitempty" jsonschema:"description=Only tasks in this project."`
	Source    string   `json:"source,omitempty" jsonschema:"enum=user,enum=agent,description=Only tasks the user or a coding agent created."`
}

func (tb *toolbox) find(ctx context.Context, a findArgs) (reply, error) {
	f := domain.TaskFilter{ProjectID: a.ProjectID, Limit: 500}
	for _, s := range a.Statuses {
		f.Statuses = append(f.Statuses, domain.Status(s))
	}
	if len(f.Statuses) == 0 {
		f.Statuses = []domain.Status{domain.StatusBacklog, domain.StatusTodo, domain.StatusDoing, domain.StatusBlocked, domain.StatusReview}
	}
	tasks, err := tb.core.ListTasks(ctx, f)
	if err != nil {
		return fail(ctx, err)
	}
	out := reply{OK: true}
	for _, t := range tasks {
		if len(out.Tasks) == maxFound {
			out.Note = fmt.Sprintf("only the %d newest matches are shown", maxFound)
			break
		}
		if matchesText(t, a.Query) && (a.Source == "" || string(t.Source) == a.Source) {
			out.Tasks = append(out.Tasks, tb.view(t))
		}
	}
	return out, nil
}

// matchesText is true when every word of query appears in the title, notes or branch.
func matchesText(t domain.Task, query string) bool {
	hay := strings.ToLower(t.Title)
	if t.Notes != nil {
		hay += "\n" + strings.ToLower(*t.Notes)
	}
	if t.Branch != nil {
		hay += "\n" + strings.ToLower(*t.Branch)
	}
	for _, w := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}

type projectsArgs struct{}

func (tb *toolbox) listProjects(context.Context, projectsArgs) (reply, error) {
	out := reply{OK: true}
	for _, p := range tb.projects {
		if p.ArchivedAt == nil {
			out.Projects = append(out.Projects, projectView{ID: p.ID, Name: p.Name})
		}
	}
	return out, nil
}

type askArgs struct {
	Question string   `json:"question" jsonschema:"description=One short question."`
	Options  []string `json:"options,omitempty" jsonschema:"description=2 to 4 short answers the user can pick."`
}

func (tb *toolbox) ask(_ context.Context, a askArgs) (reply, error) {
	q := &Question{Text: strings.TrimSpace(a.Question)}
	for _, o := range a.Options {
		if o = strings.TrimSpace(o); o != "" && !slices.Contains(q.Options, o) && len(q.Options) < maxOptions {
			q.Options = append(q.Options, o)
		}
	}
	tb.mu.Lock()
	tb.question = q
	tb.mu.Unlock()
	return reply{OK: true, Note: "question shown to the user"}, nil
}

func toolsNode(tools []tool.BaseTool) compose.ToolsNodeConfig {
	return compose.ToolsNodeConfig{Tools: tools, ExecuteSequentially: true}
}
