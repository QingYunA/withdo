package data

import (
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"
)

// Fake is an in-memory Store with realistic seed data, for development and for the design
// review. It is safe for concurrent use.
type Fake struct {
	// Now is the clock; tests and screenshots pin it.
	Now func() time.Time

	mu         sync.Mutex
	projects   []Project
	milestones []Milestone
	tasks      []Task
	reports    []Report
	nextID     int
	subs       map[int]func()
	remSubs    map[int]func(Reminder)
	nextSub    int
}

// NewFake returns a Fake filled with the seed data, relative to now().
func NewFake(now func() time.Time) *Fake {
	f := &Fake{Now: now, subs: map[int]func(){}, remSubs: map[int]func(Reminder){}, nextID: 1000}
	f.seed()
	return f
}

// NewEmptyFake returns a Fake with no data at all, for the empty states.
func NewEmptyFake(now func() time.Time) *Fake {
	return &Fake{Now: now, subs: map[int]func(){}, remSubs: map[int]func(Reminder){}, nextID: 1000}
}

// AddProject adds a project with no tasks, for demos and tests of the empty states.
func (f *Fake) AddProject(name, slug string) Project {
	f.mu.Lock()
	now := f.Now()
	p := Project{ID: f.newID(), Name: name, RepoSlug: slug, CreatedAt: now, UpdatedAt: now}
	f.projects = append(f.projects, p)
	f.mu.Unlock()
	f.notify()
	return p
}

