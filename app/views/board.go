package views

import (
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// dragCard is the value a card carries while it is dragged between columns. A card of several
// selected ones takes them all: stack holds their ids (see stackKey).
type dragCard struct {
	id    int
	stack string
}

// lanes are the board's four fixed lanes. A blocked task sits in Doing, first; Backlog and
// Cancelled have no lane (the list shows them).
var lanes = []data.Status{data.Todo, data.Doing, data.Review, data.Done}

// boardColumn is one lane: every task in it, and the ones it draws.
type boardColumn struct {
	status data.Status
	tasks  []data.Task
	shown  []data.Task // the newest few of Done until "Show all"
}

// laneOf is the lane a status is drawn in.
func laneOf(s data.Status) data.Status {
	if s == data.Blocked {
		return data.Doing
	}
	return s
}

// boardColumns returns the four lanes. Doing lists its blocked tasks first; Done lists the newest
// five until "Show all" is clicked.
func (m *Main) boardColumns() []boardColumn {
	by := map[data.Status][]data.Task{}
	for _, t := range m.selection(m.app.withCompletingCards) {
		by[t.Status] = append(by[t.Status], t)
	}
	cols := make([]boardColumn, 0, len(lanes))
	for _, s := range lanes {
		tasks := by[s]
		switch s {
		case data.Done:
			sort.SliceStable(tasks, func(i, j int) bool { return doneAt(tasks[i]).After(doneAt(tasks[j])) })
		case data.Doing:
			sortForList(by[data.Blocked])
			sortForList(tasks)
			tasks = append(append([]data.Task(nil), by[data.Blocked]...), tasks...)
		default:
			sortForList(tasks)
		}
		col := boardColumn{status: s, tasks: tasks, shown: tasks}
		if s == data.Done && !m.allDone[m.nav] && len(tasks) > theme.DoneRecentLimit {
			col.shown = tasks[:theme.DoneRecentLimit]
		}
		cols = append(cols, col)
	}
	return cols
}

// boardActive is true when the project is shown as a board and has something to show; an empty
// project reads the same in both views.
func (m *Main) boardActive() bool { return m.showsBoard() && len(m.selectionTasks()) > 0 }

// boardScreen is the header and the composer in one block, and under them the lanes in a
// horizontal scroller. The block and the lanes share their left and right edges.
func (m *Main) boardScreen(c *ui.Context, t *theme.Tokens) {
	cols := m.boardColumns()
	lay := m.boardLayout(c)
	ui.Column(c).FillWidth().Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(lay.head).Children(func() {
			m.listHeader(c, t)
			ui.Box(c).Height(18).Shrink(0)
			m.pageComposer(c, t)
		})
	})
	ui.Box(c).Grow(1).MinHeight(0).Children(func() {
		ui.ScrollHorizontal(c).Fill().TrackScroll(&m.scroll).Children(func() {
			ui.Row(c).FillHeight().Padding(8, lay.edge, 0).Gap(theme.ColumnGap).AlignItems(ui.Stretch).
				Focusable().FocusRing(false).AutoFocus().Bind(&m.listRef).Children(func() {
				for _, col := range cols {
					m.column(c, t, col, lay.col)
				}
			})
		})
	})
}

// boardMetrics are the widths of the board: the header's block, the gap left of it and of the
// lanes, and a lane.
type boardMetrics struct{ head, edge, col float32 }

// boardLayout shares the room between the lanes so that they line up with the header and the
// composer above them.
func (m *Main) boardLayout(c *ui.Context) boardMetrics {
	return boardMetricsFor(m.contentWidth(c), m.sidePad(c))
}

// boardMetricsFor lays out the lanes in avail points with pad of room on each side: the block is
// as wide as the room, but no wider than BoardMax, and centered. When even the narrowest lanes do
// not fit, the block fills the room and the board scrolls sideways.
func boardMetricsFor(avail, pad float32) boardMetrics {
	room := avail - 2*pad
	n := float32(len(lanes))
	gaps := (n - 1) * theme.ColumnGap
	if room < n*theme.MinColumnWidth+gaps {
		return boardMetrics{head: room, edge: pad, col: theme.MinColumnWidth}
	}
	block := min(room, theme.BoardMax)
	return boardMetrics{head: block, edge: (avail - block) / 2, col: (block - gaps) / n}
}

// laneSpill is how far a lane's drop highlight reaches past its cards, so the cards keep the
// block's edges while a card is dragged over.
const laneSpill float32 = 8

// column is one lane: no box, only its header and its cards. A card dropped on it takes its
// status.
func (m *Main) column(c *ui.Context, t *theme.Tokens, col boardColumn, width float32) {
	box := ui.Column(c.Key(string(col.status))).Width(width+2*laneSpill).Margin(0, -laneSpill).Padding(0, laneSpill).
		Shrink(0).Radius(theme.RadiusPanel)
	if _, over := ui.DragOver[dragCard](box); over {
		box.Background(t.Accent.Alpha(0.07))
	}
	box.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	if d, ok := ui.Drop[dragCard](box); ok {
		m.dropCards(c, stackIDs(d.stack), col.status)
	}
	box.Children(func() {
		ui.Row(c.Key("header")).Height(theme.RowOneLine).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.StatusGlyph(c, t, col.status, 0)
			theme.Text(c, theme.BodyStrong, t.Text, statusLabel(m.say, col.status))
			theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(col.tasks)))
		})
		m.columnBody(c, t, col)
	})
}

// columnBody is the cards of a lane, or a quiet line saying what the empty lane is for.
func (m *Main) columnBody(c *ui.Context, t *theme.Tokens, col boardColumn) {
	ui.Scroll(c.Key("scroll")).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(8).Padding(8, 0, 16).Children(func() {
			if len(col.tasks) == 0 {
				theme.Paragraph(c, theme.Meta, t.TextTertiary, m.laneEmpty(col.status), 2)
			}
			for _, task := range col.shown {
				m.card(c, t, task)
			}
			if len(col.shown) < len(col.tasks) && link(c, t, m.say.BoardShowAll(len(col.tasks))).Margin(4, 0, 0).Clicked() {
				m.allDone[m.nav] = true
			}
		})
	})
}

// laneEmpty is what an empty lane says.
func (m *Main) laneEmpty(s data.Status) string {
	switch s {
	case data.Doing:
		return m.say.BoardEmptyDoing()
	case data.Review:
		return m.say.BoardEmptyReview()
	}
	return m.say.BoardEmptyNone()
}

// dropCards is the drop of cards on a lane: one card takes the lane's status; several selected ones
// all do, as one change with one Undo.
func (m *Main) dropCards(c *ui.Context, ids []int, to data.Status) {
	if len(ids) == 1 {
		m.dropCard(ids[0], to)
		return
	}
	tasks := only(m.tasksOf(ids), func(t data.Task) bool { return laneOf(t.Status) != to })
	title := m.say.BulkMoveToast(len(tasks), statusLabel(m.say, to))
	if to == data.Done {
		title = m.say.BulkDoneToast(len(tasks))
	}
	m.bulkUpdate(c, tasks, data.Patch{Status: data.StatusPtr(to)}, title)
}

func (m *Main) dropCard(id int, to data.Status) {
	task, ok := m.app.Task(id)
	if !ok || laneOf(task.Status) == to { // a blocked card dropped on Doing stays blocked
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
