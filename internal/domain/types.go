// Package domain is adhd-todo's data model and the operations on it: projects,
// milestones, tasks, agent progress reports, the overview and reminders.
package domain

import (
	"fmt"
	"time"
)

// Status is a task's lifecycle state.
type Status string

// Task statuses.
const (
	StatusBacklog   Status = "backlog"
	StatusTodo      Status = "todo"
	StatusDoing     Status = "doing"
	StatusBlocked   Status = "blocked"
	StatusReview    Status = "review"
	StatusDone      Status = "done"
	StatusCancelled Status = "cancelled"
)

// Statuses lists every status in board order.
var Statuses = []Status{StatusBacklog, StatusTodo, StatusDoing, StatusBlocked, StatusReview, StatusDone, StatusCancelled}

// Terminal reports whether the status ends a task's life.
func (s Status) Terminal() bool { return s == StatusDone || s == StatusCancelled }

// Active reports whether the status means an agent is working on or waiting for the task.
func (s Status) Active() bool { return s == StatusDoing || s == StatusBlocked || s == StatusReview }

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	for _, k := range Statuses {
		if s == k {
			return true
		}
	}
	return false
}

// Priority runs from P0 (urgent) to P3 (low).
type Priority string

// Task priorities.
const (
	P0 Priority = "P0"
	P1 Priority = "P1"
	P2 Priority = "P2"
	P3 Priority = "P3"
)

// Priorities lists every priority, most urgent first.
var Priorities = []Priority{P0, P1, P2, P3}

// Valid reports whether p is a known priority.
func (p Priority) Valid() bool {
	for _, k := range Priorities {
		if p == k {
			return true
		}
	}
	return false
}

// Category says what a task is for the user: work or life. A task nobody classified has none.
type Category string

// Task categories.
const (
	CategoryWork Category = "work"
	CategoryLife Category = "life"
)

// Categories lists every category.
var Categories = []Category{CategoryWork, CategoryLife}

// Valid reports whether c is a known category.
func (c Category) Valid() bool {
	for _, k := range Categories {
		if c == k {
			return true
		}
	}
	return false
}

// Source says who created a task.
type Source string

// Task sources.
const (
	SourceUser  Source = "user"
	SourceAgent Source = "agent"
)

// Project groups tasks by repository. Its JSON form matches the database columns.
type Project struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	RepoSlug   *string `json:"repo_slug"`
	RepoPath   *string `json:"repo_path"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	ArchivedAt *string `json:"archived_at"`
	// ParentID is the project this one hangs under as a sub-project; nil for a top-level project.
	ParentID *int64 `json:"parent_id"`
}

// Milestone is a dated goal inside a project.
type Milestone struct {
	ID        int64   `json:"id"`
	ProjectID int64   `json:"project_id"`
	Title     string  `json:"title"`
	DueAt     *string `json:"due_at"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

// Task is a unit of work. Timestamps are UTC ISO 8601 with milliseconds.
type Task struct {
	ID          int64    `json:"id"`
	ProjectID   *int64   `json:"project_id"`
	MilestoneID *int64   `json:"milestone_id"`
	Title       string   `json:"title"`
	Notes       *string  `json:"notes"`
	Status      Status   `json:"status"`
	Priority    Priority `json:"priority"`
	DueAt       *string  `json:"due_at"`
	RemindAt    *string  `json:"remind_at"`
	Branch      *string  `json:"branch"`
	PRURL       *string  `json:"pr_url"`
	PRNumber    *int64   `json:"pr_number"`
	Source      Source   `json:"source"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	CompletedAt *string  `json:"completed_at"`
	// RemindedAt is when the reminder for the current RemindAt fired; nil until it does.
	RemindedAt *string `json:"reminded_at"`
	// What the GitHub poller last saw of PRURL; nil until the first check and after PRURL changes.
	PRState        *PRState  `json:"pr_state"`
	PRChecks       *PRChecks `json:"pr_checks"`
	PRReview       *PRReview `json:"pr_review"`
	PRCheckedAt    *string   `json:"pr_checked_at"`
	PRClosedSeenAt *string   `json:"pr_closed_seen_at"` // the user dismissed the "closed" state
	// ParentID is the task this one is a subtask of: the task of the agent that started this
	// task's subagent. Nil for a top-level task.
	ParentID *int64 `json:"parent_id"`
	// HandedTo says the user handed the task over, until an agent first reports on it: "agent", any
	// agent (rows of an earlier build may hold an agent's id); HandedAt is when. Both are nil otherwise.
	HandedTo *string `json:"handed_to"`
	HandedAt *string `json:"handed_at"`
	// Category is what the task is for the user, work or life; nil until someone classifies it.
	Category *Category `json:"category"`
}

// Report is one progress note an agent filed against a task.
type Report struct {
	ID        int64   `json:"id"`
	TaskID    int64   `json:"task_id"`
	Agent     string  `json:"agent"`
	Status    *Status `json:"status"`
	Summary   string  `json:"summary"`
	NextStep  *string `json:"next_step"`
	CreatedAt string  `json:"created_at"`
	// Cwd is the working directory the agent reported from; "" when it did not say.
	Cwd string `json:"cwd"`
	// Kind says what a report of the GitHub poller is about; "" for any other report.
	Kind ReportKind `json:"kind,omitempty"`
	// Links are the pages the agent attached to the report; never nil.
	Links []ReportLink `json:"links"`
}

// timeLayout is JavaScript's Date.toISOString form, kept for database compatibility.
const timeLayout = "2006-01-02T15:04:05.000Z"

// FormatTime renders t the way the database stores timestamps.
func FormatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

// ParseTimestamp validates a user-supplied ISO 8601 date ("2026-12-01") or
// datetime ("2026-12-01T09:00:00+08:00") and returns it in stored form (UTC,
// milliseconds). A datetime without an offset is local time.
func ParseTimestamp(s string) (string, error) { return ParseTimestampIn(s, time.Local) }

// ParseTimestampIn is ParseTimestamp with the zone that a datetime without an offset is read in.
// A date alone is UTC midnight.
func ParseTimestampIn(s string, loc *time.Location) (string, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return FormatTime(t), nil
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return FormatTime(t), nil
		}
	}
	return "", fmt.Errorf("must be an ISO 8601 date or datetime, e.g. 2026-12-01 or 2026-12-01T09:00 (local time) or 2026-12-01T09:00:00Z")
}
