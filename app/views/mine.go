package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// mineScreen is Today, the home page: the date as the title, the composer across the whole content
// width, and under it the lists of what is mine (today, next, no date, done today) and, beside them
// when the window is wide enough, the Agents tray with what agents handed over. Narrower, that
// section shows at the top of the lists. A section with nothing in it is not drawn; with nothing in
// any of them the screen says All clear.
func (m *Main) mineScreen(c *ui.Context, t *theme.Tokens) {
	if !m.app.snap.loaded {
		return
	}
	ui.Row(c.Key("mine-title")).FillWidth().Height(32).Shrink(0).Padding(0, homeLineLeft).Margin(8, 0, 16).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Title, t.Text, m.clock.HeroDate(m.app.Now())).Shrink(1).MinWidth(0).Ellipsis("…")
	})
	m.homeComposer(c, t)
	m.belowComposer(c, func() {
		m.homeLines(c, t)
		m.filterBar(c, t)
		if !m.mineSplit() {
			m.mineLists(c, t, true)
			return
		}
		ui.Row(c.Key("mine-cols")).FillWidth().Shrink(0).Margin(8, 0, 0).Gap(theme.MineGap).AlignItems(ui.Start).Children(func() {
			ui.Column(c.Key("mine-lists")).Grow(1).MinWidth(0).Padding(0, homeLineLeft).Children(func() { m.mineLists(c, t, false) })
			m.agentsPanel(c, t)
		})
	})
}

// mineSplit says whether the Agents panel sits beside the lists: the content must be as wide as the
// lists' minimum, the gap and the panel need (theme.MineSplitBelow).
func (m *Main) mineSplit() bool { return m.width >= theme.MineSplitBelow }

// minePad is the side padding of Home: the usual one, less while the panel beside the lists needs
// the room, never under theme.MinePad.
func (m *Main) minePad(c *ui.Context) float32 {
	pad := m.sidePad(c)
	if !m.mineSplit() {
		return pad
	}
	return min(pad, max(theme.MinePad, (m.width-theme.ContentMax-theme.MineGap-theme.AgentsPanelWidth)/2))
}

// mineMax is the widest Home gets: the column of the lists and, in a window that has the room, the
// panel and its gap.
func (m *Main) mineMax() float32 {
	if !m.mineSplit() {
		return theme.ContentMax
	}
	return theme.ContentMax + theme.MineGap + theme.AgentsPanelWidth
}

// panelKeyHint is the global shortcut that opens the menu bar panel, as the hint writes it.
func panelKeyHint() string {
	if systemGOOS == "darwin" {
		return "⌥ Space"
	}
	return "Alt+Space"
}

// mineLists are the lists of Home, by time only. With inline the section of what agents handed over
// comes first, as the tray is not beside them.
func (m *Main) mineLists(c *ui.Context, t *theme.Tokens, inline bool) {
	if m.minePage.Empty() {
		m.allClear(c, t)
	}
	if inline {
		m.fromAgentsSection(c, t, m.minePage.FromAgents)
	}
	m.mineSection(c, t, m.say.SectionToday(), m.minePage.Today)
	m.mineSection(c, t, m.say.SectionNext(), m.minePage.Next)
	m.noDateSection(c, t)
	m.doneTodaySection(c, t, m.minePage.Done, m.minePage.MoreDone)
}

// noDateSection is the fold of my tasks with no date and no project, closed until opened. A task
// of a project that has no date is on its project's page, one dated beyond the week on the Calendar.
func (m *Main) noDateSection(c *ui.Context, t *theme.Tokens) {
	n := m.minePage.NoDateTotal
	if n == 0 {
		return
	}
	m.section(c, "no-date", itemTasks(m.minePage.NoDate), func() {
		name, open := m.say.FoldNoDate(), m.app.noDateOpen
		head := ui.ButtonBase(c.Key("fold-no-date")).Margin(22, 0, 6).Shrink(0).AlignSelf(ui.Start).Radius(theme.RadiusControl).
			Cursor(ui.CursorPointer).Label(m.say.FoldCount(name, n))
		head.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Section, t.TextSecondary, name)
				theme.Text(c, theme.Meta, t.TextSecondary, itoa(n))
				chevron := theme.IconForward
				if open {
					chevron = theme.IconChevron
				}
				ui.Icon(c, chevron).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary).Shrink(0)
			})
		})
		if head.Clicked() {
			m.app.noDateOpen = !open
			m.app.Refresh()
		}
		if open {
			m.mineRows(c, t, m.minePage.NoDate)
		}
	})
}

