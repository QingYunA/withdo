package core

import (
	"encoding/json"
	"errors"

	"github.com/QingYunA/adhd-todo/app/data"
)

// manageError words a failure of a project, milestone or category edit: the sentences app/data
// promises for a bad input (an empty name, a taken category name) pass through, anything else is
// shortError's.
func manageError(err error) error {
	var taken *data.RepoTakenError
	for _, known := range []error{
		data.ErrNameEmpty, data.ErrRepoInvalid, data.ErrCategoryNameLong, data.ErrCategoryNameTaken,
		data.ErrCategoryBuiltinName, data.ErrCategoryBuiltin, data.ErrTooManyCategories, data.ErrIconEmpty,
		data.ErrCategoryColor,
	} {
		if errors.Is(err, known) {
			return known
		}
	}
	if errors.As(err, &taken) {
		return taken
	}
	return shortError(err)
}

// projectRow is one row of the Projects tab.
type projectRow struct {
	Project data.Project
	// Open counts the project's tasks on the plate: not done, not cancelled, not parked in the
	// backlog, and no subtasks (the desktop sidebar's count).
	Open int
	// Total counts all of its tasks, subtasks and finished ones too: what Delete would remove.
	Total int
	// Blocked says one of the open tasks is blocked: the sidebar's red dot instead of the count.
	Blocked bool
	// Depth is 0 for a project and 1 for its sub-project (and so on).
	Depth int
}

// ProjectList returns the Projects tab as JSON: the projects that are not archived in the order the
// desktop's sidebar lists them (top-level projects by name, each followed by its sub-projects),
// with the numbers the row shows, in one call:
//
//	[{"Project":{...},"Open":3,"Total":8,"Blocked":false,"Depth":0}, …]
//
// Project is data.Project (see Projects). A sub-project whose parent is not listed reads as a
// top-level project, as on the desktop.
func (c *Core) ProjectList() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	known := make(map[int]bool, len(snap.Projects))
	for _, p := range snap.Projects {
		known[p.ID] = true
	}
	children := make(map[int][]data.Project, len(snap.Projects))
	var top []data.Project
	for _, p := range snap.Projects {
		if p.ParentID != nil && known[*p.ParentID] {
			children[*p.ParentID] = append(children[*p.ParentID], p)
			continue
		}
		top = append(top, p)
	}
	counts := map[int]*projectRow{}
	for _, t := range snap.Tasks {
		if t.ProjectID == nil {
			continue
		}
		row := counts[*t.ProjectID]
		if row == nil {
			row = &projectRow{}
			counts[*t.ProjectID] = row
		}
		row.Total++
		if !t.IsSubtask() && !t.Status.Terminal() && t.Status != data.Backlog {
			row.Open++
			row.Blocked = row.Blocked || t.Status == data.Blocked
		}
	}
	out := make([]projectRow, 0, len(snap.Projects))
	seen := map[int]bool{}
	var draw func(p data.Project, depth int)
	draw = func(p data.Project, depth int) {
		if seen[p.ID] { // a parent cycle is not expected; never draw forever
			return
		}
		seen[p.ID] = true
		row := projectRow{Project: p, Depth: depth}
		if n := counts[p.ID]; n != nil {
			row.Open, row.Total, row.Blocked = n.Open, n.Total, n.Blocked
		}
		out = append(out, row)
		for _, child := range children[p.ID] {
			draw(child, depth+1)
		}
	}
	for _, p := range top {
		draw(p, 0)
	}
	return marshal(out)
}

// ArchivedProjects returns the archived projects as JSON, a []data.Project (see Projects), the
// Projects tab's Archived fold. Unarchive one with UnarchiveProject.
func (c *Core) ArchivedProjects() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	return marshal(emptyIfNil(snap.Archived))
}

// CreateProject makes a project, a sub-project of parentID when that is not 0, and returns it as a
// data.Project JSON. An empty name fails with "name must not be empty".
func (c *Core) CreateProject(name string, parentID int64) (string, error) {
	project, err := c.real.CreateProject(data.NewProject{Name: name, ParentID: intPointer(&parentID)})
	if err != nil {
		return "", manageError(err)
	}
	return marshal(project)
}

// RenameProject gives a project a new name.
func (c *Core) RenameProject(id int64, name string) error {
	if _, err := c.real.RenameProject(int(id), name); err != nil {
		return manageError(err)
	}
	return nil
}

// ArchiveProject hides a project with its tasks and its sub-projects.
func (c *Core) ArchiveProject(id int64) error {
	if err := c.real.ArchiveProject(int(id)); err != nil {
		return manageError(err)
	}
	return nil
}

// UnarchiveProject brings back an archived project and the sub-projects archived with it.
func (c *Core) UnarchiveProject(id int64) error {
	if err := c.real.RestoreProject(int(id)); err != nil {
		return manageError(err)
	}
	return nil
}

// DeleteProject removes a project with its tasks, their reports and its milestones, or nothing,
// and returns what went as JSON: {"Tasks":8,"Reports":12,"Milestones":1}. It cannot be undone; the
// Projects tab asks first with the project's Total (ProjectList).
func (c *Core) DeleteProject(id int64) (string, error) {
	removed, err := c.real.DeleteProject(int(id))
	if err != nil {
		return "", manageError(err)
	}
	return marshal(removed)
}

// CreateMilestone adds an open milestone to a project and returns it as a data.Milestone JSON
// (ID, ProjectID, Title, DueAt, Open, CreatedAt). dueAt is ISO 8601, "" for none.
func (c *Core) CreateMilestone(projectID int64, title string, dueAt string) (string, error) {
	due, err := optionalTime(dueAt)
	if err != nil {
		return "", err
	}
	milestone, err := c.real.CreateMilestone(int(projectID), title, due)
	if err != nil {
		return "", manageError(err)
	}
	return marshal(milestone)
}

// updateMilestoneJSON is the patch of UpdateMilestone.
//
//	{"title":"v0.13","due_at":"2026-11-14","closed":false}
//
// An absent key is left alone; a null or empty due_at clears the date.
type updateMilestoneJSON struct {
	Title  *string          `json:"title"`
	DueAt  optional[string] `json:"due_at"`
	Closed *bool            `json:"closed"`
}

// UpdateMilestone renames a milestone, moves or clears its due date, or closes or reopens it (see
// updateMilestoneJSON for the keys) and returns it as a data.Milestone JSON.
func (c *Core) UpdateMilestone(id int64, patchJSON string) (string, error) {
	var in updateMilestoneJSON
	if err := json.Unmarshal([]byte(patchJSON), &in); err != nil {
		return "", errors.New("could not read the change")
	}
	patch := data.MilestonePatch{Title: in.Title}
	if in.DueAt.Set {
		due, err := clearedTime(in.DueAt.Value)
		if err != nil {
			return "", err
		}
		patch.DueAt = due
	}
	if in.Closed != nil {
		patch.Open = data.BoolPtr(!*in.Closed)
	}
	milestone, err := c.real.UpdateMilestone(int(id), patch)
	if err != nil {
		return "", manageError(err)
	}
	return marshal(milestone)
}
