package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// agentsRefreshEvery redraws the pane so that "2m" keeps up with the clock.
const agentsRefreshEvery = 30 * time.Second

// agentAvatarSize is the pixel avatar of an agent in the list.
const agentAvatarSize float32 = 36

// agentsCache is the agents that reported, read when the data changed or the day turned rather
// than every frame.
type agentsCache struct {
	rev   int
	day   string
	list  []data.AgentActivity
	valid bool
}

// agentsPane is the agents that ever reported, each with its avatar, how many times it reported
// today and when it last did, then how to connect one.
func (s *Settings) agentsPane(c *ui.Context, t *theme.Tokens) {
	c.After(agentsRefreshEvery)
	if list := s.agentList(); len(list) > 0 {
		groupLabel(c, t, s.say.SettingsAgentsConnected())
		group(c, t, "agent-list", func() {
			for i, a := range list {
				s.agentRow(c, t, a, i == 0)
			}
		})
	}
	group(c, t, "agents", func() {
		ui.Column(c).Shrink(0).Padding(12, 14, 14).Gap(8).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, s.say.SettingsConnectTitle())
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsConnectBody(), 0)
			commandWell(c, t, s.app.Config.ConnectCommand, func() { s.copyButton(c, t) })
		})
	})
}

// agentList is the agents, latest first, from the store when it can say; read again when the data
// changed or a new day began.
func (s *Settings) agentList() []data.AgentActivity {
	src, ok := s.app.Store.(data.AgentSource)
	if !ok {
		return nil
	}
	now := s.app.Now()
	day := now.Format("2006-01-02")
	if !s.agents.valid || s.agents.rev != s.app.rev || s.agents.day != day {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		list, err := src.AgentActivity(start)
		if err != nil {
			list = nil
		}
		s.agents = agentsCache{rev: s.app.rev, day: day, list: list, valid: true}
	}
	return s.agents.list
}

func (s *Settings) agentRow(c *ui.Context, t *theme.Tokens, a data.AgentActivity, first bool) {
	sub := s.say.SettingsAgentIdle()
	if a.Since > 0 {
		sub = s.say.SettingsAgentReports(a.Since)
	}
	settingsRowLead(c, t, "agent-"+a.Agent, agentName(a.Agent), sub, first, func() {
		theme.Text(c, theme.Meta, t.TextSecondary, s.clock.Age(a.Last, s.app.Now()))
	}, func() { theme.AvatarIcon(c, theme.AgentAvatar(a.Agent), agentAvatarSize).Shrink(0) })
}

// commandWell is a command in the mono face on a quiet well, with what sits beside it.
func commandWell(c *ui.Context, t *theme.Tokens, command string, beside func()) {
	ui.Row(c).Gap(8).AlignItems(ui.Start).Padding(10, 10, 10, 12).Radius(theme.RadiusCard).Background(t.BgSubtle).Children(func() {
		theme.Paragraph(c, theme.Mono, t.Text, command, 0).Grow(1).MinWidth(0).Selectable()
		if beside != nil {
			beside()
		}
	})
}

func (s *Settings) copyButton(c *ui.Context, t *theme.Tokens) {
	copied := !s.copied.IsZero() && s.app.Now().Sub(s.copied) < copiedFor
	btn := ui.ButtonBase(c.Key("copy")).Height(theme.HitTarget).Padding(0, 6).Gap(4).Radius(theme.RadiusControl).Shrink(0).
		AlignItems(ui.Center).Label(s.say.A11yCopyCommand()).TextColor(t.TextSecondary).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		if copied {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.TextSecondary)
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsCopied())
			return
		}
		ui.Icon(c, theme.IconCopy).Size(14, 14)
		theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsCopy())
	})
	if btn.Clicked() {
		c.WriteClipboard(s.app.Config.ConnectCommand)
		s.copied = s.app.Now()
		c.After(copiedFor + 100*time.Millisecond)
	}
}
