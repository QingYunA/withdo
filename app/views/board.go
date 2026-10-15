package views

import (
	"sort"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// dragCard is the value a card carries while it is dragged between columns.
type dragCard struct{ id int }

// boardColumn is one status on the board.
type boardColumn struct {
	status data.Status
	tasks  []data.Task
}

// boardColumns returns the columns the board draws: those with tasks, and To do always, since it
// is where a new task goes. There are no strips and no zero-count columns. Backlog and Done follow
// the header's "Done · Backlog" button: folded away, they would push the live columns off the
// screen, and they stay reachable as drop targets.
func (m *Main) boardColumns() []boardColumn {
	by := map[data.Status][]data.Task{}
	for _, t := range m.selectionTasks() {
		by[t.Status] = append(by[t.Status], t)
	}
	var cols []boardColumn
	for _, s := range data.BoardStatuses {
		parked := s == data.Backlog || s == data.Done
		if (len(by[s]) == 0 && s != data.Todo) || (parked && !m.expanded[m.nav]) {
			continue
		}
		tasks := by[s]
		if s == data.Done {
			sort.SliceStable(tasks, func(i, j int) bool { return doneAt(tasks[i]).After(doneAt(tasks[j])) })
		} else {
			sortForList(tasks)
		}
		cols = append(cols, boardColumn{status: s, tasks: tasks})
	}
	return cols
}

// hiddenStatuses are the board statuses with no column, which show as drop targets while a card
// is dragged.
func hiddenStatuses(cols []boardColumn) []data.Status {
	shown := map[data.Status]bool{}
	for _, col := range cols {
		shown[col.status] = true
	}
	var out []data.Status
	for _, s := range data.BoardStatuses {
		if !shown[s] {
			out = append(out, s)
		}
	}
	return out
}

// boardActive is true when the project is shown as a board and has something to show; an empty
// project reads the same in both views.
func (m *Main) boardActive() bool { return m.showsBoard() && len(m.selectionTasks()) > 0 }

// boardScreen is the header and the composer in one block, and under them the columns in a
// horizontal scroller. The block and the columns share their left and right edges.
func (m *Main) boardScreen(c *ui.Context, t *theme.Tokens) {
	cols := m.boardColumns()
	hidden := hiddenStatuses(cols)
	lay := m.boardLayout(c, len(cols))
	ui.Column(c).FillWidth().Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(lay.head).Children(func() {
			m.listHeader(c, t)
			ui.Box(c).Height(18).Shrink(0)
			m.pageComposer(c, t)
		})
	})
	// The drop targets float over the scroller's right edge, so they are in reach however far
	// the board is scrolled.
	ui.Box(c).Grow(1).MinHeight(0).Children(func() {
		ui.ScrollHorizontal(c).Fill().TrackScroll(&m.scroll).Children(func() {
			ui.Row(c).FillHeight().Padding(16, lay.edge, 0).Gap(theme.ColumnGap).AlignItems(ui.Stretch).
				Focusable().FocusRing(false).AutoFocus().Bind(&m.listRef).Children(func() {
				for _, col := range cols {
					m.column(c, t, col, lay.col)
				}
			})
		})
		m.dropTargets(c, t, hidden)
	})
}

// boardMetrics are the widths of the board: the header's block, the gap left of it and of the
// columns, and a column.
type boardMetrics struct{ head, edge, col float32 }

// boardLayout shares the room between n columns so that they line up with the header and the
// composer above them. The block is as wide as the columns want to be (ColumnWidth each) but at
// least as wide as the lists' column, and no wider than the window less its margins; it is centered.
// When even the narrowest columns do not fit, the board scrolls and the block fills the window.
func (m *Main) boardLayout(c *ui.Context, n int) boardMetrics {
	avail := m.contentWidth(c)
	return boardMetricsFor(avail, min(avail, theme.ContentMax)-2*m.sidePad(c), n)
}

// boardMetricsFor lays out n columns in avail points, given the width of the lists' column.
func boardMetricsFor(avail, listW float32, n int) boardMetrics {
	room := avail - 2*theme.BoardPadding
	if n == 0 {
		return boardMetrics{head: room, edge: theme.BoardPadding, col: theme.ColumnWidth}
	}
	gaps := float32(n-1) * theme.ColumnGap
	block := min(max(float32(n)*theme.ColumnWidth+gaps, listW), room)
	if need := float32(n)*theme.MinColumnWidth + gaps; block < need {
		return boardMetrics{head: room, edge: theme.BoardPadding, col: theme.MinColumnWidth}
	}
	return boardMetrics{head: block, edge: (avail - block) / 2, col: (block - gaps) / float32(n)}
}