// agentsPanel is the tray at the right of Home: what agents handed over, one card each, and while a
// row of mine is dragged the well where it hands its task over.
func (m *Main) agentsPanel(c *ui.Context, t *theme.Tokens) {
	items := m.minePage.FromAgents
	zone := ui.Column(c.Key("agents-panel")).Width(theme.AgentsPanelWidth).Shrink(0).Padding(theme.TrayInset).Gap(theme.TrayInset).
		Radius(theme.RadiusPanel).Background(t.BgTray).Border(theme.Hairline, t.Border)
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
	dragging := m.ho.carry.seen || m.ho.stackSeen
	zone.Children(func() {
		ui.Row(c.Key("panel-head")).Shrink(0).Gap(8).Padding(8, 10, 4).AlignItems(ui.Center).Children(func() {
			m.agentBot(c, t, theme.AvatarRow)
			theme.Text(c, theme.BodyStrong, t.TextSecondary, m.say.AgentsPanelTitle()).Shrink(1).MinWidth(0).Ellipsis("…")
			ui.Spacer(c)
			if len(items) > 0 {
				theme.Text(c, theme.Meta, t.TextSecondary, itoa(len(items))).Shrink(0)
			}
		})
		switch {
		case len(items) > 0:
			m.section(c, "from-agents", itemTasks(items), func() {
				ui.Column(c).Shrink(0).Gap(theme.TrayInset).Children(func() {
					// The tray is narrow: its cards lay out as in a narrow window, buttons under the summary.
					wide := m.width
					m.width, m.inPanel = theme.AgentsPanelWidth, true
					defer func() { m.width, m.inPanel = wide, false }()
					m.handedRows(c, t, items)
				})
			})
		case !dragging:
			quietLine(c, t, m.say.AgentsPanelEmpty(), nil)
		}
		if dragging {
			m.dropTarget(c, t, over)
		}
	})
}

// dropTarget is the well of the Agents tray, drawn only while a row of mine is dragged: iris 2 with a
// 0.5 pt iris 8 edge (a full point while the row is over it).
func (m *Main) dropTarget(c *ui.Context, t *theme.Tokens, over bool) {
	edge := theme.Hairline
	if over {
		edge = 1
	}
	ui.Row(c.Key("panel-drop")).FillWidth().Shrink(0).MinHeight(120).Gap(8).AlignItems(ui.Center).Justify(ui.Center).
		Radius(theme.RadiusRow).Background(t.AccentWell).Border(edge, t.AccentEdge).Children(func() {
		ui.Icon(c, theme.IconMove).Size(theme.IconSmall, theme.IconSmall).TextColor(t.AccentInk).Shrink(0)
		theme.Text(c, theme.Meta, t.AccentInk, m.say.AgentsPanelDrop()).Shrink(0)
	})
}

// belowComposer holds what the page shows under its composer. It slides down when the answer's
// line appears and back up when it goes, instead of jumping (with Reduce Motion it just moves).
func (m *Main) belowComposer(c *ui.Context, body func()) {
	ui.Column(c.Key("below")).Shrink(0).Transition(ui.ElementTransition{Position: true, Duration: theme.MotionFast}).Children(body)
}

// pageHeader is the title with a quiet companion, and whatever controls sit at the right.
func (m *Main) pageHeader(c *ui.Context, t *theme.Tokens, title, sub string, controls func()) {
	m.pageHeaderOf(c, t, func() { theme.Text(c, theme.Title, t.Text, title).Shrink(0) }, sub, controls)
}

