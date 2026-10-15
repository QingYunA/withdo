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

	// OnTitle is told the open pane's name, for the window's title, whenever it changes. The window
	// itself never changes size: see ContentSize.
	OnTitle   func(title string)
	lastTitle string

	agents agentsCache
}

// ContentSize is the size of the window's content: one size for every pane, so that changing the
// pane never resizes the window. A pane that needs more room scrolls.
func (s *Settings) ContentSize() (width, height float32) {
	return theme.SettingsWidth, theme.SettingsHeight
}

// NewSettings opens on General.
func NewSettings(a *App) *Settings { return &Settings{app: a} }

// Show opens a pane.
func (s *Settings) Show(p Pane) { s.pane = p }

// View builds the window for one frame. The toolbar keeps its height; the pane under it fills the
// rest and scrolls, and has a key of its own so that its scroll position is its own.
func (s *Settings) View(c *ui.Context) {
	t := theme.Resolve(c)
	s.say, s.clock = locales.In(c.Locale()), words.New(c.Locale(), c.Region())
	ui.Column(c).Fill().Background(t.SettingsPane()).Children(func() {
		s.toolbar(c, t)
		ui.Scroll(c.Key("pane-" + itoa(int(s.pane)))).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Shrink(0).Padding(18, theme.SettingsPad, 6).Children(func() { s.paneBody(c, t) })
		})
	})
	if title := s.paneName(s.pane); s.OnTitle != nil && title != s.lastTitle {
		s.lastTitle = title
		s.OnTitle(title)
	}
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
	PaneGeneral: theme.IconGeneral, PaneAssistant: theme.IconStar, PaneAgents: theme.IconTerminal, PaneGitHub: theme.IconOctocat,
}

// SettingsTitleBarHeight is the height of the band's title row: tall enough for the traffic
// lights.
const SettingsTitleBarHeight float32 = 38

// toolbar is one band for the whole top of the window, under and around the traffic lights: the
// open pane's name centered, the panes below it each as an icon over its name, and a hairline.
func (s *Settings) toolbar(c *ui.Context, t *theme.Tokens) ui.Element {
	return ui.Column(c.Key("toolbar")).Shrink(0).Background(t.SettingsToolbar()).Children(func() {
		ui.Row(c).Height(max(c.TitleBar().Height, SettingsTitleBarHeight)).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).
			DragWindow().Children(func() {
			theme.Text(c, theme.Section, t.Text, s.paneName(s.pane))
		})
		ui.Row(c).Justify(ui.Center).Gap(4).Padding(0, 0, 12).Children(func() {
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
