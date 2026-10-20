package syncclient_test

import (
	"database/sql"
	"fmt"
	"math/rand"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// TestS2b2Convergence is acceptance criterion 2: a seeded random sequence of 300 operations spread
// over three devices, with Sync calls of random devices in between, converges after two rounds of
// Sync on all three. It runs five seeds; the seed is logged and named on failure.
func TestS2b2Convergence(t *testing.T) {
	for _, seed := range []int64{1, 2, 3, 4, 5} {
		seed := seed
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Logf("seed %d", seed)
			h := newHarness(t)
			runConvergence(t, h, seed)
		})
	}
}

func runConvergence(t *testing.T, h *harness, seed int64) {
	t.Helper()
	devices := [3]*device{
		h.newDevice(t, nil),
		h.newDevice(t, nil),
		h.newDevice(t, nil),
	}
	labels := [3]string{"A", "B", "C"}

	// A starts with a little of everything, then seeds the workspace; B and C join.
	seedSmall(t, devices[0].core)
	devices[0].link(t, labels[0])
	for i := 1; i < len(devices); i++ {
		devices[i].link(t, labels[i])
	}

	r := rand.New(rand.NewSource(seed))
	for n := 0; n < 300; n++ {
		i := r.Intn(len(devices))
		if err := randomOp(t, devices[i], labels[i], r, n); err != nil {
			t.Fatalf("seed %d: operation %d on %s: %v", seed, n, labels[i], err)
		}
		if r.Intn(4) == 0 {
			j := r.Intn(len(devices))
			if err := devices[j].client.Sync(testCtx); err != nil {
				t.Fatalf("seed %d: sync %s after operation %d: %v", seed, labels[j], n, err)
			}
		}
	}
	for round := 0; round < 2; round++ {
		for i, d := range devices {
			if err := d.client.Sync(testCtx); err != nil {
				t.Fatalf("seed %d: final sync %s round %d: %v", seed, labels[i], round, err)
			}
		}
	}

	for i := 1; i < len(devices); i++ {
		for _, entity := range syncedEntities {
			want := dumpTable(t, devices[0].db, entity)
			got := dumpTable(t, devices[i].db, entity)
			if !rowsEqual(got, want) {
				t.Fatalf("seed %d: %s differs between A and %s\nA: %s\n%s: %s",
					seed, entity, labels[i], renderRows(want), labels[i], renderRows(got))
			}
		}
	}
	// Every live server row must be on every device with the same values, and no device may keep
	// a row the server has tombstoned.
	for i, d := range devices {
		if t.Failed() {
			return
		}
		t.Run("server/"+labels[i], func(t *testing.T) {
			assertDeviceMatchesServer(t, h, d)
		})
	}
}

// seedSmall makes the few rows the random operations start from.
func seedSmall(t *testing.T, c *domain.Core) {
	t.Helper()
	project := newProject(t, c, "alpha", nil)
	newProject(t, c, "beta", nil)
	milestone := must[*domain.Milestone](t)(c.CreateMilestone(testCtx,
		domain.CreateMilestoneInput{ProjectID: project.ID, Title: "v1"}))
	category := must[*domain.Category](t)(c.CreateCategory(testCtx,
		domain.CreateCategoryInput{Name: "home", Icon: dotIcon}))
	newTask(t, c, domain.CreateTaskInput{Title: "inbox", CategoryID: &category.ID})
	task := newTask(t, c, domain.CreateTaskInput{
		Title: "project task", ProjectID: &project.ID, MilestoneID: &milestone.ID,
	})
	must[*domain.Report](t)(c.AddReport(testCtx,
		domain.AddReportInput{TaskID: task.ID, Agent: "agent", Summary: "started"}))
}