// pageHeaderOf is pageHeader with a title that draws itself.
func (m *Main) pageHeaderOf(c *ui.Context, t *theme.Tokens, title func(), sub string, controls func()) {
	ui.Row(c).Height(36).Shrink(0).Gap(10).AlignItems(ui.Center).Children(func() {
		title()
		if sub != "" {
			theme.Text(c, theme.Body, t.TextSecondary, sub).Shrink(1).MinWidth(0).Ellipsis("…")
		}
		ui.Spacer(c)
		if m.pg.Kind != page.KindColumns { // there each column's head has it
			m.selectControls(c, t)
		}
		if controls != nil {
			controls()
		}
	})
}

// sectionHeader is a section's title in sentence case, with its count.
func sectionHeader(c *ui.Context, t *theme.Tokens, title string, n int) {
	sectionHeaderWith(c, t, title, n, nil)
}

// sectionHeaderWith is sectionHeader with something at the right end of the line.
func sectionHeaderWith(c *ui.Context, t *theme.Tokens, title string, n int, end func()) {
	ui.Row(c).Margin(22, 0, 6).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, title)
		theme.Text(c, theme.Meta, t.TextSecondary, itoa(n))
		if end != nil {
			ui.Spacer(c)
			end()
		}
	})
}

// allClear is the calm state: nothing is handed over, nothing is mine.
func (m *Main) allClear(c *ui.Context, t *theme.Tokens) {
	emptyState(c, t, m.say.NowAllClear(), m.say.NowAllClearBody(), 72, func() {
		if link(c, t, m.say.LinkHowToConnect()).Margin(6, 0, 0).Clicked() {
			m.openSettings(PaneAgents)
		}
	})
}

// fromAgentsSection is the one boxed thing on the screen: what agents handed over, blocked first.
// Each row has the agent's avatar, why it is here in one line, and a button that does the next thing.
func (m *Main) fromAgentsSection(c *ui.Context, t *theme.Tokens, items []data.Item) {
	if len(items) == 0 {
		return
	}
	m.section(c, "from-agents", itemTasks(items), func() {
		sectionHeader(c, t, m.say.SectionFromAgents(), len(items))
		ui.Column(c).Shrink(0).Padding(4, 14).Radius(theme.RadiusPanel).Border(theme.Hairline, t.Border).Children(func() { m.handedRows(c, t, items) })
	})
}

// handedRows are the rows of what agents handed over, split by hairlines.
func (m *Main) handedRows(c *ui.Context, t *theme.Tokens, items []data.Item) {
	phs := make([]phase, len(items))
	for i, it := range items {
		phs[i] = m.rowPhase(c, it.Task)
	}
	rules := ruleShares(phs)
	for i, it := range items {
		if i > 0 && !m.inPanel { // the cards of the tray have no rules between them
			m.rule(c, t, it.ID, rules[i], phs[i].opacity)
		}
		m.handedRow(c, t, it, phs[i])
	}
}

// rule is the hairline above a row of From your agents, share of its height still there as it folds
// away with its row.
func (m *Main) rule(c *ui.Context, t *theme.Tokens, id int, share, opacity float32) {
	line := ui.Box(c.Key("rule-"+itoa(id))).Height(share).Shrink(0).Margin(0, -needRow.horz).Background(t.Border)
	if opacity < 1 {
		line.Opacity(opacity)
	}
}

