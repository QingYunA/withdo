package core

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The select mode's bulk bar (ui-spec, Select): one change to several tasks that either applies to
// all of them or to none, with an Undo for each.

// readIDs reads a JSON array of task ids; an empty one is an error, as nothing is selected.
func readIDs(idsJSON string) ([]int, []int64, error) {
	var ids []int64
	if err := json.Unmarshal([]byte(idsJSON), &ids); err != nil {
		return nil, nil, errors.New("could not read the tasks")
	}
	if len(ids) == 0 {
		return nil, nil, errors.New("no tasks selected")
	}
	small := make([]int, len(ids))
	for i, id := range ids {
		small[i] = int(id)
	}
	return small, ids, nil
}

// bulkPatchJSON is the patch of UpdateTasks. An absent key is left alone.
//
//	{"status":"done","priority":"P1","project_id":2}
//
// project_id 0 moves the tasks to the Inbox; their milestones stay behind. Priority P1 is what the
// desktop's "important" star sets.
type bulkPatchJSON struct {
	Status    *string `json:"status"`
	Priority  *string `json:"priority"`
	ProjectID *int64  `json:"project_id"`
}

// UpdateTasks applies one patch (status, priority, project_id) to every task in idsJSON, a JSON
// array of task ids, as one change: the open subtasks of a task that becomes done or cancelled
// close with it. It returns the tasks as they were before, as JSON (a []data.Task, with the
// subtasks that closed), which RestoreTaskStates takes as the Undo.
func (c *Core) UpdateTasks(idsJSON, patchJSON string) (string, error) {
	ids, _, err := readIDs(idsJSON)
	if err != nil {
		return "", err
	}
	var in bulkPatchJSON
	if err := json.Unmarshal([]byte(patchJSON), &in); err != nil {
		return "", errors.New("could not read the change")
	}
	var patch data.BulkPatch
	if in.ProjectID != nil {
		project := int(*in.ProjectID)
		patch.ProjectID = &project
	}
	if in.Status != nil {
		status := data.Status(*in.Status)
		patch.Status = &status
	}
	if in.Priority != nil {
		priority := data.Priority(*in.Priority)
		patch.Priority = &priority
	}
	before, err := c.real.UpdateTasks(ids, patch)
	if err != nil {
		return "", shortError(err)
	}
	return marshal(emptyIfNil(before))
}

// RestoreTaskStates is the Undo of UpdateTasks: it puts the tasks back as UpdateTasks returned them
// (where they were, their status, priority and finish time). A task that is gone since is skipped.
func (c *Core) RestoreTaskStates(beforeJSON string) error {
	var before []data.Task
	if err := json.Unmarshal([]byte(beforeJSON), &before); err != nil {
		return errors.New("could not read the earlier state")
	}
	if err := c.real.RestoreStates(before); err != nil {
		return shortError(err)
	}
	return nil
}

// DeleteTasks removes every task in idsJSON with its reports, or none, and returns the snapshots
// RestoreTasks needs as JSON (a list of internal/domain.TaskSnapshot). The JSON is opaque: pass it
// back unchanged to undo the delete.
func (c *Core) DeleteTasks(idsJSON string) (string, error) {
	_, ids, err := readIDs(idsJSON)
	if err != nil {
		return "", err
	}
	snaps, err := c.dom.DeleteTasks(context.Background(), ids)
	if err != nil {
		return "", shortError(err)
	}
	c.real.CollectSoon()
	return marshal(emptyIfNil(snaps))
}

// RestoreTasks puts back the tasks DeleteTasks removed, with their ids, fields and reports, from
// the snapshots JSON it returned. It is the Undo of DeleteTasks.
func (c *Core) RestoreTasks(snapshotsJSON string) error {
	var snaps []domain.TaskSnapshot
	if err := json.Unmarshal([]byte(snapshotsJSON), &snaps); err != nil {
		return errors.New("could not read the deleted tasks")
	}
	if err := c.dom.RestoreTasks(context.Background(), snaps); err != nil {
		return shortError(err)
	}
	return nil
}

// doneToday is one task DoTodayMany moved, with the due date it had.
type doneToday struct {
	ID     int64  `json:"id"`
	Before string `json:"before"` // RFC 3339, "" when there was none
}

// DoTodayMany is DoToday for the tasks in idsJSON, a JSON array of task ids: every task of mine that
// is not finished and not due today is moved to today, keeping its time of day. The others are left
// alone. It returns the tasks it moved with their old due dates, as JSON:
//
//	[{"id":3,"before":"2026-10-25T09:00:00+08:00"}, {"id":5,"before":""}]
//
// The Undo is UpdateTask(id, {"due_at": before}) for each ("due_at": null when before is "").
func (c *Core) DoTodayMany(idsJSON string) (string, error) {
	_, ids, err := readIDs(idsJSON)
	if err != nil {
		return "", err
	}
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	byID := make(map[int64]data.Task, len(snap.Tasks))
	for _, t := range snap.Tasks {
		byID[int64(t.ID)] = t
	}
	now := c.nowTime()
	moved := []doneToday{}
	for _, id := range ids {
		task, ok := byID[id]
		if !ok || !canDoToday(task, now) {
			continue
		}
		before, err := c.moveToToday(task, now)
		if err != nil {
			return "", err
		}
		moved = append(moved, doneToday{ID: id, Before: before})
	}
	return marshal(moved)
}
