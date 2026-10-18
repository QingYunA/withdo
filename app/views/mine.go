package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// mineScreen is the home screen: the date, the composer, and what is mine: what agents handed over,
// then today, then later. A section with nothing in it is not drawn; with nothing in any of them the
// screen says All clear. What was finished today closes the page.
func (m *Main) mineScreen(c *ui.Context, t *theme.Tokens) {
	m.pageHeader(c, t, m.mineTitle(), m.clock.Heading(m.app.Now()), nil)
	ui.Box(c).Height(18).Shrink(0)
	m.pageComposer(c, t)
	if !m.app.snap.loaded {
		return
	}
	m.belowComposer(c, func() {
		if m.minePage.Empty() {
			m.allClear(c, t)
		}
		m.fromAgentsSection(c, t, m.minePage.FromAgents)
		m.mineSection(c, t, m.say.SectionToday(), m.minePage.Today)
		m.mineSection(c, t, m.say.SectionLater(), m.minePage.Later)
		if m.minePage.MoreLater > 0 && link(c, t, m.say.NowShowAllInInbox()).Margin(8, 0, 0).Clicked() {
			m.goTo(nav{screen: screenInbox})
		}
		m.doneTodaySection(c, t, m.minePage.Done, m.minePage.MoreDone)
	})
}

// belowComposer holds what the page shows under its composer. It slides down when the answer's
// line appears and back up when it goes, instead of jumping (with Reduce Motion it just moves).
func (m *Main) belowComposer(c *ui.Context, body func()) {
	ui.Column(c.Key("below")).Shrink(0).Transition(ui.ElementTransition{Position: true, Duration: theme.MotionFast}).Children(body)
}

// pageHeader is the title with a quiet companion, and whatever controls sit at the right.
func (m *Main) pageHeader(c *ui.Context, t *theme.Tokens, title, sub string, controls func()) {
	ui.Row(c).Height(36).Shrink(0).Gap(10).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Title, t.Text, title).Shrink(0)
		if sub != "" {
			theme.Text(c, theme.Body, t.TextTertiary, sub).Shrink(1).MinWidth(0).Ellipsis("…")
		}
		ui.Spacer(c)
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
		ui.Column(c).Shrink(0).Padding(4, 14).Radius(theme.RadiusPanel).Border(1, t.Border).Children(func() {
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
		})
	})
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
		theme.AvatarIcon(c.Key("author"), m.app.taskWho(it.Task), theme.AvatarRow).Margin(2, 0, 0, 0)
		body(c, func() {
			m.titleLine(c, t, it.Task, "", false)
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
func (m *Main) mineSection(c *ui.Context, t *theme.Tokens, title string, items []data.Item) {
	if len(items) == 0 {
		return
	}
	m.section(c, "section-"+title, itemTasks(items), func() {
		sectionHeader(c, t, title, len(items))
		for _, it := range items {
			ph := m.rowPhase(c, it.Task)
			row := m.rowShell(c, t, it.ID, plainRow, ph).AlignItems(ui.Center)
			row.Children(func() {
				m.glyphSlot(c, t, it.Task, ph.fill)
				body(c, func() { m.titleLine(c, t, it.Task, it.Project, true) })
				right(c, func() {
					if theme.ShowsPriority(it.Priority) {
						theme.PriorityBadge(c, t, it.Priority)
					}
					m.rowThumb(c, t, it.ID)
					if label := m.clock.DueLabel(it.Task, m.app.Now()); label != "" {
						theme.Text(c, theme.Meta, t.TextTertiary, label).Shrink(0)
					}
				})
			})
			if m.rowClicked(row, it.ID) {
				m.openRow(it.ID)
			}
		}
	})
}