func (m *Main) handedRow(c *ui.Context, t *theme.Tokens, it data.Item, ph phase) {
	style := needRow
	if m.inPanel {
		style = cardRow
	}
	row := m.rowShell(c, t, it.ID, style, ph)
	if m.inPanel { // a card on bg, radius 8 (the tray's 12 less its 4 pt inset), a 0.5 pt edge
		row.Margin(0).Radius(theme.RadiusRow).Border(theme.Hairline, t.Border)
		if !(row.Hovered() || m.sel == it.ID || m.openID == it.ID) {
			row.Background(t.Bg)
		}
	}
	actions := m.actionsFor(it)
	showNotNow := row.Hovered() || m.sel == it.ID
	// Two buttons beside a title in a narrow row leave it no room: they go under the summary, and
	// in a very narrow one so does a single button.
	stacked := (len(actions) > 1 && m.width < stackActionsBelow) || m.width < stackAllActionsBelow
	buttons := func() { m.actionButtons(c, t, it.ID, actions, ph) }
	row.Children(func() {
		m.leadOf(c, t, it.Task, theme.AvatarRow).Margin(2, 0, 0, 0)
		body(c, func() {
			if m.inPanel {
				// The age sits at the right of the title line, which ellipsizes before it. Its right edge
				// is the panel's inner edge, like the header's count: the row reserves the Not now slot
				// (and its gap) there, so the line reaches over it, and the age yields to the x while it shows.
				ui.Row(c).MinWidth(0).Gap(8).AlignItems(ui.Center).Margin(0, -(notNowSlot + rowRightGap), 0, 0).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Children(func() { m.titleLine(c, t, it.Task, "", false) })
					m.panelAge(c, t, it, showNotNow)
				})
			} else {
				m.titleLine(c, t, it.Task, "", false)
			}
			summaryLine(c, t, m.handedSummary(it))
			if m.openFail.id == it.ID && m.openFail.err != nil {
				theme.Paragraph(c, theme.Meta, t.Blocked(), m.openFailText(m.openFail.err), 2).Margin(2, 0, 0)
			}
			if stacked {
				ui.Row(c).Shrink(0).Gap(8).Margin(8, 0, 0).Children(buttons)
			}
		})
		right(c, func() {
			m.rowThumb(c, t, it.ID)
			if m.width >= dropPillsBelow { // beside the inspector the title needs the room more
				m.ageLabel(c, t, it)
			}
			if !stacked {
				buttons()
			}
			if m.notNowButton(c, t, it, showNotNow).Clicked() {
				m.app.dismiss(it)
			}
		})
	})
	if m.rowClicked(row, it.ID) {
		m.openRow(it.ID)
	}
}

// cardRow is the padding of a card of the Agents tray.
var cardRow = rowStyle{10, 12}

// rowRightGap is the gap between the parts of the cluster at the right of a row (see right).
const rowRightGap float32 = 10

// panelAge is a row's age in the Agents panel; it is drawn invisible while the Not now x shows
// in its place, so the title does not move.
func (m *Main) panelAge(c *ui.Context, t *theme.Tokens, it data.Item, hidden bool) {
	color := t.TextTertiary
	if it.Stale {
		color = t.Stale
	}
	age := theme.Text(c, theme.Meta, color, m.clock.Age(it.Since, m.app.Now())).Shrink(0)
	if hidden {
		age.Opacity(0)
	}
}

// notNowSlot is the side of the square slot of the x on a handed-over row.
const notNowSlot float32 = 20

// notNowButton is the small x at the right end of a handed-over row that puts the row aside until
// the agent reports again. The slot is always there, so the row does not jump when the x shows.
func (m *Main) notNowButton(c *ui.Context, t *theme.Tokens, it data.Item, shown bool) ui.Element {
	btn := ui.ButtonBase(c.Key("not-now-"+itoa(it.ID))).Size(notNowSlot, notNowSlot).Shrink(0).Center().
		Radius(theme.RadiusControl).Label(m.say.A11yNotNow(it.Title)).Tooltip(m.say.ActionNotNow()).Cursor(ui.CursorPointer)
	color := t.TextTertiary
	if btn.Hovered() {
		btn.Background(t.BgSelected)
		color = t.TextSecondary
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() {
		if shown {
			ui.Icon(c, theme.IconClose).Size(theme.IconSmall, theme.IconSmall).TextColor(color)
		}
	})
	return btn
}

