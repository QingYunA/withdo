package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// mineScreen is Home: the date, a greeting and the big input; under them the lists of what is mine
// (today, later, done today) and, beside them when the window is wide enough, the Agents panel with
// what agents handed over. Narrower, that section shows at the top of the lists. A section with
// nothing in it is not drawn; with nothing in any of them the screen says All clear.
func (m *Main) mineScreen(c *ui.Context, t *theme.Tokens) {
	// The title is the greeting. A category's page keeps its name at the left, over it.
	m.pageHeaderOf(c, t, func() {
		if m.nav.cat != 0 {
			theme.Text(c, theme.Title, t.Text, m.mineTitle()).Shrink(0)
		}
	}, "", nil)
	if !m.app.snap.loaded {
		return
	}
	m.mineHero(c, t)
	m.belowComposer(c, func() {
		if !m.mineSplit() {
			m.mineLists(c, t, true)
			return
		}
		ui.Row(c.Key("mine-cols")).FillWidth().Shrink(0).Gap(theme.MineGap).AlignItems(ui.Start).Children(func() {
			ui.Column(c.Key("mine-lists")).Grow(1).MinWidth(0).Children(func() { m.mineLists(c, t, false) })
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

// mineHero is the top of Home: the day, the greeting, the big input (the composer, in its hero
// size) and under it the hint. What the input says back (the answer, the receipt with Undo, a voice
// or picture line) shows under the box where the hint is, and the hint is back when it is gone.
func (m *Main) mineHero(c *ui.Context, t *theme.Tokens) {
	ui.Column(c.Key("hero")).FillWidth().Shrink(0).AlignItems(ui.Center).Padding(14, 0, 22).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, m.clock.HeroDate(m.app.Now())).Shrink(0)
		theme.Paragraph(c, theme.Title, t.Text, m.say.MineHeroGreeting(), 0).TextAlign(ui.Center).Margin(4, 0, 0)
		ui.Column(c.Key("hero-input")).FillWidth().MaxWidth(theme.HeroMax).Margin(18, 0, 0).Children(func() {
			m.composerBoxAs(c, t, surfacePage, theme.Input, true)
			if m.replyIn(surfacePage, 0) {
				ui.Box(c).Height(8).Shrink(0)
				m.answerView(c, t, false)
			}
			m.voiceLine(c, t)
			m.notice(c, t)
			m.pictureNotice(c, t, surfacePage)
			if !m.inputLineLive(c) {
				theme.Paragraph(c, theme.Meta, t.TextTertiary, m.say.MineHeroHint(panelKeyHint()), 0).TextAlign(ui.Center).Margin(10, 0, 0)
			}
		})
	})
}

// panelKeyHint is the global shortcut that opens the menu bar panel, as the hint writes it.
func panelKeyHint() string {
	if systemGOOS == "darwin" {
		return "⌥ Space"
	}
	return "Alt+Space"
}

// inputLineLive reports whether a line shows under the big input (an answer, a voice line, a
// receipt, a picture problem, the sign-in line), which takes the hint's place.
func (m *Main) inputLineLive(c *ui.Context) bool {
	cs, now := &m.cmp[surfacePage], m.app.Now()
	length := cs.noticeLen
	if length == 0 {
		length = noticeFor
	}
	switch {
	case m.replyIn(surfacePage, 0):
	case m.voice.phase == voiceListening || m.voice.phase == voiceTranscribing:
	case m.voice.problem != voiceNoProblem && now.Sub(m.voice.problemAt) < voiceProblemFor:
	case !cs.noticeAt.IsZero() && now.Sub(cs.noticeAt) < length:
	case strings.TrimSpace(cs.problem) != "" && now.Sub(cs.problemAt) < pictureNoticeFor:
	case m.app.Assistant != nil && !m.assistReady() && cs.buf == "" && (cs.ref.Focused(c) || cs.setUpHover):
	default:
		return false
	}
	return true
}

// mineLists are the lists of Home. With inline the section of what agents handed over comes first,
// as the panel is not beside them.
func (m *Main) mineLists(c *ui.Context, t *theme.Tokens, inline bool) {
	if m.minePage.Empty() {
		m.allClear(c, t)
	}
	if inline {
		m.fromAgentsSection(c, t, m.minePage.FromAgents)
	}
	m.mineSection(c, t, m.say.SectionToday(), m.minePage.Today)
	m.mineSection(c, t, m.say.SectionLater(), m.minePage.Later)
	if m.minePage.MoreLater > 0 && link(c, t, m.say.NowShowAllInInbox()).Margin(8, 0, 0).Clicked() {
		m.goTo(nav{screen: screenInbox})
	}
	m.doneTodaySection(c, t, m.minePage.Done, m.minePage.MoreDone)
}

// agentsPanel is the column at the right of Home on the tray color: what agents handed over, the
// same rows as the inline section, and at its foot the drop target where a row of mine hands its
// task over.
func (m *Main) agentsPanel(c *ui.Context, t *theme.Tokens) {
	items := m.minePage.FromAgents
	zone := ui.Column(c.Key("agents-panel")).Width(theme.AgentsPanelWidth).Shrink(0).Padding(14-trayRing, 16-trayRing, 16-trayRing).
		Radius(theme.TrayRadius).Background(t.BgTray).Border(trayRing, ui.Transparent)
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
	if over {
		zone.Background(t.BgTray.Mix(t.Accent, 0.07)).Border(trayRing, t.Accent.Alpha(0.35))
	}
	zone.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	zone.Children(func() {
		ui.Row(c.Key("panel-head")).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			m.agentBot(c, t, theme.AvatarRow)
			theme.Text(c, theme.BodyStrong, t.Text, m.say.AgentsPanelTitle()).Shrink(1).MinWidth(0).Ellipsis("…")
			ui.Spacer(c)
			if len(items) > 0 {
				theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(items))).Shrink(0)
			}
		})
		if len(items) == 0 {
			quietLine(c, t, m.say.AgentsPanelEmpty(), nil)
		} else {
			m.section(c, "from-agents", itemTasks(items), func() {
				ui.Column(c).Shrink(0).Margin(8, 0, 0).Children(func() {
					// The panel is narrow: its rows lay out as in a narrow window, buttons under the summary.
					wide := m.width
					m.width, m.inPanel = theme.AgentsPanelWidth, true
					defer func() { m.width, m.inPanel = wide, false }()
					m.handedRows(c, t, items)
				})
			})
		}
		m.dropTarget(c, t, over)
	})
}

