package data

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The same contract for both stores: the fake the views are tested against must behave as the real
// one does.
func eachStore(t *testing.T, run func(t *testing.T, s Store, clock *time.Time)) {
	t.Helper()
	t.Run("fake", func(t *testing.T) {
		now := fixedNow()
		run(t, NewEmptyFake(func() time.Time { return now }), &now)
	})
	t.Run("real", func(t *testing.T) {
		now := fixedNow()
		r, err := OpenReal(bg, filepath.Join(t.TempDir(), "t.db"), RealOptions{Now: func() time.Time { return now }})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = r.Close() })
		run(t, r, &now)
	})
}

func must[T any](t *testing.T) func(T, error) T {
	t.Helper()
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func titlesOf(ts []Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Title
	}
	return out
}

func wantTitles(t *testing.T, what string, got []Task, want ...string) {
	t.Helper()
	g := titlesOf(got)
	if len(g) != len(want) {
		t.Errorf("%s = %q, want %q", what, g, want)
		return
	}
	for i := range g {
		if g[i] != want[i] {
			t.Errorf("%s = %q, want %q", what, g, want)
			return
		}
	}
}

func TestCreateProject(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, _ *time.Time) {
		plain := must[Project](t)(s.CreateProject(NewProject{Name: "  Garden "}))
		if plain.Name != "Garden" || plain.RepoSlug != "" {
			t.Errorf("plain = %+v", plain)
		}
		linked := must[Project](t)(s.CreateProject(NewProject{Name: "Rocket", RepoSlug: "https://github.com/acme/rocket"}))
		if linked.RepoSlug != "acme/rocket" {
			t.Errorf("a pasted address is read as owner/repo: %+v", linked)
		}
		ps := must[[]Project](t)(s.ListProjects())
		if len(ps) != 2 || ps[0].Name != "Garden" || ps[1].Name != "Rocket" {
			t.Errorf("projects = %+v", ps)
		}

		if _, err := s.CreateProject(NewProject{Name: "  "}); !errors.Is(err, ErrNameEmpty) {
			t.Errorf("blank name: %v", err)
		}
		if _, err := s.CreateProject(NewProject{Name: "x", RepoSlug: "rocket"}); !errors.Is(err, ErrRepoInvalid) {
			t.Errorf("bad repository: %v", err)
		}
		_, err := s.CreateProject(NewProject{Name: "x", RepoSlug: "acme/rocket"})
		var taken *RepoTakenError
		if !errors.As(err, &taken) || taken.Project != "Rocket" {
			t.Errorf("taken repository: %v", err)
		}
		if n := len(must[[]Project](t)(s.ListProjects())); n != 2 {
			t.Errorf("%d projects after three failures", n)
		}
	})
}

func TestRenameArchiveAndRestoreProject(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, _ *time.Time) {
		p := must[Project](t)(s.CreateProject(NewProject{Name: "Old", RepoSlug: "acme/old"}))
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "in it", ProjectID: IntPtr(p.ID), Status: Doing, Source: SourceAgent}))

		if got := must[Project](t)(s.RenameProject(p.ID, " New ")); got.Name != "New" {
			t.Errorf("rename = %+v", got)
		}
		if _, err := s.RenameProject(p.ID, ""); !errors.Is(err, ErrNameEmpty) {
			t.Errorf("blank rename: %v", err)
		}
		if _, err := s.RenameProject(999, "x"); err == nil {
			t.Error("a missing project must be an error")
		}

		if err := s.ArchiveProject(p.ID); err != nil {
			t.Fatal(err)
		}
		if ps := must[[]Project](t)(s.ListProjects()); len(ps) != 0 {
			t.Errorf("archived project still listed: %+v", ps)
		}
		arch := must[[]Project](t)(s.ListArchivedProjects())
		if len(arch) != 1 || arch[0].ID != p.ID || arch[0].ArchivedAt == nil {
			t.Errorf("archived = %+v", arch)
		}
		if ts := must[[]Task](t)(s.ListTasks(Filter{})); len(ts) != 0 {
			t.Errorf("tasks of an archived project are listed: %+v", ts)
		}
		if ov := must[Overview](t)(s.Overview()); len(ov.Groups) != 0 {
			t.Errorf("overview shows an archived project: %+v", ov.Groups)
		}
		if _, err := s.GetTask(task.ID); err != nil {
			t.Errorf("the task is still there to open: %v", err)
		}

		if err := s.RestoreProject(p.ID); err != nil {
			t.Fatal(err)
		}
		if ps := must[[]Project](t)(s.ListProjects()); len(ps) != 1 || ps[0].Name != "New" {
			t.Errorf("restored = %+v", ps)
		}
		if ts := must[[]Task](t)(s.ListTasks(Filter{})); len(ts) != 1 {
			t.Errorf("tasks after restore: %+v", ts)
		}
		if len(must[[]Project](t)(s.ListArchivedProjects())) != 0 {
			t.Error("restored project still archived")
		}
	})
}