// handedSummary is the second line of a handed-over row: why it is here, and the project it is in.
func (m *Main) handedSummary(it data.Item) string {
	reason := m.needSummary(it)
	if it.Project != "" && m.width >= dropProjectBelow {
		return reason + ", " + it.Project
	}
	return reason
}

// stackActionsBelow is the content width under which a row with two buttons puts them under its
// summary.
const stackActionsBelow float32 = 560

// stackAllActionsBelow is the content width under which every row puts its button under the summary.
const stackAllActionsBelow float32 = 420

// rowAction is a button of a handed-over row: its text, what it says to assistive technology, and
// what it does. Accept stamps once the task is being accepted.
type rowAction struct {
	text, label string
	accepts     bool
	icon        *ui.SVG // drawn before the text; nil for none
	run         func()
}

// actionsFor names the buttons of a handed-over row, left to right. Most rows have one.
func (m *Main) actionsFor(it data.Item) []rowAction {
	switch {
	case it.Kind == data.NeedReminder:
		return []rowAction{{text: m.say.ActionDone(), label: m.say.A11yMarkDone(it.Title), run: func() { m.app.SetStatus(it.Task, data.Done) }}}
	case it.Kind == data.NeedPRClosed:
		return []rowAction{
			{text: m.say.ActionCancelTask(), label: m.say.A11yCancelTask(it.Title), run: func() { m.app.SetStatus(it.Task, data.Cancelled) }},
			{text: m.say.ActionKeep(), label: m.say.A11yKeepTask(it.Title), run: func() { m.app.KeepClosedPR(it.ID) }},
		}
	case it.Kind == data.NeedPRFailing || it.Kind == data.NeedPRChanges:
		return []rowAction{m.openPRAction(it.Task)}
	case it.Kind == data.NeedBlocked || it.Kind == data.NeedReview:
		return m.agentActions(it.Task)
	}
	return []rowAction{{text: m.say.ActionOpen(), label: m.say.A11yOpenTask(it.Title), run: func() { m.openRow(it.ID) }}}
}

// mineSection is a plain section of my own tasks: Today or Next.
func (m *Main) mineSection(c *ui.Context, t *theme.Tokens, title string, items []data.Item) {
	if len(items) == 0 {
		return
	}
	m.section(c, "section-"+title, itemTasks(items), func() {
		sectionHeader(c, t, title, len(items))
		m.mineRows(c, t, items)
	})
}

// mineRows are the rows of my own tasks. Each has the status glyph, which ticks the task off, its
// title with its category, and at the right how pressing it is and when it is due. Beside the Agents
// tray a row drags onto it, to hand the task over.
func (m *Main) mineRows(c *ui.Context, t *theme.Tokens, items []data.Item) {
	for _, it := range items {
		ph := m.rowPhase(c, it.Task)
		fill := func(row ui.Element, _ bool) {
			row.AlignItems(ui.Center).Children(func() {
				m.glyphSlot(c, t, it.Task, ph.fill)
				body(c, func() { m.titleLine(c, t, it.Task, it.Project, true) })
				right(c, func() {
					if it.Status == data.Review { // mine, and it waits for my look
						m.pillBox(c, "to-review", t.TextSecondary, t.BgSubtle, m.say.PillToReview())
					}
					m.rowThumb(c, t, it.ID)
					if label := m.clock.DueLabel(it.Task, m.app.Now()); label != "" || starred(it.Task) {
						ui.Row(c).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
							priorityMark(c, t, it.Task) // the star sits before the date
							if label != "" {
								theme.Text(c, theme.Meta, t.TextSecondary, label).Shrink(0)
							}
						})
					}
				})
			})
		}
		if m.mineSplit() { // the Agents tray is there to drop a row on
			m.draggableRow(c, t, it.Task, false, ph, fill)
			continue
		}
		row := m.rowShell(c, t, it.ID, plainRow, ph)
		fill(row, false)
		if m.rowClicked(row, it.ID) {
			m.openRow(it.ID)
		}
	}
}
