package syncserver_test

import (
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// TestS2a1Isolation is acceptance criterion 1: two users' devices never see or change each
// other's rows, another user's device id is 403, and a missing or unknown token is 401.
func TestS2a1Isolation(t *testing.T) {
	h := newHarness(t)
	alice := h.device(t, aliceToken, "alice-laptop")
	bob := h.device(t, bobToken, "bob-phone")

	var out syncproto.CreateDeviceResponse
	requireStatus(t, "POST /v1/devices without a token",
		h.post(t, "/v1/devices", "", syncproto.CreateDeviceRequest{Name: "x"}, &out), http.StatusUnauthorized)
	requireStatus(t, "POST /v1/devices with an unknown token",
		h.post(t, "/v1/devices", "not-a-token", syncproto.CreateDeviceRequest{Name: "x"}, &out), http.StatusUnauthorized)

	code, _ := h.push(t, aliceToken, alice.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "alice-project"},
	})
	requireStatus(t, "alice's snapshot push", code, http.StatusOK)

	_, bobPull := h.pull(t, bobToken, bob.DeviceID, 0, 100)
	if len(bobPull.Changes) != 0 {
		t.Fatalf("bob sees alice's changes: %+v", bobPull.Changes)
	}

	code, _ = h.push(t, bobToken, alice.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 2, Op: syncproto.OpPut,
		Data: map[string]any{"name": "bob-writes-alice"},
	})
	requireStatus(t, "bob pushing alice's device", code, http.StatusForbidden)

	code, _ = h.push(t, bobToken, bob.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "bob-project"},
	})
	requireStatus(t, "bob's snapshot push", code, http.StatusOK)

	_, alicePull := h.pull(t, aliceToken, alice.DeviceID, 0, 100)
	if len(alicePull.Changes) != 1 {
		t.Fatalf("alice sees %d changes, want only her own: %+v", len(alicePull.Changes), alicePull.Changes)
	}
	if got := findChange(t, alicePull.Changes, syncproto.EntityProjects, 1).Data["name"]; got != "alice-project" {
		t.Fatalf("alice's project name = %v, want alice-project", got)
	}
}

// TestS2a2PushIdempotency is acceptance criterion 2: the same push sent twice applies once, a
// lower or equal mutation id is skipped, and falling ids in one request change nothing.
func TestS2a2PushIdempotency(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	snapshot := syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 100, Op: syncproto.OpPut,
		Data: map[string]any{"title": "one"},
	}
	code, first := h.push(t, aliceToken, dev.DeviceID, snapshot)
	requireStatus(t, "first push", code, http.StatusOK)
	if first.LastMutationID != 1 {
		t.Fatalf("last_mutation_id = %d, want 1", first.LastMutationID)
	}
	code, second := h.push(t, aliceToken, dev.DeviceID, snapshot)
	requireStatus(t, "duplicate push", code, http.StatusOK)
	if second.LastMutationID != 1 {
		t.Fatalf("duplicate push last_mutation_id = %d, want 1", second.LastMutationID)
	}
	_, pulled := h.pull(t, aliceToken, dev.DeviceID, 0, 100)
	if len(pulled.Changes) != 1 || pulled.Changes[0].Version != 1 {
		t.Fatalf("duplicate push produced changes %+v, want one version 1", pulled.Changes)
	}

	// A mutation id at last_mutation_id is skipped even when its data differs.
	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 100, Op: syncproto.OpPut,
		Data: map[string]any{"title": "changed-after-ack"},
	})
	requireStatus(t, "equal mutation id", code, http.StatusOK)
	_, pulled = h.pull(t, aliceToken, dev.DeviceID, 0, 100)
	if got := findChange(t, pulled.Changes, syncproto.EntityTasks, 100).Data["title"]; got != "one" {
		t.Fatalf("title after skipped duplicate = %v, want one", got)
	}

	// Falling ids: the first would apply, the second is lower, so nothing may commit.
	code, _ = h.push(t, aliceToken, dev.DeviceID,
		syncproto.Mutation{ID: 3, Entity: syncproto.EntityTasks, RowID: 2, Op: syncproto.OpPut,
			Data: map[string]any{"title": "two"}},
		syncproto.Mutation{ID: 2, Entity: syncproto.EntityTasks, RowID: 3, Op: syncproto.OpPut,
			Data: map[string]any{"title": "three"}},
	)
	requireStatus(t, "falling mutation ids", code, http.StatusBadRequest)
	_, pulled = h.pull(t, aliceToken, dev.DeviceID, 0, 100)
	if len(pulled.Changes) != 1 || pulled.LastMutationID != 1 {
		t.Fatalf("a falling-id push changed state: %+v (last %d)", pulled.Changes, pulled.LastMutationID)
	}

	// Equal ids in one request are not rising either.
	code, _ = h.push(t, aliceToken, dev.DeviceID,
		syncproto.Mutation{ID: 3, Entity: syncproto.EntityTasks, RowID: 2, Op: syncproto.OpPut,
			Data: map[string]any{"title": "two"}},
		syncproto.Mutation{ID: 3, Entity: syncproto.EntityTasks, RowID: 3, Op: syncproto.OpPut,
			Data: map[string]any{"title": "three"}},
	)
	requireStatus(t, "equal mutation ids", code, http.StatusBadRequest)

	code, rising := h.push(t, aliceToken, dev.DeviceID,
		syncproto.Mutation{ID: 2, Entity: syncproto.EntityTasks, RowID: 2, Op: syncproto.OpPut,
			Data: map[string]any{"title": "two"}},
		syncproto.Mutation{ID: 3, Entity: syncproto.EntityTasks, RowID: 3, Op: syncproto.OpPut,
			Data: map[string]any{"title": "three"}},
	)
	requireStatus(t, "rising mutation ids", code, http.StatusOK)
	if rising.LastMutationID != 3 {
		t.Fatalf("last_mutation_id = %d, want 3", rising.LastMutationID)
	}
	_, pulled = h.pull(t, aliceToken, dev.DeviceID, 1, 100)
	if len(pulled.Changes) != 2 || pulled.Changes[0].Version != 2 || pulled.Changes[1].Version != 3 {
		t.Fatalf("changes after rising push = %+v, want versions 2 and 3", pulled.Changes)
	}
}

