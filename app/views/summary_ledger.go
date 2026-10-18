package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The ledger (ui-spec, Summary): one row per person in the scene's order, then the total. When the
// scene has played its rows light up one by one and each count runs up.
const (
	ledgerLight = 420 * time.Millisecond // between two rows lighting up
	ledgerCount = 60 * time.Millisecond  // a step of a count running up
	ledgerRow   = 34
	ledgerWait  = 30 * time.Millisecond // between frames while the ledger animates
)

// ledgerCountStyle is the count of a row: the body's size at the weight of a title.
var ledgerCountStyle = theme.Style{Size: theme.Body.Size, Line: theme.Body.Line, Weight: 700}

// lit says how far a row has come: how long it has been lit at time at of the scene, with the scene
// over at end, row k of the ledger. Negative while it waits.
func ledgerAge(at, end time.Duration, k int) time.Duration {
	return at - end - time.Duration(k)*ledgerLight
}

// ledgerCountAt is the count a row shows age after it lit: it runs up a step every 60 ms.
func ledgerCountAt(to int, age time.Duration) int {
	if age < 0 {
		return 0
	}
	return min(to, int(age/ledgerCount))
}

// ledgerDone is when the ledger has finished animating, for n rows and the biggest count.
func ledgerDone(end time.Duration, rows, biggest int) time.Duration {
	return end + time.Duration(max(rows-1, 0))*ledgerLight + time.Duration(biggest)*ledgerCount
}

// ledger draws the people, and under them the total. It draws nothing for a period with nothing
// finished.
func (m *Main) ledger(c *ui.Context, t *theme.Tokens, v *summaryView, now time.Time) {
	if v.sum.Total() == 0 {
		return
	}
	at, calm := m.sceneTime(c, v, now)
	end := v.sceneEnd()
	if calm {
		at = end + time.Hour
	}
	people := v.sum.People
	ui.Column(c.Key("summary-ledger")).FillWidth().Shrink(0).Margin(12, 0, 0).Children(func() {
		for i, p := range people {
			m.ledgerRow(c, t, v, p, ledgerAge(at, end, i))
		}
		m.ledgerTotal(c, t, v, ledgerAge(at, end, len(people)))
	})
	if !calm && m.sum.freeze == nil && at < ledgerDone(end, len(people)+1, v.sum.Total()) {
		c.After(ledgerWait)
	}
}

// ledgerOpacity is 0 while a row waits to light up and 1 once it has.
func ledgerOpacity(age time.Duration) float32 {
	if age < 0 {
		return 0
	}
	return 1
}

// ledgerRow is one person: avatar, name, a dotted leader, the pull requests merged, the count and a
// chevron. It opens to the person's tasks.
func (m *Main) ledgerRow(c *ui.Context, t *theme.Tokens, v *summaryView, p data.SummaryPerson, age time.Duration) {
	key := v.key + "/" + p.Agent
	open := m.sum.open[key]
	name := m.say.DoneByYou()
	who := theme.You
	if !p.You() {
		name, who = agentName(p.Agent), theme.AgentAvatar(p.Agent)
	}
	row := ui.ButtonBase(c.Key("ledger-"+key)).FillWidth().Height(ledgerRow).Shrink(0).Radius(theme.RadiusControl).Cursor(ui.CursorPointer).
		Label(m.say.SummaryA11yRow(name, len(p.Tasks))).Opacity(ledgerOpacity(age)).Padding(0, 4, 0, 0).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(t.BgHover)
	}
	row.Children(func() {
		theme.AvatarIcon(c.Key("who"), who, theme.AvatarRow).Shrink(0)
		theme.Text(c, theme.Body, t.Text, name).Shrink(1).MinWidth(0).Ellipsis("…")
		leader(c, t)
		if p.Merged > 0 {
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.SummaryMerged(p.Merged)).Shrink(0)
		}
		m.ledgerCount(c, t, ledgerCountAt(len(p.Tasks), age), len(p.Tasks), false)
		icon := theme.IconForward
		if open {
			icon = theme.IconChevron
		}
		ui.Icon(c.Key("chevron"), icon).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary).Shrink(0)
	})
	if row.Clicked() && age >= 0 {
		if m.sum.open == nil {
			m.sum.open = map[string]bool{}
		}
		m.sum.open[key] = !open
	}
	hairline(c, t)
	if open {
		for _, task := range p.Tasks {
			m.ledgerTask(c, t, v, task)
		}
	}
}

