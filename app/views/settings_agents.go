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

// The sizes of the two kinds of agent row: the mark and the row itself (docs/design/mockups/v14).
const (
	agentMarkSize     float32 = 24
	agentMarkSizeMore float32 = 18
	agentRowSize      float32 = 48
	agentMoreRowSize  float32 = 31
	agentsMoreGap     float32 = 18
)

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

// agentsPane is the agents adhd-todo can set up, fourteen of them, in two groups: On this Mac,
// the ones found here (two columns of rows: mark, name, how it stands, and whether the MCP is
// installed or can be), and More, the rest, with just the name and where to get it. Both groups
// fit the window before it scrolls; under them, the one command for Claude Code to copy.
func (s *Settings) agentsPane(c *ui.Context, t *theme.Tokens) {
	c.After(agentsRefreshEvery)
	// The title's margins are a little tighter than a group label's: both sections have to fit the
	// 520 pt window before it scrolls (docs/design/mockups/v14.html).
	theme.Text(c, theme.Section, t.TextSecondary, s.say.SettingsAgentsTitle()).Margin(0, 4, 4)
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsAgentsBody(), 0).Margin(0, 4, 4)
	status, reported := s.scanAgents(), s.reports()
	local, more := splitAgents(status, reported)
	s.agentsSectionHead(c, t, "local", s.say.SettingsAgentsLocal(), len(local), 0, 3)
	if len(local) == 0 {
		s.agentsEmptyLocal(c, t)
	} else {
		s.agentsLocal(c, t, local, status, reported)
	}
	s.agentsSectionHead(c, t, "more", s.say.SettingsAgentsMore(), len(more), 14, 0)
	s.agentsMore(c, t, more)
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
	if env := s.app.Config.AgentSetup; env != nil {
		group(c, t, "other-agent", func() {
			settingsRow(c, t, "other-agent", s.say.SettingsAgentOther(), s.say.SettingsAgentOtherBody(), true, func() {
				s.copyGuideButton(c, t, "copy-guide", env)
			})
		})
	}
}

// splitAgents sorts the catalog into the ones on this Mac and the ones still to get. An agent
// that ever reported counts as here, even when this look no longer finds its command: its MCP is
// installed somewhere. The order stays the catalog's in both groups, so rows do not jump.
func splitAgents(status map[string]agentsetup.Status, reported map[string]agentReports) (local, more []agentsetup.Agent) {
	for _, a := range agentsetup.Agents {
		if status[a.ID].Found || reported[a.Name].ever {
			local = append(local, a)
		} else {
			more = append(more, a)
		}
	}
	return local, more
}

// agentsSectionHead is a group label with its quiet count: "On this Mac 6".
func (s *Settings) agentsSectionHead(c *ui.Context, t *theme.Tokens, key, label string, count int, above, below float32) {
	ui.Row(c.Key("agents-head-"+key)).Shrink(0).Gap(6).AlignItems(ui.Center).Margin(above, 4, below).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, label)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(count))
	})
}

// agentsLocal is the two white columns of On this Mac: rows read left to right, then top to
// bottom, so the odd row of the catalog is in the left column.
func (s *Settings) agentsLocal(c *ui.Context, t *theme.Tokens, local []agentsetup.Agent, status map[string]agentsetup.Status, reported map[string]agentReports) {
	ui.Row(c).Shrink(0).Gap(12).AlignItems(ui.Start).Children(func() {
		for col := 0; col < 2; col++ {
			ui.Column(c.Key("agent-col-"+itoa(col))).Grow(1).Basis(0).MinWidth(0).Radius(theme.SettingsRadius).
				Background(t.SettingsGroup()).Border(theme.Hairline, t.Border.Alpha(0.08)).Children(func() {
				first := true
				for i, a := range local {
					if i%2 != col {
						continue
					}
					s.agentRow(c, t, a, status[a.ID], reported[a.Name], first)
					first = false
				}
			})
		}
	})
}

// agentsEmptyLocal is the one row On this Mac has when nothing was found and nothing reported:
// what that means, and where to read how to connect an agent. More still lists the catalog.
func (s *Settings) agentsEmptyLocal(c *ui.Context, t *theme.Tokens) {
	ui.Column(c.Key("agent-local-empty")).Shrink(0).Radius(theme.SettingsRadius).Background(t.SettingsGroup()).
		Border(1, t.Border.Alpha(0.08)).Children(func() {
		ui.Row(c.Key("agent-empty")).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(8, 14).Gap(10).AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsAgentsEmpty()).Grow(1).MinWidth(0).Ellipsis("…")
			if settingsLink(c, t, "connect", s.say.SettingsAgentsConnect(), s.say.SettingsAgentsConnect()).Clicked() {
				s.app.Act.OpenURL(repoURL + "#connect-claude-code")
			}
		})
	})
}