// dropTarget is the dashed box at the foot of the Agents panel; it takes the accent while a row of
// mine is over the panel.
func (m *Main) dropTarget(c *ui.Context, t *theme.Tokens, over bool) {
	line, color := t.Border, t.TextTertiary
	if over {
		line, color = t.Accent.Alpha(0.5), t.Accent
	}
	ui.Row(c.Key("panel-drop")).FillWidth().Shrink(0).Margin(14, 0, 0).Padding(11, 12).Gap(8).AlignItems(ui.Start).
		Radius(theme.RadiusControl).Border(1, line).BorderStyle(ui.BorderDashed).Children(func() {
		ui.Icon(c, theme.IconMove).Size(theme.IconSmall, theme.IconSmall).TextColor(color).Shrink(0).Margin(1, 0, 0)
		theme.Paragraph(c, theme.Meta, color, m.say.AgentsPanelDrop(), 0).Shrink(1).MinWidth(0)
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
			theme.Text(c, theme.Body, t.TextTertiary, sub).Shrink(1).MinWidth(0).Ellipsis("…")
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
	ui.Row(c).Margin(22, 0, 6).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, title)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n))
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
		ui.Column(c).Shrink(0).Padding(4, 14).Radius(theme.RadiusPanel).Border(1, t.Border).Children(func() { m.handedRows(c, t, items) })
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
		if i > 0 {
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
	row := m.rowShell(c, t, it.ID, needRow, ph)
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
		return reason + " · " + it.Project
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

// mineSection is a plain section of my own tasks: Today or Later. Each row has the status glyph,
// which ticks the task off, its title, and at the right how pressing it is and when it is due.
// Beside the Agents panel a row drags onto it, to hand the task over.
func (m *Main) mineSection(c *ui.Context, t *theme.Tokens, title string, items []data.Item) {
	if len(items) == 0 {
		return
	}
	m.section(c, "section-"+title, itemTasks(items), func() {
		sectionHeader(c, t, title, len(items))
		for _, it := range items {
			ph := m.rowPhase(c, it.Task)
			fill := func(row ui.Element, _ bool) {
				row.AlignItems(ui.Center).Children(func() {
					m.glyphSlot(c, t, it.Task, ph.fill)
					body(c, func() { m.titleLine(c, t, it.Task, it.Project, true) })
					right(c, func() {
						priorityMark(c, t, it.Task)
						m.rowThumb(c, t, it.ID)
						if label := m.clock.DueLabel(it.Task, m.app.Now()); label != "" {
							theme.Text(c, theme.Meta, t.TextTertiary, label).Shrink(0)
						}
					})
				})
			}
			if m.mineSplit() { // the Agents panel is there to drop a row on
				m.draggableRow(c, t, it.Task, false, ph, fill)
				continue
			}
			row := m.rowShell(c, t, it.ID, plainRow, ph)
			fill(row, false)
			if m.rowClicked(row, it.ID) {
				m.openRow(it.ID)
			}
		}
	})
}