// ledgerTotal is the last row: what was shipped, and how many.
func (m *Main) ledgerTotal(c *ui.Context, t *theme.Tokens, v *summaryView, age time.Duration) {
	ui.Row(c.Key("ledger-total")).FillWidth().Height(ledgerRow).Shrink(0).Padding(0, 4, 0, 0).Gap(8).AlignItems(ui.Center).
		Opacity(ledgerOpacity(age)).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, m.shippedLabel(v.period)).Shrink(1).MinWidth(0).Ellipsis("…")
		leader(c, t)
		total := v.sum.Total()
		m.ledgerCount(c, t, ledgerCountAt(total, age), total, true)
		ui.Box(c).Width(theme.IconSmall).Shrink(0)
	})
}

// shippedLabel is the total's label: Shipped today, Shipped that day, Shipped this week.
func (m *Main) shippedLabel(p data.Period) string {
	over := p.Over(m.app.Now())
	switch {
	case p.Week && over:
		return m.say.SummaryShippedPastWeek()
	case p.Week:
		return m.say.SummaryShippedWeek()
	case over:
		return m.say.SummaryShippedDay()
	}
	return m.say.SummaryShippedToday()
}

// ledgerCount is a count in title weight and its unit. It is as wide as the biggest it will be, so
// the row does not shift as it runs up.
func (m *Main) ledgerCount(c *ui.Context, t *theme.Tokens, shown, final int, total bool) {
	ink := t.Text
	ui.Row(c).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
		theme.Text(c, ledgerCountStyle, ink, itoa(shown)).Width(float32(8*len(itoa(final)) + 4)).TextAlign(ui.End)
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.SummaryUnit(final)).Shrink(0)
	})
}

// leader is the dotted line between a name and its count: 2 pt dots, 4 pt apart.
func leader(c *ui.Context, t *theme.Tokens) {
	dot := t.TextTertiary
	ui.Box(c.Key("leader")).Grow(1).Height(2).Margin(8, 6, 0).MinWidth(0).Draw(func(p *ui.Painter, r ui.Rect) {
		for x := float32(0); x+2 <= r.W; x += 4 {
			p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 2, H: 2}, dot, 0)
		}
	})
}

// ledgerTask is a task under its person: the title, the last report line with the pull request, the
// project and the time. A click opens it in the inspector.
func (m *Main) ledgerTask(c *ui.Context, t *theme.Tokens, v *summaryView, task data.SummaryTask) {
	row := ui.ButtonBase(c.Key("ledger-task-"+itoa(task.ID))).FillWidth().Shrink(0).Radius(theme.RadiusControl).Cursor(ui.CursorPointer).
		Padding(5, 4, 4, 26).Gap(10).AlignItems(ui.Start).Label(m.say.A11yOpenTask(task.Title))
	if row.Hovered() {
		row.Background(t.BgHover)
	}
	row.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			theme.Text(c, theme.Body, t.Text, task.Title).Ellipsis("…")
			if line := m.taskLine(task); line != "" {
				theme.Text(c, theme.Meta, t.TextSecondary, line).Ellipsis("…")
			}
		})
		if task.Project != "" && m.width >= dropProjectBelow {
			ui.Row(c).Shrink(1).MinWidth(0).Padding(0, 6).Radius(5).Background(t.BgSubtle).Margin(2, 0, 0).Children(func() {
				theme.Text(c, theme.Meta, t.TextSecondary, task.Project).Ellipsis("…")
			})
		}
		theme.Text(c, theme.Meta, t.TextTertiary, m.taskTime(v.period, task)).Shrink(0).Margin(2, 0, 0)
	})
	if row.Clicked() {
		m.openRow(task.ID)
	}
}

// taskTime is when a task was finished: the time of day, and in a week the day before it.
func (m *Main) taskTime(p data.Period, task data.SummaryTask) string {
	if p.Week {
		return m.clock.Weekday(task.At) + " " + m.clock.Time(task.At)
	}
	return m.clock.Time(task.At)
}

// taskLine is the quiet line under a task: its pull request, merged or not, and what the agent last
// reported.
func (m *Main) taskLine(task data.SummaryTask) string {
	var parts []string
	switch {
	case task.PRNumber > 0 && task.Merged:
		parts = append(parts, m.say.SummaryRowMerged(itoa(task.PRNumber)))
	case task.PRNumber > 0:
		parts = append(parts, "PR #"+itoa(task.PRNumber))
	}
	if task.Said != "" {
		parts = append(parts, task.Said)
	}
	return strings.Join(parts, " · ")
}
