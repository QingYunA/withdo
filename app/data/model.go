// Package data is the UI's view of adhd-todo: the domain types, the Store the views read and
// write through, and an in-memory fake with realistic seed data.
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
}

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

	// HandedTo is the agent the user handed the task to (an agentsetup ID: "claude", "codex"),
	// until that agent first reports on it; empty otherwise. HandedAt is when.
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
	DueAt     *time.Time
}

// Patch changes a task. A nil field is left alone. To clear a nullable field set the zero value:
// "" for text, 0 for MilestoneID, the zero time for DueAt and RemindAt.
type Patch struct {
	Title       *string
	Notes       *string
	Status      *Status
	Priority    *Priority
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

// Store is everything the UI needs. internal/domain and internal/watch will implement it.
type Store interface {
	Manager

	ListProjects() ([]Project, error)
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

	// KeepClosedPR dismisses the Needs you row of a task whose pull request was closed without
	// merging; the task stays as it is.
	KeepClosedPR(id int) error

	// HandOff gives a task of the user's to a coding agent. The task becomes the agent's (Source
	// agent, To do, HandedTo set) and waits there until the agent reports on it. The agent is the
	// one that last reported in the task's project, else the first one found on this Mac. When its
	// command and the project's folder are found, it starts in a Terminal window with prompt and
	// Launched is true; otherwise the view copies prompt for the user to paste.
	HandOff(id int, prompt string) (HandOffResult, error)
	// TakeBack makes an agent's task the user's again: Source user, HandedTo cleared. Its status
	// stays.
	TakeBack(id int) (Task, error)
	// RestoreHandOff puts a task's Source and hand-off back as they were, for the Undo of TakeBack:
	// agent "" and at nil mean "not waiting". It starts nothing and leaves the status alone.
	RestoreHandOff(id int, agent string, at *time.Time, source Source) (Task, error)
}

// HandOffResult is what HandOff did.
type HandOffResult struct {
	Task      Task
	Agent     string // the agentsetup ID, as in Task.HandedTo
	AgentName string // as people write it: "Claude Code"
	Launched  bool   // the agent was started; false: the prompt is for the clipboard
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
