// Package theme holds every design token of the app in one place: color for light and dark, type,
// space, shape and motion, plus the vector glyphs. Views never write a color or a size of their
// own; they ask for a token.
package theme

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Tokens are the colors of one appearance.
type Tokens struct {
	Dark bool

	Bg            ui.Color // window content
	BgSubtle      ui.Color // board wells, input wells
	BgTray        ui.Color // the Agents column of a project: half as dark as BgSubtle
	BgHover       ui.Color // row hover
	BgSelected    ui.Color // selected row, keyboard highlight
	Border        ui.Color // hairlines
	Text          ui.Color
	TextSecondary ui.Color
	TextTertiary  ui.Color
	Accent        ui.Color
	AccentText    ui.Color
	Stale         ui.Color
	Shadow        ui.Color // floating things only

	status map[data.Status]ui.Color
}

// Status returns the muted color of a status.
func (t *Tokens) Status(s data.Status) ui.Color {
	if c, ok := t.status[s]; ok {
		return c
	}
	return t.TextTertiary
}

// Blocked is the only red in the app.
func (t *Tokens) Blocked() ui.Color { return t.Status(data.Blocked) }

var light = &Tokens{
	Bg:            ui.Hex("#FFFFFF"),
	BgSubtle:      ui.Hex("#F5F5F7"),
	BgTray:        ui.Hex("#F8F8FA"),
	BgHover:       ui.Hex("#0000000A"),
	Border:        ui.Hex("#0000001A"),
	Text:          ui.Hex("#1D1D1F"),
	TextSecondary: ui.Hex("#6E6E73"),
	TextTertiary:  ui.Hex("#AEAEB2"),
	Accent:        ui.Hex("#5B5BD6"),
	AccentText:    ui.Hex("#FFFFFF"),
	Stale:         ui.Hex("#D97706"),
	Shadow:        ui.RGBA(0, 0, 0, 0.26),
	status: map[data.Status]ui.Color{
		data.Backlog:   ui.Hex("#AEAEB2"),
		data.Todo:      ui.Hex("#8E8E93"),
		data.Doing:     ui.Hex("#3B82F6"),
		data.Blocked:   ui.Hex("#E5484D"),
		data.Review:    ui.Hex("#E0A800"),
		data.Done:      ui.Hex("#30A46C"),
		data.Cancelled: ui.Hex("#AEAEB2"),
	},
}

var dark = &Tokens{
	Dark:          true,
	Bg:            ui.Hex("#1C1C1E"),
	BgSubtle:      ui.Hex("#232326"),
	BgTray:        ui.Hex("#202023"),
	BgHover:       ui.Hex("#FFFFFF0F"),
	Border:        ui.Hex("#FFFFFF1A"),
	Text:          ui.Hex("#F5F5F7"),
	TextSecondary: ui.Hex("#A1A1A6"),
	TextTertiary:  ui.Hex("#636366"),
	Accent:        ui.Hex("#7C7CF0"),
	AccentText:    ui.Hex("#FFFFFF"),
	Stale:         ui.Hex("#F59E0B"),
	Shadow:        ui.RGBA(0, 0, 0, 0.5),
	status: map[data.Status]ui.Color{
		data.Backlog:   ui.Hex("#636366"),
		data.Todo:      ui.Hex("#8E8E93"),
		data.Doing:     ui.Hex("#60A5FA"),
		data.Blocked:   ui.Hex("#FF6369"),
		data.Review:    ui.Hex("#F5C542"),
		data.Done:      ui.Hex("#4CC38A"),
		data.Cancelled: ui.Hex("#636366"),
	},
}

// For returns the tokens of an appearance. accent, when it has alpha, replaces the default
// accent: the system accent the user chose.
func For(isDark bool, accent ui.Color) *Tokens {
	base := light
	if isDark {
		base = dark
	}
	t := *base
	if accent.A > 0 {
		t.Accent = accent
		t.AccentText = ui.Hex("#FFFFFF")
	}
	if isDark {
		t.BgSelected = t.Accent.Alpha(0.22)
	} else {
		t.BgSelected = t.Accent.Alpha(0.12)
	}
	return &t
}

