package syncclient_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// TestS2b8ResumeSeeding is the review's takeover test: a device with local data sends
// resume_seeding and takes the workspace's seeding over from the device that had it.
func TestS2b8ResumeSeeding(t *testing.T) {
	h := newHarness(t)
	var raw syncproto.CreateDeviceResponse
	if code := h.post(t, "/v1/devices", syncproto.CreateDeviceRequest{Name: "raw"}, &raw); code != 200 {
		t.Fatalf("register raw device = %d, want 200", code)
	}
	if !raw.Seeding {
		t.Fatal("the first device on an empty workspace is not seeding")
	}
	var pushed syncproto.PushResponse
	code := h.post(t, "/v1/push", syncproto.PushRequest{
		DeviceID: raw.DeviceID,
		Mutations: []syncproto.Mutation{{
			ID: 1, Entity: syncproto.EntityProjects, RowID: 100, Op: syncproto.OpPut,
			Data: map[string]any{
				"name": "A project", "repo_slug": nil, "created_at": seedTimestamp,
				"updated_at": seedTimestamp, "archived_at": nil, "parent_id": nil,
			},
			Full: true,
		}},
	}, &pushed)
	if code != 200 {
		t.Fatalf("raw snapshot push = %d, want 200", code)
	}

	b := h.newDevice(t, nil)
	bProject := newProject(t, b.core, "B project", nil)
	newTask(t, b.core, domain.CreateTaskInput{Title: "B task", ProjectID: &bProject.ID})
	b.link(t, "B")

	if got := h.serverSeedingDevices(t); got != 0 {
		t.Errorf("seeding devices after B took over = %d, want 0", got)
	}
	if code := h.post(t, "/v1/devices/seeded", syncproto.SeededRequest{DeviceID: raw.DeviceID}, nil); code != 403 {
		t.Errorf("old device ending seeding = %d, want 403", code)
	}
	got := must[*domain.Project](t)(b.core.GetProject(testCtx, 100))
	if got == nil || got.Name != "A project" {
		t.Fatalf("B did not pull the old seeder's project: %+v", got)
	}
	server := h.serverRows(t, syncproto.EntityProjects)
	if row, ok := server[100]; !ok || row.Deleted {
		t.Errorf("server project 100 = %+v, want live", row)
	}
	if row, ok := server[bProject.ID]; !ok || row.Deleted {
		t.Errorf("server project %d = %+v, want live", bProject.ID, row)
	}
	assertDeviceMatchesServer(t, h, b)
}

// TestS2b9ProjectDeleteRace is the review's concurrent delete test: A deletes a project while B
// adds a task under it; the server's reference cascade turns B's put into a delete, and neither the
// server nor either device keeps the task.
func TestS2b9ProjectDeleteRace(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	project := newProject(t, a.core, "doomed", nil)
	old := newTask(t, a.core, domain.CreateTaskInput{Title: "old", ProjectID: &project.ID})
	a.link(t, "A")
	b := h.newDevice(t, nil)
	b.link(t, "B")

	race := newTask(t, b.core, domain.CreateTaskInput{Title: "race", ProjectID: &project.ID})
	if _, err := a.core.DeleteProject(testCtx, project.ID); err != nil {
		t.Fatal(err)
	}
	a.sync(t)
	b.sync(t)
	a.sync(t)

	for name, d := range map[string]*device{"A": a, "B": b} {
		for _, table := range []string{"projects", "milestones", "tasks", "reports"} {
			if got := count(t, d.db, "SELECT COUNT(*) FROM "+table); got != 0 {
				t.Errorf("%s has %d %s, want 0", name, got, table)
			}
		}
		assertDeviceMatchesServer(t, h, d)
	}
	for name, id := range map[string]int64{"old": old.ID, "race": race.ID} {
		row, ok := h.serverRows(t, syncproto.EntityTasks)[id]
		if !ok || !row.Deleted {
			t.Errorf("server task %s %d = %+v, want a tombstone", name, id, row)
		}
	}
	if row, ok := h.serverRows(t, syncproto.EntityProjects)[project.ID]; !ok || !row.Deleted {
		t.Errorf("server project = %+v, want a tombstone", row)
	}
}

