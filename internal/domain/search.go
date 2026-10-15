package domain

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// liveProject limits a query on tasks to those not in an archived project.
const liveProject = "(project_id IS NULL OR project_id IN (SELECT id FROM projects WHERE archived_at IS NULL))"

// likeEscaper makes a word safe inside LIKE ... ESCAPE '\'.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// SearchTasks finds tasks, whatever their status, whose title, notes or branch contain every word
// of query, or whose pull request number is a word of it ("#42" and "42" both match). Tasks of
// archived projects are left out. The newest come first; limit is DefaultTaskLimit when 0. A blank
// query finds nothing.
func (c *Core) SearchTasks(ctx context.Context, query string, limit int) ([]Task, error) {
	words := strings.Fields(query)
	if len(words) == 0 {
		return []Task{}, nil
	}
	where := []string{liveProject}
	var args []any
	for _, w := range words {
		like := "%" + likeEscaper.Replace(w) + "%"
		clause := `(title LIKE ? ESCAPE '\' OR notes LIKE ? ESCAPE '\' OR branch LIKE ? ESCAPE '\'`
		args = append(args, like, like, like)
		if n, err := strconv.ParseInt(strings.TrimPrefix(w, "#"), 10, 64); err == nil {
			clause += " OR pr_number = ?"
			args = append(args, n)
		}
		where = append(where, clause+")")
	}
	if limit <= 0 {
		limit = DefaultTaskLimit
	}
	rows, err := c.db.QueryContext(ctx,
		"SELECT "+taskCols+" FROM tasks WHERE "+strings.Join(where, " AND ")+" ORDER BY updated_at DESC, id DESC LIMIT ?",
		append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	return collect(rows, scanTask)
}

// ListLogbook returns the done and cancelled tasks, newest first by completion (by the last edit for
// a row that has no completion time). Tasks of archived projects are left out; limit is
// DefaultTaskLimit when 0.
func (c *Core) ListLogbook(ctx context.Context, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = DefaultTaskLimit
	}
	rows, err := c.db.QueryContext(ctx, "SELECT "+taskCols+" FROM tasks WHERE status IN ('done','cancelled') AND "+liveProject+
		" ORDER BY COALESCE(completed_at, updated_at) DESC, id DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("list logbook: %w", err)
	}
	return collect(rows, scanTask)
}
