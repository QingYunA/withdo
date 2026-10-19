// Package data is the UI's view of adhd-todo: the domain types, the Store the views read and
// write through, and Real, its one implementation over internal/domain and SQLite. OpenDemo opens
// it on a temporary file with realistic seed data.
//
// The types mirror packages/core/src/domain/types.ts. Nullable strings are empty when absent,
// nullable times and ids are pointers, so integration with internal/domain stays mechanical.
package data

import "time"

// Status is a task's lifecycle state.
type Status string

const (
	Backlog   Status = "backlog"
	Todo      Status = "todo"
	Doing     Status = "doing"
	Blocked   Status = "blocked"
	Review    Status = "review"
	Done      Status = "done"
	Cancelled Status = "cancelled"
)

// Statuses lists every status in board order.
var Statuses = []Status{Backlog, Todo, Doing, Blocked, Review, Done, Cancelled}

// BoardStatuses are the statuses that have a column on the board.
var BoardStatuses = []Status{Backlog, Todo, Doing, Blocked, Review, Done}

// Terminal reports whether the status ends a task's life.
func (s Status) Terminal() bool { return s == Done || s == Cancelled }

// Active reports whether an agent is working on, or waiting on, the task.
func (s Status) Active() bool { return s == Doing || s == Blocked || s == Review }

// Priority is P0 (urgent) to P3 (someday); P2 is the default.
type Priority string

const (
	P0 Priority = "P0"
	P1 Priority = "P1"
	P2 Priority = "P2"
	P3 Priority = "P3"
)

// Priorities lists every priority from most to least urgent.
var Priorities = []Priority{P0, P1, P2, P3}

// Source says who created a task.
type Source string

const (
	SourceUser  Source = "user"
	SourceAgent Source = "agent"
)

// Project is a repository or area of work.
type Project struct {
	ID         int
	Name       string
	RepoSlug   string
	RepoPath   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ArchivedAt *time.Time
	// ParentID is the project this one hangs under as a sub-project; nil for a top-level project.
	ParentID *int
}

// HasRepo reports whether the project is a repository: its tasks count as work.
func (p Project) HasRepo() bool { return p.RepoSlug != "" || p.RepoPath != "" }

// Milestone groups a project's tasks toward a date.
type Milestone struct {
	ID        int
	ProjectID int
	Title     string
	DueAt     *time.Time
	Open      bool
	CreatedAt time.Time
}

// Task is one unit of work, written by the user or reported by an agent.
type Task struct {
	ID          int
	ParentID    *int // a subtask's parent: the task of the agent that started the subagent
	ProjectID   *int
	MilestoneID *int
	Title       string
	Notes       string
	Status      Status
	Priority    Priority
	CategoryID  int // the category the user sorted the task under, 0 for none; see CategoryOf
	DueAt       *time.Time
	RemindAt    *time.Time
	RemindedAt  *time.Time
	Branch      string
	PRURL       string
	PRNumber    int
	Source      Source
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time

	// What the GitHub poller last saw of PRURL; empty before the first check.
	PRState        PRState
	PRChecks       PRChecks
	PRReview       PRReview
	PRCheckedAt    *time.Time
	PRClosedSeenAt *time.Time // the user dismissed the closed pull request

	// HandedTo says the user handed the task over, until an agent first reports on it; empty
	// otherwise. It is "agent": any agent (older rows may hold an agent's id). HandedAt is when.
	HandedTo string
	HandedAt *time.Time
}

// IsSubtask reports whether the task belongs to another task.
func (t Task) IsSubtask() bool { return t.ParentID != nil }

// Subtasks returns the tasks among tasks whose parent is id, in the order given.
func Subtasks(tasks []Task, id int) []Task {
	var out []Task
	for _, t := range tasks {
		if t.ParentID != nil && *t.ParentID == id {
			out = append(out, t)
		}
	}
	return out
}

// Report is one progress note an agent filed against a task.
type Report struct {
	ID        int
	TaskID    int
	Agent     string
	Status    Status
	Summary   string
	NextStep  string
	CreatedAt time.Time
	Cwd       string // the working directory the agent reported from; empty when it did not say
	Links     []Link // pages the agent attached, in the order it gave them
}

// Link is a page an agent attached to a report. Title may be empty.
type Link struct {
	URL   string
	Title string
}

// WorkDir is where an agent works on a task: the directory of the newest report that names one
// (reports are newest first, as TaskDetail has them), else the project's folder, else "".
func WorkDir(reports []Report, projectPath string) string {
	for _, r := range reports {
		if r.Cwd != "" {
			return r.Cwd
		}
	}
	return projectPath
}