// Resolve returns the tokens for the frame and makes the window's widget theme follow them. It
// uses the system accent only when the user changed it from the macOS default blue.
func Resolve(c *ui.Context) *Tokens {
	sys := c.Preferences().Accent
	if isDefaultSystemAccent(sys) {
		sys = ui.Transparent
	}
	t := For(c.Theme().Dark, sys)
	c.SetTheme(t.widgetTheme())
	return t
}

// isDefaultSystemAccent recognizes macOS's default blue in both appearances.
func isDefaultSystemAccent(c ui.Color) bool {
	if c.A == 0 {
		return true
	}
	near := func(a, b uint8) bool { return int(a)+6 >= int(b) && int(b)+6 >= int(a) }
	for _, d := range [][3]uint8{{0, 122, 255}, {10, 132, 255}, {0, 125, 255}} {
		if near(c.R, d[0]) && near(c.G, d[1]) && near(c.B, d[2]) {
			return true
		}
	}
	return false
}

// widgetTheme maps the tokens onto mygo's theme so the controls (text inputs, menus, scroll bars)
// match the views drawn by hand.
func (t *Tokens) widgetTheme() *ui.Theme {
	w := ui.LightTheme()
	if t.Dark {
		w = ui.DarkTheme()
	}
	w.Background, w.Surface = t.Bg, t.BgSubtle
	w.SurfaceHover = t.BgSubtle.Mix(t.Text, 0.06)
	w.SurfacePressed = t.BgSubtle.Mix(t.Text, 0.12)
	w.Border = t.Border
	w.Text, w.TextMuted = t.Text, t.TextSecondary
	w.Accent, w.AccentText = t.Accent, t.AccentText
	w.AccentHover, w.AccentPressed = t.Accent.Mix(t.Text, 0.12), t.Accent.Mix(t.Text, 0.24)
	w.Danger, w.Warning, w.Success = t.Blocked(), t.Stale, t.Status(data.Done)
	w.Selection, w.Focus = t.Accent.Alpha(0.3), t.Accent.Alpha(0.55)
	w.Scrollbar = t.Text.Alpha(0.28)
	w.Radius, w.Spacing, w.FontSize = RadiusControl, 4, Body.Size
	return w
}

// Radii in points.
const (
	RadiusControl float32 = 6  // pills, inputs
	RadiusCard    float32 = 9  // cards, rows
	RadiusPanel   float32 = 12 // composer, From your agents container, popovers, sheets
	RadiusTile    float32 = 10 // board cards
)

// Space on the 4 pt grid.
const (
	Space4  float32 = 4
	Space8  float32 = 8
	Space12 float32 = 12
	Space16 float32 = 16
	Space24 float32 = 24
)

// Motion durations.
const (
	MotionFast   = 160 * time.Millisecond // hover, selection, drawer
	MotionSpring = 220 * time.Millisecond // card settling, completing
)

// Layout metrics shared by several surfaces.
const (
	GlyphSize      float32 = 18
	BadgeSize      float32 = 18      // the red count beside Mine
	IconProp       float32 = IconRow // the icon before a property's label in the inspector
	IconMenu       float32 = IconRow // the icon before an item of the drawn task menu
	HitTarget      float32 = 24
	RowOneLine     float32 = 32
	SidebarWidth   float32 = 216
	InspectorWidth float32 = 360
	SidebarRow     float32 = 32  // a row of the sidebar, in the 14 pt sidebar size
	ContentMax     float32 = 780 // the centered column of lists
	ContentPad     float32 = 48  // its side padding when there is room
	AskHeight      float32 = 46
	BoardMax       float32 = 980 // the board's block (header, composer and lanes) at its widest
	MinColumnWidth float32 = 176 // a board lane at its narrowest, before the board scrolls
	ColumnGap      float32 = 20

	WindowMinWidth  = 860
	WindowMinHeight = 540
	WindowWidth     = 1100
	WindowHeight    = 720
	DoneRecentLimit = 5 // Done cards the board's Done lane shows before "Show all"
	DoneListRows    = 3 // Done rows a list shows before "Show all"
)
