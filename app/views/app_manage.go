package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The writes of managing projects, milestones and tasks by hand. Each refreshes the snapshot; an
// error is kept for the calm banner and also returned, for a sheet that can say more.

// do runs a write and refreshes, or keeps the error for the banner.
func (a *App) do(write func() error) error {
	if err := write(); err != nil {
		return err
	}
	a.Refresh()
	return nil
}

// fail keeps err for the banner, unless it is a problem the caller words itself.
func (a *App) fail(err error) error {
	if err != nil {
		a.snap.err = err
	}
	return err
}

// CreateProject adds a project and returns it. The sheet words ErrNameEmpty, ErrRepoInvalid and
// *RepoTakenError; any other failure goes to the banner.
func (a *App) CreateProject(in data.NewProject) (data.Project, error) {
	var p data.Project
	err := a.do(func() (err error) { p, err = a.Store.CreateProject(in); return })
	return p, err
}

// CreateCategory adds a category and returns it. The sheet words the store's problems with a name or
// the list (ErrNameEmpty, ErrCategoryNameLong, ErrCategoryNameTaken, ErrCategoryBuiltinName,
// ErrTooManyCategories); any other failure goes to the banner.
func (a *App) CreateCategory(in data.NewCategory) (data.Category, error) {
	var c data.Category
	err := a.do(func() (err error) { c, err = a.Store.CreateCategory(in); return })
	return c, err
}

// UpdateCategory changes a category; the sheet words its problems like CreateCategory's.
func (a *App) UpdateCategory(id int, p data.CategoryPatch) error {
	return a.do(func() error { _, err := a.Store.UpdateCategory(id, p); return err })
}

// MoveCategory puts a category at a place of the list, 0 being the first.
func (a *App) MoveCategory(id, index int) error {
	return a.fail(a.do(func() error { return a.Store.MoveCategory(id, index) }))
}

// DeleteCategory removes a category; its tasks become unsorted. The result is what RestoreCategory
// needs to put it back.
func (a *App) DeleteCategory(id int) (data.DeletedCategory, error) {
	var d data.DeletedCategory
	err := a.fail(a.do(func() (err error) { d, err = a.Store.DeleteCategory(id); return }))
	return d, err
}

// RestoreCategory is the Undo of DeleteCategory: the category comes back where it was, and so do
// the tasks it held that are still unsorted.
func (a *App) RestoreCategory(d data.DeletedCategory) error {
	return a.fail(a.do(func() error { _, err := a.Store.RestoreCategory(d); return err }))
}

func (a *App) RenameProject(id int, name string) error {
	return a.fail(a.do(func() error { _, err := a.Store.RenameProject(id, name); return err }))
}

func (a *App) ArchiveProject(id int) error {
	return a.fail(a.do(func() error { return a.Store.ArchiveProject(id) }))
}

func (a *App) RestoreProject(id int) error {
	return a.fail(a.do(func() error { return a.Store.RestoreProject(id) }))
}

func (a *App) DeleteProject(id int) error {
	return a.fail(a.do(func() error { _, err := a.Store.DeleteProject(id); return err }))
}

func (a *App) DeleteTask(id int) error {
	return a.fail(a.do(func() error { return a.Store.DeleteTask(id) }))
}

// SnapshotTask reads a task with its reports before it is deleted; RestoreTask puts it back.
func (a *App) SnapshotTask(id int) (data.TaskSnapshot, error) {
	snap, err := a.Store.SnapshotTask(id)
	a.fail(err)
	return snap, err
}

func (a *App) RestoreTask(s data.TaskSnapshot) error {
	return a.fail(a.do(func() error { _, err := a.Store.RestoreTask(s); return err }))
}

// MoveTask moves a task to a project, or to the Inbox for project 0.
func (a *App) MoveTask(t data.Task, project int) {
	if (t.ProjectID == nil && project == 0) || (t.ProjectID != nil && *t.ProjectID == project) {
		return
	}
	_ = a.Update(t.ID, data.Patch{ProjectID: data.IntPtr(project)})
}

func (a *App) CreateMilestone(project int, title string, due *time.Time) error {
	return a.fail(a.do(func() error { _, err := a.Store.CreateMilestone(project, title, due); return err }))
}

func (a *App) UpdateMilestone(id int, p data.MilestonePatch) error {
	return a.fail(a.do(func() error { _, err := a.Store.UpdateMilestone(id, p); return err }))
}

// taskCount is how many tasks a project holds, whatever their status.
func (a *App) taskCount(project int) int {
	n := 0
	for _, t := range a.snap.tasks {
		if t.ProjectID != nil && *t.ProjectID == project {
			n++
		}
	}
	return n
}

// DetachImage takes a picture off a task; RestoreImage puts it back, for the Undo.
func (a *App) DetachImage(id int, hash string) error {
	return a.fail(a.do(func() error { return a.Store.DetachImage(id, hash) }))
}

func (a *App) RestoreImage(im data.TaskImage) error {
	return a.fail(a.do(func() error { return a.Store.RestoreImage(im) }))
}
