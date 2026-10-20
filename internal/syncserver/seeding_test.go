package syncserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// TestSeedingManyPushes is the first seeding test: one device uploads 2500 new rows in three
// pushes, including an id above 2^53, and the block leased after seeding starts above them all.
func TestSeedingManyPushes(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	if !dev.Seeding {
		t.Fatal("the first device on an empty workspace is not seeding")
	}

	const huge = int64(9007199254740993)
	nextMutation := int64(0)
	nextRow := int64(0)
	for _, count := range []int{1000, 1000, 500} {
		mutations := make([]syncproto.Mutation, 0, count)
		for i := 0; i < count; i++ {
			nextMutation++
			nextRow++
			rowID := nextRow
			if count == 500 && i == count-1 {
				rowID = huge
			}
			mutations = append(mutations, syncproto.Mutation{
				ID: nextMutation, Entity: syncproto.EntityTasks, RowID: rowID, Op: syncproto.OpPut,
				Data: map[string]any{"title": "snapshot row"},
			})
		}
		code, pushed := h.push(t, aliceToken, dev.DeviceID, mutations...)
		requireStatus(t, fmt.Sprintf("seeding push %d", nextMutation), code, http.StatusOK)
		if pushed.LastMutationID != nextMutation {
			t.Fatalf("last_mutation_id = %d, want %d", pushed.LastMutationID, nextMutation)
		}
	}
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, dev.DeviceID), http.StatusOK)

	code, block := h.ids(t, aliceToken, dev.DeviceID, syncproto.EntityTasks, 5)
	requireStatus(t, "id lease after seeding", code, http.StatusOK)
	if block.Start <= huge {
		t.Fatalf("block starts at %d, want above the largest snapshot id %d", block.Start, huge)
	}
	if block.End-block.Start != 5 {
		t.Fatalf("block = [%d,%d), want 5 ids", block.Start, block.End)
	}
}

// TestSeedingBlocksOtherDevices is the second seeding test: while a device seeds, another device's
// push, pull and id lease are 409, and they work again after the seeding device ends it.
func TestSeedingBlocksOtherDevices(t *testing.T) {
	h := newHarness(t)
	a := h.device(t, aliceToken, "old")
	b := h.resumingDevice(t, aliceToken, "new")
	if !a.Seeding || !b.Seeding {
		t.Fatalf("devices seeding flags = %v/%v, want both true", a.Seeding, b.Seeding)
	}

	code, _ := h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"title": "blocked"},
	})
	requireStatus(t, "non-seeding push", code, http.StatusConflict)
	code, _ = h.pull(t, aliceToken, a.DeviceID, 0, 100)
	requireStatus(t, "non-seeding pull", code, http.StatusConflict)
	code, _ = h.ids(t, aliceToken, a.DeviceID, syncproto.EntityTasks, 1)
	requireStatus(t, "non-seeding ids", code, http.StatusConflict)

	code, _ = h.push(t, aliceToken, b.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"title": "seeded"},
	})
	requireStatus(t, "seeding push", code, http.StatusOK)
	requireStatus(t, "end seeding", h.seeded(t, aliceToken, b.DeviceID), http.StatusOK)

	code, pulled := h.pull(t, aliceToken, a.DeviceID, 0, 100)
	requireStatus(t, "pull after seeding", code, http.StatusOK)
	findChange(t, pulled.Changes, syncproto.EntityTasks, 1)

	code, block := h.ids(t, aliceToken, a.DeviceID, syncproto.EntityTasks, 1)
	requireStatus(t, "ids after seeding", code, http.StatusOK)
	code, _ = h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: block.Start, Op: syncproto.OpPut,
		Data: map[string]any{"title": "after"},
	})
	requireStatus(t, "push after seeding", code, http.StatusOK)
}

// TestSeedingSeededForbidden is the third seeding test: only the device that is actually seeding
// may call POST /v1/devices/seeded.
func TestSeedingSeededForbidden(t *testing.T) {
	h := newHarness(t)
	a := h.device(t, aliceToken, "old")
	b := h.resumingDevice(t, aliceToken, "new")
	requireStatus(t, "old device ends seeding", h.seeded(t, aliceToken, a.DeviceID), http.StatusForbidden)
	requireStatus(t, "seeding device ends seeding", h.seeded(t, aliceToken, b.DeviceID), http.StatusOK)
	requireStatus(t, "ended device ends seeding again", h.seeded(t, aliceToken, b.DeviceID), http.StatusForbidden)

	other := h.device(t, bobToken, "bob-phone")
	requireStatus(t, "another user's device", h.seeded(t, aliceToken, other.DeviceID), http.StatusForbidden)
}

