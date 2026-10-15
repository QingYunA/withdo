package data

import (
	"errors"
	"time"
)

// Manager is the editing of projects, milestones and tasks by hand: what an agent can do through the
// MCP server, and what it cannot, done from the window.
type Manager interface {
	// CreateProject adds a project. A repository is optional; a GitHub address pasted in its place
	// is read as owner/repo. Failures are ErrNameEmpty, ErrRepoInvalid and *RepoTakenError.
	CreateProject(NewProject) (Project, error)
	RenameProject(id int, name string) (Project, error)
	// ArchiveProject hides a project and its tasks; RestoreProject brings them back. A report from
	// an agent in the project's repository restores it too.
	ArchiveProject(id int) error
	RestoreProject(id int) error
	ListArchivedProjects() ([]Project, error)
	// DeleteProject removes the project with its tasks, their reports and its milestones, or nothing.
	DeleteProject(id int) (Removed, error)
	// DeleteTask removes a task and its reports.
	DeleteTask(id int) error

	CreateMilestone(projectID int, title string, dueAt *time.Time) (Milestone, error)
	UpdateMilestone(id int, p MilestonePatch) (Milestone, error)

	// SearchTasks finds tasks of any status whose title, notes or branch contain every word of the
	// query, or whose pull request number is one of them; newest first, at most limit (0 for the
	// default). Tasks of archived projects are left out.
	SearchTasks(query string, limit int) ([]Task, error)
	// Logbook lists the done and cancelled tasks, newest completion first, at most limit.
	Logbook(limit int) ([]Task, error)
}

// NewProject is what the New project sheet sends.
type NewProject struct {
	Name     string
	RepoSlug string // owner/repo, or empty
}

// Removed counts what DeleteProject took with the project.
type Removed struct {
	Tasks, Reports, Milestones int
}

// MilestonePatch changes a milestone. A nil field is left alone; the zero time clears DueAt.
type MilestonePatch struct {
	Title *string
	DueAt *time.Time
	Open  *bool
}

// Problems with a new project's input, for the views to word.
var (
	ErrNameEmpty   = errors.New("name must not be empty")
	ErrRepoInvalid = errors.New("repository must look like owner/repo")
)

// RepoTakenError says another project already has the repository.
type RepoTakenError struct{ Project string }

func (e *RepoTakenError) Error() string {
	return "repository is already used by project " + e.Project
}

// BoolPtr returns a pointer to v.
func BoolPtr(v bool) *bool { return &v }
