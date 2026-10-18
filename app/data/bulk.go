package data

import (
	"errors"
	"fmt"
	"slices"
)

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

// DeleteTasks implements Bulk.
func (f *Fake) DeleteTasks(ids []int) ([]TaskSnapshot, error) {
	if len(ids) == 0 {
		return nil, errNoTasks
	}
	f.mu.Lock()
	snaps := make([]TaskSnapshot, 0, len(ids))
	drop := map[int]bool{}
	for _, id := range ids {
		t, ok := f.find(id)
		if !ok {
			f.mu.Unlock()
			return nil, fmt.Errorf("task %d not found", id)
		}
		snaps = append(snaps, TaskSnapshot{Task: *t, Reports: f.reportsOf(id), raw: slices.Clone(f.pics.links[id])})
		drop[id] = true
	}
	var kept []Task
	for _, t := range f.tasks {
		if drop[t.ID] {
			f.pics.dropTask(t.ID)
			continue
		}
		kept = append(kept, t)
	}
	f.tasks = kept
	f.reports, _ = dropReports(f.reports, drop)
	f.mu.Unlock()
	f.notify()
	return snaps, nil
}

// RestoreTasks implements Bulk.
func (f *Fake) RestoreTasks(snaps []TaskSnapshot) error {
	f.mu.Lock()
	for _, s := range snaps {
		if _, ok := f.find(s.Task.ID); ok {
			f.mu.Unlock()
			return fmt.Errorf("task %d already exists", s.Task.ID)
		}
	}
	for _, s := range snaps {
		t := s.Task
		if t.ProjectID != nil {
			if _, ok := f.project(*t.ProjectID); !ok {
				t.ProjectID, t.MilestoneID = nil, nil
			}
		}
		f.tasks = append(f.tasks, t)
		f.reports = append(f.reports, s.Reports...)
		if links, ok := s.raw.([]TaskImage); ok {
			for _, l := range links {
				f.pics.add(t.ID, l)
			}
		}
	}
	f.mu.Unlock()
	f.notify()
	return nil
}

// UpdateTasks implements Bulk.
func (f *Fake) UpdateTasks(ids []int, p Patch) ([]Task, error) {
	if len(ids) == 0 {
		return nil, errNoTasks
	}
	f.mu.Lock()
	snapshot, reports := slices.Clone(f.tasks), slices.Clone(f.reports)
	var before []Task
	for _, id := range ids {
		t, ok := f.find(id)
		if !ok {
			f.tasks, f.reports = snapshot, reports
			f.mu.Unlock()
			return nil, fmt.Errorf("task %d not found", id)
		}
		before = append(before, *t)
		if p.Status != nil && p.Status.Terminal() && !t.Status.Terminal() {
			before = append(before, f.openSubtasks(id)...)
		}
		if err := f.apply(t, p); err != nil {
			f.tasks, f.reports = snapshot, reports
			f.mu.Unlock()
			return nil, err
		}
	}
	f.mu.Unlock()
	f.notify()
	return before, nil
}

// openSubtasks are the subtasks of id, and theirs, that are not finished.
func (f *Fake) openSubtasks(id int) []Task {
	var out []Task
	for _, k := range f.tasks {
		if k.ParentID != nil && *k.ParentID == id && !k.Status.Terminal() {
			out = append(out, k)
			out = append(out, f.openSubtasks(k.ID)...)
		}
	}
	return out
}

// HandOffTasks implements Bulk.
func (f *Fake) HandOffTasks(ids []int) ([]Task, error) {
	if len(ids) == 0 {
		return nil, errNoTasks
	}
	f.mu.Lock()
	var before []Task
	for _, id := range ids {
		if t, ok := f.find(id); ok {
			before = append(before, *t)
			continue
		}
		f.mu.Unlock()
		return nil, fmt.Errorf("task %d not found", id)
	}
	now := f.Now()
	for _, id := range ids {
		t, _ := f.find(id)
		t.Source, t.Status, t.HandedTo, t.HandedAt, t.CompletedAt, t.UpdatedAt = SourceAgent, Todo, AnyAgent, &now, nil, now
	}
	f.mu.Unlock()
	f.notify()
	return before, nil
}

// RestoreStates implements Bulk.
func (f *Fake) RestoreStates(before []Task) error {
	f.mu.Lock()
	now := f.Now()
	for _, was := range before {
		t, ok := f.find(was.ID)
		if !ok {
			continue
		}
		t.ProjectID, t.MilestoneID = was.ProjectID, was.MilestoneID
		if t.ProjectID != nil {
			if _, ok := f.project(*t.ProjectID); !ok {
				t.ProjectID, t.MilestoneID = nil, nil
			}
		}
		t.Status, t.Priority, t.Source = was.Status, was.Priority, was.Source
		t.HandedTo, t.HandedAt, t.CompletedAt, t.UpdatedAt = was.HandedTo, was.HandedAt, was.CompletedAt, now
	}
	f.mu.Unlock()
	f.notify()
	return nil
}