// TestS2a3ColumnMerge is acceptance criterion 3: two devices put different columns of one row and
// both survive; a later put of the same column wins.
func TestS2a3ColumnMerge(t *testing.T) {
	h := newHarness(t)
	a := h.device(t, aliceToken, "laptop")
	code, _ := h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "project", "repo_slug": "owner/repo"},
	})
	requireStatus(t, "snapshot push", code, http.StatusOK)
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, a.DeviceID), http.StatusOK)
	b := h.device(t, aliceToken, "phone")

	code, _ = h.push(t, aliceToken, b.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"archived_at": "2026-10-19T00:00:00.000Z"},
	})
	requireStatus(t, "second device's column", code, http.StatusOK)

	_, pulled := h.pull(t, aliceToken, a.DeviceID, 0, 100)
	row := findChange(t, pulled.Changes, syncproto.EntityProjects, 1)
	for column, want := range map[string]string{
		"name": "project", "repo_slug": "owner/repo", "archived_at": "2026-10-19T00:00:00.000Z",
	} {
		if got := row.Data[column]; got != want {
			t.Errorf("merged %s = %v, want %v", column, got, want)
		}
	}

	code, _ = h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "first"},
	})
	requireStatus(t, "first name", code, http.StatusOK)
	code, _ = h.push(t, aliceToken, b.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "second"},
	})
	requireStatus(t, "later name", code, http.StatusOK)
	_, pulled = h.pull(t, aliceToken, a.DeviceID, 0, 100)
	if got := findChange(t, pulled.Changes, syncproto.EntityProjects, 1).Data["name"]; got != "second" {
		t.Fatalf("name = %v, want the later push's second", got)
	}
}

// TestS2a4DeleteWins is acceptance criterion 4: a put after a delete leaves the row deleted.
func TestS2a4DeleteWins(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	code, _ := h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "doomed"},
	})
	requireStatus(t, "put", code, http.StatusOK)
	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpDelete,
	})
	requireStatus(t, "delete", code, http.StatusOK)
	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 3, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "back-from-the-dead"},
	})
	requireStatus(t, "put after delete", code, http.StatusOK)

	_, pulled := h.pull(t, aliceToken, dev.DeviceID, 0, 100)
	if len(pulled.Changes) != 1 {
		t.Fatalf("changes = %+v, want only the tombstone", pulled.Changes)
	}
	row := pulled.Changes[0]
	if !row.Deleted || row.Data != nil || row.Version != 2 {
		t.Fatalf("row after put on tombstone = %+v, want deleted nil data version 2", row)
	}
	if pulled.LastMutationID != 3 {
		t.Fatalf("last_mutation_id = %d, want 3", pulled.LastMutationID)
	}
}

