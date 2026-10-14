package domain_test

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// Two Cores on separate connection pools stand in for two processes (the MCP
// server and the app) writing the same database at once.
func TestTwoConnectionsWriteConcurrentlyWithoutLosingRows(t *testing.T) {
	testutil.IsolateGit(t)
	path := filepath.Join(t.TempDir(), "shared.db")
	dbA, _ := testutil.OpenDBAt(t, path)
	dbB, _ := testutil.OpenDBAt(t, path)
	cores := []*domain.Core{domain.New(dbA, domain.Options{}), domain.New(dbB, domain.Options{})}
	repo := widgetsRepo(t, "feat/shared")

	const perCore = 12
	var wg sync.WaitGroup
	errs := make(chan error, 2*perCore)
	for i, c := range cores {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perCore; j++ {
				if _, err := c.CreateTask(ctx, domain.CreateTaskInput{Title: fmt.Sprintf("c%d-%d", i, j)}); err != nil {
					errs <- err
				}
				// Both cores resolve the same repo: exactly one project must result.
				if _, err := c.ReportProgress(ctx, domain.ProgressInput{Summary: "s", Agent: "x", Cwd: repo}); err != nil {
					errs <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	tasks := must[[]domain.Task](t)(cores[0].ListTasks(ctx, domain.TaskFilter{Limit: 200}))
	// perCore*2 created tasks, plus one shared agent task on feat/shared.
	eq(t, "tasks", len(tasks), 2*perCore+1)
	eq(t, "projects", len(must[[]domain.Project](t)(cores[1].ListProjects(ctx))), 1)
}
