package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/words"
	"github.com/egoist/mygo/ui"
)

// Pane is one toolbar pane of the Settings window.
type Pane int

const (
	PaneGeneral Pane = iota
	PaneAssistant
	PaneAgents
	PaneGitHub
)

// settingsWidth is the window's width.
const settingsWidth = theme.SettingsWidth

const copiedFor = 1500 * time.Millisecond

// Settings is the content of the Settings window: a toolbar of panes over grouped lists. It has
// the state of its panes; the data comes from the App the main window shares.
type Settings struct {
	app    *App
	say    locales.Messages
	clock  words.Clock
	pane   Pane
	assist assistState
	copied time.Time

	// OnLayout is told the open pane's name, for the window's title, and the height the window's
	// content needs, whenever either changes. The shell sets the title and the size.
	OnLayout   func(title string, height float32)
	lastTitle  string
	lastHeight float32
}

// NewSettings opens on General.
func NewSettings(a *App) *Settings { return &Settings{app: a} }

// Show opens a pane.
func (s *Settings) Show(p Pane) { s.pane = p }

// View builds the window for one frame.
func (s *Settings) View(c *ui.Context) {
	t := theme.Resolve(c)
	s.say, s.clock = locales.In(c.Locale()), words.New(c.Locale(), c.Region())
	var toolbar, body ui.Element
	ui.Column(c).Fill().Background(t.SettingsPane()).Children(func() {
		toolbar = s.toolbar(c, t)
		body = ui.Column(c.Key("pane")).Shrink(0).Padding(18, theme.SettingsPad, 6).Children(func() { s.paneBody(c, t) })
	})
	s.layout(toolbar.Bounds().H + body.Bounds().H)
}

// layout tells the shell the title and the height, when they changed. The bounds are the previous
// frame's, so the first frame reports nothing.
func (s *Settings) layout(height float32) {
	title := s.paneName(s.pane)
	if s.OnLayout == nil || height == 0 || (title == s.lastTitle && abs(height-s.lastHeight) < 1) {
		return
	}
	s.lastTitle, s.lastHeight = title, height
	s.OnLayout(title, height)
}

func abs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func (s *Settings) paneName(p Pane) string {
	switch p {
	case PaneAssistant:
		return s.say.SettingsPaneAssistant()
	case PaneAgents:
		return s.say.SettingsPaneAgents()
	case PaneGitHub:
		return s.say.SettingsGitHub()
	}
	return s.say.SettingsGeneral()
}

var paneIcons = map[Pane]*ui.SVG{
	PaneGeneral: theme.IconGear, PaneAssistant: theme.IconSparkle, PaneAgents: theme.IconAgents, PaneGitHub: theme.IconGitHub,
}

// toolbar is the band of panes, each an icon over its name, over a hairline.
func (s *Settings) toolbar(c *ui.Context, t *theme.Tokens) ui.Element {
	return ui.Column(c.Key("toolbar")).Shrink(0).Background(t.SettingsToolbar()).Children(func() {
		ui.Row(c).Justify(ui.Center).Gap(4).Padding(10, 0, 12).Children(func() {
			for _, p := range []Pane{PaneGeneral, PaneAssistant, PaneAgents, PaneGitHub} {
				s.tab(c, t, p)
			}
		})
		hairline(c, t)
	})
}

func (s *Settings) tab(c *ui.Context, t *theme.Tokens, p Pane) {
	on := s.pane == p
	name := s.paneName(p)
	btn := ui.ButtonBase(c.Key("pane-"+itoa(int(p)))).Width(theme.SettingsTab).Padding(6, 0).Gap(2).Radius(theme.RadiusCard).
		Label(s.say.A11yPane(name)).Cursor(ui.CursorPointer).Column().AlignItems(ui.Center)
	switch {
	case on:
		btn.Background(t.SettingsTabOn())
	case btn.Hovered():
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		ink, label := t.TextSecondary, t.TextSecondary
		if on {
			ink, label = t.Accent, t.Text
		}
		ui.Icon(c, paneIcons[p]).Size(20, 20).TextColor(ink)
		theme.Text(c, theme.Meta, label, name)
	})
	if btn.Clicked() {
		s.pane = p
	}
}

func (s *Settings) paneBody(c *ui.Context, t *theme.Tokens) {
	switch s.pane {
	case PaneAssistant:
		s.assistantPane(c, t)
	case PaneAgents:
		s.agentsPane(c, t)
	case PaneGitHub:
		s.githubPane(c, t)
	default:
		s.generalPane(c, t)
	}
}

// settingsLink is a link in a row: accent text with no box.
func settingsLink(c *ui.Context, t *theme.Tokens, key, text, label string) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Padding(2, 4).Radius(theme.RadiusControl).Shrink(0).Label(label).Cursor(ui.CursorPointer)
	ink := t.Accent
	if btn.Hovered() {
		ink = t.Accent.Mix(t.Text, 0.2)
	}
	btn.Children(func() { theme.Text(c, theme.Body, ink, text) })
	return btn
}