// TaskDetail is a task with its reports, newest first.
type TaskDetail struct {
	Task    Task
	Reports []Report
}

// OverviewTask is an active task with its latest report and staleness.
type OverviewTask struct {
	Task
	LatestReport       *Report
	Stale              bool
	HoursSinceActivity int
}

// OverviewGroup holds the active tasks of one project; Project is nil for the inbox.
type OverviewGroup struct {
	Project *Project
	Tasks   []OverviewTask
}

// Overview is what the agents are doing right now, grouped by project.
type Overview struct {
	GeneratedAt time.Time
	StaleHours  int
	Groups      []OverviewGroup
}

// DefaultStaleHours is how long an active task may stay silent before it counts as stale.
const DefaultStaleHours = 48

// Filter selects tasks. A zero Filter selects everything.
type Filter struct {
	ProjectID   *int
	Inbox       bool // only tasks without a project
	Status      []Status
	MilestoneID *int
	Source      Source
}

// CreateTaskInput is what the UI sends to create a task.
type CreateTaskInput struct {
	ParentID  *int
	Title     string
	Source    Source
	ProjectID *int
	Status    Status
	Priority  Priority
	Category  int // a category id; 0 leaves the task unsorted
	DueAt     *time.Time
}

// Patch changes a task. A nil field is left alone. To clear a nullable field set the zero value:
// "" for text, 0 for Category and MilestoneID, the zero time for DueAt and RemindAt.
type Patch struct {
	Title       *string
	Notes       *string
	Status      *Status
	Priority    *Priority
	Category    *int // a category id; 0 unsorts the task
	ProjectID   *int // 0 moves the task to the Inbox; a move without MilestoneID drops the milestone
	MilestoneID *int
	DueAt       *time.Time
	RemindAt    *time.Time
	Branch      *string
}

// Reminder is a reminder that has just come due.
type Reminder struct {
	TaskID      int
	Title       string
	ProjectName string // empty for the inbox
}

// Store is everything the UI needs. Real implements it over internal/domain and internal/watch.
type Store interface {
	Manager
	Images
	Summaries
	Bulk

	ListProjects() ([]Project, error)
	// ListCategories lists the categories in the user's order: Work and Life, and the user's own.
	ListCategories() ([]Category, error)
	Overview() (Overview, error)
	ListTasks(Filter) ([]Task, error)
	GetTask(id int) (TaskDetail, error)
	CreateTask(CreateTaskInput) (Task, error)
	UpdateTask(id int, p Patch) (Task, error)
	ListMilestones(projectID int) ([]Milestone, error)

	// Subscribe calls fn, on an arbitrary goroutine, after every change to the data, including
	// changes made by other processes such as the MCP server. The returned func cancels it.
	Subscribe(fn func()) (cancel func())
	// SubscribeReminders calls fn for every reminder that comes due, once.
	SubscribeReminders(fn func(Reminder)) (cancel func())
	// SubscribeEvents calls fn for every event, once, as the app first sees its report: an agent
	// handed a task back or got stuck, a pull request's checks failed or it merged. What happened
	// before the app started is not told, and neither is what the user did in the app.
	SubscribeEvents(fn func(Event)) (cancel func())

	// KeepClosedPR dismisses the Needs you row of a task whose pull request was closed without
	// merging; the task stays as it is.
	KeepClosedPR(id int) error

	// HandOff gives a task of the user's to any agent. The task becomes the agents' (Source
	// agent, To do, HandedTo "agent") and waits there until an agent reports on it. Nothing is
	// started: the user tells an agent to take it, or an agent finds it by itself.
	HandOff(id int) (Task, error)
	// TakeBack makes an agent's task the user's again: Source user, HandedTo cleared. Its status
	// stays.
	TakeBack(id int) (Task, error)
	// RestoreHandOff puts a task's Source and hand-off back as they were, for the Undo of TakeBack:
	// agent "" and at nil mean "not waiting". It starts nothing and leaves the status alone.
	RestoreHandOff(id int, agent string, at *time.Time, source Source) (Task, error)
}

// IntPtr returns a pointer to v.
func IntPtr(v int) *int { return &v }

// StrPtr returns a pointer to v.
func StrPtr(v string) *string { return &v }

// StatusPtr returns a pointer to v.
func StatusPtr(v Status) *Status { return &v }

// PriorityPtr returns a pointer to v.
func PriorityPtr(v Priority) *Priority { return &v }

// TimePtr returns a pointer to v.
func TimePtr(v time.Time) *time.Time { return &v }
