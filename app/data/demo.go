package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// demoClock is the clock of a demo store: the injected one, held at a past moment while a seed
// writes history through the domain layer.
type demoClock struct {
	mu   sync.Mutex
	base func() time.Time
	held *time.Time
}

func (c *demoClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.held != nil {
		return *c.held
	}
	return c.base()
}

// hold pins the clock at t until the returned func is called; the zero time pins nothing.
func (c *demoClock) hold(t time.Time) func() {
	if t.IsZero() {
		return func() {}
	}
	c.mu.Lock()
	c.held = &t
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		c.held = nil
		c.mu.Unlock()
	}
}

// OpenDemo opens a Real on a fresh SQLite file in a temporary folder, which Close removes. With
// seeded it holds the seed workspace relative to now(); without, nothing but the built-in
// categories. It is what the tests, the design screens and the -fake and -empty flags run on.
func OpenDemo(now func() time.Time, seeded bool) (*Real, error) {
	dir, err := os.MkdirTemp("", "adhd-todo-demo-")
	if err != nil {
		return nil, err
	}
	clock := &demoClock{base: now}
	r, err := OpenReal(context.Background(), filepath.Join(dir, "demo.db"), RealOptions{Now: clock.now, DataFolder: dir})
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	r.demo, r.tempDir = clock, dir
	if seeded {
		if err := r.seed(now()); err != nil {
			_ = r.Close()
			return nil, fmt.Errorf("seed the demo: %w", err)
		}
	}
	return r, nil
}

// The Seed methods write fixtures through the domain layer, for the demo and for tests. They hold
// the clock at the moment a row should carry and panic on a bad fixture, or on a store that
// OpenDemo did not open: they never write to the user's data.

// mustBeDemo panics unless OpenDemo opened r.
func (r *Real) mustBeDemo() {
	if r.demo == nil {
		panic("data: Seed is for a store OpenDemo opened")
	}
}

// SeedProject adds a project with no tasks; slug may be empty for one without a repository.
func (r *Real) SeedProject(name, slug string) Project {
	r.mustBeDemo()
	p, err := r.core.CreateProject(r.ctx(), domain.CreateProjectInput{Name: name, RepoSlug: slug})
	if err != nil {
		panic(err)
	}
	return fromProject(*p)
}

// SeedTask adds a task as given, a shape CreateTask does not make (a pull request, an age). Zero
// CreatedAt and UpdatedAt mean now; a non-zero ID renames the new row, so give it one the
// database has not reached.
func (r *Real) SeedTask(t Task) Task {
	r.mustBeDemo()
	defer r.holdClock(t.CreatedAt)()
	in := domain.CreateTaskInput{
		Title: t.Title, Source: domain.Source(t.Source), ProjectID: int64Ptr(t.ProjectID), MilestoneID: int64Ptr(t.MilestoneID),
		Status: domain.Status(t.Status), Priority: domain.Priority(t.Priority), DueAt: formatTimePtr(t.DueAt),
		RemindAt: formatTimePtr(t.RemindAt), ParentID: int64Ptr(t.ParentID), CategoryID: nonZero(t.CategoryID),
	}
	if t.Notes != "" {
		in.Notes = &t.Notes
	}
	if t.Branch != "" {
		in.Branch = &t.Branch
	}
	if t.PRURL != "" {
		in.PRURL = &t.PRURL
	}
	made, err := r.core.CreateTask(r.ctx(), in)
	if err != nil {
		panic(err)
	}
	id := int(made.ID)
	if t.ID != 0 && t.ID != id {
		r.exec(`UPDATE tasks SET id = ? WHERE id = ?`, t.ID, id)
		id = t.ID
	}
	r.stampTask(id, t.UpdatedAt, t.CompletedAt)
	return r.mustTask(id)
}

// SeedReport files a report as given; a zero CreatedAt means now.
func (r *Real) SeedReport(rep Report) {
	r.mustBeDemo()
	in := domain.AddReportInput{TaskID: int64(rep.TaskID), Agent: rep.Agent, Summary: rep.Summary, Cwd: rep.Cwd}
	if rep.Status != "" {
		st := domain.Status(rep.Status)
		in.Status = &st
	}
	if rep.NextStep != "" {
		in.NextStep = &rep.NextStep
	}
	if !rep.CreatedAt.IsZero() {
		in.CreatedAt = domain.FormatTime(rep.CreatedAt)
	}
	for _, l := range rep.Links {
		in.Links = append(in.Links, domain.ReportLink{URL: l.URL, Title: l.Title})
	}
	if _, err := r.core.AddReport(r.ctx(), in); err != nil {
		panic(err)
	}
}

// SeedReportAs files a report by an agent against a task, for tests of Settings > Agents.
func (r *Real) SeedReportAs(taskID int, agent, summary string, at time.Time) {
	r.SeedReport(Report{TaskID: taskID, Agent: agent, Summary: summary, CreatedAt: at})
}

// SeedPullRequest sets what the poller would have found about a task's pull request. The poller
// never writes it for a finished task or one without the link, so this goes to the row directly.
func (r *Real) SeedPullRequest(id int, state PRState, checks PRChecks, review PRReview) {
	r.mustBeDemo()
	r.exec(`UPDATE tasks SET pr_state = NULLIF(?, ''), pr_checks = NULLIF(?, ''), pr_review = NULLIF(?, ''),
		pr_checked_at = ?, pr_closed_seen_at = NULL WHERE id = ?`,
		string(state), string(checks), string(review), domain.FormatTime(r.opts.Now()), id)
}

// SeedCompleteAt marks a task done at a moment, for tests that want a day with things finished at
// different hours.
func (r *Real) SeedCompleteAt(id int, at time.Time) {
	r.mustBeDemo()
	done := domain.StatusDone
	release := r.holdClock(at)
	_, err := r.core.UpdateTask(r.ctx(), int64(id), domain.TaskPatch{Status: &done})
	release()
	if err != nil {
		panic(err)
	}
	r.stampTask(id, at, &at)
}

func (r *Real) holdClock(t time.Time) func() { return r.demo.hold(t) }

// stampTask sets updated_at and completed_at, which the domain layer stamps with the clock; a zero
// updated, or a nil completed on a task that is open, leaves them.
func (r *Real) stampTask(id int, updated time.Time, completed *time.Time) {
	if !updated.IsZero() {
		r.exec(`UPDATE tasks SET updated_at = ? WHERE id = ?`, domain.FormatTime(updated), id)
	}
	if completed != nil {
		r.exec(`UPDATE tasks SET completed_at = ? WHERE id = ? AND completed_at IS NOT NULL`, domain.FormatTime(*completed), id)
	}
}

func (r *Real) exec(query string, args ...any) {
	if _, err := r.db.ExecContext(r.ctx(), query, args...); err != nil {
		panic(err)
	}
}

func (r *Real) mustTask(id int) Task {
	d, err := r.GetTask(id)
	if err != nil {
		panic(err)
	}
	return d.Task
}
