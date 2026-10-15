package views

import (
	"sort"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// listIndent is where a row's text starts: under the group header's name, past its glyph and the
// 8 pt gap.
const listIndent = theme.GlyphSize + 8

// listGroup is one status of a list: a header and its rows.
type listGroup struct {
	status data.Status
	tasks  []data.Task
}

// activeGroups are the groups every list draws, in order. A group says the status, so rows carry
// no glyph.
var activeGroups = []listGroup{
	{status: data.Blocked},
	{status: data.Review},
	{status: data.Doing},
	{status: data.Todo},
}

// listModel is the tasks of the current project or the Inbox, grouped. Empty groups are dropped.
type listModel struct {
	groups  []listGroup // Blocked, In review, Doing, To do
	done    []data.Task // newest first
	backlog []data.Task
	total   int // every task of the screen, whatever its status
}

// selectionTasks returns the tasks of the current project or the Inbox, filtered by milestone.
func (m *Main) selectionTasks() []data.Task {
	in := func(t data.Task) bool {
		switch {
		case m.nav.screen == screenInbox:
			return t.ProjectID == nil
		case t.ProjectID == nil || *t.ProjectID != m.nav.projectID:
			return false
		}
		ms := m.milestone[m.nav.projectID]
		return ms == 0 || (t.MilestoneID != nil && *t.MilestoneID == ms)
	}
	var out []data.Task
	for _, t := range m.app.snap.tasks {
		if in(t) {
			out = append(out, t)
		}
	}
	return m.app.withCompleting(out, in)
}

func (m *Main) listModel() listModel {
	var lm listModel
	by := map[data.Status][]data.Task{}
	for _, t := range m.selectionTasks() {
		lm.total++
		by[t.Status] = append(by[t.Status], t)
	}
	for _, g := range activeGroups {
		tasks := by[g.status]
		if len(tasks) == 0 {
			continue
		}
		sortForList(tasks)
		g.tasks = tasks
		lm.groups = append(lm.groups, g)
	}
	lm.done, lm.backlog = by[data.Done], by[data.Backlog]
	sort.SliceStable(lm.done, func(i, j int) bool { return doneAt(lm.done[i]).After(doneAt(lm.done[j])) })
	sortForList(lm.backlog)
	return lm
}

// sortForList puts what asks for a date first, in date order, then the most urgent, then the
// most recently touched.
func sortForList(ts []data.Task) {
	sort.SliceStable(ts, func(i, j int) bool {
		ai, iok := data.NextMoment(ts[i])
		aj, jok := data.NextMoment(ts[j])
		switch {
		case iok != jok:
			return iok
		case iok && !ai.Equal(aj):
			return ai.Before(aj)
		case ts[i].Priority != ts[j].Priority:
			return ts[i].Priority < ts[j].Priority
		}
		return ts[i].UpdatedAt.After(ts[j].UpdatedAt)
	})
}

// shownDone is the Done rows drawn: the newest three, or all once "Show all" was clicked.
func (m *Main) shownDone(lm listModel) []data.Task {
	if m.allDone[m.nav] || len(lm.done) <= theme.DoneListRows {
		return lm.done
	}
	return lm.done[:theme.DoneListRows]
}

// listIDs is the task ids of the rows the list draws, for the arrow keys.
func (m *Main) listIDs() []int {
	lm := m.listModel()
	var ids []int
	add := func(ts []data.Task) {
		for _, t := range ts {
			ids = append(ids, t.ID)
		}
	}
	for _, g := range lm.groups {
		add(g.tasks)
	}
	add(m.shownDone(lm))
	if m.expanded[m.nav] {
		add(lm.backlog)
	}
	return ids
}

// listScreen is a project or the Inbox as a list grouped by status.
func (m *Main) listScreen(c *ui.Context, t *theme.Tokens) {
	m.listHeader(c, t)
	ui.Box(c).Height(18).Shrink(0)
	m.pageComposer(c, t)
	if !m.app.snap.loaded {
		return
	}
	lm := m.listModel()
	m.belowComposer(c, func() {
		if lm.total == 0 {
			m.emptyList(c, t)
			return
		}
		for _, g := range lm.groups {
			m.group(c, t, g.status, g.tasks)
		}
		m.doneAndBacklog(c, t, lm)
	})
}

// group is a header, glyph and name and count, and its rows indented under the name.
func (m *Main) group(c *ui.Context, t *theme.Tokens, s data.Status, tasks []data.Task) {
	ui.Row(c.Key("group-"+string(s))).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.StatusGlyph(c, t, s, 0)
		theme.Text(c, theme.BodyStrong, t.Text, statusLabel(m.say, s))
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(tasks)))
	})
	for _, task := range tasks {
		m.listRow(c, t, task)
	}
}

// listRow is one task: title, branch and pull request, a summary line for agent work, and on the
// right when it is due, how long ago it moved, or that it went stale.
func (m *Main) listRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	if task.Status == data.Done {
		m.finishedRow(c, t, task, finishedOpts{when: m.clock.Day(doneAt(task), m.app.Now())})
		return
	}
	_, opacity, _ := m.app.completionPhase(c, task.ID)
	ot := m.app.overviewTask(task)
	row := m.rowShell(c, t, task.ID, plainRow, opacity).AlignItems(ui.Center)
	row.Children(func() {
		m.authorSlot(c, task) // in the glyph's column
		body(c, func() {
			m.titleLine(c, t, task, "", false)
			summaryLine(c, t, listMeta(task, ot))
		})
		right(c, func() { m.listRight(c, t, task, ot) })
	})
	if row.Clicked() {
		m.openRow(task.ID)
	}
}

