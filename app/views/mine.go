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
	m.pageHeader(c, t, m.say.NavMine(), m.clock.Heading(m.app.Now()), nil)
	ui.Box(c).Height(18).Shrink(0)
	m.pageComposer(c, t)
	if !m.app.snap.loaded {
		return
	}
	m.belowComposer(c, func() {
		if m.mine.Empty() {
			m.allClear(c, t)
		}
		m.fromAgentsSection(c, t, m.mine.FromAgents)
		m.mineSection(c, t, m.say.SectionToday(), m.mine.Today)
		m.mineSection(c, t, m.say.SectionLater(), m.mine.Later)
		if m.mine.MoreLater > 0 && link(c, t, m.say.NowShowAllInInbox()).Margin(8, 0, 0).Clicked() {
			m.goTo(nav{screen: screenInbox})
		}
		m.doneTodaySection(c, t, m.mine.Done, m.mine.MoreDone)
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
	sectionHeader(c, t, m.say.SectionFromAgents(), len(items))
	ui.Column(c).Shrink(0).Padding(4, 14).Radius(theme.RadiusPanel).Border(1, t.Border).Children(func() {
		for i, it := range items {
			if i > 0 {
				ui.Box(c.Key("rule-"+itoa(it.ID))).Height(1).Shrink(0).Margin(0, -needRow.horz).Background(t.Border)
			}
			m.handedRow(c, t, it)
		}
	})
}

func (m *Main) handedRow(c *ui.Context, t *theme.Tokens, it data.Item) {
	_, opacity, _ := m.app.completionPhase(c, it.ID)
	row := m.rowShell(c, t, it.ID, needRow, opacity)
	actions := m.actionsFor(it)
	// Two buttons beside a title in a narrow row leave it no room: they go under the summary, and
	// in a very narrow one so does a single button.
	stacked := (len(actions) > 1 && m.width < stackActionsBelow) || m.width < stackAllActionsBelow
	buttons := func() {
		for i, a := range actions {
			if actionButtonKeyed(c, t, "action-"+itoa(i)+"-"+itoa(it.ID), a.text, a.label).Clicked() {
				a.run()
			}
		}
	}
	row.Children(func() {
		theme.AvatarIcon(c.Key("author"), m.app.taskWho(it.Task), theme.AvatarRow).Margin(2, 0, 0, 0)
		body(c, func() {
			m.titleLine(c, t, it.Task, "", false)
			summaryLine(c, t, m.handedSummary(it))
			if stacked {
				ui.Row(c).Shrink(0).Gap(8).Margin(8, 0, 0).Children(buttons)
			}
		})
		right(c, func() {
			if m.width >= dropPillsBelow { // beside the inspector the title needs the room more
				m.ageLabel(c, t, it)
			}
			if !stacked {
				buttons()
			}
		})
	})
	if row.Clicked() {
		m.openRow(it.ID)
	}
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
// what it does.
type rowAction struct {
	text, label string
	run         func()
}

// actionsFor names the buttons of a handed-over row, left to right. Most rows have one.
func (m *Main) actionsFor(it data.Item) []rowAction {
	openPR := rowAction{m.say.ActionOpenPR(), m.say.A11yOpenPR(itoa(it.PRNumber)), func() { m.app.Act.OpenURL(it.PRURL) }}
	open := func(text string) []rowAction {
		return []rowAction{{text, m.say.A11yOpenTask(it.Title), func() { m.openRow(it.ID) }}}
	}
	switch {
	case it.Kind == data.NeedReminder:
		return []rowAction{{m.say.ActionDone(), m.say.A11yMarkDone(it.Title), func() { m.app.SetStatus(it.Task, data.Done) }}}
	case it.Kind == data.NeedPRClosed:
		return []rowAction{
			{m.say.ActionCancelTask(), m.say.A11yCancelTask(it.Title), func() { m.app.SetStatus(it.Task, data.Cancelled) }},
			{m.say.ActionKeep(), m.say.A11yKeepTask(it.Title), func() { m.app.KeepClosedPR(it.ID) }},
		}
	case it.Kind == data.NeedPRFailing || it.Kind == data.NeedPRChanges, it.Kind == data.NeedReview && it.PRURL != "":
		return []rowAction{openPR}
	case it.Kind == data.NeedBlocked:
		return open(m.say.ActionAnswer())
	case it.Kind == data.NeedReview:
		return open(m.say.ActionReview())
	}
	return open(m.say.ActionOpen())
}

// mineSection is a plain section of my own tasks: Today or Later. Each row has the status glyph,
// which ticks the task off, its title, and at the right how pressing it is and when it is due.
func (m *Main) mineSection(c *ui.Context, t *theme.Tokens, title string, items []data.Item) {
	if len(items) == 0 {
		return
	}
	sectionHeader(c, t, title, len(items))
	for _, it := range items {
		fill, opacity, _ := m.app.completionPhase(c, it.ID)
		row := m.rowShell(c, t, it.ID, plainRow, opacity).AlignItems(ui.Center)
		row.Children(func() {
			m.glyphSlot(c, t, it.Task, fill)
			body(c, func() { m.titleLine(c, t, it.Task, it.Project, true) })
			right(c, func() {
				if theme.ShowsPriority(it.Priority) {
					theme.PriorityBadge(c, t, it.Priority)
				}
				if label := m.clock.DueLabel(it.Task, m.app.Now()); label != "" {
					theme.Text(c, theme.Meta, t.TextTertiary, label).Shrink(0)
				}
			})
		})
		if row.Clicked() {
			m.openRow(it.ID)
		}
	}
}
