package data

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

var _ Manager = (*Fake)(nil)

func (f *Fake) project(id int) (*Project, bool) {
	for i := range f.projects {
		if f.projects[i].ID == id {
			return &f.projects[i], true
		}
	}
	return nil, false
}

// inArchived reports whether a task belongs to an archived project.
func (f *Fake) inArchived(t Task) bool {
	if t.ProjectID == nil {
		return false
	}
	p, ok := f.project(*t.ProjectID)
	return ok && p.ArchivedAt != nil
}

func (f *Fake) CreateProject(in NewProject) (Project, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Project{}, ErrNameEmpty
	}
	slug := ""
	if strings.TrimSpace(in.RepoSlug) != "" {
		s, ok := domain.NormalizeRepoSlug(in.RepoSlug)
		if !ok {
			return Project{}, ErrRepoInvalid
		}
		slug = s
	}
	f.mu.Lock()
	if slug != "" {
		for _, p := range f.projects {
			if p.RepoSlug == slug {
				f.mu.Unlock()
				return Project{}, &RepoTakenError{Project: p.Name}
			}
		}
	}
	now := f.Now()
	p := Project{ID: f.newID(), Name: name, RepoSlug: slug, CreatedAt: now, UpdatedAt: now}
	f.projects = append(f.projects, p)
	f.mu.Unlock()
	f.notify()
	return p, nil
}

func (f *Fake) RenameProject(id int, name string) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, ErrNameEmpty
	}
	return f.editProject(id, func(p *Project) { p.Name = name })
}

func (f *Fake) ArchiveProject(id int) error {
	now := f.Now()
	_, err := f.editProject(id, func(p *Project) {
		if p.ArchivedAt == nil {
			p.ArchivedAt = &now
		}
	})
	return err
}

func (f *Fake) RestoreProject(id int) error {
	_, err := f.editProject(id, func(p *Project) { p.ArchivedAt = nil })
	return err
}

func (f *Fake) editProject(id int, change func(*Project)) (Project, error) {
	f.mu.Lock()
	p, ok := f.project(id)
	if !ok {
		f.mu.Unlock()
		return Project{}, fmt.Errorf("project %d not found", id)
	}
	change(p)
	p.UpdatedAt = f.Now()
	out := *p
	f.mu.Unlock()
	f.notify()
	return out, nil
}

func (f *Fake) ListArchivedProjects() ([]Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Project
	for _, p := range f.projects {
		if p.ArchivedAt != nil {
			out = append(out, p)
		}
	}
	sortByName(out)
	return out, nil
}

func (f *Fake) DeleteProject(id int) (Removed, error) {
	f.mu.Lock()
	if _, ok := f.project(id); !ok {
		f.mu.Unlock()
		return Removed{}, fmt.Errorf("project %d not found", id)
	}
	var out Removed
	gone := map[int]bool{}
	var tasks []Task
	for _, t := range f.tasks {
		if t.ProjectID != nil && *t.ProjectID == id {
			gone[t.ID] = true
			out.Tasks++
			continue
		}
		tasks = append(tasks, t)
	}
	f.tasks = tasks
	for id := range gone {
		f.pics.dropTask(id)
	}
	f.reports, out.Reports = dropReports(f.reports, gone)
	var ms []Milestone
	for _, m := range f.milestones {
		if m.ProjectID == id {
			out.Milestones++
			continue
		}
		ms = append(ms, m)
	}
	f.milestones = ms
	var ps []Project
	for _, p := range f.projects {
		if p.ID != id {
			ps = append(ps, p)
		}
	}
	f.projects = ps
	f.mu.Unlock()
	f.notify()
	return out, nil
}

func dropReports(rs []Report, tasks map[int]bool) (kept []Report, dropped int) {
	for _, r := range rs {
		if tasks[r.TaskID] {
			dropped++
			continue
		}
		kept = append(kept, r)
	}
	return kept, dropped
}

