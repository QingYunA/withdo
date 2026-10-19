package data

import (
	"testing"
	"time"
)

func TestSubprojects(t *testing.T) {
	withStore(t, func(t *testing.T, s Store, clock *time.Time) {
		root := must[Project](t)(s.CreateProject(NewProject{Name: "adhd-todo"}))
		app := must[Project](t)(s.CreateProject(NewProject{Name: "App", ParentID: IntPtr(root.ID)}))
		ui := must[Project](t)(s.CreateProject(NewProject{Name: "UI", ParentID: IntPtr(app.ID)}))
		if app.ParentID == nil || *app.ParentID != root.ID {
			t.Errorf("the sub-project's parent = %v, want %d", app.ParentID, root.ID)
		}
		if ui.ParentID == nil || *ui.ParentID != app.ID {
			t.Errorf("a sub-project of a sub-project = %v, want %d", ui.ParentID, app.ID)
		}

		missing := 999
		if _, err := s.CreateProject(NewProject{Name: "Orphan", ParentID: &missing}); err == nil {
			t.Error("a project under a missing parent was made")
		}

		if ps := must[[]Project](t)(s.ListProjects()); len(ps) != 3 {
			t.Fatalf("projects = %+v", ps)
		}

		// Archiving a project takes its sub-projects, so no row is left hanging.
		if err := s.ArchiveProject(root.ID); err != nil {
			t.Fatal(err)
		}
		if ps := must[[]Project](t)(s.ListProjects()); len(ps) != 0 {
			t.Errorf("sub-projects of an archived project are listed: %+v", ps)
		}
		if arch := must[[]Project](t)(s.ListArchivedProjects()); len(arch) != 3 {
			t.Errorf("archived = %+v", arch)
		}
		if _, err := s.CreateProject(NewProject{Name: "Late", ParentID: IntPtr(root.ID)}); err == nil {
			t.Error("a project under an archived parent was made")
		}

		// Restoring a sub-project brings back what was archived with it, not the project above.
		if err := s.RestoreProject(app.ID); err != nil {
			t.Fatal(err)
		}
		if ps := must[[]Project](t)(s.ListProjects()); len(ps) != 2 {
			t.Fatalf("restored = %+v", ps)
		}
		for _, p := range must[[]Project](t)(s.ListProjects()) {
			if p.ID == root.ID {
				t.Error("restoring a sub-project restored the project above it")
			}
		}
		if err := s.RestoreProject(root.ID); err != nil {
			t.Fatal(err)
		}
		if ps := must[[]Project](t)(s.ListProjects()); len(ps) != 3 {
			t.Errorf("restoring the project above = %+v", ps)
		}

		// A sub-project archived earlier stays archived when a later action archives and restores
		// the branch it hangs in.
		*clock = clock.Add(time.Millisecond)
		if err := s.ArchiveProject(ui.ID); err != nil {
			t.Fatal(err)
		}
		*clock = clock.Add(time.Millisecond)
		if err := s.ArchiveProject(root.ID); err != nil {
			t.Fatal(err)
		}
		if err := s.RestoreProject(root.ID); err != nil {
			t.Fatal(err)
		}
		archivedUI := false
		for _, p := range must[[]Project](t)(s.ListArchivedProjects()) {
			if p.ID == ui.ID {
				archivedUI = true
			}
		}
		if !archivedUI {
			t.Error("a sub-project archived earlier came back with a later archive action")
		}
		if err := s.RestoreProject(ui.ID); err != nil {
			t.Fatal(err)
		}

		// Deleting a parent leaves the sub-project standing as a top-level project.
		if _, err := s.DeleteProject(root.ID); err != nil {
			t.Fatal(err)
		}
		ps := must[[]Project](t)(s.ListProjects())
		if len(ps) != 2 {
			t.Fatalf("projects after deleting the parent = %+v", ps)
		}
		for _, p := range ps {
			if p.ID == app.ID && p.ParentID != nil {
				t.Errorf("the sub-project still hangs under the deleted project: %+v", p)
			}
			if p.ID == ui.ID && (p.ParentID == nil || *p.ParentID != app.ID) {
				t.Errorf("the project below it lost its parent: %+v", p)
			}
		}
	})
}
