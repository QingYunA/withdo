package syncserver_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// lastChanges keys the last change of every row by entity and id: applying a pull in version order
// leaves the row in this state.
func lastChanges(changes []syncproto.Change) map[string]syncproto.Change {
	out := map[string]syncproto.Change{}
	for _, change := range changes {
		out[change.Entity+"/"+strconv.FormatInt(change.RowID, 10)] = change
	}
	return out
}

func changeKey(entity string, rowID int64) string {
	return entity + "/" + strconv.FormatInt(rowID, 10)
}

// TestS2a10ReferencesCascade checks the reference table end to end: deleting a project deletes its
// milestones, tasks and their reports, clears a sub-project's parent, and leaves unrelated rows.
func TestS2a10ReferencesCascade(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	mutations := []syncproto.Mutation{
		{ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"name": "parent", "created_at": "t", "updated_at": "t"}},
		{ID: 2, Entity: syncproto.EntityProjects, RowID: 2, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"name": "child", "parent_id": 1, "created_at": "t", "updated_at": "t"}},
		{ID: 3, Entity: syncproto.EntityProjects, RowID: 3, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"name": "other", "created_at": "t", "updated_at": "t"}},
		{ID: 4, Entity: syncproto.EntityCategories, RowID: 10, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"name": "Reading", "icon": "x", "color": "blue", "position": 3, "created_at": "t"}},
		{ID: 5, Entity: syncproto.EntityMilestones, RowID: 20, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"project_id": 1, "title": "v1", "status": "open", "created_at": "t"}},
		{ID: 6, Entity: syncproto.EntityTasks, RowID: 30, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"project_id": 1, "milestone_id": 20, "category_id": 10,
				"title": "one", "source": "user", "created_at": "t", "updated_at": "t"}},
		{ID: 7, Entity: syncproto.EntityTasks, RowID: 31, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"project_id": 1, "parent_id": 30,
				"title": "sub", "source": "user", "created_at": "t", "updated_at": "t"}},
		{ID: 8, Entity: syncproto.EntityReports, RowID: 40, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"task_id": 30, "agent": "a", "summary": "s", "created_at": "t", "links": "[]"}},
	}
	code, _ := h.push(t, aliceToken, dev.DeviceID, mutations...)
	requireStatus(t, "snapshot", code, http.StatusOK)
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, dev.DeviceID), http.StatusOK)

	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 9, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpDelete,
	})
	requireStatus(t, "delete project", code, http.StatusOK)

	code, pulled := h.pull(t, aliceToken, dev.DeviceID, 0, syncproto.MaxPullLimit)
	requireStatus(t, "pull", code, http.StatusOK)
	rows := lastChanges(pulled.Changes)

	for _, key := range []string{
		changeKey(syncproto.EntityProjects, 1),
		changeKey(syncproto.EntityMilestones, 20),
		changeKey(syncproto.EntityTasks, 30),
		changeKey(syncproto.EntityTasks, 31),
		changeKey(syncproto.EntityReports, 40),
	} {
		if row, ok := rows[key]; !ok || !row.Deleted {
			t.Errorf("%s = %+v, want a tombstone", key, row)
		}
	}
	if row := rows[changeKey(syncproto.EntityProjects, 2)]; row.Deleted {
		t.Errorf("sub-project was deleted: %+v", row)
	} else if row.Data["parent_id"] != nil {
		t.Errorf("sub-project parent_id = %v, want NULL", row.Data["parent_id"])
	}
	if row := rows[changeKey(syncproto.EntityProjects, 3)]; row.Deleted {
		t.Errorf("unrelated project was deleted: %+v", row)
	}
	if row := rows[changeKey(syncproto.EntityCategories, 10)]; row.Deleted {
		t.Errorf("unrelated category was deleted: %+v", row)
	}
}

