package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// ErrNothingToUndo is returned for a receipt that was already undone or never existed.
var ErrNothingToUndo = errors.New("nothing to undo")

// journal remembers, per receipt, the one function that reverts exactly what a tool changed.
type journal struct {
	mu    sync.Mutex
	next  int
	undos map[int]func(context.Context) error
}

func newJournal() *journal { return &journal{undos: map[int]func(context.Context) error{}} }

// add records an undo and returns the receipt id for it.
func (j *journal) add(undo func(context.Context) error) int {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.next++
	j.undos[j.next] = undo
	return j.next
}

// run reverts one receipt, once. A failed undo stays available to try again.
func (j *journal) run(ctx context.Context, id int) error {
	j.mu.Lock()
	undo, ok := j.undos[id]
	j.mu.Unlock()
	if !ok {
		return ErrNothingToUndo
	}
	if err := undo(ctx); err != nil {
		return err
	}
	j.mu.Lock()
	delete(j.undos, id)
	j.mu.Unlock()
	return nil
}

// undoCreate deletes the task the agent created. A task that is already gone is as good as undone.
func undoCreate(core *domain.Core, id int64) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := core.DeleteTask(ctx, id); err != nil && !domain.IsNotFound(err) {
			return err
		}
		return nil
	}
}

// undoPatch puts back the fields a patch changed, and only those, from the task as it was before.
func undoPatch(core *domain.Core, before domain.Task, applied domain.TaskPatch) func(context.Context) error {
	revert := revertPatch(before, applied)
	return func(ctx context.Context) error {
		if err := dropGoneCategory(ctx, core, &revert); err != nil {
			return err
		}
		if _, err := core.UpdateTask(ctx, before.ID, revert); err != nil {
			if domain.IsNotFound(err) {
				return fmt.Errorf("the task no longer exists")
			}
			return err
		}
		return nil
	}
}

// dropGoneCategory turns a revert to a category that was deleted meanwhile into leaving the task
// unsorted, which is where the deletion left every other task of it.
func dropGoneCategory(ctx context.Context, core *domain.Core, revert *domain.TaskPatch) error {
	if !revert.CategoryID.Present || revert.CategoryID.Value == nil {
		return nil
	}
	cats, err := core.ListCategories(ctx)
	if err != nil {
		return err
	}
	for _, c := range cats {
		if c.ID == *revert.CategoryID.Value {
			return nil
		}
	}
	revert.CategoryID = domain.Clear[int64]()
	return nil
}

// revertPatch is the patch that undoes applied on a task that looked like before.
func revertPatch(before domain.Task, applied domain.TaskPatch) domain.TaskPatch {
	var r domain.TaskPatch
	if applied.Title != nil {
		r.Title = &before.Title
	}
	if applied.Status != nil {
		r.Status = &before.Status
	}
	if applied.Priority != nil {
		r.Priority = &before.Priority
	}
	r.CategoryID = restore(applied.CategoryID, before.CategoryID)
	r.Notes = restore(applied.Notes, before.Notes)
	r.ProjectID = restore(applied.ProjectID, before.ProjectID)
	r.MilestoneID = restore(applied.MilestoneID, before.MilestoneID)
	r.DueAt = restore(applied.DueAt, before.DueAt)
	r.RemindAt = restore(applied.RemindAt, before.RemindAt)
	r.Branch = restore(applied.Branch, before.Branch)
	r.PRURL = restore(applied.PRURL, before.PRURL)
	return r
}

// restore sets a field back to its old value (clearing it when it had none) if the patch touched it.
func restore[T any](applied domain.Nullable[T], old *T) domain.Nullable[T] {
	if !applied.Present {
		return domain.Nullable[T]{}
	}
	return domain.Nullable[T]{Present: true, Value: old}
}
