package views

import (
	"sort"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// agentRow is an active agent task with what the panel shows of it.
type agentRow struct {
	data.OverviewTask
	Project string
}

type agentGroup struct {
	Name string
	Rows []agentRow
}

// panelModel is what the panel lists for a query.
type panelModel struct {
	Groups []agentGroup
	Mine   []data.Task
	Active int // agent tasks shown
}

// entries returns the task ids in selection order: agents first, then mine.
func (m panelModel) entries() []int {
	var ids []int
	for _, g := range m.Groups {
		for _, r := range g.Rows {
			ids = append(ids, r.ID)
		}
	}
	for _, t := range m.Mine {
		ids = append(ids, t.ID)
	}
	return ids
}

func (m panelModel) empty() bool { return len(m.Groups) == 0 && len(m.Mine) == 0 }

// buildPanel lists what agents are doing and what the user has on their plate, filtered by the
// query. Agents: source agent in doing, blocked or review, grouped by project, blocked first.
// Mine: source user in todo or doing, by reminder or due time, then priority.
func (a *App) buildPanel(query string) panelModel {
	q := strings.ToLower(strings.TrimSpace(query))
	m := panelModel{Groups: a.agentGroups(q), Mine: a.mineTasks(q)}
	for _, g := range m.Groups {
		m.Active += len(g.Rows)
	}
	return m
}

func (a *App) agentGroups(q string) []agentGroup {
	isAgentActive := func(t data.Task) bool { return t.Source == data.SourceAgent && t.Status.Active() }
	listed := map[int]bool{}
	var groups []agentGroup
	for _, g := range a.snap.overview.Groups {
		name := "Inbox"
		if g.Project != nil {
			name = g.Project.Name
		}
		var rows []agentRow
		for _, ot := range g.Tasks {
			if !isAgentActive(ot.Task) {
				continue
			}
			listed[ot.ID] = true
			if comp, ok := a.completing[ot.ID]; ok && isAgentActive(comp.task) {
				ot = a.overviewTask(comp.task)
			}
			if r := (agentRow{OverviewTask: ot, Project: name}); r.matches(q) {
				rows = append(rows, r)
			}
		}
		if len(rows) > 0 {
			groups = append(groups, agentGroup{Name: name, Rows: rows})
		}
	}
	groups = a.addCompletingAgents(groups, listed, q, isAgentActive)
	sortGroups(groups)
	return groups
}

// addCompletingAgents puts agent tasks that were just marked done back where they were.
func (a *App) addCompletingAgents(groups []agentGroup, listed map[int]bool, q string, active func(data.Task) bool) []agentGroup {
	a.pruneCompletions()
	for id, comp := range a.completing {
		if listed[id] || !active(comp.task) {
			continue
		}
		name := a.projectName(comp.task.ProjectID)
		if name == "" {
			name = "Inbox"
		}
		row := agentRow{OverviewTask: a.overviewTask(comp.task), Project: name}
		if !row.matches(q) {
			continue
		}
		i := indexOfGroup(groups, name)
		if i < 0 {
			groups = append(groups, agentGroup{Name: name})
			i = len(groups) - 1
		}
		groups[i].Rows = append(groups[i].Rows, row)
	}
	return groups
}

func indexOfGroup(gs []agentGroup, name string) int {
	for i, g := range gs {
		if g.Name == name {
			return i
		}
	}
	return -1
}

// sortGroups puts projects with a blocked task first, then by name; rows by attention.
func sortGroups(gs []agentGroup) {
	rank := map[data.Status]int{data.Blocked: 0, data.Review: 1, data.Doing: 2}
	for _, g := range gs {
		sort.SliceStable(g.Rows, func(i, j int) bool {
			if ri, rj := rank[g.Rows[i].Status], rank[g.Rows[j].Status]; ri != rj {
				return ri < rj
			}
			return g.Rows[i].ID > g.Rows[j].ID
		})
	}
	blocked := func(g agentGroup) bool { return len(g.Rows) > 0 && g.Rows[0].Status == data.Blocked }
	sort.SliceStable(gs, func(i, j int) bool {
		if bi, bj := blocked(gs[i]), blocked(gs[j]); bi != bj {
			return bi
		}
		return gs[i].Name < gs[j].Name
	})
}

func (r agentRow) matches(q string) bool {
	if q == "" {
		return true
	}
	hay := r.Title + " " + r.Branch + " " + r.Project
	if r.LatestReport != nil {
		hay += " " + r.LatestReport.Summary
	}
	return strings.Contains(strings.ToLower(hay), q)
}

func (a *App) mineTasks(q string) []data.Task {
	mine := func(t data.Task) bool {
		return t.Source == data.SourceUser && (t.Status == data.Todo || t.Status == data.Doing)
	}
	var tasks []data.Task
	for _, t := range a.snap.tasks {
		if mine(t) {
			tasks = append(tasks, t)
		}
	}
	tasks = a.withCompleting(tasks, mine)
	out := tasks[:0:0]
	for _, t := range tasks {
		hay := strings.ToLower(t.Title + " " + a.projectName(t.ProjectID))
		if q == "" || strings.Contains(hay, q) {
			out = append(out, t)
		}
	}
	sortMine(out)
	return out
}

// sortMine orders tasks by the moment they ask for attention, then priority; tasks without a
// moment come after those with one.
func sortMine(ts []data.Task) {
	sort.SliceStable(ts, func(i, j int) bool {
		mi, mj := nextMoment(ts[i]), nextMoment(ts[j])
		if mi.IsZero() != mj.IsZero() {
			return !mi.IsZero()
		}
		if !mi.Equal(mj) {
			return mi.Before(mj)
		}
		if ts[i].Priority != ts[j].Priority {
			return ts[i].Priority < ts[j].Priority
		}
		return ts[i].ID > ts[j].ID
	})
}

// overviewTask describes a task the way Store.Overview does.
func (a *App) overviewTask(t data.Task) data.OverviewTask {
	ot := data.OverviewTask{Task: t}
	since := t.UpdatedAt
	if d, ok := a.snap.details[t.ID]; ok && len(d.Reports) > 0 {
		r := d.Reports[0]
		ot.LatestReport = &r
		since = r.CreatedAt
	}
	ot.HoursSinceActivity = int(a.Now().Sub(since) / time.Hour)
	stale := a.snap.overview.StaleHours
	if stale == 0 {
		stale = data.DefaultStaleHours
	}
	ot.Stale = ot.HoursSinceActivity >= stale
	return ot
}

// moveSelection moves the keyboard highlight by dir. -1 means no row is highlighted; moving up
// from the first row returns to the input.
func moveSelection(cur, dir, n int) int {
	if n == 0 {
		return -1
	}
	next := cur + dir
	return max(-1, min(next, n-1))
}
