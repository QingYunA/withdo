package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/words"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/egoist/mygo/ui"
)

// Pane is one row of the Settings sidebar.
type Pane int

const (
	PaneAccount Pane = iota
	PaneGeneral
	PaneCategories
	PaneAgents
	PaneModel
	PaneAbout
)

const copiedFor = 1500 * time.Millisecond

// navPanes are the sidebar's rows, top to bottom. About sits at its foot, away from them.
var navPanes = []Pane{PaneAccount, PaneGeneral, PaneCategories, PaneAgents, PaneModel}

// panes are all the rows the arrow keys walk: the five above, then About.
var panes = []Pane{PaneAccount, PaneGeneral, PaneCategories, PaneAgents, PaneModel, PaneAbout}

// Settings is the content of the Settings window: a sidebar of panes beside grouped lists. It has
// the state of its panes; the data comes from the App the main window shares.
type Settings struct {
	app       *App
	say       locales.Messages
	clock     words.Clock
	pane      Pane
	navFocus  *Pane // the row the arrow keys moved to, focused once
	assist    assistState
	copied    time.Time
	copiedKey string // the copy button that was pressed
	setup     setupState

	// OnTitle is told the open pane's name, for the window's title, whenever it changes. The window
	// itself never changes size: see ContentSize.
	OnTitle   func(title string)
	lastTitle string

	agents agentsCache
	cats   categoriesState // Settings > Categories
	// The app mark in About, decoded once from Config.AppIcon.
	mark      *ui.Bitmap
	markTried bool

	// Settings > Account: whether this opening asked for the devices, and the inline Change field.
	syncAsked         bool
	syncLastSynced    time.Time
	syncHadConflict   bool
	syncChoice        SyncChoice
	syncHadOther      bool
	syncOtherChoice   int
	syncEditingServer bool
	syncServerDraft   string
	syncServerErr     bool
	// Settings > Account's sign-in fields. The password leaves with the pane.
	syncFields        fields
	syncLogin         string
	syncPassword      string
	syncShowPassword  bool
	syncFocusAccount  bool
	syncFocusPassword bool
	syncPrevPhase     data.SyncPhase
	// The code flow's card: the digits typed, the code already sent, the step last drawn, and
	// which server's kind was asked.
	syncCode      string
	syncCodeTried string
	syncFocusCode bool
	syncPrevStep  data.SignInStep
	syncProbed    string
}

// ContentSize is the size of the window's content: one size for every pane, so that changing the
// pane never resizes the window. A pane that needs more room scrolls.
func (s *Settings) ContentSize() (width, height float32) {
	return theme.SettingsWidth, theme.SettingsHeight
}

// NewSettings opens on General.
func NewSettings(a *App) *Settings {
	return &Settings{app: a, pane: PaneGeneral, syncFields: fields{}}
}

// Show opens a pane. Opening Account again asks the server for its devices again.
func (s *Settings) Show(p Pane) {
	if p == PaneAccount {
		s.syncAsked = false
		s.syncEditingServer, s.syncServerErr = false, false
		s.syncFocusAccount, s.syncFocusPassword = true, false
		s.syncPassword = ""
		s.syncCode, s.syncCodeTried, s.syncProbed = "", "", ""
	}
	if p == PaneAgents && s.pane != PaneAgents {
		s.setup.status = nil // look at this computer again as the pane opens
	}
	s.pane = p
}

// ClearSyncPassword forgets the password when the Settings window closes or leaves the pane.
func (s *Settings) ClearSyncPassword() {
	s.syncPassword = ""
	s.syncCode, s.syncCodeTried = "", ""
	s.syncFocusPassword = false
}

// openPane switches panes from the sidebar.
func (s *Settings) openPane(p Pane) {
	if p == PaneAccount && s.pane != PaneAccount {
		s.syncAsked = false
		s.syncFocusAccount, s.syncFocusPassword = true, false
	}
	if s.pane == PaneAccount && p != PaneAccount {
		s.syncPassword = ""
	}
	s.syncEditingServer, s.syncServerErr = false, false
	if p == PaneAgents && s.pane != PaneAgents {
		s.setup.status = nil // look at this computer again as the pane opens
	}
	s.pane = p
}