// TestS2a5IDs is acceptance criterion 5: leased blocks never overlap, a new row outside a block
// is 409, and the snapshot push raises the counters.
func TestS2a5IDs(t *testing.T) {
	h := newHarness(t)
	a := h.device(t, aliceToken, "laptop")
	code, _ := h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 500, Op: syncproto.OpPut,
		Data: map[string]any{"title": "snapshot"},
	})
	requireStatus(t, "snapshot push", code, http.StatusOK)
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, a.DeviceID), http.StatusOK)
	b := h.device(t, aliceToken, "phone")
	code, first := h.ids(t, aliceToken, a.DeviceID, syncproto.EntityTasks, 5)
	requireStatus(t, "id lease", code, http.StatusOK)
	if first.Start < 501 {
		t.Fatalf("ids after a snapshot at 500 start at %d, want above 500", first.Start)
	}
	if first.End-first.Start != 5 {
		t.Fatalf("block = [%d,%d), want 5 ids", first.Start, first.End)
	}

	type block struct{ start, end int64 }
	const requests = 20
	blocks := make(chan block, requests)
	errs := make(chan error, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			token, device := aliceToken, a.DeviceID
			if i%2 == 1 {
				device = b.DeviceID
			}
			got, err := h.idsE(token, device, syncproto.EntityTasks, 3)
			if err != nil {
				errs <- err
				return
			}
			blocks <- block{got.Start, got.End}
		}(i)
	}
	wg.Wait()
	close(blocks)
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	all := make([]block, 0, requests)
	for got := range blocks {
		all = append(all, got)
	}
	if len(all) != requests {
		t.Fatalf("got %d leases, want %d", len(all), requests)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].start < all[j].start })
	for i, got := range all {
		if got.end-got.start != 3 {
			t.Errorf("block %d = [%d,%d), want 3 ids", i, got.start, got.end)
		}
		if i > 0 && got.start < all[i-1].end {
			t.Errorf("blocks overlap: [%d,%d) and [%d,%d)", all[i-1].start, all[i-1].end, got.start, got.end)
		}
	}

	// The workspace has rows, so a fresh row above the counters is refused.
	code, _ = h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityTasks, RowID: 100000, Op: syncproto.OpPut,
		Data: map[string]any{"title": "unleased"},
	})
	requireStatus(t, "put outside every block", code, http.StatusConflict)

	// A row inside a leased block is accepted.
	code, _ = h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityTasks, RowID: first.Start, Op: syncproto.OpPut,
		Data: map[string]any{"title": "leased"},
	})
	requireStatus(t, "put inside a block", code, http.StatusOK)
}

