package views

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/egoist/mygo/ui"
)

// agentsRefreshEvery redraws the pane, and looks at this Mac again, so that an agent installed
// meanwhile shows up.
const agentsRefreshEvery = 30 * time.Second

// The compact rows of More: the mark and the row itself.
const (
	agentMarkSizeMore float32 = 18
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
// is running, why one failed, the snippet to copy for a config file that cannot be rewritten, and
// which of the pane's two disclosures are open.
type setupState struct {
	status    map[string]agentsetup.Status
	scannedAt time.Time
	busy      string            // the agent whose install is running
	failed    map[string]string // agent id -> why its last install failed, in words
	manual    *manualHint
	moreOpen  bool // the "N more agents" list is open in place
	handOpen  bool // Connect by hand is open in place
}

// manualHint is a config file the app will not rewrite, and what to add to it by hand.
type manualHint struct {
	id, file, snippet string
}

// agentsPane is the agents adhd-todo can set up: the ones found here as pixel tiles, then one
// group of rows for the rest, the command to copy and where an agent's work opens. The GitHub
// group follows it.
func (s *Settings) agentsPane(c *ui.Context, t *theme.Tokens) {
	c.After(agentsRefreshEvery)
	paneHeader(c, t, s.say.SettingsPaneAgents(), s.say.SettingsAgentsBody())
	status, reported := s.scanAgents(), s.reports()
	local, more := splitAgents(status, reported)
	s.agentsSectionHead(c, t, "local", s.say.SettingsAgentsLocal(), len(local))
	if len(local) == 0 {
		s.agentsEmptyLocal(c, t)
	} else {
		s.agentsLocal(c, t, local, status, reported)
	}
	agentsGroup(c, t, "agent-actions", func() {
		first := true
		if len(more) > 0 {
			first = false
			s.agentsMoreDisclosure(c, t, more, true)
		}
		s.agentsHandDisclosure(c, t, first)
		settingsRow(c, t, "open-with", s.say.SettingsOpenWith(), s.say.SettingsOpenWithBody(), false, func() { s.openWithControl(c, t) })
	})
	if h := s.setup.manual; h != nil {
		s.manualPanel(c, t, h)
	}
	agentsGroupLabel(c, t, s.say.SettingsGitHub())
	s.githubPane(c, t)
}

// agentsGroupLabel is the small title above the GitHub group, tighter than a pane's first label
// so that the whole pane fits the window.
func agentsGroupLabel(c *ui.Context, t *theme.Tokens, text string) ui.Element {
	return theme.Text(c, theme.Section, t.TextSecondary, text).Margin(2, 4, 3)
}

// agentsGroup is a block of rows with a little less below it than a pane's other groups, so that
// the whole Agents pane fits the 580 pt window.
func agentsGroup(c *ui.Context, t *theme.Tokens, key string, rows func()) {
	ui.Column(c.Key("group-"+key)).Shrink(0).Margin(0, 0, 10).Radius(theme.SettingsRadius).Background(t.SettingsGroup()).
		Border(1, t.Border.Alpha(0.08)).Children(rows)
}

// agentsSettingsRowInk is a compact row of the GitHub group: a label and one quiet line, in the
// stale ink when something is missing.
func agentsSettingsRowInk(c *ui.Context, t *theme.Tokens, key, label, sub string, subInk ui.Color, first bool, control func()) {
	if !first {
		hairline(c, t).Margin(0, 14)
	}
	ui.Row(c.Key("row-"+key)).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(4, 14).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			theme.Text(c, theme.Compact, t.Text, label).Ellipsis("…")
			if sub != "" {
				theme.Text(c, theme.Meta, subInk, sub).Ellipsis("…")
			}
		})
		if control != nil {
			control()
		}
	})
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

// agentsSectionHead is a group label with its quiet count: "On this computer 6".
func (s *Settings) agentsSectionHead(c *ui.Context, t *theme.Tokens, key, label string, count int) {
	ui.Row(c.Key("agents-head-"+key)).Shrink(0).Gap(6).AlignItems(ui.Center).Margin(0, 4, 2).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, label)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(count))
	})
}

