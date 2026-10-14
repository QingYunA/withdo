package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// forcedOpen is the set of columns that stay in full whatever the width: the ones the user opened,
// the one a task is being added to, and the one whose task is open in the inspector.
func (m *Main) forcedOpen() map[data.Status]bool {
	out := map[data.Status]bool{}
	for s, on := range m.pinned[m.nav] {
		out[s] = on
	}
	if m.adding != "" && m.addNav == m.nav {
		out[m.adding] = true
	}
	if task, ok := m.app.Task(m.openID); ok {
		out[task.Status] = true
	}
	return out
}

func (m *Main) pin(s data.Status, on bool) {
	if m.pinned[m.nav] == nil {
		m.pinned[m.nav] = map[data.Status]bool{}
	}
	m.pinned[m.nav][s] = on
}

func (m *Main) isAdding(s data.Status) bool { return m.adding == s && m.addNav == m.nav }

// column is one status. A card dropped on it takes its status, also when it is a strip, which
// opens while a card is dragged over it.
func (m *Main) column(c *ui.Context, t *theme.Tokens, s data.Status, tasks []data.Task, plan columnPlan) {
	col := ui.Column(c.Key(string(s))).Shrink(0).Radius(theme.RadiusPanel + 2)
	_, over := ui.DragOver[dragCard](col)
	strip := !plan.expanded && !over
	switch {
	case strip:
		col.Width(theme.StripWidth)
	case plan.expanded:
		col.Width(plan.width).Padding(0, 8)
	default:
		col.Width(theme.ColumnWidth).Padding(0, 8)
	}
	switch {
	case over:
		col.Background(t.BgSubtle)
	case strip && col.Hovered():
		col.Background(t.BgHover)
	}
	col.Transition(ui.ElementTransition{Size: true, Colors: true, Duration: theme.MotionFast})
	if d, ok := ui.Drop[dragCard](col); ok {
		m.dropCard(d.id, s)
	}
	if strip {
		col.Cursor(ui.CursorPointer).Tooltip(stripTip(s, len(tasks)))
		col.Children(func() { m.stripBody(c, t, s, len(tasks)) })
		if col.Clicked() {
			m.pin(s, true)
		}
		return
	}
	col.Children(func() {
		m.columnHeader(c, t, s, len(tasks))
		m.columnBody(c, t, s, tasks, over)
	})
}

func stripTip(s data.Status, n int) string {
	if n == 0 {
		return s.Label() + ": " + hintFor(s)
	}
	return fmt.Sprintf("%s: %d. Click to open", s.Label(), n)
}

func (m *Main) dropCard(id int, to data.Status) {
	task, ok := m.app.Task(id)
	m.dragging = 0
	if !ok || task.Status == to {
		return
	}
	m.app.SetStatus(task, to)
}

// stripBody is a collapsed column: its glyph, its name turned on its side, its count, and a quiet
// plus to add a task (always there for Todo and Backlog, on hover for the rest).
func (m *Main) stripBody(c *ui.Context, t *theme.Tokens, s data.Status, n int) {
	ink := t.TextSecondary
	if n == 0 {
		ink = t.TextTertiary
	}
	ui.Column(c).FillWidth().Padding(12, 0, 8).Gap(10).AlignItems(ui.Center).Children(func() {
		theme.StatusGlyph(c, t, s, 0)
		sideways(c, theme.BodyStrong, ink, s.Label())
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n))
		ui.Spacer(c)
		if s != data.Done {
			m.plusButton(c, t, s, alwaysPlus(s))
		}
	})
}

func alwaysPlus(s data.Status) bool { return s == data.Todo || s == data.Backlog }

// sideways draws text reading downward in a box 16 pt wide.
func sideways(c *ui.Context, style theme.Style, color ui.Color, text string) {
	const length, thickness = 56, 16
	ui.Box(c).Size(thickness, length).Shrink(0).Children(func() {
		theme.Text(c, style, color, text).Absolute().Left((thickness - length) / 2).Top((length - thickness) / 2).
			Width(length).TextAlign(ui.Center).Rotate(90).PassThrough()
	})
}