// randomOp runs one of the operations acceptance criterion 2 names against d. It returns a
// not-found or validation failure instead of failing the test, so the caller can add the seed.
func randomOp(t *testing.T, d *device, label string, r *rand.Rand, n int) error {
	t.Helper()
	name := fmt.Sprintf("%s-%d", label, n)
	switch r.Intn(13) {
	case 0: // create a task
		var projectID, categoryID *int64
		if id, ok, err := pickID(d.db, "SELECT id FROM projects ORDER BY id", r); err != nil {
			return err
		} else if ok {
			projectID = &id
		}
		if id, ok, err := pickID(d.db, "SELECT id FROM categories WHERE builtin IS NULL ORDER BY id", r); err != nil {
			return err
		} else if ok {
			categoryID = &id
		}
		_, err := d.core.CreateTask(testCtx, domain.CreateTaskInput{
			Title: "task-" + name, ProjectID: projectID, CategoryID: categoryID,
		})
		return err
	case 1: // update a task
		id, ok, err := pickID(d.db, "SELECT id FROM tasks ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateTask(testCtx, domain.CreateTaskInput{Title: "task-" + name})
			return err
		}
		_, err = d.core.UpdateTask(testCtx, id, domain.TaskPatch{Title: ptr("title-" + name)})
		return err
	case 2: // complete a task
		id, ok, err := pickID(d.db, "SELECT id FROM tasks ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateTask(testCtx, domain.CreateTaskInput{Title: "task-" + name})
			return err
		}
		_, err = d.core.UpdateTask(testCtx, id, domain.TaskPatch{Status: ptr(domain.StatusDone)})
		return err
	case 3: // delete a task
		id, ok, err := pickID(d.db, "SELECT id FROM tasks ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateTask(testCtx, domain.CreateTaskInput{Title: "task-" + name})
			return err
		}
		return d.core.DeleteTask(testCtx, id)
	case 4: // delete and restore a task
		id, ok, err := pickID(d.db, "SELECT id FROM tasks ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateTask(testCtx, domain.CreateTaskInput{Title: "task-" + name})
			return err
		}
		snap, err := d.core.SnapshotTask(testCtx, id)
		if err != nil {
			return err
		}
		if err := d.core.DeleteTask(testCtx, id); err != nil {
			return err
		}
		_, err = d.core.RestoreTask(testCtx, *snap)
		return err
	case 5: // add a report
		id, ok, err := pickID(d.db, "SELECT id FROM tasks ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			task, err := d.core.CreateTask(testCtx, domain.CreateTaskInput{Title: "task-" + name})
			if err != nil {
				return err
			}
			id = task.ID
		}
		_, err = d.core.AddReport(testCtx, domain.AddReportInput{
			TaskID: id, Agent: "agent", Summary: "report-" + name,
		})
		return err
	case 6: // create a project
		_, err := d.core.CreateProject(testCtx, domain.CreateProjectInput{Name: "project-" + name})
		return err
	case 7: // rename a project
		id, ok, err := pickID(d.db, "SELECT id FROM projects ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateProject(testCtx, domain.CreateProjectInput{Name: "project-" + name})
			return err
		}
		_, err = d.core.RenameProject(testCtx, id, "renamed-"+name)
		return err
	case 8: // archive a project
		id, ok, err := pickID(d.db, "SELECT id FROM projects ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateProject(testCtx, domain.CreateProjectInput{Name: "project-" + name})
			return err
		}
		_, err = d.core.ArchiveProject(testCtx, id)
		return err
	case 9: // delete a project
		id, ok, err := pickID(d.db, "SELECT id FROM projects ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateProject(testCtx, domain.CreateProjectInput{Name: "project-" + name})
			return err
		}
		_, err = d.core.DeleteProject(testCtx, id)
		return err
	case 10: // create a category
		existing, err := countRows(d.db, "SELECT COUNT(*) FROM categories")
		if err != nil {
			return err
		}
		if existing >= 20 {
			return nil
		}
		_, err = d.core.CreateCategory(testCtx, domain.CreateCategoryInput{
			Name: "cat-" + name, Icon: dotIcon,
		})
		return err
	case 11: // update a category
		id, ok, err := pickID(d.db, "SELECT id FROM categories WHERE builtin IS NULL ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateCategory(testCtx, domain.CreateCategoryInput{
				Name: "cat-" + name, Icon: dotIcon,
			})
			return err
		}
		_, err = d.core.UpdateCategory(testCtx, id, domain.CategoryPatch{Name: ptr("cat-ren-" + name)})
		return err
	default: // delete a category
		id, ok, err := pickID(d.db, "SELECT id FROM categories WHERE builtin IS NULL ORDER BY id", r)
		if err != nil {
			return err
		}
		if !ok {
			_, err := d.core.CreateCategory(testCtx, domain.CreateCategoryInput{
				Name: "cat-" + name, Icon: dotIcon,
			})
			return err
		}
		_, err = d.core.DeleteCategory(testCtx, id)
		return err
	}
}

// countRows is count without t.Fatal, for the random operations that report errors to their caller.
func countRows(db *sql.DB, query string) (int, error) {
	var n int
	if err := db.QueryRowContext(testCtx, query).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// pickID returns a random id of the query's result, deterministic for a seed and the database state.
func pickID(db *sql.DB, query string, r *rand.Rand) (int64, bool, error) {
	rows, err := db.QueryContext(testCtx, query)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = rows.Close() }()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, false, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, false, err
	}
	if len(ids) == 0 {
		return 0, false, nil
	}
	return ids[r.Intn(len(ids))], true, nil
}

func rowsEqual(a, b []map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for k, v := range a[i] {
			if b[i][k] != v {
				return false
			}
		}
	}
	return true
}