func (f *Fake) ListProjects() ([]Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Project
	for _, p := range f.projects {
		if p.ArchivedAt == nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (f *Fake) ListMilestones(projectID int) ([]Milestone, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Milestone
	for _, m := range f.milestones {
		if m.ProjectID == projectID {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *Fake) ListTasks(flt Filter) ([]Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Task
	for _, t := range f.tasks {
		if matches(t, flt) && !f.inArchived(t) {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func matches(t Task, f Filter) bool {
	switch {
	case f.Inbox && t.ProjectID != nil:
		return false
	case f.ProjectID != nil && (t.ProjectID == nil || *t.ProjectID != *f.ProjectID):
		return false
	case f.MilestoneID != nil && (t.MilestoneID == nil || *t.MilestoneID != *f.MilestoneID):
		return false
	case f.Source != "" && t.Source != f.Source:
		return false
	case len(f.Status) > 0 && !slices.Contains(f.Status, t.Status):
		return false
	}
	return true
}

func (f *Fake) GetTask(id int) (TaskDetail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.find(id)
	if !ok {
		return TaskDetail{}, fmt.Errorf("task %d not found", id)
	}
	return TaskDetail{Task: *t, Reports: f.reportsOf(id)}, nil
}

func (f *Fake) find(id int) (*Task, bool) {
	for i := range f.tasks {
		if f.tasks[i].ID == id {
			return &f.tasks[i], true
		}
	}
	return nil, false
}

// reportsOf returns the reports of a task, newest first.
func (f *Fake) reportsOf(id int) []Report {
	var out []Report
	for _, r := range f.reports {
		if r.TaskID == id {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (f *Fake) CreateTask(in CreateTaskInput) (Task, error) {
	if in.Title == "" {
		return Task{}, fmt.Errorf("title must not be empty")
	}
	f.mu.Lock()
	now := f.Now()
	t := Task{
		ID: f.newID(), ParentID: in.ParentID, ProjectID: in.ProjectID, Title: in.Title, Status: orStatus(in.Status, Todo),
		Priority: orPriority(in.Priority, P2), Source: orSource(in.Source, SourceUser),
		CreatedAt: now, UpdatedAt: now, DueAt: in.DueAt,
	}
	if t.Status.Terminal() {
		t.CompletedAt = &now
	}
	f.tasks = append(f.tasks, t)
	f.mu.Unlock()
	f.notify()
	return t, nil
}

func (f *Fake) UpdateTask(id int, p Patch) (Task, error) {
	f.mu.Lock()
	t, ok := f.find(id)
	if !ok {
		f.mu.Unlock()
		return Task{}, fmt.Errorf("task %d not found", id)
	}
	if err := f.apply(t, p); err != nil {
		f.mu.Unlock()
		return Task{}, err
	}
	out := *t
	f.mu.Unlock()
	f.notify()
	return out, nil
}

func (f *Fake) apply(t *Task, p Patch) error {
	if p.Title != nil {
		if *p.Title == "" {
			return fmt.Errorf("title must not be empty")
		}
		t.Title = *p.Title
	}
	if p.Notes != nil {
		t.Notes = *p.Notes
	}
	if p.Priority != nil {
		t.Priority = *p.Priority
	}
	if p.Branch != nil {
		t.Branch = *p.Branch
	}
	if p.ProjectID != nil {
		if _, ok := f.project(*p.ProjectID); *p.ProjectID != 0 && !ok {
			return fmt.Errorf("project %d not found", *p.ProjectID)
		}
		t.ProjectID = nilIfZero(*p.ProjectID)
		t.MilestoneID = nil // a milestone belongs to one project, so a move leaves it behind
	}
	if p.MilestoneID != nil {
		t.MilestoneID = nilIfZero(*p.MilestoneID)
	}
	if p.DueAt != nil {
		t.DueAt = nilIfZeroTime(*p.DueAt)
	}
	if p.RemindAt != nil {
		t.RemindAt = nilIfZeroTime(*p.RemindAt)
		t.RemindedAt = nil // any write to remind_at re-arms the reminder
	}
	now := f.Now()
	if p.Status != nil {
		wasTerminal := t.Status.Terminal()
		t.Status = *p.Status
		switch {
		case !t.Status.Terminal():
			t.CompletedAt = nil
		case !wasTerminal:
			t.CompletedAt = &now
			f.closeSubtasks(t.ID, now)
		}
	}
	t.UpdatedAt = now
	return nil
}

// closeSubtasks mirrors the core: when a task is done or cancelled, its open subtasks (and theirs)
// become done, each with a report from adhd-todo. The lock is held.
func (f *Fake) closeSubtasks(parent int, now time.Time) {
	for i := range f.tasks {
		k := &f.tasks[i]
		if k.ParentID == nil || *k.ParentID != parent || k.Status.Terminal() {
			continue
		}
		k.Status, k.CompletedAt, k.UpdatedAt = Done, &now, now
		f.reports = append(f.reports, Report{ID: f.newID(), TaskID: k.ID, Agent: "adhd-todo", Status: Done,
			Summary: "Closed with its parent task", CreatedAt: now})
		f.closeSubtasks(k.ID, now)
	}
}

func (f *Fake) Overview() (Overview, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.Now()
	byProject := map[int][]OverviewTask{}
	var inbox []OverviewTask
	for _, t := range f.tasks {
		if !t.Status.Active() {
			continue
		}
		ot := f.describe(t, now)
		if t.ProjectID == nil {
			inbox = append(inbox, ot)
		} else {
			byProject[*t.ProjectID] = append(byProject[*t.ProjectID], ot)
		}
	}
	groups := f.groups(byProject)
	if len(inbox) > 0 {
		sortByAttention(inbox)
		groups = append(groups, OverviewGroup{Tasks: inbox})
	}
	return Overview{GeneratedAt: now, StaleHours: DefaultStaleHours, Groups: groups}, nil
}

func (f *Fake) groups(byProject map[int][]OverviewTask) []OverviewGroup {
	var out []OverviewGroup
	for i := range f.projects {
		p := f.projects[i]
		if p.ArchivedAt != nil || len(byProject[p.ID]) == 0 {
			continue
		}
		tasks := byProject[p.ID]
		sortByAttention(tasks)
		out = append(out, OverviewGroup{Project: &p, Tasks: tasks})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Project.Name < out[j].Project.Name })
	return out
}

func (f *Fake) describe(t Task, now time.Time) OverviewTask {
	ot := OverviewTask{Task: t}
	since := t.UpdatedAt
	if rs := f.reportsOf(t.ID); len(rs) > 0 {
		r := rs[0]
		ot.LatestReport = &r
		since = r.CreatedAt
	}
	ot.HoursSinceActivity = int(now.Sub(since) / time.Hour)
	ot.Stale = ot.HoursSinceActivity >= DefaultStaleHours
	return ot
}

var attentionOrder = map[Status]int{Blocked: 0, Review: 1, Doing: 2}

func sortByAttention(ts []OverviewTask) {
	sort.SliceStable(ts, func(i, j int) bool {
		a, b := attentionOrder[ts[i].Status], attentionOrder[ts[j].Status]
		if a != b {
			return a < b
		}
		return ts[i].ID > ts[j].ID
	})
}

// Subscribe implements Store.
func (f *Fake) Subscribe(fn func()) func() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextSub++
	id := f.nextSub
	f.subs[id] = fn
	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		delete(f.subs, id)
	}
}

// SubscribeReminders implements Store.
func (f *Fake) SubscribeReminders(fn func(Reminder)) func() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextSub++
	id := f.nextSub
	f.remSubs[id] = fn
	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		delete(f.remSubs, id)
	}
}

// CheckReminders sends every reminder that is due and not yet sent, once. The app calls it on a
// timer; the real store does the same inside a transaction.
func (f *Fake) CheckReminders() {
	f.mu.Lock()
	now := f.Now()
	var due []Reminder
	for i := range f.tasks {
		t := &f.tasks[i]
		if t.RemindAt == nil || t.RemindedAt != nil || t.RemindAt.After(now) || t.Status.Terminal() {
			continue
		}
		t.RemindedAt = &now
		due = append(due, Reminder{TaskID: t.ID, Title: t.Title, ProjectName: f.projectName(t.ProjectID)})
	}
	subs := f.reminderSubs()
	f.mu.Unlock()
	for _, r := range due {
		for _, fn := range subs {
			fn(r)
		}
	}
}

func (f *Fake) reminderSubs() []func(Reminder) {
	out := make([]func(Reminder), 0, len(f.remSubs))
	for _, fn := range f.remSubs {
		out = append(out, fn)
	}
	return out
}

func (f *Fake) projectName(id *int) string {
	if id == nil {
		return ""
	}
	for _, p := range f.projects {
		if p.ID == *id {
			return p.Name
		}
	}
	return ""
}

func (f *Fake) notify() {
	f.mu.Lock()
	fns := make([]func(), 0, len(f.subs))
	for _, fn := range f.subs {
		fns = append(fns, fn)
	}
	f.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (f *Fake) newID() int {
	f.nextID++
	return f.nextID
}

func orStatus(s, def Status) Status {
	if s == "" {
		return def
	}
	return s
}

func orPriority(p, def Priority) Priority {
	if p == "" {
		return def
	}
	return p
}

func orSource(s, def Source) Source {
	if s == "" {
		return def
	}
	return s
}

func nilIfZero(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func nilIfZeroTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// SetPullRequest sets what the poller would have found about a task's pull request, for demos and
// screens.
func (f *Fake) SetPullRequest(id int, state PRState, checks PRChecks, review PRReview) error {
	f.mu.Lock()
	t, ok := f.find(id)
	if !ok {
		f.mu.Unlock()
		return fmt.Errorf("task %d not found", id)
	}
	now := f.Now()
	t.PRState, t.PRChecks, t.PRReview, t.PRCheckedAt, t.PRClosedSeenAt = state, checks, review, &now, nil
	f.mu.Unlock()
	f.notify()
	return nil
}

// AddReport files a report against a task, for demos and screens.
func (f *Fake) AddReport(r Report) {
	f.mu.Lock()
	r.ID = f.newID()
	f.reports = append(f.reports, r)
	f.mu.Unlock()
	f.notify()
}

// KeepClosedPR implements Store.
func (f *Fake) KeepClosedPR(id int) error {
	f.mu.Lock()
	t, ok := f.find(id)
	if !ok {
		f.mu.Unlock()
		return fmt.Errorf("task %d not found", id)
	}
	if t.PRState == PRClosed {
		now := f.Now()
		t.PRClosedSeenAt = &now
	}
	f.mu.Unlock()
	f.notify()
	return nil
}

// AddTask adds a task as given, with a new id when it has none, for demos and screens that need a
// shape CreateTask does not make (a pull request, an age).
func (f *Fake) AddTask(t Task) Task {
	f.mu.Lock()
	if t.ID == 0 {
		t.ID = f.newID()
	}
	now := f.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}
	f.tasks = append(f.tasks, orDefaults(t))
	f.mu.Unlock()
	f.notify()
	return t
}

func orDefaults(t Task) Task {
	t.Status, t.Priority, t.Source = orStatus(t.Status, Todo), orPriority(t.Priority, P2), orSource(t.Source, SourceUser)
	return t
}
