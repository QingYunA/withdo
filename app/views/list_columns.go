package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// A project with tasks is drawn as two columns under its header and composer: Mine on the left (the
// tasks the user wrote) and Agents on the right, on a faint tray (the tasks agents wrote, and the ones
// the user handed over that wait for an agent). The Inbox and the board are not drawn this way.

// side is one of the two columns.
type side int

const (
	sideMine side = iota
	sideAgents
)

// trayRing is the width of the ring the tray shows while a task is dragged over it. It is always
// there, transparent, so the tray does not change size; Mine keeps the same room at the top.
const trayRing float32 = 1.5

// trayPadBottom is the room under the tray's last row.
const trayPadBottom float32 = 14

// stackGap is the room between the two columns once they are stacked.
const stackGap float32 = 14

// columnsState is what the columns remember: the Agents column's own fold and Done state (Mine
// uses the list's), and where things were drawn the frame before, for the hand-off's motion.
type columnsState struct {
	expanded, allDone map[nav]bool // the Agents column's Backlog opened, and every Done task shown
	geo               columnsGeo
}

// columnsGeo are the rectangles of the frame before, in window points.
type columnsGeo struct {
	mine, tray ui.Rect // the Mine column and the Agents tray
	stacked    bool
}

// foldMaps are a column's "Backlog opened" and "every Done shown" states.
func (m *Main) foldMaps(s side) (expanded, allDone map[nav]bool) {
	if s == sideMine {
		return m.expanded, m.allDone
	}
	if m.cols.expanded == nil {
		m.cols.expanded, m.cols.allDone = map[nav]bool{}, map[nav]bool{}
	}
	return m.cols.expanded, m.cols.allDone
}

// columnMetrics are the widths of the block (header, composer and columns) and of a column.
type columnMetrics struct {
	block, col float32
	stacked    bool
}

// columnMetrics is as wide as the room allows, but no wider than the board, and stacked when the
// content is narrower than ColumnsStackBelow.
func (m *Main) columnMetrics(c *ui.Context) columnMetrics {
	block := min(m.width-2*m.sidePad(c), theme.BoardMax)
	if m.width < theme.ColumnsStackBelow {
		return columnMetrics{block: block, col: block, stacked: true}
	}
	return columnMetrics{block: block, col: (block - theme.ColumnsGap) / 2}
}

// columnsScreen is the header and the composer, and under them Mine and Agents.
func (m *Main) columnsScreen(c *ui.Context, t *theme.Tokens) {
	m.listHeader(c, t)
	ui.Box(c).Height(18).Shrink(0)
	if !m.app.snap.loaded {
		return
	}
	mt := m.columnMetrics(c)
	m.cols.geo.stacked = mt.stacked
	m.belowComposer(c, func() {
		both := func() {
			m.mineColumn(c, t, mt.col)
			m.agentsColumn(c, t, mt.col)
		}
		if mt.stacked {
			ui.Column(c.Key("columns")).Margin(6, 0, 0).Gap(stackGap).Shrink(0).Children(both)
			return
		}
		ui.Row(c.Key("columns")).Margin(6, 0, 0).Gap(theme.ColumnsGap).AlignItems(ui.Stretch).Shrink(0).Children(both)
	})
}

// mineColumn is the user's tasks. Dropping an agent's row on it takes that task back.
func (m *Main) mineColumn(c *ui.Context, t *theme.Tokens, width float32) {
	col := ui.Column(c.Key("col-mine")).Width(width).Shrink(0).Padding(trayRing, 0, 0).Radius(theme.RadiusPanel)
	if d, over := ui.DragOver[dragRow](col); over && d.fromAgents {
		col.Background(t.Accent.Alpha(0.07))
	}
	col.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	if d, ok := ui.Drop[dragRow](col); ok && d.fromAgents {
		m.takeBack(c, d.id)
	}
	col.Children(func() {
		m.columnHead(c, t, sideMine, m.pg.Cols[sideMine].Active, t.Border)
		if m.pg.Cols[sideMine].Total == 0 {
			quietLine(c, t, m.say.ListMineEmpty(), nil)
			return
		}
		m.sideBody(c, t, sideMine)
	})
	m.cols.geo.mine = col.Bounds()
}