func (f *Fake) DeleteTask(id int) error {
	f.mu.Lock()
	if _, ok := f.find(id); !ok {
		f.mu.Unlock()
		return fmt.Errorf("task %d not found", id)
	}
	var tasks []Task
	for _, t := range f.tasks {
		if t.ID != id {
			tasks = append(tasks, t)
		}
	}
	f.tasks = tasks
	f.pics.dropTask(id)
	f.reports, _ = dropReports(f.reports, map[int]bool{id: true})
	f.mu.Unlock()
	f.notify()
	return nil
}

func (f *Fake) SnapshotTask(id int) (TaskSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.find(id)
	if !ok {
		return TaskSnapshot{}, fmt.Errorf("task %d not found", id)
	}
	return TaskSnapshot{Task: *t, Reports: f.reportsOf(id), raw: slices.Clone(f.pics.links[id])}, nil
}

func (f *Fake) RestoreTask(s TaskSnapshot) (Task, error) {
	f.mu.Lock()
	t := s.Task
	if _, ok := f.find(t.ID); ok {
		f.mu.Unlock()
		return Task{}, fmt.Errorf("task %d already exists", t.ID)
	}
	if t.ProjectID != nil {
		if _, ok := f.project(*t.ProjectID); !ok {
			t.ProjectID, t.MilestoneID = nil, nil
		}
	}
	f.tasks = append(f.tasks, t)
	f.reports = append(f.reports, s.Reports...)
	if links, ok := s.raw.([]TaskImage); ok && len(links) > 0 {
		for _, l := range links {
			f.pics.add(t.ID, l)
		}
	}
	f.mu.Unlock()
	f.notify()
	return t, nil
}

func (f *Fake) CreateMilestone(projectID int, title string, dueAt *time.Time) (Milestone, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Milestone{}, fmt.Errorf("title must not be empty")
	}
	f.mu.Lock()
	if _, ok := f.project(projectID); !ok {
		f.mu.Unlock()
		return Milestone{}, fmt.Errorf("project %d not found", projectID)
	}
	m := Milestone{ID: f.newID(), ProjectID: projectID, Title: title, Open: true, CreatedAt: f.Now()}
	if dueAt != nil && !dueAt.IsZero() {
		m.DueAt = dueAt
	}
	f.milestones = append(f.milestones, m)
	f.mu.Unlock()
	f.notify()
	return m, nil
}

func (f *Fake) UpdateMilestone(id int, p MilestonePatch) (Milestone, error) {
	f.mu.Lock()
	var m *Milestone
	for i := range f.milestones {
		if f.milestones[i].ID == id {
			m = &f.milestones[i]
		}
	}
	if m == nil {
		f.mu.Unlock()
		return Milestone{}, fmt.Errorf("milestone %d not found", id)
	}
	if p.Title != nil {
		title := strings.TrimSpace(*p.Title)
		if title == "" {
			f.mu.Unlock()
			return Milestone{}, fmt.Errorf("title must not be empty")
		}
		m.Title = title
	}
	if p.DueAt != nil {
		m.DueAt = nilIfZeroTime(*p.DueAt)
	}
	if p.Open != nil {
		m.Open = *p.Open
	}
	out := *m
	f.mu.Unlock()
	f.notify()
	return out, nil
}

// DefaultSearchLimit is how many results SearchTasks returns when no limit is given.
const DefaultSearchLimit = 50

func (f *Fake) SearchTasks(query string, limit int) ([]Task, error) {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Task
	for _, t := range f.tasks {
		if !f.inArchived(t) && matchesAll(t, words) {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func matchesAll(t Task, words []string) bool {
	hay := strings.ToLower(t.Title + "\n" + t.Notes + "\n" + t.Branch)
	for _, w := range words {
		n, err := strconv.Atoi(strings.TrimPrefix(w, "#"))
		if !strings.Contains(hay, w) && !(err == nil && t.PRNumber == n) {
			return false
		}
	}
	return true
}

func (f *Fake) Logbook(limit int) ([]Task, error) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Task
	for _, t := range f.tasks {
		if t.Status.Terminal() && !f.inArchived(t) {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := finishedAt(out[i]), finishedAt(out[j])
		if !a.Equal(b) {
			return a.After(b)
		}
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func finishedAt(t Task) time.Time {
	if t.CompletedAt != nil {
		return *t.CompletedAt
	}
	return t.UpdatedAt
}