// TestS2a6Pull is acceptance criterion 6: paging with limit returns every change once, in version
// order, with a correct more; a cookie from the future is 409; last_mutation_id matches the push.
func TestS2a6Pull(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	code, empty := h.pull(t, aliceToken, dev.DeviceID, 0, 2)
	requireStatus(t, "empty pull", code, http.StatusOK)
	if len(empty.Changes) != 0 || empty.Cookie != 0 || empty.More || empty.LastMutationID != 0 {
		t.Fatalf("empty pull = %+v", empty)
	}

	mutations := make([]syncproto.Mutation, 0, 5)
	for i := int64(1); i <= 5; i++ {
		mutations = append(mutations, syncproto.Mutation{
			ID: i, Entity: syncproto.EntityTasks, RowID: i, Op: syncproto.OpPut,
			Data: map[string]any{"title": "task"},
		})
	}
	code, pushed := h.push(t, aliceToken, dev.DeviceID, mutations...)
	requireStatus(t, "push", code, http.StatusOK)
	if pushed.LastMutationID != 5 {
		t.Fatalf("push last_mutation_id = %d, want 5", pushed.LastMutationID)
	}

	var (
		cookie int64
		seen   []int64
		pages  int
	)
	for {
		code, page := h.pull(t, aliceToken, dev.DeviceID, cookie, 2)
		requireStatus(t, "pull page", code, http.StatusOK)
		pages++
		if page.LastMutationID != 5 {
			t.Fatalf("pull page last_mutation_id = %d, want 5", page.LastMutationID)
		}
		for _, c := range page.Changes {
			if len(seen) > 0 && c.Version <= seen[len(seen)-1] {
				t.Fatalf("versions are not rising: %v then %d", seen, c.Version)
			}
			seen = append(seen, c.Version)
		}
		cookie = page.Cookie
		if !page.More {
			break
		}
		if pages > 5 {
			t.Fatal("pull paging does not finish")
		}
	}
	if pages != 3 {
		t.Errorf("pages = %d, want 3", pages)
	}
	if len(seen) != 5 {
		t.Fatalf("saw versions %v, want 1..5 exactly once", seen)
	}
	for i, v := range seen {
		if v != int64(i+1) {
			t.Fatalf("versions = %v, want 1..5", seen)
		}
	}

	code, _ = h.pull(t, aliceToken, dev.DeviceID, 6, 2)
	requireStatus(t, "cookie from the future", code, http.StatusConflict)

	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 6, Entity: syncproto.EntityTasks, RowID: 5, Op: syncproto.OpPut,
		Data: map[string]any{"title": "six"},
	})
	requireStatus(t, "push after paging", code, http.StatusOK)
	_, page := h.pull(t, aliceToken, dev.DeviceID, 5, 100)
	if len(page.Changes) != 1 || page.Changes[0].Version != 6 || page.LastMutationID != 6 || page.More {
		t.Fatalf("last page = %+v, want one version 6 change", page)
	}
	_, page = h.pull(t, aliceToken, dev.DeviceID, 6, 100)
	if len(page.Changes) != 0 || page.Cookie != 6 || page.More || page.LastMutationID != 6 {
		t.Fatalf("pull at the head = %+v", page)
	}
}

// TestS2a7InvalidEntityOrColumn is acceptance criterion 7's first half: an unknown entity, column
// or op is a 400 and applies nothing.
func TestS2a7InvalidEntityOrColumn(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	for name, mutation := range map[string]syncproto.Mutation{
		"unknown entity": {ID: 1, Entity: "notes", RowID: 1, Op: syncproto.OpPut,
			Data: map[string]any{"title": "x"}},
		"unknown column": {ID: 1, Entity: syncproto.EntityTasks, RowID: 1, Op: syncproto.OpPut,
			Data: map[string]any{"not_a_column": "x"}},
		"unknown op": {ID: 1, Entity: syncproto.EntityTasks, RowID: 1, Op: "patch",
			Data: map[string]any{"title": "x"}},
	} {
		code, _ := h.push(t, aliceToken, dev.DeviceID, mutation)
		requireStatus(t, name, code, http.StatusBadRequest)
	}
	_, pulled := h.pull(t, aliceToken, dev.DeviceID, 0, 100)
	if len(pulled.Changes) != 0 || pulled.LastMutationID != 0 {
		t.Fatalf("invalid pushes changed state: %+v (last %d)", pulled.Changes, pulled.LastMutationID)
	}

	// The five column lists themselves are compared with migration 014 in syncproto's test.
	if _, ok := syncproto.EntityColumns(syncproto.EntityTasks); !ok {
		t.Fatal("tasks is not a known entity")
	}
}

// TestS2a8Events is acceptance criterion 8: a push wakes the open stream of another device in the
// same workspace, and not one of another workspace.
func TestS2a8Events(t *testing.T) {
	h := newHarness(t)
	a := h.device(t, aliceToken, "laptop")
	code, _ := h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "seed"},
	})
	requireStatus(t, "snapshot push", code, http.StatusOK)
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, a.DeviceID), http.StatusOK)
	b := h.device(t, aliceToken, "phone")
	other := h.device(t, bobToken, "bob-phone")

	same := h.stream(t, aliceToken, b.DeviceID)
	foreign := h.stream(t, bobToken, other.DeviceID)
	same.waitReady(t)
	foreign.waitReady(t)

	code, _ = h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "poke"},
	})
	requireStatus(t, "push", code, http.StatusOK)
	same.waitEvent(t, "poke", 2*time.Second)
	foreign.expectNoEvent(t, 300*time.Millisecond)

	code, _ = h.push(t, bobToken, other.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "bob"},
	})
	requireStatus(t, "bob's push", code, http.StatusOK)
	foreign.waitEvent(t, "poke", 2*time.Second)
	same.expectNoEvent(t, 300*time.Millisecond)
}