// agentsColumn is the agents' tasks on the tray. Dropping one of the user's rows on it hands that
// task over.
func (m *Main) agentsColumn(c *ui.Context, t *theme.Tokens, width float32) {
	zone := ui.Column(c.Key("col-agents")).Width(width).Shrink(0)
	d, over := ui.DragOver[dragRow](zone)
	over = over && !d.fromAgents
	m.ho.setOver(over, m.app.Now())
	if d, ok := ui.Drop[dragRow](zone); ok && !d.fromAgents {
		if d.n > 1 {
			m.handOffMany(c, stackIDs(d.stack))
		} else {
			m.handOff(c, d.id)
		}
	}
	zone.Children(func() {
		tray := ui.Column(c.Key("tray")).Padding(0, theme.TrayPad-trayRing, trayPadBottom-trayRing).
			Radius(theme.TrayRadius).Background(t.BgTray).Border(trayRing, ui.Transparent)
		if over {
			tray.Background(t.BgTray.Mix(t.Accent, 0.07)).Border(trayRing, t.Accent.Alpha(0.35))
		}
		tray.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
		tray.Children(func() {
			m.columnHead(c, t, sideAgents, m.pg.Cols[sideAgents].Active, t.Text.Alpha(0.06))
			if over {
				line := m.say.ListDropToHand()
				if d.n > 1 {
					line = m.say.ListDropToHandMany(d.n)
				}
				theme.Text(c, theme.Meta, t.Accent, line).Margin(12, 0, 0, 2)
			}
			if m.pg.Cols[sideAgents].Total == 0 {
				m.agentsEmpty(c, t)
				return
			}
			m.sideBody(c, t, sideAgents)
		})
		m.cols.geo.tray = tray.Bounds()
	})
}

// columnHead is the column's avatar, name and count, and a line under them.
func (m *Main) columnHead(c *ui.Context, t *theme.Tokens, s side, n int, line ui.Color) {
	name := m.say.NavMine()
	if s == sideAgents {
		name = m.say.NavAgents()
	}
	ui.Column(c.Key("head")).Shrink(0).Margin(10, 0, 0).Children(func() {
		ui.Row(c).Height(theme.ColumnHead - 1).Gap(9).AlignItems(ui.Center).Children(func() {
			if s == sideMine {
				theme.AvatarIcon(c.Key("avatar"), theme.You, theme.ColumnAvatar)
			} else {
				m.agentAvatar20(c, t)
			}
			theme.Text(c, theme.BodyStrong, t.Text, name)
			if n > 0 {
				theme.Text(c, theme.Meta, t.TextTertiary, itoa(n))
			}
		})
		ui.Box(c).FillWidth().Height(1).Background(line)
	})
}

// agentAvatar20 is the robot of the Agents header. It hops while a row of Mine is over the tray and
// squashes as it catches the row dropped on it (see agentFrame).
func (m *Main) agentAvatar20(c *ui.Context, t *theme.Tokens) {
	sprite := theme.AccentBot(t.Accent).Sprite()
	ui.Box(c.Key("avatar")).Size(theme.ColumnAvatar, theme.ColumnAvatar).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		theme.DrawSprite(p, m.agentFrame(c.Preferences().ReduceMotion, r), sprite)
	})
}

// agentsEmpty is the Agents column of a project no agent has reported in.
func (m *Main) agentsEmpty(c *ui.Context, t *theme.Tokens) {
	quietLine(c, t, m.say.ListAgentsEmpty(), func() {
		if link(c, t, m.say.ListInstallMCP()).Clicked() {
			m.openSettings(PaneAgents)
		}
	})
}

// quietLine is one line in text.tertiary under a column's header, and an optional link after it.
func quietLine(c *ui.Context, t *theme.Tokens, text string, after func()) {
	ui.Row(c).Shrink(0).Margin(18, 0, 0, 2).Gap(6).AlignItems(ui.Start).Children(func() {
		theme.Paragraph(c, theme.Meta, t.TextTertiary, text, 0).Shrink(1).MinWidth(0)
		if after != nil {
			after()
		}
	})
}

// sideBody is a column's sections: Waiting for an agent first (Agents only), then the statuses, then
// Done and the folded Backlog.
func (m *Main) sideBody(c *ui.Context, t *theme.Tokens, s side) {
	m.sections(c, t, s, m.pg.Of(int(s)), func(sec page.Section, task data.Task) {
		if sec.Status == data.Done {
			m.doneRow(c, t, s, task)
			return
		}
		m.rowOf(c, t, s, task)
	})
}

// waitingGroup is the tasks handed to an agent that has not reported on them yet.
func (m *Main) waitingGroup(c *ui.Context, t *theme.Tokens, tasks []data.Task) {
	if len(tasks) == 0 {
		return
	}
	groupHeader(c, t, "group-waiting", len(tasks), m.say.ListWaiting(), func() { theme.ParcelGlyph(c) })
	ui.Column(c.Key("rows-waiting")).Shrink(0).Margin(0, 0, 0, listIndent).Children(func() {
		for _, task := range tasks {
			m.waitingRow(c, t, task)
		}
	})
}

// rowOf is a row of the column.
func (m *Main) rowOf(c *ui.Context, t *theme.Tokens, s side, task data.Task) {
	if s == sideMine {
		m.myRow(c, t, task)
		return
	}
	m.agentRow(c, t, task)
}
