package data

import (
	"math"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// Snapshot implements Store: eight queries, whatever the number of tasks (five thousand more tasks
// add one for their reports).
func (r *Real) Snapshot() (Snapshot, error) {
	ctx := r.ctx()
	projects, err := r.core.ListProjects(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	cats, err := r.core.ListCategories(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	tasks, err := r.core.ListTasks(ctx, domain.TaskFilter{Limit: math.MaxInt32})
	if err != nil {
		return Snapshot{}, err
	}
	milestones, err := r.core.ListAllMilestones(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	overview, err := r.core.GetOverview(ctx, domain.OverviewOptions{})
	if err != nil {
		return Snapshot{}, err
	}
	s := Snapshot{Milestones: map[int][]Milestone{}, Reports: map[int][]Report{}, Overview: fromOverview(overview)}
	hidden := map[int64]bool{}
	for _, p := range projects {
		if p.ArchivedAt != nil {
			hidden[p.ID] = true
			s.Archived = append(s.Archived, fromProject(p))
			continue
		}
		s.Projects = append(s.Projects, fromProject(p))
		for _, m := range milestones[p.ID] {
			s.Milestones[int(p.ID)] = append(s.Milestones[int(p.ID)], fromMilestone(m))
		}
	}
	sortByName(s.Projects)
	sortByName(s.Archived)
	for _, c := range cats {
		s.Categories = append(s.Categories, fromCategory(c))
	}
	var ids []int64
	for _, t := range tasks {
		if t.ProjectID == nil || !hidden[*t.ProjectID] {
			s.Tasks = append(s.Tasks, fromTask(t))
			ids = append(ids, t.ID)
		}
	}
	reports, err := r.core.ListReportsOf(ctx, ids)
	if err != nil {
		return Snapshot{}, err
	}
	for id, rs := range reports {
		out := make([]Report, len(rs))
		for i, rep := range rs { // oldest first in the domain, newest first here
			out[len(rs)-1-i] = fromReport(rep)
		}
		s.Reports[int(id)] = out
	}
	return s, nil
}
