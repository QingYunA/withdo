package data

import (
	"errors"
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

var _ Manager = (*Real)(nil)

func (r *Real) CreateProject(in NewProject) (Project, error) {
	p, err := r.core.CreateProject(r.ctx(), domain.CreateProjectInput{Name: in.Name, RepoSlug: in.RepoSlug})
	if err != nil {
		return Project{}, r.projectProblem(err, in)
	}
	return fromProject(*p), nil
}

// projectProblem words a domain failure of CreateProject as the Manager promises.
func (r *Real) projectProblem(err error, in NewProject) error {
	var fe *domain.FieldError
	if !errors.As(err, &fe) {
		return err
	}
	switch {
	case fe.Field == "name":
		return ErrNameEmpty
	case fe.Field == "repo_slug" && fe.Conflict:
		slug, _ := domain.NormalizeRepoSlug(in.RepoSlug)
		owner := ""
		if ps, perr := r.core.ListProjects(r.ctx()); perr == nil {
			for _, p := range ps {
				if p.RepoSlug != nil && *p.RepoSlug == slug {
					owner = p.Name
				}
			}
		}
		return &RepoTakenError{Project: owner}
	case fe.Field == "repo_slug":
		return ErrRepoInvalid
	}
	return err
}

func (r *Real) RenameProject(id int, name string) (Project, error) {
	p, err := r.core.RenameProject(r.ctx(), int64(id), name)
	if err != nil {
		var fe *domain.FieldError
		if errors.As(err, &fe) && fe.Field == "name" && !fe.NotFound {
			return Project{}, ErrNameEmpty
		}
		return Project{}, err
	}
	return fromProject(*p), nil
}

func (r *Real) ArchiveProject(id int) error {
	_, err := r.core.ArchiveProject(r.ctx(), int64(id))
	return err
}

func (r *Real) RestoreProject(id int) error {
	_, err := r.core.RestoreProject(r.ctx(), int64(id))
	return err
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

func (r *Real) DeleteProject(id int) (Removed, error) {
	d, err := r.core.DeleteProject(r.ctx(), int64(id))
	if err == nil {
		r.CollectSoon()
	}
	return Removed{Tasks: d.Tasks, Reports: d.Reports, Milestones: d.Milestones}, err
}

func (r *Real) DeleteTask(id int) error {
	err := r.core.DeleteTask(r.ctx(), int64(id))
	if err == nil {
		r.CollectSoon()
	}
	return err
}

func (r *Real) SnapshotTask(id int) (TaskSnapshot, error) {
	snap, err := r.core.SnapshotTask(r.ctx(), int64(id))
	if err != nil {
		return TaskSnapshot{}, err
	}
	out := TaskSnapshot{Task: fromTask(snap.Task), raw: *snap}
	for i := len(snap.Reports) - 1; i >= 0; i-- { // newest first, as GetTask has them
		out.Reports = append(out.Reports, fromReport(snap.Reports[i]))
	}
	return out, nil
}

func (r *Real) RestoreTask(s TaskSnapshot) (Task, error) {
	raw, ok := s.raw.(domain.TaskSnapshot)
	if !ok {
		return Task{}, errors.New("snapshot was not taken by this store")
	}
	t, err := r.core.RestoreTask(r.ctx(), raw)
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}

func (r *Real) CreateMilestone(projectID int, title string, dueAt *time.Time) (Milestone, error) {
	var due *string
	if dueAt != nil && !dueAt.IsZero() {
		s := domain.FormatTime(*dueAt)
		due = &s
	}
	m, err := r.core.CreateMilestone(r.ctx(), domain.CreateMilestoneInput{ProjectID: int64(projectID), Title: title, DueAt: due})
	if err != nil {
		return Milestone{}, err
	}
	return fromMilestone(*m), nil
}

func (r *Real) UpdateMilestone(id int, p MilestonePatch) (Milestone, error) {
	patch := domain.MilestonePatch{Title: p.Title, DueAt: nullableTime(p.DueAt)}
	if p.Open != nil {
		s := domain.MilestoneClosed
		if *p.Open {
			s = domain.MilestoneOpen
		}
		patch.Status = &s
	}
	m, err := r.core.UpdateMilestone(r.ctx(), int64(id), patch)
	if err != nil {
		return Milestone{}, err
	}
	return fromMilestone(*m), nil
}

func (r *Real) SearchTasks(query string, limit int) ([]Task, error) {
	ts, err := r.core.SearchTasks(r.ctx(), query, limit)
	return fromTasks(ts), err
}

func (r *Real) Logbook(limit int) ([]Task, error) {
	ts, err := r.core.ListLogbook(r.ctx(), limit)
	return fromTasks(ts), err
}

func fromTasks(ts []domain.Task) []Task {
	out := make([]Task, len(ts))
	for i, t := range ts {
		out[i] = fromTask(t)
	}
	return out
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

func sortByName(ps []Project) {
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
}
