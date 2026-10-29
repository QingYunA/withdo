package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/egoist/mygo/ui"
)

// The tiles of Settings > Agents (docs/design/mockups/v41.html, frame 2): five to a row, 8 pt
// apart, each a group block with the agent's avatar on a 56 pt stage.
const (
	agentTilePerRow         = 5
	agentTileGap    float32 = 8
	agentTileH      float32 = 142
	// agentTileTextW is the room a tile's name, line and error have inside its padding.
	agentTileTextW float32 = 100
)

// agentsLocal is the row of pixel tiles of the agents found here: five to a row, eight points
// apart. A connected agent's avatar bobs; a tile that is not connected draws it gray.
func (s *Settings) agentsLocal(c *ui.Context, t *theme.Tokens, local []agentsetup.Agent, status map[string]agentsetup.Status, reported map[string]agentReports) {
	now, reduce := s.app.Now(), c.Preferences().ReduceMotion
	if !reduce {
		for _, a := range local {
			if agentInstalled(status[a.ID], reported[a.Name]) {
				c.After(theme.AgentBobNext(now))
				break
			}
		}
	}
	cols := agentTilePerRow
	ui.Column(c.Key("agent-tiles")).Shrink(0).Gap(agentTileGap).Margin(0, 0, 8).Children(func() {
		for row := 0; row*cols < len(local); row++ {
			ui.Row(c.Key("agent-tile-row-" + itoa(row))).Shrink(0).Gap(agentTileGap).AlignItems(ui.Stretch).Children(func() {
				for col := 0; col < cols; col++ {
					i := row*cols + col
					if i >= len(local) {
						ui.Box(c).Grow(1).Basis(0)
						continue
					}
					a := local[i]
					s.agentTile(c, t, a, status[a.ID], reported[a.Name], i, now, reduce)
				}
			})
		}
	})
}

// agentTile is one agent of On this computer: its avatar on the stage, its name, one line on how
// it stands and its state: Connected, Connect, or Connecting… A failed install says why under
// the line, in the blocked red.
func (s *Settings) agentTile(c *ui.Context, t *theme.Tokens, a agentsetup.Agent, st agentsetup.Status, rep agentReports, index int, now time.Time, reduce bool) {
	connected := agentInstalled(st, rep)
	tile := ui.Column(c.Key("agent-tile-"+a.ID)).Grow(1).Basis(0).MinWidth(0).MinHeight(agentTileH).
		Padding(8, 4).Radius(theme.SettingsRadius).Background(t.SettingsGroup()).
		Border(theme.Hairline, t.Border.Alpha(0.08)).AlignItems(ui.Center)
	tile.Children(func() {
		avatar := theme.AgentAvatar(a.ID)
		if connected {
			theme.AgentAvatarOnStage(c, avatar, theme.AgentAvatarSize, theme.AgentAvatarStageH,
				theme.AgentBobOffset(now, index, reduce)).Margin(0, 0, 4)
		} else {
			theme.GrayAgentAvatarOnStage(c, avatar, theme.AgentAvatarSize, theme.AgentAvatarStageH).Margin(0, 0, 4)
		}
		theme.Text(c, theme.Section, t.Text, agentTileName(a)).MaxWidth(agentTileTextW).TextAlign(ui.Center).Ellipsis("…")
		theme.Text(c, theme.Meta, t.TextSecondary, s.agentTileSub(st, rep)).MaxWidth(agentTileTextW).TextAlign(ui.Center).Ellipsis("…")
		if why := s.setup.failed[a.ID]; why != "" {
			theme.Paragraph(c, theme.Meta, t.Blocked(), why, 3).MaxWidth(agentTileTextW).TextAlign(ui.Center).Margin(4, 0, 0)
		}
		s.agentTileState(c, t, a, st, connected)
	})
}

// agentTileState is the pill or button at the foot of a tile.
func (s *Settings) agentTileState(c *ui.Context, t *theme.Tokens, a agentsetup.Agent, st agentsetup.Status, connected bool) {
	switch {
	case connected:
		agentPill(c, t, s.say.SettingsAgentInstalled(), t.Status(data.Done), t.Status(data.Done).Alpha(0.12))
	case s.setup.busy == a.ID:
		agentPill(c, t, s.say.SettingsAgentInstalling(), t.TextSecondary, t.BgSubtle)
	case st.Found:
		btn := ui.ButtonBase(c.Key("install-"+a.ID)).Shrink(0).Height(24).Margin(6, 0, 0).Padding(0, 10).Radius(7).
			Background(t.Accent).AlignItems(ui.Center).Label(s.say.A11yInstallMCP(a.Name)).Cursor(ui.CursorPointer)
		if btn.Hovered() {
			btn.Background(t.Accent.Mix(t.Text, 0.12))
		}
		btn.Children(func() {
			theme.Text(c, theme.Meta, t.AccentText, s.say.SettingsAgentInstall())
		})
		if btn.Clicked() {
			s.install(a) // inert while another install runs
		}
	}
}

// agentPill is the quiet state of a tile: Connected in the done green, or Connecting… while the
// install runs.
func agentPill(c *ui.Context, t *theme.Tokens, text string, ink, well ui.Color) {
	ui.Row(c).Shrink(0).Height(24).Margin(6, 0, 0).Padding(0, 10).Radius(7).Background(well).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Meta, ink, text)
	})
}

// agentTileName is the name a tile shows: the short name when the catalog has a parenthetical,
// as "Copilot (VS Code)" is only "Copilot" on a 103 pt tile.
func agentTileName(a agentsetup.Agent) string {
	if i := strings.Index(a.Name, " ("); i > 0 {
		return a.Name[:i]
	}
	return a.Name
}

// agentTileSub is the one line under a tile's name: its reports today, that it reported before,
// or that it was found here.
func (s *Settings) agentTileSub(st agentsetup.Status, rep agentReports) string {
	switch {
	case rep.today > 0:
		return s.say.SettingsAgentReports(rep.today)
	case rep.ever:
		return s.say.SettingsAgentIdle()
	}
	return s.say.SettingsAgentFoundOnly()
}
