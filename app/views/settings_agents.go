package views

import (
	"context"
	"errors"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/egoist/mygo/ui"
)

// agentsRefreshEvery redraws the pane, and looks at this Mac again, so that an agent installed
// meanwhile shows up.
const agentsRefreshEvery = 30 * time.Second

// agentAvatarSize is the pixel avatar of an agent in the list.
const agentAvatarSize float32 = 24

// agentsCache is the agents that reported, read when the data changed or the day turned rather
// than every frame.
type agentsCache struct {
	rev   int
	day   string
	list  []data.AgentActivity
	valid bool
}

// setupState is what Settings > Agents knows of this Mac: what the last look found, which install
// is running, why one failed, and the snippet to copy for a config file that cannot be rewritten.
type setupState struct {
	status    map[string]agentsetup.Status
	scannedAt time.Time
	busy      string            // the agent whose install is running
	failed    map[string]string // agent id -> why its last install failed, in words
	manual    *manualHint
}

// manualHint is a config file the app will not rewrite, and what to add to it by hand.
type manualHint struct {
	id, file, snippet string
}

// agentsPane is the agents adhd-todo can set up, ten of them in two columns: for each, whether it
// is on this Mac, how often it reported today, and whether adhd-todo's MCP is installed in it or
// can be, or where to get the agent. Under them, the one command for Claude Code to copy.
func (s *Settings) agentsPane(c *ui.Context, t *theme.Tokens) {
	c.After(agentsRefreshEvery)
	groupLabel(c, t, s.say.SettingsAgentsTitle())
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsAgentsBody(), 0).Margin(0, 4, 14)
	status, reported := s.scanAgents(), s.reports()
	ui.Row(c).Shrink(0).Gap(12).AlignItems(ui.Start).Margin(0, 0, 18).Children(func() {
		for col := 0; col < 2; col++ {
			ui.Column(c.Key("agent-col-"+itoa(col))).Grow(1).Basis(0).MinWidth(0).Radius(theme.SettingsRadius).
				Background(t.SettingsGroup()).Border(1, t.Border.Alpha(0.08)).Children(func() {
				for i := col; i < len(agentsetup.Agents); i += 2 {
					a := agentsetup.Agents[i]
					s.agentRow(c, t, a, status[a.ID], reported[a.Name], i < 2)
				}
			})
		}
	})
	if h := s.setup.manual; h != nil {
		s.manualPanel(c, t, h)
	}
	group(c, t, "open-with", func() {
		settingsRow(c, t, "open-with", s.say.SettingsOpenWith(), s.say.SettingsOpenWithBody(), true, func() { s.openWithControl(c, t) })
	})
	ui.Column(c).Shrink(0).Gap(8).Margin(0, 0, 18).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsAgentsCommand()).Margin(0, 4)
		commandWell(c, t, s.app.Config.ConnectCommand, func() { s.copyButton(c, t, "copy", s.app.Config.ConnectCommand) })
	})
}

// openWithControl chooses where Review and Answer on a handed-over row take the user.
func (s *Settings) openWithControl(c *ui.Context, t *theme.Tokens) {
	opts := make([]popOption, 0, len(openWiths))
	for _, how := range openWiths {
		opts = append(opts, popOption{string(how), openWithName(s.say, how)})
	}
	if v, ok := popUp(c, t, "open-with", string(s.app.OpenWith()), opts); ok {
		s.app.SetOpenWith(OpenWith(v))
	}
}

// scanAgents is what is on this Mac, looked at on first use and again every agentsRefreshEvery
// (and after an install). Without an Env nothing is found.
func (s *Settings) scanAgents() map[string]agentsetup.Status {
	env := s.app.Config.AgentSetup
	if env == nil {
		return nil
	}
	if now := s.app.Now(); s.setup.status == nil || now.Sub(s.setup.scannedAt) >= agentsRefreshEvery || now.Before(s.setup.scannedAt) {
		s.setup.status, s.setup.scannedAt = env.DetectAll(), now
	}
	return s.setup.status
}

// reports is how an agent did: its reports today, and whether it ever reported, by the agent's
// name as the list shows it. An agent that reported has the MCP installed somewhere.
type agentReports struct {
	today int
	ever  bool
}

