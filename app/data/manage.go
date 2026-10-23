package data

import (
	"errors"
	"time"
)

// Manager is the editing of projects, milestones and tasks by hand: what an agent can do through the
// MCP server, and what it cannot, done from the window.
type Manager interface {
	// CreateProject adds a project. A repository is optional; a GitHub address pasted in its place
	// is read as owner/repo, and ParentID makes the project a sub-project. Failures are ErrNameEmpty,
	// ErrRepoInvalid and *RepoTakenError; a parent that is gone or archived is any other error, for
	// the banner.
	CreateProject(NewProject) (Project, error)
	RenameProject(id int, name string) (Project, error)
	// ArchiveProject hides a project, its tasks and its sub-projects; RestoreProject brings back the
	// project and the sub-projects archived with it. A report from an agent in the project's
	// repository restores it too.
	ArchiveProject(id int) error
	RestoreProject(id int) error
	// DeleteProject removes the project with its tasks, their reports and its milestones, or nothing.
	DeleteProject(id int) (Removed, error)
	// DeleteTask removes a task and its reports.
	DeleteTask(id int) error
	// SnapshotTask reads a task with its reports, so RestoreTask can put it back after DeleteTask:
	// the Undo of deleting a task. RestoreTask keeps the id and every field; a task whose project is
	// gone returns to the Inbox. It fails when the id is taken again.
	SnapshotTask(id int) (TaskSnapshot, error)
	RestoreTask(TaskSnapshot) (Task, error)

	// CreateCategory adds a category at the end of the list. Failures are ErrNameEmpty,
	// ErrCategoryNameLong, ErrCategoryNameTaken, ErrTooManyCategories, ErrIconEmpty and ErrCategoryColor.
	CreateCategory(NewCategory) (Category, error)
	// UpdateCategory renames, redraws or recolors a category; Work and Life answer
	// ErrCategoryBuiltinName to a new name.
	UpdateCategory(id int, p CategoryPatch) (Category, error)
	// MoveCategory puts a category at index (0-based) in the list, the built-in two included.
	MoveCategory(id, index int) error
	// DeleteCategory removes a category of the user's; its tasks become unsorted. Work and Life
	// answer ErrCategoryBuiltin. RestoreCategory is its Undo: the category comes back with its id at
	// its old place, and takes back the tasks that are still unsorted.
	DeleteCategory(id int) (DeletedCategory, error)
	RestoreCategory(DeletedCategory) (Category, error)

	CreateMilestone(projectID int, title string, dueAt *time.Time) (Milestone, error)
	UpdateMilestone(id int, p MilestonePatch) (Milestone, error)

	// SearchTasks finds tasks of any status whose title, notes or branch contain every word of the
	// query, or whose pull request number is one of them; newest first, at most limit (0 for the
	// default). Tasks of archived projects are left out.
	SearchTasks(query string, limit int) ([]Task, error)
	// Logbook lists the done and cancelled tasks, newest completion first, at most limit.
	Logbook(limit int) ([]Task, error)
	// LogbookOf is Logbook for one project (0 for all of them), or with agents for the tasks agents wrote.
	LogbookOf(project int, agents bool, limit int) ([]Task, error)
}

// TaskSnapshot is a task with its reports as they were; only the store that made it can restore it.
type TaskSnapshot struct {
	Task    Task
	Reports []Report // newest first
	raw     any      // the store's own form
}

// NewProject is what the New project sheet sends. ParentID makes it a sub-project of that project.
type NewProject struct {
	Name     string
	RepoSlug string // owner/repo, or empty
	ParentID *int
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
