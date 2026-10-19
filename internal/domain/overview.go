package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// DefaultStaleHours is how long a task may go without a report before it is stale.
const DefaultStaleHours = 48

// OverviewTask is an active task with its latest report and staleness.
type OverviewTask struct {
	Task
	LatestReport       *Report `json:"latest_report"`
	Stale              bool    `json:"stale"`
	HoursSinceActivity int64   `json:"hours_since_activity"`
}

// OverviewProject identifies the project of an OverviewGroup.
type OverviewProject struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	RepoSlug *string `json:"repo_slug"`
}

// OverviewGroup is one project's active tasks; Project is nil for the inbox.
type OverviewGroup struct {
	Project *OverviewProject `json:"project"`
	Tasks   []OverviewTask   `json:"tasks"`
}

// Overview is every active task (doing, blocked, review) grouped by project.
type Overview struct {
	GeneratedAt string          `json:"generated_at"`
	StaleHours  float64         `json:"stale_hours"`
	Projects    []OverviewGroup `json:"projects"`
}

// OverviewOptions tunes GetOverview. Zero values mean defaults.
type OverviewOptions struct {
	// StaleHours: hours without a report before a task is stale (DefaultStaleHours when 0).
	StaleHours float64
}

// attentionRank orders what needs the user first: blocked, review, doing.
var attentionRank = map[Status]int{StatusBlocked: 0, StatusReview: 1, StatusDoing: 2}

// GetOverview lists active tasks per non-archived project (plus the inbox), each
// with its latest report. Projects sort by name, the inbox last; tasks sort by
// attention, then newest first.
func (c *Core) GetOverview(ctx context.Context, opts OverviewOptions) (*Overview, error) {
	staleHours := opts.StaleHours
	if staleHours <= 0 {
		staleHours = DefaultStaleHours
	}
	now := c.now()
	tasks, err := c.activeTasks(ctx)
	if err != nil {
		return nil, err
	}
	projects, err := c.activeProjectIndex(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	latest, err := c.latestReportsOf(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("latest reports: %w", err)
	}
	byProject := map[int64][]OverviewTask{}
	var inbox []OverviewTask
	for _, t := range tasks {
		var report *Report
		if r, ok := latest[t.ID]; ok {
			report = &r
		}
		ot := describe(t, report, now, staleHours)
		if t.ProjectID == nil {
			inbox = append(inbox, ot)
		} else {
			byProject[*t.ProjectID] = append(byProject[*t.ProjectID], ot)
		}
	}
	groups := make([]OverviewGroup, 0, len(byProject)+1)
	for id, list := range byProject {
		p := projects[id]
		groups = append(groups, OverviewGroup{
			Project: &OverviewProject{ID: p.ID, Name: p.Name, RepoSlug: p.RepoSlug},
			Tasks:   sortByAttention(list),
		})
	}
	sort.Slice(groups, func(i, j int) bool { return lessName(groups[i].Project.Name, groups[j].Project.Name) })
	if len(inbox) > 0 {
		groups = append(groups, OverviewGroup{Tasks: sortByAttention(inbox)})
	}
	return &Overview{GeneratedAt: FormatTime(now), StaleHours: staleHours, Projects: groups}, nil
}

// activeTasks returns doing/blocked/review tasks outside archived projects.
func (c *Core) activeTasks(ctx context.Context) ([]Task, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT "+prefixCols(taskCols)+` FROM tasks t LEFT JOIN projects p ON p.id = t.project_id
		WHERE t.status IN ('doing','blocked','review') AND (t.project_id IS NULL OR p.archived_at IS NULL)`)
	if err != nil {
		return nil, fmt.Errorf("list active tasks: %w", err)
	}
	return collect(rows, scanTask)
}

// prefixCols qualifies every unqualified column in a comma list with "t.".
func prefixCols(cols string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if !strings.Contains(p, ".") {
			p = "t." + p
		}
		parts[i] = p
	}
	return strings.Join(parts, ", ")
}

func (c *Core) activeProjectIndex(ctx context.Context) (map[int64]Project, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT "+projectCols+" FROM projects WHERE archived_at IS NULL")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	list, err := collect(rows, scanProject)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]Project, len(list))
	for _, p := range list {
		out[p.ID] = p
	}
	return out, nil
}

func describe(t Task, report *Report, now time.Time, staleHours float64) OverviewTask {
	since := t.UpdatedAt
	if report != nil {
		since = report.CreatedAt
	}
	hours := int64(0)
	if at, err := time.Parse(time.RFC3339Nano, since); err == nil {
		hours = int64(math.Floor(now.Sub(at).Hours()))
	}
	return OverviewTask{Task: t, LatestReport: report, HoursSinceActivity: hours, Stale: float64(hours) >= staleHours}
}

func sortByAttention(list []OverviewTask) []OverviewTask {
	out := append([]OverviewTask(nil), list...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := attentionRank[out[i].Status], attentionRank[out[j].Status]
		if ri != rj {
			return ri < rj
		}
		return out[i].ID > out[j].ID
	})
	return out
}

// lessName orders project names case-insensitively, lower case first on ties.
func lessName(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return la < lb
	}
	return a > b
}
