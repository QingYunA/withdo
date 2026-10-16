package data

import (
	"testing"
	"time"
)

func TestCategoryOfTheTaskItselfWinsOverItsProject(t *testing.T) {
	repo := func(p *int) bool { return p != nil && *p == 1 }
	cases := []struct {
		name string
		task Task
		want Category
	}{
		{"unclassified, no project", Task{}, ""},
		{"unclassified, project without a repository", Task{ProjectID: IntPtr(2)}, ""},
		{"unclassified, repository project counts as work", Task{ProjectID: IntPtr(1)}, CategoryWork},
		{"life in a repository project stays life", Task{ProjectID: IntPtr(1), Category: CategoryLife}, CategoryLife},
		{"work in the inbox", Task{Category: CategoryWork}, CategoryWork},
	}
	for _, c := range cases {
		if got := CategoryOf(c.task, repo); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if got := CategoryOf(Task{ProjectID: IntPtr(1)}, nil); got != "" {
		t.Errorf("without a way to tell, a project task is %q", got)
	}
}

func TestProjectHasRepo(t *testing.T) {
	for p, want := range map[Project]bool{
		{}: false, {RepoSlug: "a/b"}: true, {RepoPath: "~/Code/b"}: true, {Name: "Home"}: false,
	} {
		if got := p.HasRepo(); got != want {
			t.Errorf("%+v: %v", p, got)
		}
	}
}

func TestFakeCreatesAndPatchesACategory(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	made, err := f.CreateTask(CreateTaskInput{Title: "Pay rent", Category: CategoryLife})
	if err != nil || made.Category != CategoryLife {
		t.Fatalf("created %+v, %v", made, err)
	}
	moved, err := f.UpdateTask(made.ID, Patch{Category: CategoryPtr(CategoryWork)})
	if err != nil || moved.Category != CategoryWork {
		t.Fatalf("patched %+v, %v", moved, err)
	}
	kept, _ := f.UpdateTask(made.ID, Patch{Title: StrPtr("Pay the rent")})
	if kept.Category != CategoryWork {
		t.Errorf("a patch without a category changed it to %q", kept.Category)
	}
	cleared, err := f.UpdateTask(made.ID, Patch{Category: CategoryPtr("")})
	if err != nil || cleared.Category != "" {
		t.Errorf("cleared %+v, %v", cleared, err)
	}
	if _, err := f.UpdateTask(made.ID, Patch{Category: CategoryPtr("play")}); err == nil {
		t.Error("an unknown category was accepted")
	}
}

// categoryTasks is a store with one open task of mine in each category, one in a repository's
// project, and one nobody classified.
func categoryTasks(t *testing.T) (MineInput, func(Task) Category) {
	t.Helper()
	f := NewEmptyFake(fixedNow)
	f.projects = []Project{{ID: 1, Name: "app", RepoSlug: "me/app"}, {ID: 2, Name: "Home"}}
	f.CreateTask(CreateTaskInput{Title: "work task", Category: CategoryWork})
	f.CreateTask(CreateTaskInput{Title: "life task", Category: CategoryLife})
	f.CreateTask(CreateTaskInput{Title: "repo task", ProjectID: IntPtr(1)})
	f.CreateTask(CreateTaskInput{Title: "loose task", ProjectID: IntPtr(2)})
	of := func(task Task) Category { return CategoryOf(task, func(p *int) bool { return p != nil && *p == 1 }) }
	return mineInputOf(t, f), of
}

func TestBuildMineKeepsOnlyTheTasksOfASlice(t *testing.T) {
	in, of := categoryTasks(t)
	slice := func(c Category) []string {
		in := in
		in.Keep = func(task Task) bool { return of(task) == c }
		return titles(BuildMine(in).Later)
	}
	if got := slice(CategoryWork); !equal(got, []string{"repo task", "work task"}) {
		t.Errorf("work %q: its own category, and a repository's project", got)
	}
	if got := slice(CategoryLife); !equal(got, []string{"life task"}) {
		t.Errorf("life %q", got)
	}
	if got := slice(""); !equal(got, []string{"loose task"}) {
		t.Errorf("unsorted %q", got)
	}
	if got := len(BuildMine(in).Later); got != 4 {
		t.Errorf("without a filter %d rows, want all 4", got)
	}
}

func TestBuildMineSliceCountsItsOwnLaterOverflow(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	for i := 0; i < MaxUndated+3; i++ {
		f.CreateTask(CreateTaskInput{Title: "work " + itoaT(i), Category: CategoryWork})
	}
	f.CreateTask(CreateTaskInput{Title: "life", Category: CategoryLife})
	in := mineInputOf(t, f)
	in.Keep = func(task Task) bool { return task.Category == CategoryLife }
	m := BuildMine(in)
	if len(m.Later) != 1 || m.MoreLater != 0 {
		t.Errorf("life slice shows %d rows and %d more: the work tasks must not crowd it", len(m.Later), m.MoreLater)
	}
}

func TestASubtaskFollowsItsParentIntoASlice(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	parent, _ := f.CreateTask(CreateTaskInput{Title: "Parent", Source: SourceAgent, Status: Doing, Category: CategoryWork})
	f.CreateTask(CreateTaskInput{Title: "child", Source: SourceAgent, Status: Doing, ParentID: IntPtr(parent.ID)})
	other, _ := f.CreateTask(CreateTaskInput{Title: "Other", Source: SourceAgent, Status: Doing, Category: CategoryLife})
	f.CreateTask(CreateTaskInput{Title: "other child", Source: SourceAgent, Status: Doing, ParentID: IntPtr(other.ID)})
	in := mineInputOf(t, f)
	in.Keep = func(task Task) bool { return task.Category == CategoryWork }
	a := BuildAgents(in)
	if len(a.Working) != 1 || a.Working[0].Title != "Parent" || len(a.Working[0].Subs) != 1 || a.Working[0].Subs[0].Title != "child" {
		t.Errorf("work slice of the agents: %+v", a.Working)
	}
}

func TestMineOnlyFiltersTheSections(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	f.CreateTask(CreateTaskInput{Title: "w", Category: CategoryWork, Status: Doing})
	f.CreateTask(CreateTaskInput{Title: "l", Category: CategoryLife, Status: Doing})
	f.CreateTask(CreateTaskInput{Title: "later", Category: CategoryLife})
	m := mineOf(t, f)
	life := m.Only(func(task Task) bool { return task.Category == CategoryLife })
	if got := titles(life.Today); !equal(got, []string{"l"}) || len(life.Later) != 1 || life.Count() != 1 || life.Empty() {
		t.Errorf("life %+v", life)
	}
	none := m.Only(func(task Task) bool { return task.Category == "" })
	if !none.Empty() || none.Count() != 0 {
		t.Errorf("unsorted %+v", none)
	}
}

func TestAgentsScreenTalliesByAgentInScreenOrder(t *testing.T) {
	f := NewEmptyFake(fixedNow)
	a1, _ := f.CreateTask(CreateTaskInput{Title: "a", Source: SourceAgent, Status: Doing})
	b1, _ := f.CreateTask(CreateTaskInput{Title: "b", Source: SourceAgent, Status: Doing})
	a2, _ := f.CreateTask(CreateTaskInput{Title: "c", Source: SourceAgent, Status: Blocked})
	d, _ := f.CreateTask(CreateTaskInput{Title: "d", Source: SourceAgent, Status: Doing})
	f.ReportAs(a1.ID, "claude-code", "x", fixedNow().Add(-3*time.Minute))
	f.ReportAs(b1.ID, "Codex", "x", fixedNow().Add(-1*time.Minute))
	f.ReportAs(a2.ID, "Claude Code", "x", fixedNow().Add(-9*time.Minute))
	f.ReportAs(d.ID, "gemini", "x", fixedNow().Add(-20*time.Minute))
	f.UpdateTask(d.ID, Patch{Status: StatusPtr(Done)})
	in := mineInputOf(t, f)
	tally := BuildAgents(in).Tally(func(task Task) string {
		switch task.Title {
		case "a", "c":
			return "Claude Code"
		case "b":
			return "Codex"
		}
		return "Gemini"
	})
	if len(tally) != 3 || tally[0].Name != "Codex" || tally[0].Working != 1 ||
		tally[1].Name != "Claude Code" || tally[1].Working != 2 ||
		tally[2].Name != "Gemini" || tally[2].Working != 0 {
		t.Errorf("tally %+v: newest report first, then who only finished something", tally)
	}
}
