package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// listIndent is how far a group's rows sit in from its header: under the header's name, past its
// glyph and the 8 pt gap. A row's title is as far in again from its avatar.
const listIndent = theme.GlyphSize + 8

// selection is selectionTasks with its own way to show the tasks that are completing.
func (m *Main) selection(completing func([]data.Task, func(data.Task) bool) []data.Task) []data.Task {
	in := page.Scope{Inbox: m.nav.screen == screenInbox, Project: m.nav.projectID, Milestone: m.milestone[m.nav.projectID]}.Has
	var out []data.Task
	for _, t := range m.app.snap.tasks {
		if in(t) {
			out = append(out, t)
		}
	}
	return completing(out, in)
}

// listScreen is the Inbox, or an empty project, as a list grouped by status. A project with tasks is
// two columns (columnsScreen) or a board instead.
func (m *Main) listScreen(c *ui.Context, t *theme.Tokens) {
	m.listHeader(c, t)
	ui.Box(c).Height(18).Shrink(0)
	if !m.app.snap.loaded {
		return
	}
	m.belowComposer(c, func() {
		if m.pg.Cols[0].Total == 0 {
			m.emptyList(c, t)
			return
		}
		m.sections(c, t, sideMine, m.pg.Sections, func(_ page.Section, task data.Task) { m.listRow(c, t, task) })
	})
}

// sections draws the sections of a list or a column: a header and its rows for each status, the Done
// group with "Show all" under it while the fold hides rows, and Backlog folded into its last line. row
// draws a task of a section.
func (m *Main) sections(c *ui.Context, t *theme.Tokens, s side, secs []page.Section, row func(page.Section, data.Task)) {
	expanded, allDone := m.foldMaps(s)
	for _, sec := range secs {
		draw := func(task data.Task) { row(sec, task) }
		switch {
		case sec.Kind == page.SecWaiting:
			m.waitingGroup(c, t, sec.Tasks)
		case sec.Status == data.Backlog:
			if sec.Open {
				m.groupOf(c, t, sec.Status, sec.Tasks, draw)
			}
			if m.foldLine(c, t, sec.Total, sec.Open, 16) {
				expanded[m.nav] = !sec.Open
			}
		default:
			m.groupOf(c, t, sec.Status, sec.Tasks, draw)
			if sec.Hidden() > 0 && link(c, t, m.say.ListShowAllDone(sec.Total)).Margin(6, 0, 0, 2*listIndent).Clicked() {
				allDone[m.nav] = true
			}
		}
	}
}

// groupOf is a header, glyph and name and count, and its rows indented under the name: a row's
// avatar starts where the header's name does, so the header reads as the parent of its rows.
func (m *Main) groupOf(c *ui.Context, t *theme.Tokens, s data.Status, tasks []data.Task, row func(data.Task)) {
	m.section(c, "section-"+string(s), tasks, func() {
		groupHeader(c, t, "group-"+string(s), len(tasks), statusLabel(m.say, s), func() { theme.StatusGlyph(c, t, s, 0) })
		ui.Column(c.Key("rows-"+string(s))).Shrink(0).Margin(0, 0, 0, listIndent).Children(func() {
			for _, task := range tasks {
				row(task)
			}
		})
	})
}

// groupHeader is a glyph, a name and a count.
func groupHeader(c *ui.Context, t *theme.Tokens, key string, n int, name string, glyph func()) {
	ui.Row(c.Key(key)).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		glyph()
		theme.Text(c, theme.BodyStrong, t.Text, name)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n))
	})
}

// listRow is one task: title, branch and pull request, a summary line for agent work, and on the
// right when it is due, how long ago it moved, or that it went stale.
func (m *Main) listRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	if task.Status == data.Done {
		m.finishedRow(c, t, task, finishedOpts{when: m.clock.Day(doneAt(task), m.app.Now())})
		return
	}
	ph := m.rowPhase(c, task)
	ot := m.app.overviewTask(task)
	row := m.rowShell(c, t, task.ID, plainRow, ph).AlignItems(ui.Center)
	row.Children(func() {
		m.groupGlyph(c, t, row, task, ph, false)
		m.authorSlot(c, t, task) // under the group's name
		body(c, func() {
			m.titleLine(c, t, task, "", false)
			summaryLine(c, t, listMeta(task, ot))
		})
		right(c, func() { m.agentRight(c, t, row, task, ot, ph) })
	})
	if m.rowClicked(row, task.ID) {
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
		said = notTheTitle(task, ot.LatestReport.Summary)
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
	if !task.Status.Terminal() {
		priorityMark(c, t, task)
	}
	m.rowThumb(c, t, task.ID)
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

// foldLine is the last line of a list, which folds Backlog: "Backlog 2 — show". It reports a click.
func (m *Main) foldLine(c *ui.Context, t *theme.Tokens, n int, open bool, above float32) bool {
	count := m.say.FoldCount(statusLabel(m.say, data.Backlog), n)
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
	return line.Clicked()
}

// emptyList is a project or the Inbox with no task at all. It says what will appear and how to
// make it appear; the composer above is where to write one.
func (m *Main) emptyList(c *ui.Context, t *theme.Tokens) {
	title, body := m.say.ListEmptyTitle(), m.say.ListEmptyBody()
	if m.nav.screen == screenInbox {
		title, body = m.say.InboxEmptyTitle(), m.say.InboxEmptyBody(keyHint("⌘K"))
	}
	emptyState(c, t, title, body, 56, func() {
		if m.nav.screen == screenProject {
			if link(c, t, m.say.LinkHowToConnect()).Margin(4, 0, 0).Clicked() {
				m.openSettings(PaneAgents)
			}
		}
	})
}

// listTitle is the page's title. A project's name is an input once a double click starts renaming
// it, with the same keys as in the sidebar; Inbox stays as it is.
func (m *Main) listTitle(c *ui.Context, t *theme.Tokens, title string) {
	if m.nav.screen != screenProject {
		theme.Text(c, theme.Title, t.Text, title).Shrink(0)
		return
	}
	id := m.nav.projectID
	if r := m.manage.rename; r.id == id && r.page {
		m.renameInput(c, t, theme.Title)
		return
	}
	box := ui.Box(c.Key("page-title")).Shrink(0)
	box.Children(func() { theme.Text(c, theme.Title, t.Text, title) })
	if box.DoubleClicked() {
		m.startPageRename(id, title)
	}
}

// listHeader is the project's name, or Inbox, with the milestone filter (once the project has a
// milestone) and the List | Board switch at the right.
func (m *Main) listHeader(c *ui.Context, t *theme.Tokens) {
	title := m.say.NavInbox()
	if m.nav.screen == screenProject {
		p, _ := m.app.Project(m.nav.projectID)
		title = p.Name
	}
	m.pageHeaderOf(c, t, func() { m.listTitle(c, t, title) }, "", func() {
		m.recentLink(c, t)
		m.milestoneMenu(c, t)
		if m.nav.screen == screenProject {
			m.viewToggle(c, t)
		}
	})
}
