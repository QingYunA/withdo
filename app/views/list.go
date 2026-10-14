package views

import (
	"sort"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// listIndent is where a row's text starts: under the group header's name, past its 16 pt glyph and
// the 8 pt gap.
const listIndent float32 = 24

// listGroup is one status of a list: a header and its rows.
type listGroup struct {
	status data.Status
	name   string
	tasks  []data.Task
}

// activeGroups are the groups every list draws, in order. The names are the ones the list uses:
// the group says the status, so rows carry no glyph.
var activeGroups = []listGroup{
	{status: data.Blocked, name: "Blocked"},
	{status: data.Review, name: "In review"},
	{status: data.Doing, name: "Doing"},
	{status: data.Todo, name: "To do"},
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

// shownDone is the Done rows drawn: the recent ten, or all.
func (m *Main) shownDone(lm listModel) []data.Task {
	if m.allDone[m.nav] || len(lm.done) <= theme.DoneRecentLimit {
		return lm.done
	}
	return lm.done[:theme.DoneRecentLimit]
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
	if m.expanded[m.nav] {
		add(m.shownDone(lm))
		add(lm.backlog)
	}
	return ids
}

// listScreen is a project or the Inbox as a list grouped by status.
func (m *Main) listScreen(c *ui.Context, t *theme.Tokens) {
	m.listHeader(c, t)
	if !m.app.snap.loaded {
		return
	}
	lm := m.listModel()
	if lm.total == 0 {
		m.emptyList(c, t)
		return
	}
	for _, g := range lm.groups {
		m.group(c, t, g.status, g.name, g.tasks)
	}
	m.addTaskRow(c, t, listIndent)
	m.foldedGroups(c, t, lm)
}

// group is a header, glyph and name and count, and its rows indented under the name.
func (m *Main) group(c *ui.Context, t *theme.Tokens, s data.Status, name string, tasks []data.Task) {
	ui.Row(c.Key("group-"+string(s))).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.StatusGlyph(c, t, s, 0)
		theme.Text(c, theme.BodyStrong, t.Text, name)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(tasks)))
	})
	for _, task := range tasks {
		m.listRow(c, t, task)
	}
}

// listRow is one task: title, branch and pull request, a summary line for agent work, and on the
// right when it is due, how long ago it moved, or that it went stale.
func (m *Main) listRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	_, opacity, _ := m.app.completionPhase(c, task.ID)
	ot := m.app.overviewTask(task)
	row := m.rowShell(c, t, task.ID, plainRow, opacity).AlignItems(ui.Center)
	row.Children(func() {
		ui.Box(c).Width(12).Shrink(0) // the glyph's column, and the gap after it, make 24
		body(c, func() {
			m.titleLine(c, t, task, "", true)
			summaryLine(c, t, listSummary(task, ot))
		})
		right(c, func() { m.listRight(c, t, task, ot) })
	})
	if row.Clicked() {
		m.openRow(task.ID)
	}
}

// listSummary is what an agent last said; for a blocked task, what it is waiting for.
func listSummary(task data.Task, ot data.OverviewTask) string {
	if task.Source != data.SourceAgent || ot.LatestReport == nil || task.Status.Terminal() {
		return ""
	}
	if task.Status == data.Blocked && ot.LatestReport.NextStep != "" {
		return ot.LatestReport.NextStep
	}
	return ot.LatestReport.Summary
}

func (m *Main) listRight(c *ui.Context, t *theme.Tokens, task data.Task, ot data.OverviewTask) {
	now := m.app.Now()
	if task.Priority != data.P2 && !task.Status.Terminal() {
		theme.PriorityBars(c, t, task.Priority)
	}
	switch {
	case task.Status.Terminal():
		plainText(c, t, data.Age(doneAt(task), now))
	case task.Source == data.SourceUser:
		if label := data.DueLabel(task, now); label != "" {
			plainText(c, t, label)
		}
	case ot.Stale && task.Status.Active():
		theme.Text(c, theme.Meta, t.Stale, staleWords(ot.HoursSinceActivity)).Shrink(0)
	default:
		since := task.UpdatedAt
		if ot.LatestReport != nil {
			since = ot.LatestReport.CreatedAt
		}
		if task.Status == data.Doing && now.Sub(since) < data.LiveWithin {
			liveDot(c, t)
		}
		plainText(c, t, data.Age(since, now))
	}
}

func plainText(c *ui.Context, t *theme.Tokens, text string) {
	theme.Text(c, theme.Meta, t.TextTertiary, text).Shrink(0)
}

// addTaskRow is the quiet "+ Add a task" under To do; clicking it opens an inline input, and Enter
// creates a task in this project, or in the Inbox.
func (m *Main) addTaskRow(c *ui.Context, t *theme.Tokens, indent float32) {
	if m.adding && m.addNav == m.nav {
		m.addInput(c, t, indent)
		return
	}
	row := ui.Row(c.Key("add-task")).Height(theme.RowOneLine).Shrink(0).Margin(8, 0, 0, -6).Padding(0, 6, 0, indent+6).
		Radius(theme.RadiusCard).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label("Add a task")
	if row.Hovered() {
		row.Background(t.BgHover)
	}
	row.Children(func() { theme.Text(c, theme.Body, t.TextTertiary, "+ Add a task") })
	if row.Clicked() {
		m.adding, m.addNav, m.addBuf = true, m.nav, ""
	}
}

