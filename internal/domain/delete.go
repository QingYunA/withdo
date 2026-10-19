package domain

import (
	"context"
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// Deleted counts what DeleteProject removed besides the project itself.
type Deleted struct {
	Reports    int `json:"reports"`
	Tasks      int `json:"tasks"`
	Milestones int `json:"milestones"`
}

// DeleteProject removes a project with its reports, tasks and milestones, in that order and in one
// transaction: when any step fails nothing is removed. The schema has no ON DELETE CASCADE; the only
// foreign key actions are SET NULL: deleting a project promotes its sub-projects, as deleting a task
// promotes its subtasks.
func (c *Core) DeleteProject(ctx context.Context, id int64) (Deleted, error) {
	var out Deleted
	err := c.write(ctx, func(q store.Querier) error {
		p, err := getProject(ctx, q, id)
		if err != nil {
			return err
		}
		if p == nil {
			return notFound("project_id", "project", id)
		}
		// A task belongs through its project or through a milestone of it.
		const tasks = `SELECT id FROM tasks WHERE project_id = ?1
			OR milestone_id IN (SELECT id FROM milestones WHERE project_id = ?1)`
		steps := []struct {
			what  string
			query string
			n     *int
		}{
			{"reports", "DELETE FROM reports WHERE task_id IN (" + tasks + ")", &out.Reports},
			{"tasks", "DELETE FROM tasks WHERE id IN (" + tasks + ")", &out.Tasks},
			{"milestones", "DELETE FROM milestones WHERE project_id = ?1", &out.Milestones},
			{"project", "DELETE FROM projects WHERE id = ?1", new(int)},
		}
		for _, s := range steps {
			res, err := q.ExecContext(ctx, s.query, id)
			if err != nil {
				return fmt.Errorf("delete %s: %w", s.what, err)
			}
			n, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("delete %s: %w", s.what, err)
			}
			*s.n = int(n)
		}
		return nil
	})
	if err != nil {
		return Deleted{}, fmt.Errorf("delete project %d: %w", id, err)
	}
	return out, nil
}