// boardFold is the list's last line, in the board's header: "Done 5 · Backlog 2". It shows or
// hides those two columns.
func (m *Main) boardFold(c *ui.Context, t *theme.Tokens) {
	lm := m.listModel()
	parts := m.foldCounts(lm, true)
	if len(parts) == 0 {
		return
	}
	open := m.expanded[m.nav]
	text, label := strings.Join(parts, " · "), m.say.FoldShow(m.foldParts(parts))
	ink := t.TextTertiary
	if open {
		label, ink = m.say.FoldHide(m.foldParts(parts)), t.TextSecondary
	}
	btn := ui.ButtonBase(c.Key("fold")).Height(28).Padding(0, 8).Radius(theme.RadiusControl).Shrink(0).AlignItems(ui.Center).
		Label(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.TextSecondary
	}
	btn.Children(func() { theme.Text(c, theme.Meta, ink, text) })
	if btn.Clicked() {
		m.expanded[m.nav] = !open
	}
}

// column is one status. A card dropped on it takes its status.
func (m *Main) column(c *ui.Context, t *theme.Tokens, col boardColumn, width float32) {
	box := ui.Column(c.Key(string(col.status))).Width(width).Shrink(0).Padding(0, 8).Radius(theme.RadiusPanel).Background(t.BgSubtle)
	_, over := ui.DragOver[dragCard](box)
	if over {
		box.Border(1, t.Accent.Alpha(0.6))
	}
	if d, ok := ui.Drop[dragCard](box); ok {
		m.dropCard(d.id, col.status)
	}
	box.Children(func() {
		ui.Row(c.Key("header")).Height(40).Shrink(0).Padding(0, 0, 0, 6).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.StatusGlyph(c, t, col.status, 0)
			theme.Text(c, theme.BodyStrong, t.Text, statusLabel(m.say, col.status))
			theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(col.tasks)))
			if col.status == data.Todo {
				ui.Spacer(c)
				m.boardPlus(c, t)
			}
		})
		m.columnBody(c, t, col)
	})
}

func (m *Main) columnBody(c *ui.Context, t *theme.Tokens, col boardColumn) {
	tasks, hidden := col.tasks, 0
	if col.status == data.Done && !m.allDone[m.nav] && len(tasks) > theme.DoneRecentLimit {
		hidden = len(tasks) - theme.DoneRecentLimit
		tasks = tasks[:theme.DoneRecentLimit]
	}
	ui.Scroll(c.Key("scroll")).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(8).Padding(0, 0, 12).Children(func() {
			if len(tasks) == 0 && col.status == data.Todo {
				theme.Paragraph(c, theme.Meta, t.TextTertiary, m.say.BoardEmpty(), 3).Padding(2, 6, 0)
			}
			for _, task := range tasks {
				m.card(c, t, task)
			}
			if hidden > 0 && link(c, t, m.say.BoardShowAll(hidden+len(tasks))).Margin(4, 0, 0, 6).Clicked() {
				m.allDone[m.nav] = true
			}
		})
	})
}

// dropTargets are targets for the statuses that have no column. They appear while a card is
// dragged, in a strip at the bottom of the board, where the columns are usually empty, so they
// cover neither a column's header nor its cards.
func (m *Main) dropTargets(c *ui.Context, t *theme.Tokens, hidden []data.Status) {
	if (m.dragging == 0 && m.wasDrag == 0) || len(hidden) == 0 {
		return
	}
	ui.Row(c.Key("drop-targets")).Absolute().Left(0).Right(0).Bottom(16).Justify(ui.Center).PassThrough().Children(func() {
		ui.Row(c).Padding(8).Gap(8).Radius(theme.RadiusPanel).Background(t.Bg).Border(1, t.Border).
			Shadow(0, 8, 24, 0, t.Shadow).Children(func() {
			for _, s := range hidden {
				target := ui.Row(c.Key("drop-"+string(s))).Height(40).Shrink(0).Padding(0, 14).Gap(8).AlignItems(ui.Center).
					Radius(theme.RadiusCard).Border(1, t.Border).BorderStyle(ui.BorderDashed)
				if _, over := ui.DragOver[dragCard](target); over {
					target.Background(t.BgSelected).Border(1, t.Accent.Alpha(0.6))
				}
				if d, ok := ui.Drop[dragCard](target); ok {
					m.dropCard(d.id, s)
				}
				target.Children(func() {
					theme.StatusGlyph(c, t, s, 0)
					theme.Text(c, theme.Body, t.TextSecondary, statusLabel(m.say, s))
				})
			}
		})
	})
}

// boardPlus is the + in the To do column's header: it puts the cursor in the composer.
func (m *Main) boardPlus(c *ui.Context, t *theme.Tokens) {
	tip := m.say.BoardAdd()
	btn := ui.ButtonBase(c.Key("board-plus")).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().Radius(theme.RadiusControl).
		Label(tip).Tooltip(tip).TextColor(t.TextTertiary).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover).TextColor(t.TextSecondary)
	}
	btn.Children(func() { ui.Icon(c, theme.IconPlus).Size(14, 14) })
	if btn.Clicked() {
		m.focusPageComposer(c)
	}
}

func (m *Main) dropCard(id int, to data.Status) {
	task, ok := m.app.Task(id)
	if !ok || task.Status == to {
		return
	}
	m.app.SetStatus(task, to)
}

func doneAt(t data.Task) time.Time {
	if t.CompletedAt != nil {
		return *t.CompletedAt
	}
	return t.UpdatedAt
}
