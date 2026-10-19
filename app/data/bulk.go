package data

import "errors"

// Bulk is changing several tasks as one: each call is one change that either applies to every task
// or to none, and hands back what one Undo needs.
type Bulk interface {
	// UpdateTasks applies one patch to every task in ids. It returns the tasks as they were before,
	// and with them the open subtasks that closed along with a task that became done or
	// cancelled; RestoreStates puts them back.
	UpdateTasks(ids []int, p Patch) (before []Task, err error)
	// HandOffTasks hands every task in ids to any agent, as HandOff does for one, and returns the
	// tasks as they were before. The caller leaves out what cannot be handed over.
	HandOffTasks(ids []int) (before []Task, err error)
	// RestoreStates puts tasks back as they were before UpdateTasks or HandOffTasks: where they
	// are (project, milestone), their status, priority, whose they are, whether they wait for an
	// agent, and when they were finished. A task that is gone since is skipped.
	RestoreStates(before []Task) error
	// DeleteTasks removes every task in ids with its reports; RestoreTasks puts them back with
	// their ids and fields.
	DeleteTasks(ids []int) ([]TaskSnapshot, error)
	RestoreTasks(snaps []TaskSnapshot) error
}

var errNoTasks = errors.New("no tasks given")