// agentsEmptyLocal is the one row On this computer has when nothing was found and nothing reported:
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

// agentsMoreDisclosure is the first row under the tiles: how many agents are only in the catalog,
// and, opened, the compact list of the catalog in place.
func (s *Settings) agentsMoreDisclosure(c *ui.Context, t *theme.Tokens, more []agentsetup.Agent, first bool) {
	sep := ", "
	if strings.HasPrefix(c.Locale(), "zh") {
		sep = "、"
	}
	s.agentDisclosure(c, t, "more", s.say.SettingsAgentsMoreCount(len(more)), s.say.SettingsAgentsMoreMeta(moreNames(more, 3, sep)), first, &s.setup.moreOpen, func() {
		s.agentsMore(c, t, more)
	})
}

// moreNames lists the first agents of More, an ellipsis saying there are more. It is the quiet
// line under "N more agents".
func moreNames(more []agentsetup.Agent, n int, sep string) string {
	names := make([]string, 0, n)
	for i, a := range more {
		if i == n {
			break
		}
		names = append(names, agentTileName(a))
	}
	out := strings.Join(names, sep)
	if len(more) > len(names) {
		out += "…"
	}
	return out
}

// agentsMore is the compact two-column list of the catalog's other agents, opened in place under
// its row. Each row has its official mark, its name and the link to get it.
func (s *Settings) agentsMore(c *ui.Context, t *theme.Tokens, more []agentsetup.Agent) {
	for i := 0; i < len(more); i += 2 {
		hairline(c, t).Margin(0, 14)
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

// agentsHandDisclosure is Connect by hand: the command that adds Withdo for Claude Code, its Copy
// and the setup guide, opened in place.
func (s *Settings) agentsHandDisclosure(c *ui.Context, t *theme.Tokens, first bool) {
	s.agentDisclosure(c, t, "hand", s.say.SettingsAgentsManual(), s.say.SettingsAgentsManualBody(), first, &s.setup.handOpen, func() {
		ui.Column(c.Key("agent-hand-open")).Shrink(0).Padding(0, 14, 12).Gap(8).Children(func() {
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsAgentsCommand()).Margin(0, 4)
			commandWell(c, t, s.app.Config.ConnectCommand, func() {
				s.copyButton(c, t, "copy-command", s.app.Config.ConnectCommand)
			})
			if env := s.app.Config.AgentSetup; env != nil {
				s.copyGuideButton(c, t, "copy-guide-hand", env)
			}
		})
	})
}

// agentDisclosure is one row that opens a piece of the pane in place: a label, its quiet line, and
// a chevron that points right while it is closed. The whole row is the button.
func (s *Settings) agentDisclosure(c *ui.Context, t *theme.Tokens, key, label, sub string, first bool, open *bool, body func()) {
	if !first {
		hairline(c, t).Margin(0, 14)
	}
	btn := ui.ButtonBase(c.Key("disclose-"+key)).FillWidth().Shrink(0).MinHeight(theme.SettingsRowMin).
		Padding(4, 14).Gap(12).AlignItems(ui.Center).Label(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			theme.Text(c, theme.Compact, t.Text, label).Ellipsis("…")
			if sub != "" {
				theme.Text(c, theme.Meta, t.TextSecondary, sub).Ellipsis("…")
			}
		})
		icon := theme.IconForward
		if *open {
			icon = theme.IconChevron
		}
		ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary).Shrink(0)
	})
	if btn.Clicked() {
		*open = !*open
	}
	if *open {
		body()
	}
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

// agentInstalled is whether the agent's MCP is here: installed, or reported at least once without
// only the old adhd-todo entry (which still needs the install).
func agentInstalled(st agentsetup.Status, rep agentReports) bool {
	return st.Installed || (rep.ever && !st.Legacy)
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
