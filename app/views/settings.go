package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

const (
	sheetWidth float32 = 640
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
			if m.models.tab == tabModels && m.app.Models != nil {
				m.modelsTab(c, t)
				return
			}
			m.connectBlock(c, t)
			m.dataBlock(c, t)
			m.languageBlock(c, t)
			hairline(c, t)
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.SettingsAbout(m.app.Config.Version))
		})
	})
}

func (m *Main) sheetHeader(c *ui.Context, t *theme.Tokens) {
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Title, t.Text, m.say.SettingsTitle())
		ui.Spacer(c)
		if m.app.Models != nil {
			if i := segmented(c, t, "settings-tab", []string{m.say.SettingsGeneral(), m.say.SettingsModels()}, int(m.models.tab)); i >= 0 {
				m.models.tab = settingsTab(i)
			}
			ui.Box(c).Width(12).Shrink(0)
		}
		// Enter belongs to the fields on the Models tab, where it saves.
		enter := m.models.tab == tabGeneral && c.Shortcut(0, ui.KeyEnter)
		if ui.PrimaryButton(c.Key("done"), m.say.SettingsDone()).Label(m.say.A11ySettingsClose()).Clicked() || enter {
			m.settings = false
		}
	})
}

func (m *Main) connectBlock(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Gap(8).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, m.say.SettingsConnectTitle())
		theme.Paragraph(c, theme.Meta, t.TextSecondary, m.say.SettingsConnectBody(), 0)
		ui.Row(c).Gap(8).AlignItems(ui.Start).Padding(10, 10, 10, 12).Radius(theme.RadiusCard).Background(t.BgSubtle).Children(func() {
			theme.Paragraph(c, theme.Mono, t.Text, m.app.Config.ConnectCommand, 0).Grow(1).MinWidth(0).Selectable()
			m.copyButton(c, t)
		})
	})
}

func (m *Main) copyButton(c *ui.Context, t *theme.Tokens) {
	copied := !m.copied.IsZero() && m.app.Now().Sub(m.copied) < copiedFor
	btn := ui.ButtonBase(c.Key("copy")).Height(theme.HitTarget).Padding(0, 6).Gap(4).Radius(theme.RadiusControl).Shrink(0).
		AlignItems(ui.Center).Label(m.say.A11yCopyCommand()).TextColor(t.TextSecondary)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		if copied {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.TextSecondary)
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.SettingsCopied())
			return
		}
		ui.Icon(c, theme.IconCopy).Size(14, 14)
		theme.Text(c, theme.Meta, t.TextSecondary, m.say.SettingsCopy())
	})
	if btn.Clicked() {
		c.WriteClipboard(m.app.Config.ConnectCommand)
		m.copied = m.app.Now()
		c.After(copiedFor + 100*time.Millisecond)
	}
}

// languageBlock chooses the language of the interface: the system's, or one the app is written
// in, each named in itself.
func (m *Main) languageBlock(c *ui.Context, t *theme.Tokens) {
	tags := m.app.Act.Languages()
	labels := []string{m.say.LanguageSystem()}
	selected := 0
	for i, tag := range tags {
		labels = append(labels, i18n.Autonym(tag))
		if tag == m.app.Act.Language() {
			selected = i + 1
		}
	}
	ui.Column(c).Gap(8).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, m.say.SettingsLanguage())
		ui.Row(c).Children(func() {
			if i := segmented(c, t, "language", labels, selected); i == 0 {
				m.app.Act.SetLanguage("")
			} else if i > 0 {
				m.app.Act.SetLanguage(tags[i-1])
			}
		})
	})
}

func (m *Main) dataBlock(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Gap(4).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, m.say.SettingsDataFolder())
		theme.Paragraph(c, theme.Mono, t.TextSecondary, m.app.Config.DataFolder, 0).Selectable()
	})
}
