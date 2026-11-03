package theme

import (
	"fmt"
	"image/color"
	"sync"

	"github.com/QingYunA/adhd-todo/app/categoryicons"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// A category's icon is a 12 x 12 picture kept in the database. The picker offers the library of
// app/categoryicons; this file draws it.

// CategoryIconEntry, CategoryIconGroup and the lookups are app/categoryicons', which the phone's
// core reads too.
type (
	CategoryIconEntry = categoryicons.Entry
	CategoryIconGroup = categoryicons.Group
)

var (
	// CategoryIcons is the whole library, sorted by name.
	CategoryIcons = categoryicons.All
	// CategoryIconGroups is the curated list, in the order the picker shows it.
	CategoryIconGroups = categoryicons.Groups
)

// CategoryIconNamed finds a library icon by its pixelarticons name.
func CategoryIconNamed(name string) (CategoryIconEntry, bool) { return categoryicons.Named(name) }

// SearchCategoryIcons lists the icons whose name or Chinese keywords contain q: see
// categoryicons.Search.
func SearchCategoryIcons(q string) []CategoryIconEntry { return categoryicons.Search(q) }

// PaletteColors are the colors of the letters a to l in a category's icon: the project colors (a to
// h) and white, gray, black and brown. They are the same in light and dark mode.
var PaletteColors = [pixmap.Colors]ui.Color{
	ProjectColors[0], ProjectColors[1], ProjectColors[2], ProjectColors[3],
	ProjectColors[4], ProjectColors[5], ProjectColors[6], ProjectColors[7],
	ui.Hex("#FFFFFF"), ui.Hex("#8E8E93"), ui.Hex("#1C1C1E"), ui.Hex("#8A5A3B"),
}

// PaletteRGBA is PaletteColors for the image package, to match the colors of a picture to the palette.
func PaletteRGBA() (out [pixmap.Colors]color.RGBA) {
	for i, c := range PaletteColors {
		out[i] = color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
	}
	return out
}

// PaletteColor is the color of a palette letter.
func PaletteColor(letter byte) ui.Color { return PaletteColors[letter-pixmap.FirstColor] }

var (
	categorySVGMu sync.Mutex
	categorySVGs  = map[pixmap.Picture]*ui.SVG{}
)

// CategoryIcon is the picture as an SVG: its Own pixels in the color of the surrounding text
// (tint it with TextColor), its palette pixels in their own colors. Draw it with ui.Image, which
// keeps the SVG's own colors; ui.Icon would flatten it to one. Parsed once per picture. Draw it at
// the same whole multiples of 12 pt as the interface icons.
func CategoryIcon(p pixmap.Picture) *ui.SVG {
	categorySVGMu.Lock()
	defer categorySVGMu.Unlock()
	if s, ok := categorySVGs[p]; ok {
		return s
	}
	var fills [pixmap.Colors]string
	for i, c := range PaletteColors {
		fills[i] = fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	s := ui.MustParseSVG(p.SVG(fills))
	categorySVGs[p] = s
	return s
}
