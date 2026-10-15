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
	for _, want := range []string{"Name", "GitHub repository", "owner/repo", "Agent reports from this repository land in the project.", "Cancel", "Create"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
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
	if !tt.HasText("Tasks appear here") && !tt.HasText("Add a task") {
		t.Errorf("the empty project does not invite a task: %q", tt.Texts())
	}
}

func TestNewProjectWithARepository(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	openNewProject(t, tt)
	tt.Type("Rocket")
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
	_ = tt.Click("owner/repo")
	tt.Type("rocket")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("Use the form owner/repo, like QingYunA/adhd-todo.") {
		t.Errorf("no word about the repository: %q", tt.Texts())
	}
	if len(app.snap.projects) != 3 {
		t.Error("a project was created with a bad repository")
	}

	// Fix it to one the seed already uses.
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("QingYunA/adhd-todo")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("“adhd-todo” already uses this repository.") {
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
	if err := tt.RightClick("picpocket"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.Menu(), []string{"Rename", "Archive", "-", "Delete…"}; !slices.Equal(got, want) {
		t.Errorf("menu %q, want %q", got, want)
	}
}

func TestRenameAProjectInTheSidebar(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("picpocket")
	if err := tt.ChooseMenuItem("Rename"); err != nil {
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
	_ = tt.ChooseMenuItem("Rename")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("zzz")
	tt.Key(0, ui.KeyEscape)
	if !slices.Contains(projectNames(app), "pocket photos") || !tt.HasText("pocket photos") {
		t.Errorf("Esc kept the edit: %v %q", projectNames(app), tt.Texts())
	}

	// A blank name changes nothing.
	_ = tt.RightClick("pocket photos")
	_ = tt.ChooseMenuItem("Rename")
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
	if err := tt.ChooseMenuItem("Archive"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(projectNames(app), "picpocket") {
		t.Errorf("the archived project is still listed: %v", projectNames(app))
	}
	if m.nav.screen != screenNow {
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
	_ = tt.RightClick("adhd-todo")
	if err := tt.ChooseMenuItem("Delete…"); err != nil {
		t.Fatal(err)
	}
	n := app.taskCount(1)
	for _, want := range []string{"Delete “adhd-todo”?", "This removes " + itoa(n) + " tasks, their reports and the project's milestones. It can't be undone. To keep the work, archive the project instead.", "Cancel", "Delete"} {
		if !tt.HasText(want) {
			t.Errorf("confirmation lacks %q in %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(projectNames(app), "adhd-todo") {
		t.Fatal("Cancel deleted the project")
	}

	_ = tt.RightClick("adhd-todo")
	_ = tt.ChooseMenuItem("Delete…")
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("Delete “adhd-todo”?") || !slices.Contains(projectNames(app), "adhd-todo") {
		t.Fatal("Esc did not cancel")
	}

	_ = tt.RightClick("adhd-todo")
	_ = tt.ChooseMenuItem("Delete…")
	tt.Key(0, ui.KeyEnter) // the default button is Cancel
	if !slices.Contains(projectNames(app), "adhd-todo") {
		t.Fatal("Enter deleted the project: the default button must be Cancel")
	}
	if tt.HasText("Delete “adhd-todo”?") {
		t.Error("Enter on Cancel left the question open")
	}

	_ = tt.RightClick("adhd-todo")
	_ = tt.ChooseMenuItem("Delete…")
	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(projectNames(app), "adhd-todo") || app.taskCount(1) != 0 {
		t.Errorf("project %v, tasks %d", projectNames(app), app.taskCount(1))
	}
	if m.nav.screen != screenNow {
		t.Errorf("nav = %+v: deleting the open project goes to Now", m.nav)
	}
}

func TestDeleteAnEmptyProjectSaysSo(t *testing.T) {
	app, p := emptyProjectApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	_ = tt.RightClick(p.Name)
	_ = tt.ChooseMenuItem("Delete…")
	if !tt.HasText("This removes the project and its milestones. It can't be undone.") {
		t.Errorf("body: %q", tt.Texts())
	}
}
