package theme

import (
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The important star of the user's own tasks (v31): a 12 x 12 pixel drawing like every icon, from the
// mockup's PIX.star and PIX.staro. IconImportant is filled (set, in status.review); IconImportantOff
// is the outline the inspector shows while it is not set. They fill with currentColor, so ui.Icon
// tints them, and are drawn only at 12, 18, 24 or 36 pt.
var (
	starFilled = []string{
		".....##.....", ".....##.....", "....####....", "....####....", "############", ".##########.",
		"..########..", "...######...", "...######...", "..###..###..", "..##....##..", ".##......##.",
	}
	starOutline = []string{
		".....##.....", ".....##.....", "....#..#....", "....#..#....", "#####..#####", ".#........#.",
		"..#......#..", "...#....#...", "...#....#...", "..#..##..#..", "..#.#..#.#..", ".##......##.",
	}

	IconImportant    = gridIcon(starFilled)
	IconImportantOff = gridIcon(starOutline)
)

// gridIcon turns a 12 x 12 grid of '#' pixels into a currentColor icon on the 24 unit grid: one
// 2-unit square per lit pixel, like the pixelarticons.
func gridIcon(rows []string) *ui.SVG {
	var d strings.Builder
	for y, r := range rows {
		for x, ch := range r {
			if ch == '#' {
				fmt.Fprintf(&d, "M%d %dh2v2h-2Z", x*2, y*2)
			}
		}
	}
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 24 24"><path d="` + d.String() + `"/></svg>`))
}

// Important is the color of the set star: status.review, the one warm color of the palette.
func (t *Tokens) Important() ui.Color { return t.Status(data.Review) }

// Star draws the important star at size pt (12, 18 or 24): filled in status.review when on, else
// the outline in text.tertiary.
func Star(c *ui.Context, t *Tokens, on bool, size float32) ui.Element {
	if on {
		return ui.Icon(c, IconImportant).Size(size, size).TextColor(t.Important()).Shrink(0)
	}
	return ui.Icon(c, IconImportantOff).Size(size, size).TextColor(t.TextTertiary).Shrink(0)
}
