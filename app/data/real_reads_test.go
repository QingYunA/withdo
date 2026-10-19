package data

import (
	"fmt"
	"math"
	"slices"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The single reads below are what the windows used before Store.Snapshot. The tests keep them, to
// look at one thing at a time and as the reference the snapshot is checked against.

func (r *Real) ListProjects() ([]Project, error) {
	ps, err := r.core.ListProjects(r.ctx())
	if err != nil {
		return nil, err
	}
	var out []Project
	for _, p := range ps {
		if p.ArchivedAt == nil {
			out = append(out, fromProject(p))
		}
	}
	slices.SortStableFunc(out, func(a, b Project) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		}
		return 0
	})
	return out, nil
}

func (r *Real) ListCategories() ([]Category, error) {
	cs, err := r.core.ListCategories(r.ctx())
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(cs))
	for i, c := range cs {
		out[i] = fromCategory(c)
	}
	return out, nil
}

func (r *Real) ListMilestones(projectID int) ([]Milestone, error) {
	ms, err := r.core.ListMilestones(r.ctx(), int64(projectID))
	if err != nil {
		return nil, err
	}
	out := make([]Milestone, len(ms))
	for i, m := range ms {
		out[i] = fromMilestone(m)
	}
	return out, nil
}

func (r *Real) ListTasks(f Filter) ([]Task, error) {
	ts, err := r.core.ListTasks(r.ctx(), domain.TaskFilter{
		ProjectID: int64Ptr(f.ProjectID), Inbox: f.Inbox, Statuses: toStatuses(f.Status),
		MilestoneID: int64Ptr(f.MilestoneID), Limit: math.MaxInt32,
	})
	if err != nil {
		return nil, err
	}
	hidden, err := r.archivedIDs()
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, t := range ts {
		inArchived := t.ProjectID != nil && hidden[int(*t.ProjectID)]
		if !inArchived && (f.Source == "" || Source(t.Source) == f.Source) {
			out = append(out, fromTask(t))
		}
	}
	return out, nil
}

func (r *Real) GetTask(id int) (TaskDetail, error) {
	t, err := r.core.GetTask(r.ctx(), int64(id))
	if err != nil {
		return TaskDetail{}, err
	}
	if t == nil {
		return TaskDetail{}, fmt.Errorf("task %d not found", id)
	}
	rs, err := r.core.ListReports(r.ctx(), int64(id))
	if err != nil {
		return TaskDetail{}, err
	}
	d := TaskDetail{Task: fromTask(*t)}
	for i := len(rs) - 1; i >= 0; i-- { // newest first
		d.Reports = append(d.Reports, fromReport(rs[i]))
	}
	return d, nil
}

func (r *Real) Overview() (Overview, error) {
	o, err := r.core.GetOverview(r.ctx(), domain.OverviewOptions{})
	if err != nil {
		return Overview{}, err
	}
	return fromOverview(o), nil
}

func (r *Real) ListArchivedProjects() ([]Project, error) {
	ps, err := r.core.ListProjects(r.ctx())
	if err != nil {
		return nil, err
	}
	var out []Project
	for _, p := range ps {
		if p.ArchivedAt != nil {
			out = append(out, fromProject(p))
		}
	}
	sortByName(out)
	return out, nil
}

// archivedIDs are the projects hidden from the lists.
func (r *Real) archivedIDs() (map[int]bool, error) {
	ps, err := r.core.ListProjects(r.ctx())
	if err != nil {
		return nil, err
	}
	out := map[int]bool{}
	for _, p := range ps {
		if p.ArchivedAt != nil {
			out[int(p.ID)] = true
		}
	}
	return out, nil
}

// Filter selects tasks. A zero Filter selects everything.
type Filter struct {
	ProjectID   *int
	Inbox       bool // only tasks without a project
	Status      []Status
	MilestoneID *int
	Source      Source
}

func toStatuses(in []Status) []domain.Status {
	out := make([]domain.Status, len(in))
	for i, s := range in {
		out[i] = domain.Status(s)
	}
	return out
}