// TestS2a11PutReferences checks the put side: a put that points at a deleted category has the
// column cleared, and a put that points at a deleted project becomes a delete.
func TestS2a11PutReferences(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	code, _ := h.push(t, aliceToken, dev.DeviceID,
		syncproto.Mutation{ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"name": "doomed", "created_at": "t", "updated_at": "t"}},
		syncproto.Mutation{ID: 2, Entity: syncproto.EntityCategories, RowID: 10, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"name": "Reading", "icon": "x", "color": "blue", "position": 3, "created_at": "t"}},
		syncproto.Mutation{ID: 3, Entity: syncproto.EntityTasks, RowID: 30, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"title": "one", "source": "user", "created_at": "t", "updated_at": "t"}},
		syncproto.Mutation{ID: 4, Entity: syncproto.EntityTasks, RowID: 31, Op: syncproto.OpPut, Full: true,
			Data: map[string]any{"title": "two", "source": "user", "created_at": "t", "updated_at": "t"}},
	)
	requireStatus(t, "snapshot", code, http.StatusOK)
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, dev.DeviceID), http.StatusOK)
	code, _ = h.push(t, aliceToken, dev.DeviceID,
		syncproto.Mutation{ID: 5, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpDelete},
		syncproto.Mutation{ID: 6, Entity: syncproto.EntityCategories, RowID: 10, Op: syncproto.OpDelete},
	)
	requireStatus(t, "delete target", code, http.StatusOK)

	code, block := h.ids(t, aliceToken, dev.DeviceID, syncproto.EntityTasks, 5)
	requireStatus(t, "lease task ids", code, http.StatusOK)
	orphan := block.Start
	code, _ = h.push(t, aliceToken, dev.DeviceID,
		syncproto.Mutation{ID: 7, Entity: syncproto.EntityTasks, RowID: 30, Op: syncproto.OpPut,
			Data: map[string]any{"category_id": 10}},
		syncproto.Mutation{ID: 8, Entity: syncproto.EntityTasks, RowID: orphan, Op: syncproto.OpPut,
			Data: map[string]any{"title": "orphan", "project_id": 1}},
	)
	requireStatus(t, "put with dead references", code, http.StatusOK)

	code, pulled := h.pull(t, aliceToken, dev.DeviceID, 0, syncproto.MaxPullLimit)
	requireStatus(t, "pull", code, http.StatusOK)
	rows := lastChanges(pulled.Changes)
	if row := rows[changeKey(syncproto.EntityTasks, 30)]; row.Deleted {
		t.Errorf("task 30 was deleted: %+v", row)
	} else if row.Data["category_id"] != nil {
		t.Errorf("task 30 category_id = %v, want NULL", row.Data["category_id"])
	}
	if row, ok := rows[changeKey(syncproto.EntityTasks, orphan)]; !ok || !row.Deleted {
		t.Errorf("task from a deleted project = %+v, want a tombstone", row)
	}
}

// TestS2a12FullPutRevivesTombstone checks that a partial put on a tombstone is ignored, and a full
// put brings the row back with its data.
func TestS2a12FullPutRevivesTombstone(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	code, _ := h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 30, Op: syncproto.OpPut, Full: true,
		Data: map[string]any{"title": "original", "source": "user", "created_at": "t", "updated_at": "t"},
	})
	requireStatus(t, "snapshot", code, http.StatusOK)
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, dev.DeviceID), http.StatusOK)
	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityTasks, RowID: 30, Op: syncproto.OpDelete,
	})
	requireStatus(t, "delete", code, http.StatusOK)
	code, ignored := h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 3, Entity: syncproto.EntityTasks, RowID: 30, Op: syncproto.OpPut,
		Data: map[string]any{"title": "edited"},
	})
	requireStatus(t, "partial put on a tombstone", code, http.StatusOK)
	if ignored.LastMutationID != 3 {
		t.Errorf("last_mutation_id after the ignored put = %d, want 3", ignored.LastMutationID)
	}
	code, pulled := h.pull(t, aliceToken, dev.DeviceID, 0, syncproto.MaxPullLimit)
	requireStatus(t, "pull after ignored put", code, http.StatusOK)
	if row := lastChanges(pulled.Changes)[changeKey(syncproto.EntityTasks, 30)]; !row.Deleted {
		t.Errorf("partial put revived the tombstone: %+v", row)
	}

	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 4, Entity: syncproto.EntityTasks, RowID: 30, Op: syncproto.OpPut, Full: true,
		Data: map[string]any{"title": "revived", "source": "user", "created_at": "t", "updated_at": "t"},
	})
	requireStatus(t, "full put on a tombstone", code, http.StatusOK)
	code, pulled = h.pull(t, aliceToken, dev.DeviceID, 0, syncproto.MaxPullLimit)
	requireStatus(t, "pull after full put", code, http.StatusOK)
	row := lastChanges(pulled.Changes)[changeKey(syncproto.EntityTasks, 30)]
	if row.Deleted {
		t.Fatalf("full put did not revive the tombstone: %+v", row)
	}
	if row.Data["title"] != "revived" {
		t.Errorf("revived title = %v, want revived", row.Data["title"])
	}
}