// View builds the window for one frame: the sidebar down the left, and the open pane beside it.
// The pane scrolls, and has a key of its own so that its scroll position is its own.
func (s *Settings) View(c *ui.Context) {
	t := theme.Resolve(c)
	s.say, s.clock = locales.In(c.Locale()), words.New(c.Locale(), c.Region())
	focus := s.navFocus
	s.navFocus = nil
	ui.Row(c).Fill().Background(t.SettingsPane()).Children(func() {
		s.sidebar(c, t, focus)
		ui.Column(c.Key("pane")).Grow(1).MinWidth(0).FillHeight().Children(func() {
			ui.Scroll(c.Key("pane-" + itoa(int(s.pane)))).Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Shrink(0).Padding(22, theme.SettingsPad, 6).Children(func() { s.paneBody(c, t) })
			})
		})
	})
	if s.pane == PaneCategories {
		s.categorySheetView(c, t)
	}
	if s.pane == PaneAccount && s.app.Sync != nil {
		snap := s.app.Sync.SyncSnapshot()
		s.syncSheet(c, t, snap)
		s.syncOtherSheet(c, t, snap)
	}
	if title := s.paneName(s.pane); s.OnTitle != nil && title != s.lastTitle {
		s.lastTitle = title
		s.OnTitle(title)
	}
}

func (s *Settings) paneName(p Pane) string {
	switch p {
	case PaneAccount:
		return s.say.SettingsPaneAccount()
	case PaneCategories:
		return s.say.SettingsCategories()
	case PaneAgents:
		return s.say.SettingsPaneAgents()
	case PaneModel:
		return s.say.SettingsPaneModel()
	case PaneAbout:
		return s.say.SettingsPaneAbout()
	}
	return s.say.SettingsGeneral()
}

var paneIcons = map[Pane]*ui.SVG{
	PaneAccount: theme.IconUser, PaneGeneral: theme.IconGeneral, PaneCategories: theme.IconFolder,
	PaneAgents: theme.IconAgent, PaneModel: theme.IconSparkle,
}

// SettingsTitleBarHeight is the height of the sidebar's first row: the room of the traffic lights,
// which drags the window.
const SettingsTitleBarHeight float32 = 38

// sidebar is the window's left side: the traffic lights' row, the pane rows, and About at the foot.
// Up and Down move between the rows while it has the keyboard.
func (s *Settings) sidebar(c *ui.Context, t *theme.Tokens, focus *Pane) {
	side := ui.Column(c.Key("sidebar")).Width(theme.SettingsSide).FillHeight().Shrink(0).
		Background(t.SettingsToolbar()).BorderWidth(0, theme.Hairline, 0, 0).BorderColor(t.Border)
	side.Children(func() {
		ui.Row(c.Key("sidebar-drag")).Height(max(c.TitleBar().Height, SettingsTitleBarHeight)).Shrink(0).DragWindow()
		ui.Column(c.Key("sidebar-panes")).Shrink(0).Gap(2).Padding(0, theme.SettingsSidePad).Children(func() {
			for _, p := range navPanes {
				s.paneRow(c, t, p, focus)
			}
		})
		ui.Box(c.Key("sidebar-space")).Grow(1)
		ui.Column(c.Key("sidebar-foot")).Shrink(0).Padding(0, theme.SettingsSidePad, 12).Children(func() {
			s.aboutRow(c, t, focus)
		})
	})
	switch {
	case side.Shortcut(0, ui.KeyDown):
		s.navMove(1)
	case side.Shortcut(0, ui.KeyUp):
		s.navMove(-1)
	}
}

// paneRow is one pane of the sidebar: its pixel icon and its name, in the well of the open pane.
func (s *Settings) paneRow(c *ui.Context, t *theme.Tokens, p Pane, focus *Pane) {
	on := s.pane == p
	name := s.paneName(p)
	btn := ui.ButtonBase(c.Key("pane-"+itoa(int(p)))).Width(theme.SettingsSide-2*theme.SettingsSidePad).
		Height(theme.SettingsSideRow).Padding(0, 10).Gap(10).Radius(8).Label(s.say.A11yPane(name)).
		Cursor(ui.CursorPointer).AlignItems(ui.Center)
	switch {
	case on:
		btn.Background(t.SettingsTabOn())
	case btn.Hovered():
		btn.Background(t.BgHover)
	}
	if focus != nil && *focus == p {
		btn.Focus()
	}
	btn.Children(func() {
		ink, label := t.TextSecondary, t.Text
		weight := 400
		if on {
			ink, weight = t.Accent, 600
		}
		ui.Icon(c, paneIcons[p]).Size(theme.IconRow, theme.IconRow).TextColor(ink).Shrink(0)
		theme.Text(c, navStyle(theme.Body, weight), label, name).Grow(1).MinWidth(0).Ellipsis("…")
		if p == PaneAgents && !on && s.navAgentAlert() {
			ui.Box(c.Key("pane-agent-alert")).Size(6, 6).Shrink(0).Margin(0, 2, 0, 0).Background(t.Accent)
		}
	})
	if btn.Clicked() {
		s.openPane(p)
	}
}

