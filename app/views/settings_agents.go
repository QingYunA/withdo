package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// agentsPane is how to connect Claude Code: the command to run once, and a line on what agents
// then do.
func (s *Settings) agentsPane(c *ui.Context, t *theme.Tokens) {
	group(c, t, "agents", func() {
		ui.Column(c).Shrink(0).Padding(12, 14, 14).Gap(8).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, s.say.SettingsConnectTitle())
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsConnectBody(), 0)
			commandWell(c, t, s.app.Config.ConnectCommand, func() { s.copyButton(c, t) })
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
