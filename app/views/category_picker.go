package views

import (
	"slices"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// The color swatches and the icon picker of the category sheet.

const (
	swatchCell   float32 = 24 // a swatch's hit area; its ring is drawn at this size
	swatchSize   float32 = 18
	pickerHeight float32 = 240
	iconCell     float32 = 36
	iconsPerRow          = 9
)

// swatchDisc and swatchRing are a color's 12 x 12 pixel circle and the ring around the current one.
var swatchDisc, swatchRing = swatchPictures()

func swatchPictures() (disc, ring pixmap.Picture) {
	circle, err := pixmap.ParseRows([]string{
		"....####....",
		"..########..",
		".##########.",
		".##########.",
		"############",
		"############",
		"############",
		"############",
		".##########.",
		".##########.",
		"..########..",
		"....####....",
	})
	if err != nil {
		panic(err)
	}
	disc = circle.Picture()
	return disc, disc.Outline()
}

// categorySwatches is the eight colors a category can have, in the order of the project colors; the
// current one has a ring.
func (m *Main) categorySwatches(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	ui.Row(c).Shrink(0).Gap(8).Children(func() {
		for _, key := range data.CategoryColors {
			on := s.color == key
			btn := ui.ButtonBase(c.Key("swatch-"+key)).Size(swatchCell, swatchCell).Shrink(0).Center().
				Label(m.say.A11yCategoryColor(m.categoryColorName(key))).Cursor(ui.CursorPointer)
			btn.Children(func() {
				if on {
					ui.Icon(c, theme.CategoryIcon(swatchRing)).Size(swatchCell, swatchCell).TextColor(t.Text).Absolute().Left(0).Top(0)
				}
				ui.Icon(c, theme.CategoryIcon(swatchDisc)).Size(swatchSize, swatchSize).TextColor(theme.CategoryColor(key))
			})
			if btn.Clicked() {
				s.color, s.picked = key, true
			}
		}
	})
}

// categoryPicker is the search field and the 240 pt area under it: the curated groups while nothing
// is typed, the icons whose name holds the text after.
func (m *Main) categoryPicker(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	ui.Row(c.Key("icon-search")).Height(30).Shrink(0).Padding(0, 10).Gap(8).Radius(theme.RadiusControl).Background(t.BgSubtle).
		AlignItems(ui.Center).Children(func() {
		ui.Icon(c, theme.IconSearch).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary).Shrink(0)
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			theme.Body.Apply(ui.TextInputBase(c.Key("icon-search-input"), &s.query)).TextColor(t.Text).Label(m.say.CategorySearchIcons())
			if s.query == "" {
				theme.Text(c, theme.Body, t.TextSecondary, m.say.CategorySearchIcons()).Absolute().Left(0).Top(0).PassThrough()
			}
		})
	})
	ui.Scroll(c.Key("icon-scroll")).Height(pickerHeight).Shrink(0).Radius(theme.RadiusPanel).Border(theme.Hairline, t.Border).Children(func() {
		ui.Column(c).Padding(8).Gap(4).Children(func() {
			if strings.TrimSpace(s.query) != "" {
				m.iconResults(c, t, s, theme.SearchCategoryIcons(s.query))
				return
			}
			for _, g := range theme.CategoryIconGroups {
				theme.Text(c, theme.Meta, t.TextSecondary, m.categoryGroupName(g.Name)).Margin(4, 0, 2, 0)
				m.iconGrid(c, t, s, g.Icons)
			}
		})
	})
}

// iconResults is what a search found, or the line that says it found nothing.
func (m *Main) iconResults(c *ui.Context, t *theme.Tokens, s *categorySheet, hits []theme.CategoryIconEntry) {
	if len(hits) == 0 {
		theme.Text(c, theme.Body, t.TextSecondary, m.say.CategoryNoIconMatch()).Margin(80, 0, 0, 0).AlignSelf(ui.Center)
		return
	}
	m.iconGrid(c, t, s, hits)
}

// iconGrid lays icons out nine to a row.
func (m *Main) iconGrid(c *ui.Context, t *theme.Tokens, s *categorySheet, icons []theme.CategoryIconEntry) {
	for row := range slices.Chunk(icons, iconsPerRow) {
		ui.Row(c).Shrink(0).Children(func() {
			for _, e := range row {
				m.iconCell(c, t, s, e)
			}
		})
	}
}

// iconCell is one icon of the picker: its name for assistive technology, the icon in the text color,
// and on the chosen one a 2 pt accent ring, a faint accent fill and the icon in the chosen color.
func (m *Main) iconCell(c *ui.Context, t *theme.Tokens, s *categorySheet, e theme.CategoryIconEntry) {
	cell := ui.ButtonBase(c.Key("icon-"+e.Name)).Size(iconCell, iconCell).Shrink(0).Center().Radius(7).Label(e.Name).Cursor(ui.CursorPointer)
	ink := t.Text
	switch {
	case e.Bitmap.Picture() == s.icon:
		cell.Background(t.Accent.Alpha(0.12)).Border(2, t.Accent)
		ink = theme.CategoryColor(s.color)
	case cell.Hovered():
		cell.Background(t.BgHover)
	}
	cell.Children(func() {
		ui.Icon(c, theme.CategoryIcon(e.Bitmap.Picture())).Size(theme.IconRow, theme.IconRow).TextColor(ink)
	})
	if cell.Clicked() {
		s.icon = e.Bitmap.Picture()
	}
}

// categoryGroupName is a curated group's heading in the user's language; the groups are keyed by
// their English name.
func (m *Main) categoryGroupName(key string) string {
	switch key {
	case "Work":
		return m.say.CategoryGroupWork()
	case "Study":
		return m.say.CategoryGroupStudy()
	case "Health":
		return m.say.CategoryGroupHealth()
	case "Money":
		return m.say.CategoryGroupMoney()
	case "Home":
		return m.say.CategoryGroupHome()
	case "Hobby":
		return m.say.CategoryGroupHobby()
	case "Travel":
		return m.say.CategoryGroupTravel()
	case "People":
		return m.say.CategoryGroupPeople()
	}
	return key
}

// categoryColorName is a color's name in the user's language.
func (m *Main) categoryColorName(key string) string {
	switch key {
	case "indigo":
		return m.say.CategoryColorIndigo()
	case "yellow":
		return m.say.CategoryColorYellow()
	case "green":
		return m.say.CategoryColorGreen()
	case "red":
		return m.say.CategoryColorRed()
	case "blue":
		return m.say.CategoryColorBlue()
	case "pink":
		return m.say.CategoryColorPink()
	case "teal":
		return m.say.CategoryColorTeal()
	case "orange":
		return m.say.CategoryColorOrange()
	}
	return key
}