func TestDeleteProjectAndTask(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, _ *time.Time) {
		p := must[Project](t)(s.CreateProject(NewProject{Name: "Doomed"}))
		keep := must[Project](t)(s.CreateProject(NewProject{Name: "Keep"}))
		must[Milestone](t)(s.CreateMilestone(p.ID, "v1", nil))
		must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a", ProjectID: IntPtr(p.ID)}))
		must[Task](t)(s.CreateTask(CreateTaskInput{Title: "b", ProjectID: IntPtr(p.ID), Status: Done}))
		kept := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "kept", ProjectID: IntPtr(keep.ID)}))
		inbox := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "inbox"}))

		got, err := s.DeleteProject(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Tasks != 2 || got.Milestones != 1 {
			t.Errorf("removed %+v", got)
		}
		if ps := must[[]Project](t)(s.ListProjects()); len(ps) != 1 || ps[0].ID != keep.ID {
			t.Errorf("projects = %+v", ps)
		}
		wantTitles(t, "tasks left", must[[]Task](t)(s.ListTasks(Filter{})), "inbox", "kept")
		if _, err := s.DeleteProject(p.ID); err == nil {
			t.Error("deleting twice must be an error")
		}

		if err := s.DeleteTask(kept.ID); err != nil {
			t.Fatal(err)
		}
		wantTitles(t, "after DeleteTask", must[[]Task](t)(s.ListTasks(Filter{})), "inbox")
		if _, err := s.GetTask(kept.ID); err == nil {
			t.Error("a deleted task can still be opened")
		}
		if err := s.DeleteTask(kept.ID); err == nil {
			t.Error("deleting twice must be an error")
		}
		_ = inbox
	})
}

func TestMovingATaskBetweenProjectsAndTheInbox(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, _ *time.Time) {
		a := must[Project](t)(s.CreateProject(NewProject{Name: "A"}))
		b := must[Project](t)(s.CreateProject(NewProject{Name: "B"}))
		ms := must[Milestone](t)(s.CreateMilestone(a.ID, "v1", nil))
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "t", ProjectID: IntPtr(a.ID)}))
		must[Task](t)(s.UpdateTask(task.ID, Patch{MilestoneID: IntPtr(ms.ID)}))

		moved := must[Task](t)(s.UpdateTask(task.ID, Patch{ProjectID: IntPtr(b.ID)}))
		if moved.ProjectID == nil || *moved.ProjectID != b.ID || moved.MilestoneID != nil {
			t.Errorf("moved = %+v: the milestone stays behind", moved)
		}
		home := must[Task](t)(s.UpdateTask(task.ID, Patch{ProjectID: IntPtr(0)}))
		if home.ProjectID != nil {
			t.Errorf("to the Inbox = %+v", home)
		}
		if _, err := s.UpdateTask(task.ID, Patch{ProjectID: IntPtr(999)}); err == nil {
			t.Error("a move to a missing project must be an error")
		}
		if got := must[Task](t)(s.UpdateTask(task.ID, Patch{Priority: PriorityPtr(P1)})); got.ProjectID != nil {
			t.Errorf("a patch without ProjectID moved the task: %+v", got)
		}
	})
}

func TestMilestones(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, _ *time.Time) {
		p := must[Project](t)(s.CreateProject(NewProject{Name: "A"}))
		due := time.Date(2026, 12, 1, 0, 0, 0, 0, time.Local)
		m := must[Milestone](t)(s.CreateMilestone(p.ID, " v1 ", &due))
		if m.Title != "v1" || !m.Open || m.DueAt == nil || !m.DueAt.Equal(due) {
			t.Errorf("created = %+v", m)
		}
		if _, err := s.CreateMilestone(p.ID, " ", nil); err == nil {
			t.Error("a blank title must be an error")
		}
		if _, err := s.CreateMilestone(999, "x", nil); err == nil {
			t.Error("a missing project must be an error")
		}

		got := must[Milestone](t)(s.UpdateMilestone(m.ID, MilestonePatch{Title: StrPtr("v1.0")}))
		if got.Title != "v1.0" || got.DueAt == nil || !got.Open {
			t.Errorf("renamed = %+v", got)
		}
		later := due.AddDate(0, 1, 0)
		got = must[Milestone](t)(s.UpdateMilestone(m.ID, MilestonePatch{DueAt: &later}))
		if got.DueAt == nil || !got.DueAt.Equal(later) {
			t.Errorf("due = %+v", got)
		}
		got = must[Milestone](t)(s.UpdateMilestone(m.ID, MilestonePatch{DueAt: TimePtr(time.Time{})}))
		if got.DueAt != nil {
			t.Errorf("due not cleared: %+v", got)
		}
		got = must[Milestone](t)(s.UpdateMilestone(m.ID, MilestonePatch{Open: BoolPtr(false)}))
		if got.Open || got.Title != "v1.0" {
			t.Errorf("closed = %+v", got)
		}
		got = must[Milestone](t)(s.UpdateMilestone(m.ID, MilestonePatch{Open: BoolPtr(true)}))
		if !got.Open {
			t.Errorf("reopened = %+v", got)
		}
		if _, err := s.UpdateMilestone(m.ID, MilestonePatch{Title: StrPtr("")}); err == nil {
			t.Error("a blank title must be an error")
		}
		if _, err := s.UpdateMilestone(999, MilestonePatch{Title: StrPtr("x")}); err == nil {
			t.Error("a missing milestone must be an error")
		}
		if ms := must[[]Milestone](t)(s.ListMilestones(p.ID)); len(ms) != 1 || ms[0].Title != "v1.0" {
			t.Errorf("listed = %+v", ms)
		}
	})
}

