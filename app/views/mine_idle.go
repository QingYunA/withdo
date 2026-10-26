package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// idleCard is the Agents tray when no agent has anything for the user: a card in the style of the
// agents' own, with the robot asleep, who reported last and how long ago, and under a hairline how
// to hand something over.
func (m *Main) idleCard(c *ui.Context, t *theme.Tokens) {
	ui.Column(c.Key("agents-idle")).FillWidth().Shrink(0).Padding(14, 12, 12).Radius(theme.RadiusRow).
		Background(t.Bg).Border(theme.Hairline, t.Border).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			m.sleepingBot(c, t)
			ui.Column(c).Shrink(1).MinWidth(0).Gap(2).Children(func() {
				theme.Text(c, theme.BodyStrong, t.Text, m.say.AgentsIdleTitle()).Shrink(1).MinWidth(0).Ellipsis("…")
				m.lastReporter(c, t)
			})
		})
		ui.Box(c).FillWidth().Height(theme.Hairline).Shrink(0).Margin(12, 0, 0).Background(t.Border)
		m.idleHint(c, t)
	})
}

// sleepingBot is the robot of the idle card with its z's rising. Its box reaches above and to the
// right of the 40 x 48 pt the layout gives it, where the z's float.
func (m *Main) sleepingBot(c *ui.Context, t *theme.Tokens) {
	now, reduce := m.app.Now(), c.Preferences().ReduceMotion
	if !reduce {
		c.After(theme.SleepNext(now))
	}
	theme.SleepingBot(c, t, theme.SleepFrame(now, reduce)).Margin(-theme.SleepZRoom, -6, 0, 0)
}

// lastReporter is the line under the title: the avatar of the agent that reported most recently
// and "name · age". Without a report ever it is left out.
func (m *Main) lastReporter(c *ui.Context, t *theme.Tokens) {
	r := m.app.snap.lastReport
	if r == nil {
		return
	}
	ui.Row(c.Key("idle-last")).Gap(6).AlignItems(ui.Center).MinWidth(0).Children(func() {
		theme.AvatarIcon(c.Key("idle-who"), theme.AgentAvatar(r.Agent), 14).Shrink(0)
		line := m.say.AgentsIdleLast(agentName(r.Agent), m.clock.Age(r.CreatedAt, m.app.Now()))
		theme.Text(c, theme.Meta, t.TextSecondary, line).Shrink(1).MinWidth(0).Ellipsis("…")
	})
}

// idleHint is how to hand a task over, its key words in the primary text color and weight 500.
func (m *Main) idleHint(c *ui.Context, t *theme.Tokens) {
	hint, emph := m.say.AgentsIdleHint(), m.say.AgentsIdleHintEmph()
	before, after, found := strings.Cut(hint, emph)
	ui.Row(c.Key("idle-hint")).Margin(10, 0, 0).MinWidth(0).Children(func() {
		if !found {
			theme.Text(c, theme.Meta, t.TextSecondary, hint).Shrink(1).MinWidth(0).Ellipsis("…")
			return
		}
		for _, part := range []struct {
			text  string
			style theme.Style
			color ui.Color
		}{{before, theme.Meta, t.TextSecondary}, {emph, theme.MetaStrong, t.Text}, {after, theme.Meta, t.TextSecondary}} {
			if part.text != "" {
				theme.Text(c, part.style, part.color, part.text).Shrink(0)
			}
		}
	})
}
