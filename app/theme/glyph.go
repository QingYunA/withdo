package theme

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// StatusGlyph draws a status as a pixel glyph on a 12 x 12 grid, GlyphSize points square (1.5 pt a
// pixel). The shape carries the meaning, the color repeats it. complete runs from 0 to 1 while a
// task is being marked done, in two steps: the disc fills, then the check appears.
func StatusGlyph(c *ui.Context, t *Tokens, s data.Status, complete float32) ui.Element {
	return StatusGlyphSized(c, t, s, GlyphSize, complete)
}

// StatusGlyphSized is StatusGlyph at another size: a card's blocked mark is 12 pt. Whole
// multiples of 12 (12, 24, 36) put every pixel on the device grid; 18 does on a Retina screen.
func StatusGlyphSized(c *ui.Context, t *Tokens, s data.Status, size, complete float32) ui.Element {
	return ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		drawStatus(p, r, t, s, complete)
	})
}

// The 12 x 12 grids of docs/design/mockups/v6.html. A cell is a pixel: the ring is the stepped
// circle, the disc is the same circle filled.
var (
	glyphDisc = []string{"....####....", "..########..", ".##########.", ".##########.", "############", "############",
		"############", "############", ".##########.", ".##########.", "..########..", "....####...."}
	glyphRing = []string{"....####....", "..##....##..", ".#........#.", ".#........#.", "#..........#", "#..........#",
		"#..........#", "#..........#", ".#........#.", ".#........#.", "..##....##..", "....####...."}
)

// Pixels laid over the ring or the disc, as x, y cells.
var (
	glyphDoingHalf   = [][2]int{{6, 3}, {7, 3}, {6, 4}, {7, 4}, {8, 4}, {6, 5}, {7, 5}, {8, 5}, {6, 6}, {7, 6}, {8, 6}, {6, 7}, {7, 7}, {8, 7}, {6, 8}, {7, 8}}
	glyphReviewDot   = [][2]int{{5, 4}, {6, 4}, {4, 5}, {5, 5}, {6, 5}, {7, 5}, {4, 6}, {5, 6}, {6, 6}, {7, 6}, {5, 7}, {6, 7}}
	glyphCheck       = [][2]int{{3, 6}, {3, 7}, {4, 7}, {4, 8}, {5, 8}, {5, 9}, {6, 7}, {6, 8}, {7, 6}, {7, 7}, {8, 5}, {8, 6}, {9, 4}, {9, 5}}
	glyphBang        = [][2]int{{5, 2}, {6, 2}, {5, 3}, {6, 3}, {5, 4}, {6, 4}, {5, 5}, {6, 5}, {5, 6}, {6, 6}, {5, 8}, {6, 8}, {5, 9}, {6, 9}}
	glyphCancelSlash = [][2]int{{3, 8}, {4, 7}, {5, 6}, {6, 5}, {7, 4}, {8, 3}}
)

// glyphRows lays the cells over a grid: 'c' is the status color, 'w' the white of a mark on a disc.
func glyphRows(base []string, ch byte, cells ...[][2]int) []string {
	rows := make([][]byte, len(base))
	for y, r := range base {
		rows[y] = []byte(r)
		for x := range rows[y] {
			if rows[y][x] == '#' {
				rows[y][x] = 'c'
			}
		}
	}
	for _, set := range cells {
		for _, xy := range set {
			rows[xy[1]][xy[0]] = ch
		}
	}
	out := make([]string, len(rows))
	for y, r := range rows {
		out[y] = string(r)
	}
	return out
}

// dashedRing lists every third pixel of the ring, in reading order: the backlog's mark leaves
// them out.
func dashedRing() [][2]int {
	var drop [][2]int
	i := 0
	for y, r := range glyphRing {
		for x := range r {
			if r[x] == '#' {
				if i%3 == 2 {
					drop = append(drop, [2]int{x, y})
				}
				i++
			}
		}
	}
	return drop
}

