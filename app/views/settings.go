package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const (
	sheetWidth float32 = 460
	copiedFor          = 1500 * time.Millisecond
)

// settingsSheet is a small sheet over the dimmed window: how to connect Claude Code, where the
// data lives, and an About line. There are no shortcut settings.
func (m *Main) settingsSheet(c *ui.Context, t *theme.Tokens) {
	ui.DialogBase(c, &m.settings, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.32))
		panel.Width(sheetWidth).Padding(24).Radius(theme.RadiusPanel).Background(t.Bg).Border(1, t.Border).
			Shadow(0, 20, 48, 0, t.Shadow)
		ui.Column(c).Gap(20).Children(func() {
			m.sheetHeader(c, t)
			m.connectBlock(c, t)
			m.dataBlock(c, t)
			hairline(c, t)
			theme.Text(c, theme.Meta, t.TextSecondary, "adhd-todo "+m.app.Config.Version+"  ·  Open source, MIT license")
		})
	})
}

func (m *Main) sheetHeader(c *ui.Context, t *theme.Tokens) {
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Title, t.Text, "Settings")
		ui.Spacer(c)
		if ui.PrimaryButton(c.Key("done"), "Done").Label("Close settings").Clicked() || c.Shortcut(0, ui.KeyEnter) {
			m.settings = false
		}
	})
}

func (m *Main) connectBlock(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Gap(8).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, "Connect Claude Code")
		theme.Paragraph(c, theme.Meta, t.TextSecondary, "Run this once in a terminal. Agents then report their progress here.", 0)
		ui.Row(c).Gap(8).AlignItems(ui.Start).Padding(10, 10, 10, 12).Radius(theme.RadiusCard).Background(t.BgSubtle).Children(func() {
			theme.Paragraph(c, theme.Mono, t.Text, m.app.Config.ConnectCommand, 0).Grow(1).MinWidth(0).Selectable()
			m.copyButton(c, t)
		})
	})
}

func (m *Main) copyButton(c *ui.Context, t *theme.Tokens) {
	copied := !m.copied.IsZero() && m.app.Now().Sub(m.copied) < copiedFor
	btn := ui.ButtonBase(c.Key("copy")).Height(theme.HitTarget).Padding(0, 6).Gap(4).Radius(theme.RadiusControl).Shrink(0).
		AlignItems(ui.Center).Label("Copy command").TextColor(t.TextSecondary)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		if copied {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.Status(data.Done))
			theme.Text(c, theme.Meta, t.TextSecondary, "Copied")
			return
		}
		ui.Icon(c, theme.IconCopy).Size(14, 14)
		theme.Text(c, theme.Meta, t.TextSecondary, "Copy")
	})
	if btn.Clicked() {
		c.WriteClipboard(m.app.Config.ConnectCommand)
		m.copied = m.app.Now()
		c.After(copiedFor + 100*time.Millisecond)
	}
}

func (m *Main) dataBlock(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Gap(4).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, "Data folder")
		theme.Paragraph(c, theme.Mono, t.TextSecondary, m.app.Config.DataFolder, 0).Selectable()
	})
}
