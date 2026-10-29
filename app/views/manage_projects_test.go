package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func projectNames(app *App) []string {
	var out []string
	for _, p := range app.snap.projects {
		out = append(out, p.Name)
	}
	return out
}

func openNewProject(t *testing.T, tt *ui.Tester) {
	t.Helper()
	if err := tt.Click("New project"); err != nil {
		t.Fatal(err)
	}
}

func TestPlusOnTheProjectsHeaderOpensTheNewProjectSheet(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.Frame()
	if tt.HasText("GitHub repository") {
		t.Fatal("the sheet is open before the plus is clicked")
	}
	openNewProject(t, tt)
	for _, want := range []string{"Name", "GitHub repository", "Cancel", "Create"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	// The repository field is folded until the user asks for it.
	if tt.HasText("owner/repo") || tt.HasText("Agent reports from this repository land in the project.") {
		t.Errorf("the repository field is open by default: %q", tt.Texts())
	}
	if err := tt.Click("GitHub repository"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"owner/repo", "Agent reports from this repository land in the project."} {
		if !tt.HasText(want) {
			t.Errorf("missing %q once unfolded in %q", want, tt.Texts())
		}
	}
}

func TestTheRepositoryFieldIsFoldedAgainInTheNextSheet(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	openNewProject(t, tt)
	_ = tt.Click("GitHub repository")
	_ = tt.Click("Cancel")
	m.openNewProject(0)
	tt.Frame()
	if tt.HasText("owner/repo") {
		t.Errorf("the repository field stays unfolded: %q", tt.Texts())
	}
}

func TestNewProjectNeedsANameAndOpensWhenCreated(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	tt.Frame()
	openNewProject(t, tt)
	_ = tt.Click("Create") // disabled while the name is blank
	if len(app.snap.projects) != 3 {
		t.Fatalf("a project without a name was created: %v", projectNames(app))
	}

	tt.Type("Garden")
	tt.Key(0, ui.KeyEnter)
	if !slices.Contains(projectNames(app), "Garden") {
		t.Fatalf("projects %v", projectNames(app))
	}
	if tt.HasText("GitHub repository") {
		t.Error("the sheet stays open after Create")
	}
	garden := app.snap.projects[slices.IndexFunc(app.snap.projects, func(p data.Project) bool { return p.Name == "Garden" })]
	if m.nav.screen != screenProject || m.nav.projectID != garden.ID {
		t.Errorf("nav = %+v, want the new project", m.nav)
	}
	if !tt.HasText("No tasks yet.") || !tt.HasText("Add a task, or ask me anything") {
		t.Errorf("the empty project does not invite a task: %q", tt.Texts())
	}
}

func TestNewProjectWithARepository(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	openNewProject(t, tt)
	tt.Type("Rocket")
	if err := tt.Click("GitHub repository"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("owner/repo"); err != nil {
		t.Fatal(err)
	}
	tt.Type("https://github.com/acme/rocket")
	if err := tt.Click("Create"); err != nil {
		t.Fatal(err)
	}
	var got data.Project
	for _, p := range app.snap.projects {
		if p.Name == "Rocket" {
			got = p
		}
	}
	if got.RepoSlug != "acme/rocket" {
		t.Errorf("project %+v: a pasted address is read as owner/repo", got)
	}
}

func TestNewProjectSaysWhatIsWrongWithTheRepositoryAndStaysOpen(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	openNewProject(t, tt)
	tt.Type("Rocket")
	_ = tt.Click("GitHub repository")
	_ = tt.Click("owner/repo")
	tt.Type("rocket")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("Use the form owner/repo, like QingYunA/withdo.") {
		t.Errorf("no word about the repository: %q", tt.Texts())
	}
	if len(app.snap.projects) != 3 {
		t.Error("a project was created with a bad repository")
	}

	// Fix it to one the seed already uses.
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("QingYunA/withdo")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("“withdo” already uses this repository.") {
		t.Errorf("no word about the taken repository: %q", tt.Texts())
	}
	if !tt.HasText("Create") {
		t.Error("the sheet closed")
	}
}

func TestEscAndCancelCloseTheNewProjectSheet(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	openNewProject(t, tt)
	tt.Type("Nope")
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("GitHub repository") {
		t.Error("Esc did not close the sheet")
	}
	openNewProject(t, tt)
	if tt.HasText("Nope") {
		t.Error("the sheet remembers a cancelled name")
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("GitHub repository") || len(app.snap.projects) != 3 {
		t.Error("Cancel did not close the sheet cleanly")
	}
}

func TestProjectContextMenu(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.Frame()
	got := drawnMenu(tt, func() { _ = tt.RightClick("picpocket") })
	if want := []string{"Open QingYunA/picpocket on GitHub", "New milestone…", "New sub-project…", "Rename", "Archive", "Delete…"}; !slices.Equal(got, want) {
		t.Errorf("menu %q, want %q", got, want)
	}
}

// The menu is drawn in the window, in the app's theme: the system shows none.
func TestProjectContextMenuIsDrawnInTheWindow(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("picpocket")
	tt.Frame()
	if got := tt.Menu(); got != nil {
		t.Errorf("the system shows a menu too: %q", got)
	}
	for _, item := range []string{"New milestone…", "New sub-project…", "Rename", "Archive", "Delete…"} {
		if !tt.HasText(item) {
			t.Errorf("the window shows no item %q: %q", item, tt.Texts())
		}
	}
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("New sub-project…") {
		t.Error("Esc left the menu open")
	}
}

// The sidebar rows take no focus, so the menu opens by the pointer; once open it has the keys.
func TestProjectContextMenuWorksFromTheKeyboard(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("picpocket")
	tt.Frame()
	if !tt.HasText("New sub-project…") {
		t.Fatalf("no menu: %q", tt.Texts())
	}
	// Escape closes it.
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("New sub-project…") {
		t.Fatal("Esc left the menu open")
	}
	// Down walks the items (GitHub, New milestone, New sub-project, Rename, Archive) and Enter
	// chooses the fifth.
	_ = tt.RightClick("picpocket")
	tt.Frame()
	for range 5 {
		tt.Key(0, ui.KeyDown)
	}
	tt.Key(0, ui.KeyEnter)
	if slices.Contains(projectNames(app), "picpocket") {
		t.Error("Enter on Archive did not archive the project")
	}
}

func TestProjectContextMenuOpensTheRepositoryAndLeavesItOutWithoutOne(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	_ = tt.RightClick("withdo")
	if err := chooseItem(tt, "Open QingYunA/withdo on GitHub"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || rec.urls[0] != "https://github.com/QingYunA/withdo" {
		t.Errorf("opened %v", rec.urls)
	}

	f := demoStore(t, fixedNow, false)
	p := f.SeedProject("local-only", "")
	plain := NewApp(f, &recorder{}, testConfig())
	plain.Now = fixedNow
	tt2, _, _ := newTesterFor(plain, 1100)
	tt2.Frame()
	got := drawnMenu(tt2, func() { _ = tt2.RightClick(p.Name) })
	if want := []string{"New milestone…", "New sub-project…", "Rename", "Archive", "Delete…"}; !slices.Equal(got, want) {
		t.Errorf("menu without a repository %q, want %q", got, want)
	}
}

func TestRenameAProjectInTheSidebar(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("picpocket")
	if err := chooseItem(tt, "Rename"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("pocket photos")
	tt.Key(0, ui.KeyEnter)
	if !slices.Contains(projectNames(app), "pocket photos") || slices.Contains(projectNames(app), "picpocket") {
		t.Errorf("projects %v", projectNames(app))
	}
	if !tt.HasText("pocket photos") {
		t.Errorf("the sidebar does not show the new name: %q", tt.Texts())
	}

	// Esc puts the old name back.
	_ = tt.RightClick("pocket photos")
	_ = chooseItem(tt, "Rename")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("zzz")
	tt.Key(0, ui.KeyEscape)
	if !slices.Contains(projectNames(app), "pocket photos") || !tt.HasText("pocket photos") {
		t.Errorf("Esc kept the edit: %v %q", projectNames(app), tt.Texts())
	}

	// A blank name changes nothing.
	_ = tt.RightClick("pocket photos")
	_ = chooseItem(tt, "Rename")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyEnter)
	if !slices.Contains(projectNames(app), "pocket photos") {
		t.Errorf("a blank name was saved: %v", projectNames(app))
	}
}

func TestArchivingTheOpenProjectGoesToNow(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 3})
	tt.Frame()
	_ = tt.RightClick("picpocket")
	if err := chooseItem(tt, "Archive"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(projectNames(app), "picpocket") {
		t.Errorf("the archived project is still listed: %v", projectNames(app))
	}
	if m.nav.screen != screenMine {
		t.Errorf("nav = %+v: archiving the open project goes to Now", m.nav)
	}
	if tt.HasText("Export thumbnails to HEIC") {
		t.Error("tasks of an archived project are still shown")
	}
}

func TestDeleteAProjectAsksFirstAndNamesTheTasks(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	_ = tt.RightClick("withdo")
	if err := chooseItem(tt, "Delete…"); err != nil {
		t.Fatal(err)
	}
	n := app.taskCount(1)
	for _, want := range []string{"Delete “withdo”?", "This removes " + itoa(n) + " tasks, their reports and the project's milestones. It can't be undone. To keep the work, archive the project instead.", "Cancel", "Delete"} {
		if !tt.HasText(want) {
			t.Errorf("confirmation lacks %q in %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(projectNames(app), "withdo") {
		t.Fatal("Cancel deleted the project")
	}

	_ = tt.RightClick("withdo")
	_ = chooseItem(tt, "Delete…")
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("Delete “withdo”?") || !slices.Contains(projectNames(app), "withdo") {
		t.Fatal("Esc did not cancel")
	}

	_ = tt.RightClick("withdo")
	_ = chooseItem(tt, "Delete…")
	tt.Key(0, ui.KeyEnter) // the default button is Cancel
	if !slices.Contains(projectNames(app), "withdo") {
		t.Fatal("Enter deleted the project: the default button must be Cancel")
	}
	if tt.HasText("Delete “withdo”?") {
		t.Error("Enter on Cancel left the question open")
	}

	_ = tt.RightClick("withdo")
	_ = chooseItem(tt, "Delete…")
	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(projectNames(app), "withdo") || app.taskCount(1) != 0 {
		t.Errorf("project %v, tasks %d", projectNames(app), app.taskCount(1))
	}
	if m.nav.screen != screenMine {
		t.Errorf("nav = %+v: deleting the open project goes to Now", m.nav)
	}
}

func TestDeleteAnEmptyProjectSaysSo(t *testing.T) {
	app, p := emptyProjectApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	_ = tt.RightClick(p.Name)
	_ = chooseItem(tt, "Delete…")
	if !tt.HasText("This removes the project and its milestones. It can't be undone.") {
		t.Errorf("body: %q", tt.Texts())
	}
}