// glyphSprites are the pixels of every status; the 'c' pixels take the status color at draw time.
var glyphSprites = map[data.Status][]string{
	data.Backlog:   glyphRows(glyphRing, '.', dashedRing()),
	data.Todo:      glyphRows(glyphRing, 'c'),
	data.Doing:     glyphRows(glyphRing, 'c', glyphDoingHalf),
	data.Review:    glyphRows(glyphRing, 'c', glyphReviewDot),
	data.Blocked:   glyphRows(glyphDisc, 'w', glyphBang),
	data.Done:      glyphRows(glyphDisc, 'w', glyphCheck),
	data.Cancelled: glyphRows(glyphRing, 'c', glyphCancelSlash),
}

// glyphDiscOnly is the disc of a task that is being completed, before its check.
var glyphDiscOnly = glyphRows(glyphDisc, 'c')

var white = ui.Hex("#FFFFFF")

func drawStatus(p *ui.Painter, r ui.Rect, t *Tokens, s data.Status, complete float32) {
	if complete > 0 {
		drawCompleting(p, r, t, complete)
		return
	}
	rows, ok := glyphSprites[s]
	if !ok {
		rows = glyphSprites[data.Todo]
	}
	DrawSprite(p, r, &Sprite{Rows: rows, Palette: map[rune]ui.Color{'c': t.Status(s), 'w': white}})
}

// drawCompleting plays the completion in two steps: the green disc fills, then the check appears.
// Reduce Motion never gets here with less than 1, so it goes straight to done.
func drawCompleting(p *ui.Painter, r ui.Rect, t *Tokens, k float32) {
	done := t.Status(data.Done)
	if k < completeCheckAt {
		DrawSprite(p, r, &Sprite{Rows: glyphDiscOnly, Palette: map[rune]ui.Color{'c': done}})
		return
	}
	DrawSprite(p, r, &Sprite{Rows: glyphSprites[data.Done], Palette: map[rune]ui.Color{'c': done, 'w': white}})
}

// completeCheckAt is how far through the completion the check appears.
const completeCheckAt float32 = 0.5

// RadioGlyph draws a choice's radio on the status ring's grid at one point a pixel: the ring, and
// when on, the centered dot of the review glyph inside it.
func RadioGlyph(c *ui.Context, col ui.Color, on bool) ui.Element {
	rows := glyphRadioOff
	if on {
		rows = glyphRadioOn
	}
	return ui.Box(c).Size(IconSmall, IconSmall).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		DrawSprite(p, r, &Sprite{Rows: rows, Palette: map[rune]ui.Color{'c': col}})
	})
}

var (
	glyphRadioOff = glyphRows(glyphRing, 'c')
	glyphRadioOn  = glyphRows(glyphRing, 'c', glyphReviewDot)
)

// ShowsPriority reports whether a priority is drawn on a row or a card: only the deliberate ones.
// P2 is the default and P3 means "whenever", so both stay quiet.
func ShowsPriority(p data.Priority) bool { return p == data.P0 || p == data.P1 }

// PriorityBadge is the small outlined "P0" or "P1" of a row or a card: P0 in the blocked color,
// P1 in secondary text.
func PriorityBadge(c *ui.Context, t *Tokens, p data.Priority) ui.Element {
	ink, ring := t.TextSecondary, t.Border
	if p == data.P0 {
		ink, ring = t.Blocked(), t.Blocked().Alpha(0.35)
	}
	return ui.Row(c.Key("badge-"+string(p))).Shrink(0).Padding(0, 5).Radius(5).Border(Hairline, ring).AlignItems(ui.Center).
		Children(func() { Text(c, Badge, ink, string(p)) })
}

// ClockGlyph draws a small clock, the mark of a reminder, at one point a pixel.
func ClockGlyph(c *ui.Context, col ui.Color) ui.Element {
	return ui.Icon(c, IconClock).Size(IconSmall, IconSmall).TextColor(col).Shrink(0)
}

// Dot draws a filled dot of the given diameter.
func Dot(c *ui.Context, col ui.Color, size float32) ui.Element {
	return ui.Box(c).Size(size, size).Shrink(0).Radius(size / 2).Background(col)
}