// listMeta is the quiet second line of a row: who it is (an agent's name), where (its branch and
// pull request) and what the agent last said; for a blocked task, what it is waiting for. A
// task of mine with no branch has none.
func listMeta(task data.Task, ot data.OverviewTask) string {
	var parts []string
	said := ""
	if task.Source == data.SourceAgent && ot.LatestReport != nil && !task.Status.Terminal() {
		parts = append(parts, agentName(ot.LatestReport.Agent))
		said = ot.LatestReport.Summary
		if task.Status == data.Blocked && ot.LatestReport.NextStep != "" {
			said = ot.LatestReport.NextStep
		}
	}
	if task.Branch != "" {
		parts = append(parts, task.Branch)
	}
	if task.PRNumber > 0 {
		parts = append(parts, "#"+itoa(task.PRNumber))
	}
	if said != "" {
		parts = append(parts, said)
	}
	return strings.Join(parts, " · ")
}

func (m *Main) listRight(c *ui.Context, t *theme.Tokens, task data.Task, ot data.OverviewTask) {
	now := m.app.Now()
	if theme.ShowsPriority(task.Priority) && !task.Status.Terminal() {
		theme.PriorityBadge(c, t, task.Priority)
	}
	switch {
	case task.Status.Terminal():
		plainText(c, t, m.clock.Age(doneAt(task), now))
	case task.Source == data.SourceUser:
		if label := m.clock.DueLabel(task, now); label != "" {
			plainText(c, t, label)
		}
	case ot.Stale && task.Status.Active():
		theme.Text(c, theme.Meta, t.Stale, m.clock.Stale(ot.HoursSinceActivity)).Shrink(0)
	default:
		since := task.UpdatedAt
		if ot.LatestReport != nil {
			since = ot.LatestReport.CreatedAt
		}
		if task.Status == data.Doing && now.Sub(since) < data.LiveWithin {
			liveDot(c, t)
		}
		plainText(c, t, m.clock.Age(since, now))
	}
}

func plainText(c *ui.Context, t *theme.Tokens, text string) {
	theme.Text(c, theme.Meta, t.TextTertiary, text).Shrink(0)
}

// doneAndBacklog closes a list: the Done group, expanded, with its newest three rows; then Backlog
// folded into the last line, "Backlog 2 — show", which opens it.
func (m *Main) doneAndBacklog(c *ui.Context, t *theme.Tokens, lm listModel) {
	if len(lm.done) > 0 {
		m.group(c, t, data.Done, m.shownDone(lm))
		m.showAllDone(c, t, lm)
	}
	if len(lm.backlog) == 0 {
		return
	}
	open := m.expanded[m.nav]
	if open {
		m.group(c, t, data.Backlog, lm.backlog)
	}
	m.foldLine(c, t, lm, open, 16)
}

func (m *Main) showAllDone(c *ui.Context, t *theme.Tokens, lm listModel) {
	if m.allDone[m.nav] || len(lm.done) <= theme.DoneListRows {
		return
	}
	if link(c, t, m.say.ListShowAllDone(len(lm.done))).Margin(6, 0, 0, listIndent).Clicked() {
		m.allDone[m.nav] = true
	}
}

// foldLine is the last line of a list, which folds Backlog: "Backlog 2 — show".
func (m *Main) foldLine(c *ui.Context, t *theme.Tokens, lm listModel, open bool, above float32) {
	count := m.say.FoldCount(statusLabel(m.say, data.Backlog), len(lm.backlog))
	label, verb := m.say.FoldShow(count), m.say.FoldShowShort()
	if open {
		label, verb = m.say.FoldHide(count), m.say.FoldHideShort()
	}
	line := ui.Row(c.Key("fold")).Height(theme.RowOneLine).Shrink(0).Margin(above, 0, 0).Gap(6).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Label(label)
	line.Children(func() {
		theme.Text(c, theme.BodyStrong, t.TextTertiary, count)
		verbColor := t.TextTertiary
		if line.Hovered() {
			verbColor = t.TextSecondary
		}
		theme.Text(c, theme.Body, verbColor, verb)
	})
	if line.Clicked() {
		m.expanded[m.nav] = !open
	}
}

// emptyList is a project or the Inbox with no task at all. It says what will appear and how to
// make it appear; the composer above is where to write one.
func (m *Main) emptyList(c *ui.Context, t *theme.Tokens) {
	title, body := m.say.ListEmptyTitle(), m.say.ListEmptyBody()
	if m.nav.screen == screenInbox {
		title, body = m.say.InboxEmptyTitle(), m.say.InboxEmptyBody()
	}
	emptyState(c, t, title, body, 56, func() {
		if m.nav.screen == screenProject {
			if link(c, t, m.say.LinkHowToConnect()).Margin(4, 0, 0).Clicked() {
				m.openSettings(PaneAgents)
			}
		}
	})
}

// listHeader is the project's name, or Inbox, with the milestone filter (once the project has a
// milestone) and the List | Board switch at the right.
func (m *Main) listHeader(c *ui.Context, t *theme.Tokens) {
	title := m.say.NavInbox()
	if m.nav.screen == screenProject {
		p, _ := m.app.Project(m.nav.projectID)
		title = p.Name
	}
	m.pageHeader(c, t, title, "", func() {
		m.milestoneMenu(c, t)
		if m.nav.screen == screenProject {
			m.viewToggle(c, t)
		}
	})
}
