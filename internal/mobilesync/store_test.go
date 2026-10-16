package mobilesync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var mutationSequence atomic.Int64

const stamp = "2026-10-16T00:00:00.000Z"

func uid(n int64) string { return fmt.Sprintf("%08x-0000-4000-8000-%012x", n, n) }

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.Context(), filepath.Join(t.TempDir(), "sync.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func change(kind, id string, version int64, data any) Mutation {
	raw, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return Mutation{ID: uid(100000 + mutationSequence.Add(1)), Type: kind, EntityID: id, BaseVersion: version, Data: raw}
}

func sampleProject(n int64) project {
	return project{ID: uid(n), Name: "Project", CreatedAt: stamp, UpdatedAt: stamp}
}

func sampleTask(n int64) task {
	return task{ID: uid(n), Title: "Task", Status: "todo", Priority: "P2", Source: "user", CreatedAt: stamp, UpdatedAt: stamp}
}

func push(t *testing.T, s *Store, mutations ...Mutation) []PushResult {
	t.Helper()
	out, err := s.Push(t.Context(), PushRequest{DeviceID: uid(999999), Mutations: mutations})
	if err != nil {
		t.Fatal(err)
	}
	return out.Results
}

func wantStatus(t *testing.T, result PushResult, status, code string) {
	t.Helper()
	if result.Status != status || result.Error != code {
		t.Fatalf("got %+v, want %s %q", result, status, code)
	}
}

func TestLostAcknowledgementRetryAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.db")
	s, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	m := change("task", uid(1), 0, sampleTask(1))
	first := push(t, s, m)[0]
	wantStatus(t, first, "accepted", "")
	if first.Record.Version != 1 {
		t.Fatal(first.Record)
	}
	// The client never received first, then retries after the service restarts.
	before, err := s.Snapshot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	retry := push(t, s, m)[0]
	if !reflect.DeepEqual(first, retry) {
		t.Fatalf("retry changed receipt: first=%+v retry=%+v", first, retry)
	}
	after, err := s.Snapshot(t.Context())
	if err != nil || !reflect.DeepEqual(before, after) || after.Cursor != 1 {
		t.Fatalf("durability or duplicate failure: before=%+v after=%+v err=%v", before, after, err)
	}
	changed := sampleTask(1)
	changed.Title = "A different payload using the same mutation ID"
	m.Data, _ = json.Marshal(changed)
	wantStatus(t, push(t, s, m)[0], "rejected", "mutation_payload_mismatch")
}

func TestConflictsDoNotBlockIndependentMutations(t *testing.T) {
	s := openTestStore(t)
	wantStatus(t, push(t, s, change("task", uid(1), 0, sampleTask(1)))[0], "accepted", "")
	stale := change("task", uid(1), 0, sampleTask(1))
	results := push(t, s, stale, change("task", uid(2), 0, sampleTask(2)))
	wantStatus(t, results[0], "conflict", "")
	if results[0].Record == nil || results[0].Record.Version != 1 {
		t.Fatal(results[0])
	}
	wantStatus(t, results[1], "accepted", "")
	newer := sampleTask(1)
	newer.Title = "Newest"
	wantStatus(t, push(t, s, change("task", uid(1), 1, newer))[0], "accepted", "")
	// A conflict receipt is stable even if the entity later changes again.
	if got := push(t, s, stale)[0]; !reflect.DeepEqual(got, results[0]) {
		t.Fatalf("conflict retry changed: %+v", got)
	}
}

