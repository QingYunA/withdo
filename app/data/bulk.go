package data

import "errors"

// BulkPatch is what a bulk change may touch: only the fields RestoreStates puts back, so an Undo
// can never lose part of a change. A field the Undo cannot restore has no place here.
type BulkPatch struct {
	Status    *Status
	Priority  *Priority
	ProjectID *int // 0 moves the tasks to the Inbox; the milestone stays behind, as with Patch
}

// patch is p as the single-task Patch the store applies.
func (p BulkPatch) patch() Patch {
	return Patch{Status: p.Status, Priority: p.Priority, ProjectID: p.ProjectID}
}

// Bulk is changing several tasks as one: each call is one change that either applies to every task
// or to none, and hands back what one Undo needs.
type Bulk interface {
	// UpdateTasks applies one patch to every task in ids. It returns the tasks as they were before,
	// and with them the open subtasks that closed along with a task that became done or
	// cancelled; RestoreStates puts them back.
	UpdateTasks(ids []int, p BulkPatch) (before []Task, err error)
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