func (m *Main) addInput(c *ui.Context, t *theme.Tokens, indent float32) {
	ui.Row(c.Key("add-input-row")).Height(theme.RowOneLine).Shrink(0).Margin(8, 0, 0).Padding(0, 0, 0, indent).AlignItems(ui.Center).Children(func() {
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			in := theme.Body.Apply(ui.TextInputBase(c.Key("add-input"), &m.addBuf)).TextColor(t.Text).
				Label("New task title").AutoFocus()
			if in.Submitted() {
				m.submitAdding()
			}
			if m.addBuf == "" {
				theme.Text(c, theme.Body, t.TextTertiary, "Task title").Absolute().Left(0).Top(0).PassThrough()
			}
		})
	})
}

func (m *Main) cancelAdding() { m.adding, m.addBuf = false, "" }

// submitAdding creates a todo in the current project (or the Inbox) and keeps the input open and
// empty, so several thoughts can be caught in a row.
func (m *Main) submitAdding() {
	title := strings.TrimSpace(m.addBuf)
	if title == "" {
		return
	}
	var project *int
	if m.nav.screen == screenProject {
		project = data.IntPtr(m.nav.projectID)
	}
	if err := m.app.AddTask(data.CreateTaskInput{Title: title, Source: data.SourceUser, ProjectID: project, Status: data.Todo}); err == nil {
		m.addBuf = ""
	}
}

// foldedGroups is the last line: "Done 4 · Backlog 2 — show". Clicking opens both groups, and the
// line then reads "hide".
func (m *Main) foldedGroups(c *ui.Context, t *theme.Tokens, lm listModel) {
	if len(lm.done) == 0 && len(lm.backlog) == 0 {
		return
	}
	open := m.expanded[m.nav]
	if open {
		if len(lm.backlog) > 0 {
			m.group(c, t, data.Backlog, "Backlog", lm.backlog)
		}
		if len(lm.done) > 0 {
			m.group(c, t, data.Done, "Done", m.shownDone(lm))
			m.showAllDone(c, t, lm)
		}
	}
	m.foldLine(c, t, lm, open, 16)
}

func (m *Main) showAllDone(c *ui.Context, t *theme.Tokens, lm listModel) {
	if m.allDone[m.nav] || len(lm.done) <= theme.DoneRecentLimit {
		return
	}
	if link(c, t, "Show all "+itoa(len(lm.done))+" done").Margin(6, 0, 0, 24).Clicked() {
		m.allDone[m.nav] = true
	}
}

func (m *Main) foldLine(c *ui.Context, t *theme.Tokens, lm listModel, open bool, above float32) {
	var parts []string
	if len(lm.done) > 0 {
		parts = append(parts, "Done "+itoa(len(lm.done)))
	}
	if len(lm.backlog) > 0 {
		parts = append(parts, "Backlog "+itoa(len(lm.backlog)))
	}
	verb := "show"
	if open {
		verb = "hide"
	}
	line := ui.Row(c.Key("fold")).Height(theme.RowOneLine).Shrink(0).Margin(above, 0, 0).Gap(6).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Label(verb + " " + strings.Join(parts, " and "))
	line.Children(func() {
		theme.Text(c, theme.BodyStrong, t.TextTertiary, strings.Join(parts, " · "))
		verbColor := t.TextTertiary
		if line.Hovered() {
			verbColor = t.TextSecondary
		}
		theme.Text(c, theme.Body, verbColor, "— "+verb)
	})
	if line.Clicked() {
		m.expanded[m.nav] = !open
	}
}

// emptyList is a project or the Inbox with no task at all. It says what will appear and how to
// make it appear, and offers the input.
func (m *Main) emptyList(c *ui.Context, t *theme.Tokens) {
	title, body := "No tasks yet.", "Coding agents add tasks here through the adhd-todo MCP server. You can add one yourself."
	if m.nav.screen == screenInbox {
		title, body = "Your inbox is clear.", "Tasks without a project land here. Add one below, or press ⌘K on Now."
	}
	ui.Column(c).Padding(32, 0, 0).Gap(4).Shrink(0).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, title)
		theme.Paragraph(c, theme.Body, t.TextSecondary, body, 0)
		if m.nav.screen == screenProject {
			if link(c, t, "How to connect").Margin(4, 0, 0).Clicked() {
				m.settings = true
			}
		}
	})
	m.addTaskRow(c, t, 0)
}

// listHeader is the project's name and repository (or Inbox), the milestone menu, and the
// List | Board switch.
func (m *Main) listHeader(c *ui.Context, t *theme.Tokens) {
	title, sub := m.headerText()
	m.pageHeader(c, t, title, sub, func() {
		m.milestoneMenu(c, t)
		if m.nav.screen == screenProject {
			m.viewToggle(c, t)
		}
	})
}

func (m *Main) headerText() (title, sub string) {
	if m.nav.screen == screenInbox {
		return "Inbox", "Tasks without a project"
	}
	p, _ := m.app.Project(m.nav.projectID)
	return p.Name, p.RepoSlug
}