// aboutRow is the quiet row at the foot of the sidebar. It opens the About pane.
func (s *Settings) aboutRow(c *ui.Context, t *theme.Tokens, focus *Pane) {
	on := s.pane == PaneAbout
	name := s.say.SettingsPaneAbout()
	btn := ui.ButtonBase(c.Key("pane-"+itoa(int(PaneAbout)))).Width(theme.SettingsSide-2*theme.SettingsSidePad).
		Height(theme.SettingsSideRow).Padding(0, 10).Gap(10).Radius(8).Label(s.say.A11yPane(name)).
		Cursor(ui.CursorPointer).AlignItems(ui.Center)
	ink, weight := t.TextSecondary, 400
	switch {
	case on:
		btn.Background(t.SettingsTabOn())
		ink, weight = t.Text, 600
	case btn.Hovered():
		btn.Background(t.BgHover)
	}
	if focus != nil && *focus == PaneAbout {
		btn.Focus()
	}
	btn.Children(func() {
		icon := t.TextSecondary
		if on {
			icon = t.Accent
		}
		ui.Icon(c, theme.IconDiamond).Size(theme.IconRow, theme.IconRow).TextColor(icon).Shrink(0)
		theme.Text(c, navStyle(theme.Meta, weight), ink, s.say.SettingsSidebarAbout()).
			Grow(1).MinWidth(0).Ellipsis("…")
	})
	if btn.Clicked() {
		s.openPane(PaneAbout)
	}
}

// navStyle is a text style at another weight, for the open row of the sidebar.
func navStyle(base theme.Style, weight int) theme.Style {
	base.Weight = weight
	return base
}

// navMove moves the sidebar's focus by dir through the rows, and opens the row it reaches.
func (s *Settings) navMove(dir int) {
	at := 0
	for i, p := range panes {
		if p == s.pane {
			at = i
			break
		}
	}
	next := panes[max(0, min(at+dir, len(panes)-1))]
	if next != s.pane {
		s.openPane(next)
	}
	s.navFocus = &next
}

// navAgentAlert reports whether an agent is on this computer without Withdo's MCP: the square the
// Agents row carries while its pane is closed.
func (s *Settings) navAgentAlert() bool {
	status := s.scanAgents()
	if status == nil {
		return false
	}
	reported := s.reports()
	for _, a := range agentsetup.Agents {
		st := status[a.ID]
		installed := st.Installed || (reported[a.Name].ever && !st.Legacy)
		if st.Found && !installed {
			return true
		}
	}
	return false
}

func (s *Settings) paneBody(c *ui.Context, t *theme.Tokens) {
	switch s.pane {
	case PaneAccount:
		s.paneAccount(c, t)
	case PaneCategories:
		s.categoriesPane(c, t)
	case PaneAgents:
		s.agentsPane(c, t)
	case PaneModel:
		s.modelPane(c, t)
	case PaneAbout:
		s.aboutPane(c, t)
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

// settingsOutLink is a settingsLink that leaves the app (a web page, System Settings): the pixel
// external-link icon follows the words, never a text arrow.
func settingsOutLink(c *ui.Context, t *theme.Tokens, key, text, label string) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Padding(2, 4).Radius(theme.RadiusControl).Shrink(0).Label(label).Cursor(ui.CursorPointer)
	ink := t.Accent
	if btn.Hovered() {
		ink = t.Accent.Mix(t.Text, 0.2)
	}
	btn.Children(func() {
		ui.Row(c.Key(key + "-in")).Gap(4).AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.Body, ink, text)
			ui.Icon(c.Key(key+"-out"), theme.IconArrowOut).Size(theme.IconSmall, theme.IconSmall).TextColor(ink).Shrink(0)
		})
	})
	return btn
}
