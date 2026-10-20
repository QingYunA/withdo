package syncclient_test

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// toggleTransport lets a test take a device offline for one write.
type toggleTransport struct {
	base    http.RoundTripper
	offline atomic.Bool
}

func (t *toggleTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.offline.Load() {
		return nil, errors.New("test: network is down")
	}
	return t.base.RoundTrip(r)
}

// TestS2b3StableNumbers is acceptance criterion 3: a task made on B while it cannot reach the
// server takes an id from B's block and appears on A under the same id after both sync.
func TestS2b3StableNumbers(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	project := newProject(t, a.core, "alpha", nil)
	newTask(t, a.core, domain.CreateTaskInput{Title: "seed", ProjectID: &project.ID})
	a.link(t, "A")

	transport := &toggleTransport{base: http.DefaultTransport}
	b := h.newDevice(t, &http.Client{Transport: transport})
	b.link(t, "B")

	wantID := int64Value(t, b.db, "SELECT next FROM sync_id_blocks WHERE entity = 'tasks' ORDER BY next LIMIT 1")
	transport.offline.Store(true)
	task, err := b.core.CreateTask(testCtx, domain.CreateTaskInput{Title: "made offline", ProjectID: &project.ID})
	if err != nil {
		t.Fatalf("offline create: %v", err)
	}
	if task.ID != wantID {
		t.Fatalf("offline task id = %d, want the next leased id %d", task.ID, wantID)
	}
	if err := b.client.Sync(testCtx); err == nil {
		t.Fatal("sync succeeded while the network was down")
	}
	if got := int64Value(t, b.db, "SELECT COUNT(*) FROM sync_outbox WHERE entity = 'tasks'"); got == 0 {
		t.Fatal("the offline task was not queued in the outbox")
	}

	transport.offline.Store(false)
	b.sync(t)
	a.sync(t)

	got := must[*domain.Task](t)(a.core.GetTask(testCtx, wantID))
	if got == nil {
		t.Fatalf("A does not have task %d", wantID)
	}
	if got.Title != "made offline" {
		t.Errorf("A's task title = %q, want made offline", got.Title)
	}
	if got.ID != task.ID {
		t.Errorf("A's task id = %d, B's = %d", got.ID, task.ID)
	}
}

// TestS2b4Columns is acceptance criterion 4: different columns of one task changed on A and B both
// survive; the same column changed on both keeps the value of the push the server took last.
func TestS2b4Columns(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	project := newProject(t, a.core, "alpha", nil)
	task := newTask(t, a.core, domain.CreateTaskInput{Title: "one", ProjectID: &project.ID})
	a.link(t, "A")
	b := h.newDevice(t, nil)
	b.link(t, "B")

	must[*domain.Task](t)(a.core.UpdateTask(testCtx, task.ID,
		domain.TaskPatch{Title: ptr("title from A")}))
	a.sync(t)
	must[*domain.Task](t)(b.core.UpdateTask(testCtx, task.ID,
		domain.TaskPatch{Notes: domain.Set("notes from B")}))
	b.sync(t)
	a.sync(t)

	for name, d := range map[string]*device{"A": a, "B": b} {
		got := must[*domain.Task](t)(d.core.GetTask(testCtx, task.ID))
		if got == nil {
			t.Fatalf("%s lost task %d", name, task.ID)
		}
		if got.Title != "title from A" {
			t.Errorf("%s title = %q, want title from A", name, got.Title)
		}
		if got.Notes == nil || *got.Notes != "notes from B" {
			t.Errorf("%s notes = %v, want notes from B", name, got.Notes)
		}
	}

	// Both change the title. The server takes A's push first and B's later, so both end with B's.
	must[*domain.Task](t)(a.core.UpdateTask(testCtx, task.ID,
		domain.TaskPatch{Title: ptr("title A last")}))
	a.sync(t)
	must[*domain.Task](t)(b.core.UpdateTask(testCtx, task.ID,
		domain.TaskPatch{Title: ptr("title B last")}))
	b.sync(t)
	a.sync(t)
	b.sync(t)
	for name, d := range map[string]*device{"A": a, "B": b} {
		got := must[*domain.Task](t)(d.core.GetTask(testCtx, task.ID))
		if got == nil || got.Title != "title B last" {
			t.Errorf("%s title = %v, want title B last", name, got)
		}
	}
}

// TestS2b5Deletes is acceptance criterion 5: deleting a project on A removes its tasks, reports
// and milestones on B.
func TestS2b5Deletes(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	project := newProject(t, a.core, "doomed", nil)
	milestone := must[*domain.Milestone](t)(a.core.CreateMilestone(testCtx,
		domain.CreateMilestoneInput{ProjectID: project.ID, Title: "v1"}))
	inProject := newTask(t, a.core, domain.CreateTaskInput{Title: "in project", ProjectID: &project.ID})
	inMilestone := newTask(t, a.core, domain.CreateTaskInput{Title: "in milestone", MilestoneID: &milestone.ID})
	must[*domain.Report](t)(a.core.AddReport(testCtx,
		domain.AddReportInput{TaskID: inProject.ID, Agent: "agent", Summary: "one"}))
	must[*domain.Report](t)(a.core.AddReport(testCtx,
		domain.AddReportInput{TaskID: inMilestone.ID, Agent: "agent", Summary: "two"}))
	a.link(t, "A")
	b := h.newDevice(t, nil)
	b.link(t, "B")

	must[domain.Deleted](t)(a.core.DeleteProject(testCtx, project.ID))
	a.sync(t)
	b.sync(t)

	for name, d := range map[string]*device{"A": a, "B": b} {
		for _, check := range []struct {
			table string
			want  int
		}{
			{"projects", 0},
			{"milestones", 0},
			{"tasks", 0},
			{"reports", 0},
		} {
			if got := count(t, d.db, "SELECT COUNT(*) FROM "+check.table); got != check.want {
				t.Errorf("%s has %d %s, want %d", name, got, check.table, check.want)
			}
		}
	}
}