// TestSeedingTakeover is the fourth seeding test: a registration with resume_seeding while a
// device seeds takes the seeding over, and the old device waits until the new one is done. A
// registration without it is 409.
func TestSeedingTakeover(t *testing.T) {
	h := newHarness(t)
	a := h.device(t, aliceToken, "old")
	if !a.Seeding {
		t.Fatal("first device is not seeding")
	}
	code, _ := h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 5, Op: syncproto.OpPut,
		Data: map[string]any{"title": "partial snapshot"},
	})
	requireStatus(t, "partial snapshot", code, http.StatusOK)

	// Without resume_seeding a registration does not take the seeding over.
	var plain syncproto.CreateDeviceResponse
	requireStatus(t, "registration without resume_seeding",
		h.post(t, "/v1/devices", aliceToken,
			syncproto.CreateDeviceRequest{Name: "plain"}, &plain), http.StatusConflict)

	b := h.resumingDevice(t, aliceToken, "new")
	if !b.Seeding || b.Empty {
		t.Fatalf("takeover device = %+v, want seeding in a non-empty workspace", b)
	}
	requireStatus(t, "old device ends seeding", h.seeded(t, aliceToken, a.DeviceID), http.StatusForbidden)
	code, _ = h.push(t, aliceToken, a.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityTasks, RowID: 6, Op: syncproto.OpPut,
		Data: map[string]any{"title": "old device"},
	})
	requireStatus(t, "old device push during takeover", code, http.StatusConflict)

	code, _ = h.push(t, aliceToken, b.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityTasks, RowID: 500, Op: syncproto.OpPut,
		Data: map[string]any{"title": "takeover snapshot"},
	})
	requireStatus(t, "takeover device arbitrary id", code, http.StatusOK)
	requireStatus(t, "takeover device ends seeding", h.seeded(t, aliceToken, b.DeviceID), http.StatusOK)

	code, block := h.ids(t, aliceToken, a.DeviceID, syncproto.EntityTasks, 1)
	requireStatus(t, "ids after takeover", code, http.StatusOK)
	if block.Start <= 500 {
		t.Fatalf("block starts at %d, want above the takeover snapshot id 500", block.Start)
	}
}

// TestJSONNumberPrecision is the fifth seeding test: a JSON number above 2^53 survives a push
// merge and the next pull unchanged.
func TestJSONNumberPrecision(t *testing.T) {
	h := newHarness(t)
	dev := h.device(t, aliceToken, "laptop")
	const big = "9007199254740993"

	code, _ := h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 1, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"parent_id": json.Number(big), "name": "first"},
	})
	requireStatus(t, "push big number", code, http.StatusOK)
	code, _ = h.push(t, aliceToken, dev.DeviceID, syncproto.Mutation{
		ID: 2, Entity: syncproto.EntityProjects, RowID: 1, Op: syncproto.OpPut,
		Data: map[string]any{"name": "second"},
	})
	requireStatus(t, "merge big number", code, http.StatusOK)

	resp, body := h.do(t, http.MethodGet,
		fmt.Sprintf("/v1/pull?device_id=%s&cookie=0&limit=100", dev.DeviceID), aliceToken, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pull = %d, want 200", resp.StatusCode)
	}
	if !bytes.Contains(body, []byte(`"parent_id":9007199254740993`)) {
		t.Fatalf("pull body lost the number: %s", body)
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var pulled syncproto.PullResponse
	if err := dec.Decode(&pulled); err != nil {
		t.Fatal(err)
	}
	row := findChange(t, pulled.Changes, syncproto.EntityProjects, 1)
	got, ok := row.Data["parent_id"].(json.Number)
	if !ok || got.String() != big {
		t.Fatalf("parent_id = %v (%T), want json.Number %s", row.Data["parent_id"], row.Data["parent_id"], big)
	}
	if row.Data["name"] != "second" {
		t.Fatalf("name = %v, want second", row.Data["name"])
	}
}
