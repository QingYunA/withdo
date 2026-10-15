package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The model chip is at the bottom right of a focused composer: the logo of the provider in use and
// "Provider · model ⌄". It opens a drawn popover (a native menu cannot show logos) that lists the
// providers that are set up, each with its model, the one in use checked. Choosing one makes it the
// provider in use, the same as in Settings > Assistant.

const (
	modelPopWidth float32 = 270
	modelChipH    float32 = 24
	modelRowH     float32 = 32
	modelLogo     float32 = 16
)

// providerMark is the logo of a provider at size points. It is the one place the composer draws a
// logo, so it takes one line to change.
func providerMark(c *ui.Context, p data.Provider, size float32) ui.Element {
	return theme.ProviderLogo(c, p.ID, size)
}

// modelsReady are the providers that are set up, in Settings' order, and the one in use among them.
// None is in use while no model is connected; the chip is not drawn then.
func (m *Main) modelsReady() (all []data.Provider, active data.Provider, ok bool) {
	if m.app.Models == nil || !m.assistReady() {
		return nil, data.Provider{}, false
	}
	for _, p := range m.app.Models.Providers() {
		if p.Configured {
			all = append(all, p)
		}
		if p.Active {
			active, ok = p, true
		}
	}
	return all, active, ok
}

// modelText is what the chip and a row of the popover say of a provider: "DeepSeek · deepseek-chat".
func modelText(p data.Provider) string {
	if p.Model == "" {
		return p.Name
	}
	return p.Name + " · " + p.Model
}

// modelChip is the chip and, while it is open, its popover.
func (m *Main) modelChip(c *ui.Context, t *theme.Tokens, s surface) {
	cs := &m.cmp[s]
	all, active, ok := m.modelsReady()
	if !ok {
		cs.modelOpen = false
		return
	}
	text := modelText(active)
	chip := ui.ButtonBase(c.Key("model-chip-"+itoa(int(s)))).Height(modelChipH).Shrink(1).MinWidth(0).Padding(0, 8, 0, 6).Gap(6).
		Radius(7).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(m.say.ComposerModelLabel(text))
	switch {
	case cs.modelOpen || chip.Hovered():
		chip.Background(t.BgSubtle.Mix(t.Text, 0.05))
	default:
		chip.Background(t.BgSubtle)
	}
	chip.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	chip.Children(func() {
		providerMark(c, active, modelLogo).Shrink(0)
		theme.Text(c, theme.Meta, t.TextSecondary, text).Shrink(1).MinWidth(0).Ellipsis("…")
		ui.Icon(c, theme.IconChevron).Size(12, 12).TextColor(t.TextTertiary).Shrink(0)
	})
	if chip.Clicked() {
		cs.modelOpen = !cs.modelOpen
	}
	ui.PopoverBase(c, chip, &cs.modelOpen, func(panel ui.Element) {
		panel.AttachTo(chip, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0)
		panel.Width(modelPopWidth).Padding(popPad).Radius(theme.RadiusPanel).Background(t.Bg).Border(1, t.Border).
			Role(ui.RoleDialog).Label(m.say.ComposerModelTitle()).
			Shadow(0, 2, 6, 0, t.Shadow.Alpha(0.25)).Shadow(0, 12, 36, 0, t.Shadow.Alpha(0.6))
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.ComposerModelTitle()).Padding(6, 9, 2).Shrink(0)
		for _, p := range all {
			m.modelRow(c, t, s, p)
		}
	})
}

// modelRow is one provider of the popover: its logo, its name, its model and a check on the one in
// use.
func (m *Main) modelRow(c *ui.Context, t *theme.Tokens, s surface, p data.Provider) {
	cs := &m.cmp[s]
	row := ui.ButtonBase(c.Key("model-"+itoa(int(s))+"-"+p.ID)).Height(modelRowH).Shrink(0).Padding(0, 9).Gap(9).Radius(7).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(m.say.ComposerModelLabel(modelText(p)))
	switch {
	case p.Active:
		row.Background(t.BgSelected)
	case row.Hovered():
		row.Background(t.BgHover)
	}
	row.Children(func() {
		providerMark(c, p, modelLogo).Shrink(0)
		theme.Text(c, theme.Compact, t.Text, p.Name).Shrink(0)
		theme.Text(c, theme.Meta, t.TextTertiary, p.Model).Grow(1).MinWidth(0).Ellipsis("…").TextAlign(ui.End)
		if p.Active {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.Accent).Shrink(0)
		}
	})
	if row.Clicked() {
		if !p.Active {
			_ = m.app.Models.SetActive(p.ID)
		}
		cs.modelOpen = false
		m.refocus(s)
	}
}

// modelName is the model in use as an error names it: "deepseek-chat", else the provider's name.
func (m *Main) modelName() string {
	_, p, ok := m.modelsReady()
	switch {
	case !ok:
		return ""
	case p.Model != "":
		return p.Model
	}
	return p.Name
}