// TestS2b6JoinRefused is acceptance criterion 6: joining a workspace with local tasks is
// ErrLocalData and leaves the database unlinked and unchanged.
func TestS2b6JoinRefused(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	project := newProject(t, a.core, "alpha", nil)
	newTask(t, a.core, domain.CreateTaskInput{Title: "seed", ProjectID: &project.ID})
	a.link(t, "A")

	b := h.newDevice(t, nil)
	local := newTask(t, b.core, domain.CreateTaskInput{Title: "B's own task"})
	before := dumpTable(t, b.db, "tasks")

	err := b.client.Link(testCtx, "B")
	if !errors.Is(err, syncclient.ErrLocalData) {
		t.Fatalf("link with local data = %v, want ErrLocalData", err)
	}
	if got := str(t, b.db, "SELECT COALESCE((SELECT value FROM sync_state WHERE key = 'enabled'), '')"); got != "" {
		t.Errorf("B's enabled = %q, want unset", got)
	}
	if got := str(t, b.db, "SELECT COALESCE((SELECT value FROM sync_state WHERE key = 'device_id'), '')"); got != "" {
		t.Errorf("B's device_id = %q, want unset", got)
	}
	if got := count(t, b.db, "SELECT COUNT(*) FROM sync_outbox"); got != 0 {
		t.Errorf("B's outbox has %d entries, want 0", got)
	}
	after := dumpTable(t, b.db, "tasks")
	if !rowsEqual(after, before) {
		t.Errorf("B changed\nbefore: %s\nafter: %s", renderRows(before), renderRows(after))
	}
	if got := must[*domain.Task](t)(b.core.GetTask(testCtx, local.ID)); got == nil {
		t.Fatalf("B lost its task %d", local.ID)
	}
	// An unlinked database still assigns ids as before.
	next := newTask(t, b.core, domain.CreateTaskInput{Title: "still unlinked"})
	if next.ID != local.ID+1 {
		t.Errorf("B's next task id = %d, want %d", next.ID, local.ID+1)
	}
}

// seededGate holds the seeding device's POST /v1/devices/seeded until the test releases it, so a
// second device can try to link while the workspace is really still seeding.
type seededGate struct {
	base    http.RoundTripper
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func newSeededGate() *seededGate {
	return &seededGate{
		base:    http.DefaultTransport,
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (g *seededGate) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Path == "/v1/devices/seeded" {
		g.once.Do(func() { close(g.started) })
		<-g.release
	}
	return g.base.RoundTrip(r)
}

// TestS2b7SeedingWait is acceptance criterion 7: while A is seeding, linking B is ErrSeeding;
// after A finishes, B links.
func TestS2b7SeedingWait(t *testing.T) {
	h := newHarness(t)
	gate := newSeededGate()
	a := h.newDevice(t, &http.Client{Transport: gate})
	project := newProject(t, a.core, "alpha", nil)
	newTask(t, a.core, domain.CreateTaskInput{Title: "seed", ProjectID: &project.ID})

	done := make(chan error, 1)
	go func() { done <- a.client.Link(testCtx, "A") }()
	select {
	case <-gate.started:
	case <-time.After(15 * time.Second):
		t.Fatal("A never reached the end of its seeding")
	}

	b := h.newDevice(t, nil)
	err := b.client.Link(testCtx, "B")
	if !errors.Is(err, syncclient.ErrSeeding) {
		t.Fatalf("link while A seeds = %v, want ErrSeeding", err)
	}
	if got := str(t, b.db, "SELECT COALESCE((SELECT value FROM sync_state WHERE key = 'device_id'), '')"); got != "" {
		t.Errorf("B's device_id = %q, want unset after ErrSeeding", got)
	}
	if got := h.serverSeedingDevices(t); got != 1 {
		t.Errorf("seeding devices = %d after B's ErrSeeding, want A still seeding", got)
	}

	close(gate.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("A's link: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("A's link did not finish")
	}

	b.link(t, "B")
	assertSyncedEqual(t, a, b)
	if got := count(t, b.db, "SELECT COUNT(*) FROM projects"); got != 1 {
		t.Errorf("B has %d projects after joining, want 1", got)
	}
	if got := count(t, b.db, "SELECT COUNT(*) FROM tasks"); got != 1 {
		t.Errorf("B has %d tasks after joining, want 1", got)
	}
	if got := h.serverSeedingDevices(t); got != 0 {
		t.Errorf("seeding devices after A finished = %d, want 0", got)
	}
}