// plusButton starts adding a task to a column. Hidden, it keeps its room so nothing shifts.
func (m *Main) plusButton(c *ui.Context, t *theme.Tokens, s data.Status, show bool) {
	if !show && !m.isAdding(s) {
		ui.Box(c).Size(theme.HitTarget, theme.HitTarget).Shrink(0)
		return
	}
	btn := ui.ButtonBase(c.Key("add-"+string(s))).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().
		Radius(theme.RadiusControl).Label("Add task to " + s.Label()).Tooltip("Add a task").TextColor(t.TextSecondary)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() { ui.Icon(c, theme.IconPlus).Size(14, 14) })
	if btn.Clicked() {
		m.startAdding(s)
	}
}

func (m *Main) columnHeader(c *ui.Context, t *theme.Tokens, s data.Status, n int) {
	hdr := ui.Row(c.Key("header")).Height(40).Shrink(0).Padding(0, 0, 0, 4).Gap(8).AlignItems(ui.Center)
	hover := hdr.Hovered()
	hdr.Children(func() {
		theme.StatusGlyph(c, t, s, 0)
		name := theme.Text(c, theme.BodyStrong, t.Text, s.Label())
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n))
		// A column the user opened closes again with a click on its name.
		if m.pinned[m.nav][s] && !m.hasOpenTask(s) {
			name.Cursor(ui.CursorPointer).Tooltip("Collapse")
			if name.Clicked() {
				m.pin(s, false)
			}
		}
		ui.Spacer(c)
		if s != data.Done {
			m.plusButton(c, t, s, hover || alwaysPlus(s))
		}
	})
}

func (m *Main) hasOpenTask(s data.Status) bool {
	task, ok := m.app.Task(m.openID)
	return ok && task.Status == s
}

func (m *Main) columnBody(c *ui.Context, t *theme.Tokens, s data.Status, tasks []data.Task, over bool) {
	hidden := 0
	if s == data.Done {
		sort.SliceStable(tasks, func(i, j int) bool { return doneAt(tasks[i]).After(doneAt(tasks[j])) })
		if !m.showAllDone[m.nav] && len(tasks) > theme.DoneRecentLimit {
			hidden = len(tasks) - theme.DoneRecentLimit
			tasks = tasks[:theme.DoneRecentLimit]
		}
	} else {
		byPriorityThenRecent(tasks)
	}
	ui.Scroll(c.Key("scroll")).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(8).Padding(0, 0, 16).Children(func() {
			if m.isAdding(s) {
				m.addRow(c, t, s)
			}
			if len(tasks) == 0 && !over && !m.isAdding(s) {
				columnHint(c, t, s)
			}
			for _, task := range tasks {
				m.card(c, t, task)
			}
			if hidden > 0 {
				if link(c, t, fmt.Sprintf("Show all %d", hidden+len(tasks))).Margin(4, 0, 0, 4).Clicked() {
					m.showAllDone[m.nav] = true
				}
			}
		})
	})
}

// startAdding opens the inline title input at the top of a column.
func (m *Main) startAdding(s data.Status) {
	m.adding, m.addNav, m.addBuf = s, m.nav, ""
}

func (m *Main) cancelAdding() { m.adding, m.addBuf = "", "" }

// addRow is the inline input: Enter creates a task with the column's status in the current
// project (or the inbox), Escape cancels.
func (m *Main) addRow(c *ui.Context, t *theme.Tokens, s data.Status) {
	ui.Row(c.Key("add-row")).Gap(4).Padding(4, 12, 4, 8).Radius(theme.RadiusCard).Background(t.Bg).
		Border(1, t.Accent.Alpha(0.6)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().Children(func() {
			theme.StatusGlyph(c, t, s, 0)
		})
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			in := theme.BodyStrong.Apply(ui.TextInputBase(c.Key("add-input"), &m.addBuf)).TextColor(t.Text).
				Label("New task title").AutoFocus()
			if in.Submitted() {
				m.submitAdding(s)
			}
			if m.addBuf == "" {
				theme.Text(c, theme.BodyStrong, t.TextTertiary, "Task title…").Absolute().Left(0).Top(0).PassThrough()
			}
		})
	})
}

func (m *Main) submitAdding(s data.Status) {
	title := strings.TrimSpace(m.addBuf)
	if title == "" {
		return
	}
	var project *int
	if !m.nav.inbox {
		project = data.IntPtr(m.nav.projectID)
	}
	if err := m.app.AddTask(data.CreateTaskInput{Title: title, Source: data.SourceUser, ProjectID: project, Status: s}); err == nil {
		m.cancelAdding()
	}
}