func (s *Settings) reports() map[string]agentReports {
	out := map[string]agentReports{}
	for _, a := range s.agentList() {
		r := out[agentName(a.Agent)]
		r.today += a.Since
		r.ever = true
		out[agentName(a.Agent)] = r
	}
	return out
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

// agentRow is one agent: its avatar, its name, one line on how it stands and, at the right, what
// can be done: nothing (the MCP is installed), Install MCP, or Get it.
func (s *Settings) agentRow(c *ui.Context, t *theme.Tokens, a agentsetup.Agent, st agentsetup.Status, rep agentReports, first bool) {
	installed := st.Installed || rep.ever
	sub := s.agentSub(st, rep)
	if !first {
		ui.Box(c.Key("rule-agent-"+a.ID)).Height(1).Shrink(0).Margin(0, 14).Background(t.Border)
	}
	ui.Row(c.Key("agent-"+a.ID)).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(func() {
		theme.AvatarIcon(c, theme.AgentAvatar(a.Name), agentAvatarSize).Shrink(0)
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			theme.Text(c, theme.Body, t.Text, a.Name).Ellipsis("…")
			theme.Text(c, theme.Meta, t.TextSecondary, sub).Ellipsis("…")
			if why := s.setup.failed[a.ID]; why != "" {
				theme.Paragraph(c, theme.Meta, t.Blocked(), why, 3)
			}
		})
		switch {
		case installed:
			ui.Row(c).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, theme.IconCheck).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Status(data.Done))
				theme.Text(c, theme.Meta, t.Status(data.Done), s.say.SettingsAgentInstalled())
			})
		case s.setup.busy == a.ID:
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsAgentInstalling()).Shrink(0)
		case st.Found:
			if settingsButton(c, t, "install-"+a.ID, s.say.SettingsAgentInstall(), true).Label(s.say.A11yInstallMCP(a.Name)).Clicked() {
				s.install(a) // inert while another install runs
			}
		default:
			if settingsLink(c, t, "get-"+a.ID, s.say.SettingsAgentGet(), s.say.A11yGetAgent(a.Name)).Clicked() {
				s.app.Act.OpenURL(a.URL)
			}
		}
	})
}

func (s *Settings) agentSub(st agentsetup.Status, rep agentReports) string {
	switch {
	case rep.today > 0:
		return s.say.SettingsAgentReports(rep.today)
	case rep.ever:
		return s.say.SettingsAgentIdle()
	case st.Found:
		return s.say.SettingsAgentFound(st.Via)
	}
	return s.say.SettingsAgentAbsent()
}

// install adds the MCP to an agent off the UI thread: its own command can take a while.
func (s *Settings) install(a agentsetup.Agent) {
	env := s.app.Config.AgentSetup
	if env == nil || s.setup.busy != "" {
		return
	}
	s.setup.busy = a.ID
	delete(s.setup.failed, a.ID)
	if s.setup.manual != nil && s.setup.manual.id == a.ID {
		s.setup.manual = nil
	}
	post := s.app.Act.Post
	go func() {
		err := env.Install(context.Background(), a)
		post(func() { s.finishInstall(env, a, err) })
	}()
}

func (s *Settings) finishInstall(env *agentsetup.Env, a agentsetup.Agent, err error) {
	s.setup.busy = ""
	if err == nil {
		s.setup.status, s.setup.scannedAt = env.DetectAll(), s.app.Now()
		return
	}
	if s.setup.failed == nil {
		s.setup.failed = map[string]string{}
	}
	var (
		manual *agentsetup.ManualError
		cmd    *agentsetup.CommandError
	)
	switch {
	case errors.As(err, &manual):
		s.setup.manual = &manualHint{id: a.ID, file: manual.File, snippet: manual.Snippet}
		s.setup.failed[a.ID] = ""
	case errors.Is(err, agentsetup.ErrNoCore):
		s.setup.failed[a.ID] = s.say.SettingsAgentNoCore(withTilde(env.CorePath(), env.Home))
	case errors.Is(err, agentsetup.ErrNotFound):
		s.setup.failed[a.ID] = s.say.SettingsAgentGone(a.Name)
	case errors.Is(err, agentsetup.ErrTimeout):
		s.setup.failed[a.ID] = s.say.SettingsAgentTimeout(a.Name)
	case errors.As(err, &cmd) && cmd.Output != "":
		s.setup.failed[a.ID] = s.say.SettingsAgentFailed(a.Name, cmd.Output)
	default:
		s.setup.failed[a.ID] = s.say.SettingsAgentCannot(err.Error())
	}
}

// manualPanel is the snippet to add by hand to a config file the app would not rewrite.
func (s *Settings) manualPanel(c *ui.Context, t *theme.Tokens, h *manualHint) {
	ui.Column(c.Key("agent-manual")).Shrink(0).Gap(8).Margin(0, 0, 18).Children(func() {
		theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsAgentManual(withTilde(h.file, s.app.Config.AgentSetup.Home)), 0).Margin(0, 4)
		commandWell(c, t, h.snippet, func() { s.copyButton(c, t, "copy-manual", h.snippet) })
	})
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

// copyButton copies text; for copiedFor afterwards it says Copied. key tells two buttons apart.
func (s *Settings) copyButton(c *ui.Context, t *theme.Tokens, key, text string) {
	copied := s.copiedKey == key && !s.copied.IsZero() && s.app.Now().Sub(s.copied) < copiedFor
	btn := ui.ButtonBase(c.Key(key)).Height(theme.HitTarget).Padding(0, 6).Gap(4).Radius(theme.RadiusControl).Shrink(0).
		AlignItems(ui.Center).Label(s.say.A11yCopyCommand()).TextColor(t.TextSecondary).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		if copied {
			ui.Icon(c, theme.IconCheck).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextSecondary)
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsCopied())
			return
		}
		ui.Icon(c, theme.IconCopy).Size(theme.IconSmall, theme.IconSmall)
		theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsCopy())
	})
	if btn.Clicked() {
		c.WriteClipboard(text)
		s.copied, s.copiedKey = s.app.Now(), key
		c.After(copiedFor + 100*time.Millisecond)
	}
}