func TestSearchTasks(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, clock *time.Time) {
		p := must[Project](t)(s.CreateProject(NewProject{Name: "A"}))
		add := func(in CreateTaskInput, patch Patch) {
			task := must[Task](t)(s.CreateTask(in))
			*clock = clock.Add(time.Minute)
			if patch != (Patch{}) {
				must[Task](t)(s.UpdateTask(task.ID, patch))
			}
			*clock = clock.Add(time.Minute)
		}
		add(CreateTaskInput{Title: "Fix the Login form", ProjectID: IntPtr(p.ID)}, Patch{})
		add(CreateTaskInput{Title: "Docs"}, Patch{Notes: StrPtr("mention login throttling")})
		add(CreateTaskInput{Title: "Refactor", ProjectID: IntPtr(p.ID)}, Patch{Branch: StrPtr("feat/login-v2")})
		add(CreateTaskInput{Title: "Ship it", ProjectID: IntPtr(p.ID), Status: Done}, Patch{})
		add(CreateTaskInput{Title: "Gone", Status: Cancelled}, Patch{})

		wantTitles(t, "login", must[[]Task](t)(s.SearchTasks("login", 0)), "Refactor", "Docs", "Fix the Login form")
		wantTitles(t, "every word", must[[]Task](t)(s.SearchTasks("fix login", 0)), "Fix the Login form")
		wantTitles(t, "cancelled counts", must[[]Task](t)(s.SearchTasks("gone", 0)), "Gone")
		wantTitles(t, "limit", must[[]Task](t)(s.SearchTasks("login", 2)), "Refactor", "Docs")
		wantTitles(t, "blank", must[[]Task](t)(s.SearchTasks("   ", 0)))

		if err := s.ArchiveProject(p.ID); err != nil {
			t.Fatal(err)
		}
		wantTitles(t, "archived projects are left out", must[[]Task](t)(s.SearchTasks("login", 0)), "Docs")
	})
}

func TestSearchFindsAPullRequestNumber(t *testing.T) {
	r, core := realPair(t, nil)
	url := "https://github.com/acme/rocket/pull/42"
	if _, err := core.CreateTask(bg, domain.CreateTaskInput{Title: "Ship it", PRURL: &url}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"#42", "42"} {
		wantTitles(t, q, must[[]Task](t)(r.SearchTasks(q, 0)), "Ship it")
	}
	f := NewFake(fixedNow) // task 2 of the seed is PR #12
	wantTitles(t, "fake #12", must[[]Task](t)(f.SearchTasks("#12", 0)), "Native UI with mygo")
}

func TestLogbook(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, clock *time.Time) {
		p := must[Project](t)(s.CreateProject(NewProject{Name: "A"}))
		a := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a", ProjectID: IntPtr(p.ID)}))
		b := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "b"}))
		c := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "c"}))
		must[Task](t)(s.CreateTask(CreateTaskInput{Title: "open"}))
		for _, step := range []struct {
			id int
			st Status
		}{{a.ID, Done}, {c.ID, Cancelled}, {b.ID, Done}} {
			*clock = clock.Add(time.Hour)
			must[Task](t)(s.UpdateTask(step.id, Patch{Status: StatusPtr(step.st)}))
		}
		wantTitles(t, "newest first", must[[]Task](t)(s.Logbook(0)), "b", "c", "a")
		wantTitles(t, "limit", must[[]Task](t)(s.Logbook(2)), "b", "c")
		if err := s.ArchiveProject(p.ID); err != nil {
			t.Fatal(err)
		}
		wantTitles(t, "archived projects are left out", must[[]Task](t)(s.Logbook(0)), "b", "c")
	})
}

// An agent reporting in an archived project's repository brings it back, and the app hears of it.
func TestRealAnAgentReportRestoresAnArchivedProject(t *testing.T) {
	r, core := realPair(t, nil)
	p := must[Project](t)(r.CreateProject(NewProject{Name: "Rocket", RepoSlug: "acme/rocket"}))
	task := must[Task](t)(r.CreateTask(CreateTaskInput{Title: "launch", ProjectID: IntPtr(p.ID), Source: SourceAgent, Status: Doing}))
	if err := r.ArchiveProject(p.ID); err != nil {
		t.Fatal(err)
	}
	changes := make(chan struct{}, 8)
	r.Subscribe(func() { changes <- struct{}{} })
	r.Start()

	id := int64(task.ID)
	if _, err := core.ReportProgress(bg, domain.ProgressInput{Summary: "back at it", Agent: "claude", TaskID: &id}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the project to be back", func() bool {
		ps, _ := r.ListProjects()
		return len(ps) == 1 && ps[0].ID == p.ID
	})
	select {
	case <-changes:
	case <-time.After(3 * time.Second):
		t.Error("the app was not told")
	}
}
