package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// addProject makes a project with the real sheet's inputs, refusing a failure here.
func addProject(t *testing.T, app *App, name string, parent *data.Project) data.Project {
	t.Helper()
	in := data.NewProject{Name: name}
	if parent != nil {
		in.ParentID = data.IntPtr(parent.ID)
	}
	p, err := app.CreateProject(in)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestTheSidebarNestsASubprojectUnderItsProject(t *testing.T) {
	app, _, _ := newTestApp(t)
	parent := addProject(t, app, "Research", nil)
	sub := addProject(t, app, "Kitchen", &parent)
	tt, m, _ := newTesterFor(app, 1100)
	tt.Frame()

	parentLabel, ok := tt.Find("Research")
	if !ok {
		t.Fatalf("no Research row in %q", tt.Texts())
	}
	subLabel, ok := tt.Find("Kitchen")
	if !ok {
		t.Fatalf("no Kitchen row in %q", tt.Texts())
	}
	if got := subLabel.X - parentLabel.X; got != subRowIndent {
		t.Errorf("the sub-project sits %.0f pt in from its project, want %.0f", got, float32(subRowIndent))
	}
	if subLabel.Y < parentLabel.Y {
		t.Error("the sub-project is drawn above its project")
	}

	tt.Click("Kitchen")
	tt.Frame()
	if want := (nav{screen: screenProject, projectID: sub.ID}); m.nav != want {
		t.Errorf("clicking the sub-project opened %+v, want %+v", m.nav, want)
	}
}

func TestAProjectsMenuMakesASubproject(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.Frame()

	tt.RightClick("withdo")
	tt.Frame()
	if err := tt.ChooseMenuItem(m.say.ManageNewSubproject()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	has(t, tt, m.say.ManageNewSubprojectTitle(), m.say.ManageSubprojectUnder("withdo"))

	tt.Type("Docs")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	var docs data.Project
	found := false
	for _, p := range app.snap.projects {
		if p.Name == "Docs" {
			docs, found = p, true
		}
	}
	if !found {
		t.Fatalf("no Docs project in %+v", app.snap.projects)
	}
	if docs.ParentID == nil || *docs.ParentID != 1 {
		t.Errorf("the sub-project hangs under %v, want withdo (1)", docs.ParentID)
	}
	if want := (nav{screen: screenProject, projectID: docs.ID}); m.nav != want {
		t.Errorf("the new sub-project was not opened: %+v", m.nav)
	}

	parentLabel, _ := tt.Find("withdo")
	subLabel, ok := tt.Find("Docs")
	if !ok {
		t.Fatalf("no Docs row in %q", tt.Texts())
	}
	if got := subLabel.X - parentLabel.X; got != subRowIndent {
		t.Errorf("the new sub-project sits %.0f pt in from its project, want %.0f", got, float32(subRowIndent))
	}
}

func TestArchivingAProjectHidesItsSubprojectsFromTheSidebar(t *testing.T) {
	app, _, _ := newTestApp(t)
	parent := addProject(t, app, "Research", nil)
	addProject(t, app, "Kitchen", &parent)
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	has(t, tt, "Research", "Kitchen")

	if err := app.ArchiveProject(parent.ID); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	hasNot(t, tt, "Research", "Kitchen")
}

func TestASubprojectWhoseProjectIsArchivedReadsAsTopLevel(t *testing.T) {
	app, _, _ := newTestApp(t)
	parent := addProject(t, app, "Research", nil)
	sub := addProject(t, app, "Kitchen", &parent)
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()

	if err := app.ArchiveProject(parent.ID); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	hasNot(t, tt, "Research", "Kitchen")

	if err := app.RestoreProject(sub.ID); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	has(t, tt, "Kitchen")
	hasNot(t, tt, "Research")
	top, _ := tt.Find("withdo") // a top-level project of the seed
	subRow, ok := tt.Find("Kitchen")
	if !ok {
		t.Fatalf("no Kitchen row in %q", tt.Texts())
	}
	if subRow.X != top.X {
		t.Errorf("the sub-project sits at %.0f, want the top level's %.0f", subRow.X, top.X)
	}
}
