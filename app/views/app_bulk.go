package views

import "github.com/QingYunA/adhd-todo/app/data"

// The writes of changing several tasks at once. Each is one call to the store, so it is one
// change, and each refreshes the snapshot; an error is kept for the banner and returned.

// UpdateTasks applies one patch to every task in ids and returns the tasks as they were, which
// RestoreStates puts back.
func (a *App) UpdateTasks(ids []int, p data.Patch) ([]data.Task, error) {
	var before []data.Task
	err := a.do(func() (err error) { before, err = a.Store.UpdateTasks(ids, p); return })
	return before, a.fail(err)
}

// HandOffTasks hands every task in ids to any agent and returns the tasks as they were.
func (a *App) HandOffTasks(ids []int) ([]data.Task, error) {
	var before []data.Task
	err := a.do(func() (err error) { before, err = a.Store.HandOffTasks(ids); return })
	return before, a.fail(err)
}

// RestoreStates puts tasks back as UpdateTasks or HandOffTasks found them. Whatever of them was
// playing its completion stops.
func (a *App) RestoreStates(before []data.Task) error {
	for _, t := range before {
		delete(a.completing, t.ID)
	}
	return a.fail(a.do(func() error { return a.Store.RestoreStates(before) }))
}

// DeleteTasks removes every task in ids and returns what RestoreTasks needs to put them back.
func (a *App) DeleteTasks(ids []int) ([]data.TaskSnapshot, error) {
	var snaps []data.TaskSnapshot
	err := a.do(func() (err error) { snaps, err = a.Store.DeleteTasks(ids); return })
	return snaps, a.fail(err)
}

// RestoreTasks puts deleted tasks back with their ids and reports.
func (a *App) RestoreTasks(snaps []data.TaskSnapshot) error {
	return a.fail(a.do(func() error { return a.Store.RestoreTasks(snaps) }))
}
