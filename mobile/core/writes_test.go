package core

import (
	"context"
	"encoding/json"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
)

// changeRecorder counts the listener's OnChange calls.
type changeRecorder struct {
	changes atomic.Int64
	done    chan struct{}
}

func newChangeRecorder() *changeRecorder { return &changeRecorder{done: make(chan struct{}, 32)} }

func (r *changeRecorder) OnChange() {
	r.changes.Add(1)
	select {
	case r.done <- struct{}{}:
	default:
	}
}

func (r *changeRecorder) OnStatus(string) {}

// expectOne waits for one OnChange after before and checks no second one follows.
func (r *changeRecorder) expectOne(t *testing.T, before int64) {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(5 * time.Second):
		t.Fatalf("no OnChange after a write")
	}
	time.Sleep(100 * time.Millisecond)
	if got := r.changes.Load(); got != before+1 {
		t.Errorf("OnChange fired %d times for one write, want 1", got-before)
	}
}

// TestWritesMatchTheDesktop is acceptance 2: create, update, complete, accept, delete and restore
// through the phone change the database as data.Real does, and each write fires OnChange once.
func TestWritesMatchTheDesktop(t *testing.T) {
	phoneDir := demoDatabase(t, demoNow)
	desktopDir := demoDatabase(t, demoNow)
	phone, err := open(phoneDir, nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	defer func() { _ = phone.Close() }()
	desktop, err := data.OpenReal(context.Background(), desktopDir+"/adhd-todo.db", data.RealOptions{
		Now: func() time.Time { return demoNow },
	})
	if err != nil {
		t.Fatalf("open the desktop store: %v", err)
	}
	defer func() { _ = desktop.Close() }()

	recorder := newChangeRecorder()
	phone.SetListener(recorder)
	var writes int64

	// Create: the composer's plain user task.
	raw, err := phone.CreateTask(`{"title":"Write the Android core","project_id":1,"priority":"P1","category_id":2,"due_at":"2026-10-25T09:00:00Z"}`)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	writes++
	recorder.expectOne(t, writes-1)
	due := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	made, err := desktop.CreateTask(data.CreateTaskInput{
		Title: "Write the Android core", Source: data.SourceUser, ProjectID: data.IntPtr(1),
		Priority: data.P1, Category: 2, DueAt: &due,
	})
	if err != nil {
		t.Fatalf("desktop CreateTask: %v", err)
	}
	var phoneMade data.Task
	if err := json.Unmarshal([]byte(raw), &phoneMade); err != nil {
		t.Fatalf("decode the created task: %v", err)
	}
	if !reflect.DeepEqual(phoneMade, made) {
		t.Errorf("created task = %+v\nwant %+v", phoneMade, made)
	}
	assertSameSnapshot(t, phone, desktop)

	// Update: every field the plan lists, including a project move and a category change.
	title := "Port core to Go, revised"
	notes := "A note about the port."
	status := data.Blocked
	priority := data.P0
	remind := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	if err := phone.UpdateTask(1, `{"title":"Port core to Go, revised","notes":"A note about the port.","status":"blocked","priority":"P0","due_at":"2026-10-26T12:00:00Z","remind_at":"2026-10-27T08:00:00Z","category":0,"project_id":2}`); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	writes++
	recorder.expectOne(t, writes-1)
	due2 := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	project, category := 2, 0
	if _, err := desktop.UpdateTask(1, data.Patch{
		Title: &title, Notes: &notes, Status: &status, Priority: &priority,
		DueAt: &due2, RemindAt: &remind, Category: &category, ProjectID: &project,
	}); err != nil {
		t.Fatalf("desktop UpdateTask: %v", err)
	}
	assertSameSnapshot(t, phone, desktop)

	// A null clears notes, due and remind, and 0 moves the task back to the Inbox.
	if err := phone.UpdateTask(1, `{"notes":null,"due_at":null,"remind_at":null,"category":0,"project_id":0}`); err != nil {
		t.Fatalf("clearing UpdateTask: %v", err)
	}
	writes++
	recorder.expectOne(t, writes-1)
	empty, zero := "", time.Time{}
	if _, err := desktop.UpdateTask(1, data.Patch{Notes: &empty, DueAt: &zero, RemindAt: &zero, Category: &category, ProjectID: &category}); err != nil {
		t.Fatalf("desktop clearing UpdateTask: %v", err)
	}
	assertSameSnapshot(t, phone, desktop)

	// Complete and Accept are the desktop's done transition.
	if err := phone.Complete(1); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	writes++
	recorder.expectOne(t, writes-1)
	done := data.Done
	if _, err := desktop.UpdateTask(1, data.Patch{Status: &done}); err != nil {
		t.Fatalf("desktop Complete: %v", err)
	}
	assertSameSnapshot(t, phone, desktop)

	if err := phone.Accept(2); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	writes++
	recorder.expectOne(t, writes-1)
	if _, err := desktop.UpdateTask(2, data.Patch{Status: &done}); err != nil {
		t.Fatalf("desktop Accept: %v", err)
	}
	assertSameSnapshot(t, phone, desktop)

	// Delete returns the Undo snapshot; restore puts the task, its reports and its subtasks back.
	snapshot, err := phone.DeleteTask(20)
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	writes++
	recorder.expectOne(t, writes-1)
	desktopSnapshot, err := desktop.SnapshotTask(20)
	if err != nil {
		t.Fatalf("desktop SnapshotTask: %v", err)
	}
	if err := desktop.DeleteTask(20); err != nil {
		t.Fatalf("desktop DeleteTask: %v", err)
	}
	assertSameSnapshot(t, phone, desktop)

	if err := phone.RestoreTask(snapshot); err != nil {
		t.Fatalf("RestoreTask: %v", err)
	}
	writes++
	recorder.expectOne(t, writes-1)
	if _, err := desktop.RestoreTask(desktopSnapshot); err != nil {
		t.Fatalf("desktop RestoreTask: %v", err)
	}
	assertSameSnapshot(t, phone, desktop)
}

// assertSameSnapshot compares everything the desktop's pages read.
func assertSameSnapshot(t *testing.T, phone *Core, desktop *data.Real) {
	t.Helper()
	got, err := phone.real.Snapshot()
	if err != nil {
		t.Fatalf("phone snapshot: %v", err)
	}
	want, err := desktop.Snapshot()
	if err != nil {
		t.Fatalf("desktop snapshot: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the phone's database does not match the desktop's")
	}
}

// TestReplyIsTheUsersAndKeepsStaleness is acceptance 3: a reply is a report by "you" of kind
// "reply", Task(id) shows it in the activity, and it changes neither the task nor staleness.
func TestReplyIsTheUsersAndKeepsStaleness(t *testing.T) {
	phone, err := open(demoDatabase(t, demoNow), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	defer func() { _ = phone.Close() }()

	beforeTask, err := phone.dom.GetTask(context.Background(), 3) // blocked for three days: stale
	if err != nil || beforeTask == nil {
		t.Fatalf("read task 3: %v", err)
	}
	beforeMine := mineItem(t, phone, 3)
	if !beforeMine.Stale {
		t.Fatal("the fixture's blocked task is not stale; the test cannot see a reset")
	}

	if err := phone.Reply(3, "I will decide on the activation policy tomorrow."); err != nil {
		t.Fatalf("Reply: %v", err)
	}

	afterTask, err := phone.dom.GetTask(context.Background(), 3)
	if err != nil || afterTask == nil {
		t.Fatalf("read task 3 again: %v", err)
	}
	if !reflect.DeepEqual(beforeTask, afterTask) {
		t.Errorf("Reply changed the task\nbefore: %+v\nafter:  %+v", beforeTask, afterTask)
	}
	reports, err := phone.dom.ListReports(context.Background(), 3)
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	last := reports[len(reports)-1]
	if last.Agent != domain.ReplyAgent || last.Kind != domain.KindReply {
		t.Errorf("newest report = agent %q kind %q, want %q and %q", last.Agent, last.Kind, domain.ReplyAgent, domain.KindReply)
	}
	if last.Summary != "I will decide on the activation policy tomorrow." {
		t.Errorf("reply summary = %q", last.Summary)
	}

	afterMine := mineItem(t, phone, 3)
	if !reflect.DeepEqual(beforeMine, afterMine) {
		t.Errorf("Reply changed the Mine row\nbefore: %+v\nafter:  %+v", beforeMine, afterMine)
	}

	raw, err := phone.Task(3)
	if err != nil {
		t.Fatalf("Task(3): %v", err)
	}
	var page taskPage
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatalf("decode Task(3): %v", err)
	}
	found := false
	for _, report := range page.Reports {
		if report.Agent == domain.ReplyAgent && report.Summary == last.Summary {
			found = true
		}
	}
	if !found {
		t.Errorf("Task(3)'s activity does not show the reply: %+v", page.Reports)
	}
}

// mineItem returns the Mine row of a task, whichever section it is in.
func mineItem(t *testing.T, c *Core, id int) data.Item {
	t.Helper()
	raw, err := c.Mine()
	if err != nil {
		t.Fatalf("Mine: %v", err)
	}
	var mine data.Mine
	if err := json.Unmarshal([]byte(raw), &mine); err != nil {
		t.Fatalf("decode Mine: %v", err)
	}
	for _, section := range [][]data.Item{mine.FromAgents, mine.Today, mine.Next, mine.NoDate, mine.Done} {
		for _, item := range section {
			if item.ID == id {
				return item
			}
		}
	}
	t.Fatalf("task %d is not in Mine: %+v", id, mine)
	return data.Item{}
}
