package data

import (
	"errors"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

var _ Bulk = (*Real)(nil)

func int64s(ids []int) []int64 {
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = int64(id)
	}
	return out
}

// UpdateTasks implements Bulk.
func (r *Real) UpdateTasks(ids []int, p Patch) ([]Task, error) {
	if len(ids) == 0 {
		return nil, errNoTasks
	}
	before, err := r.core.UpdateTasks(r.ctx(), int64s(ids), toPatch(p))
	if err != nil {
		return nil, err
	}
	return fromTasks(before), nil
}

// HandOffTasks implements Bulk.
func (r *Real) HandOffTasks(ids []int) ([]Task, error) {
	if len(ids) == 0 {
		return nil, errNoTasks
	}
	before, err := r.core.HandOffTasks(r.ctx(), int64s(ids))
	if err != nil {
		return nil, err
	}
	return fromTasks(before), nil
}

// RestoreStates implements Bulk.
func (r *Real) RestoreStates(before []Task) error {
	states := make([]domain.TaskState, len(before))
	for i, t := range before {
		s := domain.TaskState{
			ID: int64(t.ID), ProjectID: int64Ptr(t.ProjectID), MilestoneID: int64Ptr(t.MilestoneID),
			Status: domain.Status(t.Status), Priority: domain.Priority(t.Priority), Source: domain.Source(t.Source),
			HandedAt: formatTimePtr(t.HandedAt), CompletedAt: formatTimePtr(t.CompletedAt),
		}
		if t.HandedTo != "" {
			s.HandedTo = &t.HandedTo
		}
		states[i] = s
	}
	return r.core.RestoreStates(r.ctx(), states)
}

// DeleteTasks implements Bulk.
func (r *Real) DeleteTasks(ids []int) ([]TaskSnapshot, error) {
	if len(ids) == 0 {
		return nil, errNoTasks
	}
	snaps, err := r.core.DeleteTasks(r.ctx(), int64s(ids))
	if err != nil {
		return nil, err
	}
	r.CollectSoon()
	out := make([]TaskSnapshot, len(snaps))
	for i, s := range snaps {
		out[i] = snapshotOf(s)
	}
	return out, nil
}

// RestoreTasks implements Bulk.
func (r *Real) RestoreTasks(snaps []TaskSnapshot) error {
	raw := make([]domain.TaskSnapshot, len(snaps))
	for i, s := range snaps {
		d, ok := s.raw.(domain.TaskSnapshot)
		if !ok {
			return errors.New("snapshot was not taken by this store")
		}
		raw[i] = d
	}
	return r.core.RestoreTasks(r.ctx(), raw)
}

// snapshotOf is a domain snapshot as the UI holds it: its reports newest first, as GetTask has them.
func snapshotOf(snap domain.TaskSnapshot) TaskSnapshot {
	out := TaskSnapshot{Task: fromTask(snap.Task), raw: snap}
	for i := len(snap.Reports) - 1; i >= 0; i-- {
		out.Reports = append(out.Reports, fromReport(snap.Reports[i]))
	}
	return out
}