// agentsMore is the one block of More: the catalog's other agents, two to a row, compact,
// without a status line and without a fold.
func (s *Settings) agentsMore(c *ui.Context, t *theme.Tokens, more []agentsetup.Agent) {
	ui.Column(c.Key("agent-more")).Shrink(0).Radius(theme.SettingsRadius).Background(t.SettingsGroup()).
		Border(1, t.Border.Alpha(0.08)).Children(func() {
		for i := 0; i < len(more); i += 2 {
			if i > 0 {
				ui.Box(c.Key("rule-more-"+itoa(i))).Height(theme.Hairline).Shrink(0).Margin(0, 14).Background(t.Border)
			}
			ui.Row(c.Key("agent-more-"+itoa(i))).Height(agentMoreRowSize).Shrink(0).Gap(agentsMoreGap).Padding(0, 14).Children(func() {
				for j := 0; j < 2; j++ {
					if i+j < len(more) {
						s.moreRow(c, t, more[i+j])
					} else {
						ui.Box(c).Grow(1).Basis(0)
					}
				}
			})
		}
	})
}

// moreRow is one agent of More: its official mark, its name, and the link to get it. It says
// nothing about this Mac, because the whole group is the ones not on it.
func (s *Settings) moreRow(c *ui.Context, t *theme.Tokens, a agentsetup.Agent) {
	ui.Row(c.Key("more-" + a.ID)).Grow(1).Basis(0).MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.HarnessMark(c, a.ID, agentMarkSizeMore).Shrink(0)
		theme.Text(c, theme.Meta, t.Text, a.Name).Grow(1).MinWidth(0).Ellipsis("…")
		if settingsOutLink(c, t, "get-"+a.ID, s.say.SettingsAgentGet(), s.say.A11yGetAgent(a.Name)).Clicked() {
			s.app.Act.OpenURL(a.URL)
		}
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

// agentRow is one agent of On this Mac: its official mark, its name, one line on how it stands
// and, at the right, what can be done: nothing (the MCP is installed), Install MCP, or, for an
// agent whose report alone put it here, nothing either.
func (s *Settings) agentRow(c *ui.Context, t *theme.Tokens, a agentsetup.Agent, st agentsetup.Status, rep agentReports, first bool) {
	installed := st.Installed || rep.ever
	sub := s.agentSub(st, rep)
	if !first {
		ui.Box(c.Key("rule-agent-"+a.ID)).Height(theme.Hairline).Shrink(0).Margin(0, 14).Background(t.Border)
	}
	ui.Row(c.Key("agent-"+a.ID)).MinHeight(agentRowSize).Shrink(0).Padding(0, 14).Gap(10).AlignItems(ui.Center).Children(func() {
		theme.HarnessMark(c, a.ID, agentMarkSize).Shrink(0)
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
		}
	})
}

// agentSub is the line under a name of On this Mac: its reports today, that it reported before,
// or what was found. Every row here is here because one of those is true.
func (s *Settings) agentSub(st agentsetup.Status, rep agentReports) string {
	switch {
	case rep.today > 0:
		return s.say.SettingsAgentReports(rep.today)
	case rep.ever:
		return s.say.SettingsAgentIdle()
	}
	return s.say.SettingsAgentFound(st.Via)
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
		commandWell(c, t, h.snippet, func() {
			ui.Column(c).Shrink(0).AlignItems(ui.End).Children(func() {
				s.copyButton(c, t, "copy-manual", h.snippet)
				s.copyGuideButton(c, t, "copy-guide-manual", s.app.Config.AgentSetup)
			})
		})
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
	s.copyAs(c, t, key, text, s.say.SettingsCopy(), s.say.A11yCopyCommand())
}

// copyGuideButton copies the guide any agent can follow to add adhd-todo to itself.
func (s *Settings) copyGuideButton(c *ui.Context, t *theme.Tokens, key string, env *agentsetup.Env) {
	s.copyAs(c, t, key, env.Guide(), s.say.SettingsCopyGuide(), s.say.SettingsCopyGuide())
}

// copyAs is the copy button: the pixel copy icon and word, or the check and Copied after a click.
func (s *Settings) copyAs(c *ui.Context, t *theme.Tokens, key, text, word, label string) {
	copied := s.copiedKey == key && !s.copied.IsZero() && s.app.Now().Sub(s.copied) < copiedFor
	btn := ui.ButtonBase(c.Key(key)).Height(theme.HitTarget).Padding(0, 6).Gap(4).Radius(theme.RadiusControl).Shrink(0).
		AlignItems(ui.Center).Label(label).TextColor(t.TextSecondary).Cursor(ui.CursorPointer)
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
		theme.Text(c, theme.Meta, t.TextSecondary, word)
	})
	if btn.Clicked() {
		c.WriteClipboard(text)
		s.copied, s.copiedKey = s.app.Now(), key
		c.After(copiedFor + 100*time.Millisecond)
	}
}
