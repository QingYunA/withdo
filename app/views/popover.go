package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/words"
	"github.com/egoist/mygo/ui"
)

// The menu bar panel: a pixel icon in the menu bar (macOS) or the tray icon (Windows), or Option+Space
// anywhere, opens it. It is the composer of Mine, focused, with today's first rows under it and one
// line about the agents. It is its own Main: the same composer, with its own draft, reply and
// receipts, filing into Mine, and no window around it.

// Panel sizes in points. The panel is as wide as the mockup (v31) draws it and as tall as its
// content: the shell resizes the window when the height changes.
const (
	PopoverWidth float32 = 360

	popoverPad     float32 = 12
	popoverRowH    float32 = 32 // a row of today: 18 pt glyph, 7 pt above and below
	popoverHead    float32 = 34 // "Today 3", with 12 pt above it
	popoverFoot    float32 = 41 // hairline, 8 pt above it, the line
	popoverReply   float32 = 150
	popoverLine    float32 = 28 // the quiet line under the composer: "Added to Inbox", "Set up the assistant"
	popoverRowsMax         = 5
)

// Popover is the content of the menu bar panel.
type Popover struct {
	m *Main
	// Close is set by the shell and closes the panel; Resize tells it the height the content wants,
	// in points, when that changes.
	Close  func()
	Resize func(height int)

	height int
}

// NewPopover makes the panel's content over the app. It reads and writes what the window does.
func NewPopover(a *App) *Popover {
	p := &Popover{m: NewMain(a)}
	p.m.closePanel = func() {
		if p.Close != nil {
			p.Close()
		}
	}
	p.m.pg = page.Page{Kind: page.KindMine, Composer: true}
	return p
}

// Height is the height the content asked for last, in points; 0 before it has been drawn.
func (p *Popover) Height() int { return p.height }

// Show readies the panel for opening: the composer takes the keyboard once it is drawn.
func (p *Popover) Show() {
	p.m.focusComposer, p.m.cmp[surfacePage].frames = true, 0
	p.m.app.Refresh()
}

// popoverRows are the rows of today the panel lists: the first five, in the order Mine shows them.
func popoverRows(today []data.Item) []data.Item {
	return today[:min(len(today), popoverRowsMax)]
}

// agentCounts are the agents' state for the panel's footer: how many tasks are being worked on and
// how many are stuck (blocked: they wait for me).
func agentCounts(a data.AgentsScreen) (working, stuck int) {
	for _, r := range a.Working {
		if r.Status == data.Blocked {
			stuck++
		} else {
			working++
		}
	}
	return working, stuck
}

// popoverHeight is the height the panel wants: the composer, the rows of today with their header,
// room for what shows under the composer (a reply, or a quiet line), and the footer.
func popoverHeight(rows int, reply, line bool) float32 {
	h := 2*popoverPad + theme.ComposerHeight + popoverFoot
	if rows > 0 {
		h += popoverHead + float32(rows)*popoverRowH
	}
	switch {
	case reply:
		h += popoverReply
	case line:
		h += popoverLine
	}
	return h
}

// View builds the panel for one frame.
func (p *Popover) View(c *ui.Context) {
	m := p.m
	t := theme.Resolve(c)
	m.say, m.clock = locales.In(c.Locale()), words.New(c.Locale(), c.Region())
	m.weekStart = firstWeekday(c)
	m.app.setWeekStart(m.weekStart)
	m.width = PopoverWidth
	m.mine, m.agents = m.app.buildMine(), m.app.buildAgents()
	m.dropStaleReply()

	rows := popoverRows(m.mine.Today)
	reply := m.replyIn(surfacePage, 0)
	cs := &m.cmp[surfacePage]
	line := (m.app.Assistant != nil && !m.assistReady()) || (!cs.noticeAt.IsZero() && m.app.Now().Sub(cs.noticeAt) < noticeFor)
	h := int(popoverHeight(len(rows), reply, line))
	if h != p.height {
		p.height = h
		if p.Resize != nil {
			p.Resize(h)
		}
	}

	root := ui.Column(c).Fill().Padding(popoverPad)
	if !c.Vibrancy() { // elsewhere the panel draws its own background; on a Mac the popover's material is it
		root.Background(t.Bg)
	}
	root.Children(func() {
		m.composerBox(c, t, surfacePage, theme.Input)
		ui.Scroll(c.Key("popover-body")).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				if reply {
					ui.Box(c).Height(8).Shrink(0)
					m.answerView(c, t, false)
				}
				m.notice(c, t)
				m.pictureNotice(c, t, surfacePage)
				p.today(c, t, rows)
			})
		})
		p.footer(c, t)
	})
}

// today is the header and the rows: a click opens the task in the main window and closes the panel.
func (p *Popover) today(c *ui.Context, t *theme.Tokens, rows []data.Item) {
	if len(rows) == 0 {
		return
	}
	m := p.m
	ui.Row(c).Margin(12, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, m.say.SectionToday())
		theme.Text(c, theme.Meta, t.TextSecondary, itoa(len(m.mine.Today)))
	})
	for _, it := range rows {
		ph := m.rowPhase(c, it.Task)
		row := m.rowShell(c, t, it.ID, rowStyle{7, 8}, ph).AlignItems(ui.Center)
		row.Children(func() {
			m.glyphSlot(c, t, it.Task, ph.fill)
			body(c, func() { m.titleLine(c, t, it.Task, "", false) })
			right(c, func() {
				if label := m.clock.DueLabel(it.Task, m.app.Now()); label != "" {
					theme.Text(c, theme.Meta, t.TextSecondary, label).Shrink(0)
				}
			})
		})
		if row.Clicked() {
			m.app.Act.OpenTask(it.ID)
			m.closePanel()
		}
	}
}

// footer is one line under a hairline: how many agents are working and how many are stuck (the
// stuck number in the blocked color, the only strong color here), and the way to the main window.
func (p *Popover) footer(c *ui.Context, t *theme.Tokens) {
	m := p.m
	working, stuck := agentCounts(m.agents)
	hairline(c, t).Margin(8, 0, 0)
	ui.Row(c).Height(popoverFoot - 9).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
		if working > 0 {
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.PopoverWorking(working)).Shrink(0)
		}
		if working > 0 && stuck > 0 {
			theme.Text(c, theme.Meta, t.TextSecondary, "·").Shrink(0)
		}
		if stuck > 0 {
			theme.Text(c, theme.Meta, t.Blocked(), m.say.PopoverStuck(stuck)).Shrink(0)
		}
		ui.Spacer(c)
		if link(c, t, m.say.TrayOpen()).Shrink(0).Clicked() {
			m.app.Act.OpenMain()
			m.closePanel()
		}
	})
}
