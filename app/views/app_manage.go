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