// TestS2b10RestoreSurvivesSync is the review's Undo test: deleting a task and restoring it with
// RestoreTask works before a sync and after one; the full put revives the server's tombstone.
func TestS2b10RestoreSurvivesSync(t *testing.T) {
	for _, tc := range []struct {
		name              string
		syncBeforeRestore bool
	}{
		{"before a sync", false},
		{"after a sync", true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			a := h.newDevice(t, nil)
			project := newProject(t, a.core, "alpha", nil)
			task := newTask(t, a.core, domain.CreateTaskInput{Title: "restorable", ProjectID: &project.ID})
			a.link(t, "A")
			b := h.newDevice(t, nil)
			b.link(t, "B")

			snap := must[*domain.TaskSnapshot](t)(a.core.SnapshotTask(testCtx, task.ID))
			if err := a.core.DeleteTask(testCtx, task.ID); err != nil {
				t.Fatal(err)
			}
			if tc.syncBeforeRestore {
				a.sync(t)
				b.sync(t)
				if got := must[*domain.Task](t)(a.core.GetTask(testCtx, task.ID)); got != nil {
					t.Fatalf("the delete did not sync: %+v", got)
				}
			}
			restored := must[*domain.Task](t)(a.core.RestoreTask(testCtx, *snap))
			if restored.ID != task.ID {
				t.Fatalf("restored id = %d, want %d", restored.ID, task.ID)
			}
			a.sync(t)
			b.sync(t)

			for name, d := range map[string]*device{"A": a, "B": b} {
				got := must[*domain.Task](t)(d.core.GetTask(testCtx, task.ID))
				if got == nil {
					t.Fatalf("%s lost the restored task", name)
				}
				if got.Title != "restorable" {
					t.Errorf("%s title = %q, want restorable", name, got.Title)
				}
				assertDeviceMatchesServer(t, h, d)
			}
			if row, ok := h.serverRows(t, syncproto.EntityTasks)[task.ID]; !ok || row.Deleted {
				t.Errorf("server task = %+v, want revived", row)
			}
		})
	}
}

// TestS2b11CategoryDeleteRace is the review's concurrent category delete test: A deletes a
// category while B puts a task under it; the server clears the reference, and every device ends
// with category_id NULL.
func TestS2b11CategoryDeleteRace(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	project := newProject(t, a.core, "alpha", nil)
	task := newTask(t, a.core, domain.CreateTaskInput{Title: "sort me", ProjectID: &project.ID})
	category := must[*domain.Category](t)(a.core.CreateCategory(testCtx, domain.CreateCategoryInput{
		Name: "Reading", Icon: dotIcon,
	}))
	a.link(t, "A")
	b := h.newDevice(t, nil)
	b.link(t, "B")

	must[*domain.Task](t)(b.core.UpdateTask(testCtx, task.ID,
		domain.TaskPatch{CategoryID: domain.Set(category.ID)}))
	must[domain.DeletedCategory](t)(a.core.DeleteCategory(testCtx, category.ID))
	a.sync(t)
	b.sync(t)
	a.sync(t)

	for name, d := range map[string]*device{"A": a, "B": b} {
		got := must[*domain.Task](t)(d.core.GetTask(testCtx, task.ID))
		if got == nil {
			t.Fatalf("%s lost the task", name)
		}
		if got.CategoryID != nil {
			t.Errorf("%s category_id = %d, want NULL", name, *got.CategoryID)
		}
		if cats := must[[]domain.Category](t)(d.core.ListCategories(testCtx)); len(cats) != 2 {
			t.Errorf("%s has %d categories, want the two built-ins", name, len(cats))
		}
		assertDeviceMatchesServer(t, h, d)
	}
	if row, ok := h.serverRows(t, syncproto.EntityCategories)[category.ID]; !ok || !row.Deleted {
		t.Errorf("server category = %+v, want a tombstone", row)
	}
	if row, ok := h.serverRows(t, syncproto.EntityTasks)[task.ID]; !ok || row.Deleted {
		t.Errorf("server task = %+v, want live", row)
	} else if row.Data["category_id"] != nil {
		t.Errorf("server task category_id = %v, want NULL", row.Data["category_id"])
	}
}