func TestPullPaginationSnapshotAndTombstones(t *testing.T) {
	s := openTestStore(t)
	push(t, s, change("task", uid(1), 0, sampleTask(1)), change("task", uid(2), 0, sampleTask(2)))
	push(t, s, change("task", uid(1), 1, nil))
	first, err := s.Pull(t.Context(), 0, 2, "")
	if err != nil || len(first.Changes) != 2 || first.Cursor != 2 || !first.HasMore {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	last, err := s.Pull(t.Context(), first.Cursor, 2, first.Epoch)
	if err != nil || len(last.Changes) != 1 || last.Cursor != 3 || last.HasMore || !last.Changes[0].Record.Deleted || string(last.Changes[0].Record.Data) != "null" {
		t.Fatalf("last=%+v err=%v", last, err)
	}
	snapshot, err := s.Snapshot(t.Context())
	if err != nil || snapshot.Cursor != 3 || len(snapshot.Records) != 2 || snapshot.Epoch != first.Epoch {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	if !snapshot.Records[0].Deleted {
		t.Fatal("snapshot omitted tombstone")
	}
	for _, input := range []struct {
		cursor int64
		epoch  string
	}{{4, ""}, {0, "different-epoch"}} {
		if _, err := s.Pull(t.Context(), input.cursor, 2, input.epoch); !errors.Is(err, ErrResnapshot) {
			t.Fatalf("wanted resnapshot, got %v", err)
		}
	}
	// An offline stale edit cannot resurrect a deleted task.
	got := push(t, s, change("task", uid(1), 1, sampleTask(1)))[0]
	wantStatus(t, got, "conflict", "")
	if !got.Record.Deleted {
		t.Fatal(got)
	}
	// Explicitly resolving against the tombstone version is a new restore.
	wantStatus(t, push(t, s, change("task", uid(1), 2, sampleTask(1)))[0], "accepted", "")
}

func TestRelationshipValidationAndChildFirstDeletion(t *testing.T) {
	s := openTestStore(t)
	p1, p2 := sampleProject(1), sampleProject(2)
	push(t, s, change("project", p1.ID, 0, p1), change("project", p2.ID, 0, p2))
	parent, child := sampleTask(3), sampleTask(4)
	parent.ProjectID, child.ProjectID, child.ParentID = &p1.ID, &p1.ID, &parent.ID
	push(t, s, change("task", parent.ID, 0, parent), change("task", child.ID, 0, child))
	bad := sampleTask(5)
	bad.ParentID, bad.ProjectID = &parent.ID, &p2.ID
	wantStatus(t, push(t, s, change("task", bad.ID, 0, bad))[0], "rejected", "parent_project_mismatch")
	parent.ParentID = &child.ID
	wantStatus(t, push(t, s, change("task", parent.ID, 1, parent))[0], "rejected", "invalid_parent")
	parent.ParentID, parent.ProjectID = nil, &p2.ID
	wantStatus(t, push(t, s, change("task", parent.ID, 1, parent))[0], "rejected", "child_project_mismatch")
	wantStatus(t, push(t, s, change("project", p1.ID, 1, nil))[0], "rejected", "dependent_tasks_exist")
	wantStatus(t, push(t, s, change("task", parent.ID, 1, nil))[0], "rejected", "dependent_tasks_exist")
	results := push(t, s, change("task", child.ID, 1, nil), change("task", parent.ID, 1, nil), change("project", p1.ID, 1, nil))
	for _, result := range results {
		wantStatus(t, result, "accepted", "")
	}
	bad.ProjectID, bad.ParentID = &p1.ID, nil
	wantStatus(t, push(t, s, change("task", bad.ID, 0, bad))[0], "rejected", "project_not_found")
	bad.ProjectID, bad.ParentID = nil, &parent.ID
	wantStatus(t, push(t, s, change("task", bad.ID, 0, bad))[0], "rejected", "parent_not_found")
}

func TestReportsAreAppendOnlyAndSurviveTaskDeletion(t *testing.T) {
	s := openTestStore(t)
	task := sampleTask(1)
	push(t, s, change("task", task.ID, 0, task))
	r := report{ID: uid(2), TaskID: task.ID, Agent: "codex", Summary: "Finished a step", CreatedAt: stamp}
	m := change("report", r.ID, 0, r)
	wantStatus(t, push(t, s, m)[0], "accepted", "")
	wantStatus(t, push(t, s, change("report", r.ID, 1, r))[0], "rejected", "reports_are_append_only")
	wantStatus(t, push(t, s, change("report", r.ID, 1, nil))[0], "rejected", "reports_are_append_only")
	wantStatus(t, push(t, s, change("task", task.ID, 1, nil))[0], "accepted", "")
	snap, err := s.Snapshot(t.Context())
	if err != nil || len(snap.Records) != 2 || snap.Records[0].Type != "report" || snap.Records[0].Deleted {
		t.Fatalf("report lost: %+v %v", snap, err)
	}
	r.ID = uid(3)
	wantStatus(t, push(t, s, change("report", r.ID, 0, r))[0], "rejected", "task_not_found")
}

func TestRejectInvalidEntityWithoutChangingCursor(t *testing.T) {
	s := openTestStore(t)
	valid := sampleTask(1)
	for _, modify := range []func(*task){
		func(t *task) { t.Title = " " },
		func(t *task) { t.Status = "almost-done" },
		func(t *task) { t.Priority = "high" },
		func(t *task) { t.Source = "system" },
		func(t *task) { t.CreatedAt = "today" },
		func(t *task) { d := "tomorrow"; t.DueAt = &d },
		func(t *task) { bad := "42"; t.ProjectID = &bad },
		func(t *task) { t.ID = uid(2) },
	} {
		bad := valid
		modify(&bad)
		wantStatus(t, push(t, s, change("task", valid.ID, 0, bad))[0], "rejected", "invalid_task")
	}
	m := change("task", valid.ID, -1, valid)
	wantStatus(t, push(t, s, m)[0], "rejected", "invalid_mutation")
	unknown := change("unknown", valid.ID, 0, valid)
	wantStatus(t, push(t, s, unknown)[0], "rejected", "invalid_mutation")
	m = change("task", valid.ID, 0, valid)
	m.Data = json.RawMessage(`{"id":"` + valid.ID + `","unexpected":true}`)
	wantStatus(t, push(t, s, m)[0], "rejected", "invalid_task")
	var fields map[string]any
	_ = json.Unmarshal(change("task", valid.ID, 0, valid).Data, &fields)
	fields["notes"] = nil
	wantStatus(t, push(t, s, change("task", valid.ID, 0, fields))[0], "rejected", "invalid_task")
	fields["notes"] = ""
	delete(fields, "parentId")
	wantStatus(t, push(t, s, change("task", valid.ID, 0, fields))[0], "rejected", "invalid_task")
	snap, err := s.Snapshot(t.Context())
	if err != nil || snap.Cursor != 0 || len(snap.Records) != 0 {
		t.Fatalf("invalid input wrote records: %+v %v", snap, err)
	}
	push(t, s, change("task", valid.ID, 0, valid))
	valid.CreatedAt = "2020-01-01T00:00:00Z"
	wantStatus(t, push(t, s, change("task", valid.ID, 1, valid))[0], "rejected", "created_at_is_immutable")
}

func TestConcurrentWritersAndConsistentSnapshots(t *testing.T) {
	s := openTestStore(t)
	push(t, s, change("task", uid(1), 0, sampleTask(1)))
	const count = 30
	var wg sync.WaitGroup
	results := make(chan PushResult, count)
	errs := make(chan error, count*2)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task := sampleTask(1)
			task.Title = fmt.Sprintf("writer %d", i)
			out, err := s.Push(context.Background(), PushRequest{DeviceID: uid(int64(9000 + i)), Mutations: []Mutation{change("task", task.ID, 1, task)}})
			if err != nil {
				errs <- err
				return
			}
			results <- out.Results[0]
			snap, err := s.Snapshot(context.Background())
			if err != nil {
				errs <- err
				return
			}
			if len(snap.Records) != 1 || snap.Records[0].Version != snap.Cursor {
				errs <- fmt.Errorf("inconsistent snapshot: %+v", snap)
			}
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	accepted, conflicts := 0, 0
	for r := range results {
		switch r.Status {
		case "accepted":
			accepted++
		case "conflict":
			conflicts++
		default:
			t.Errorf("unexpected result: %+v", r)
		}
	}
	if accepted != 1 || conflicts != count-1 {
		t.Fatalf("accepted=%d conflicts=%d", accepted, conflicts)
	}
}

func TestRefuseDesktopDatabase(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.db.ExecContext(t.Context(), "CREATE TABLE projects(id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	var file string
	var seq int
	var schema string
	if err := s.db.QueryRowContext(t.Context(), "PRAGMA database_list").Scan(&seq, &schema, &file); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if other, err := Open(t.Context(), file); err == nil {
		_ = other.Close()
		t.Fatal("accepted a database with desktop tables")
	}
}

func TestResetEpochForBackupRestore(t *testing.T) {
	s := openTestStore(t)
	m := change("task", uid(1), 0, sampleTask(1))
	first := push(t, s, m)[0]
	before, err := s.Snapshot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ResetEpoch(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, err := s.Snapshot(t.Context())
	if err != nil || before.Epoch == after.Epoch || before.Cursor != after.Cursor || !reflect.DeepEqual(before.Records, after.Records) {
		t.Fatalf("unexpected epoch reset: before=%+v after=%+v err=%v", before, after, err)
	}
	if _, err := s.Pull(t.Context(), before.Cursor, 10, before.Epoch); !errors.Is(err, ErrResnapshot) {
		t.Fatalf("old device was not told to resnapshot: %v", err)
	}
	if retry := push(t, s, m)[0]; !reflect.DeepEqual(first, retry) {
		t.Fatal("epoch reset changed idempotent receipts")
	}
}

func TestSnapshotSizeBound(t *testing.T) {
	s := openTestStore(t)
	// A private owner can eventually outgrow this initial bounded snapshot API.
	// Fail explicitly instead of allocating an unbounded response or truncating.
	for n := int64(1); n <= 170; n++ {
		task := sampleTask(n)
		task.Notes = strings.Repeat("n", 50000)
		wantStatus(t, push(t, s, change("task", task.ID, 0, task))[0], "accepted", "")
	}
	if _, err := s.Snapshot(t.Context()); !errors.Is(err, ErrSnapshotTooLarge) {
		t.Fatalf("wanted bounded snapshot failure, got %v", err)
	}
	// Existing devices can still consume the bounded change pages.
	page, err := s.Pull(t.Context(), 0, 5, "")
	if err != nil || len(page.Changes) != 5 || !page.HasMore {
		t.Fatalf("pull failed after snapshot cap: %+v %v", page, err)
	}
}
