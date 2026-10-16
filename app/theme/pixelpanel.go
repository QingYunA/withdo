package theme

import "github.com/egoist/mygo/ui"

// The day card is a pixel panel (docs/design/mockups/v8.html, B): its corners step in twice by
// PixelUnit, it has a PixelUnit edge in PixelEdge, and a hard shadow PixelShadowOffset right and
// down, not blurred, in PixelShadow. Its input well steps in once. Colors are the interface's own.
const (
	// PixelUnit is one pixel of the panel: a step of a corner, and the edge's width.
	PixelUnit float32 = 2
	// PixelShadowOffset is how far right and down the shadow falls.
	PixelShadowOffset float32 = 4
)

// PixelEdge is the edge of a pixel panel: a fifth of the way from the background to the text.
func (t *Tokens) PixelEdge() ui.Color { return t.Bg.Mix(t.Text, 0.2) }

// PixelShadow is the hard shadow of a pixel panel: faint in light mode, near black in dark.
func (t *Tokens) PixelShadow() ui.Color {
	if t.Dark {
		return ui.RGBA(0, 0, 0, 0.5)
	}
	return ui.RGBA(0, 0, 0, 0.1)
}

// DrawPixelPanel draws a pixel panel in r: the shadow beyond r's right and bottom, the edge, and
// the fill in t.Bg. Whatever draws it leaves PixelShadowOffset of room for the shadow.
func DrawPixelPanel(p *ui.Painter, r ui.Rect, t *Tokens) {
	shadow := ui.Rect{X: r.X + PixelShadowOffset, Y: r.Y + PixelShadowOffset, W: r.W, H: r.H}
	inner := ui.Rect{X: r.X + PixelUnit, Y: r.Y + PixelUnit, W: r.W - 2*PixelUnit, H: r.H - 2*PixelUnit}
	p.FillPath(steppedPath(shadow, 2), t.PixelShadow())
	p.FillPath(steppedPath(r, 2), t.PixelEdge())
	p.FillPath(steppedPath(inner, 1), t.Bg)
}

// DrawPixelWell fills r with its corners stepped in once.
func DrawPixelWell(p *ui.Painter, r ui.Rect, fill ui.Color) { p.FillPath(steppedPath(r, 1), fill) }

// steppedPath is r with each corner cut in steps stairs of PixelUnit.
func steppedPath(r ui.Rect, steps int) *ui.Path {
	n := float32(steps)
	// The top left corner, clockwise, in pixels from it: down the left edge, then stair by stair
	// up and right onto the top edge.
	corner := [][2]float32{{0, n}}
	for i := float32(1); i <= n; i++ {
		corner = append(corner, [2]float32{i, n - i + 1}, [2]float32{i, n - i})
	}
	right, bottom := r.X+r.W, r.Y+r.H
	// Each corner is the top left one turned a quarter clockwise more.
	turns := [4]func(a, b float32) (float32, float32){
		func(a, b float32) (float32, float32) { return r.X + a, r.Y + b },
		func(a, b float32) (float32, float32) { return right - b, r.Y + a },
		func(a, b float32) (float32, float32) { return right - a, bottom - b },
		func(a, b float32) (float32, float32) { return r.X + b, bottom - a },
	}
	path := &ui.Path{}
	for k, at := range turns {
		for j, pt := range corner {
			x, y := at(pt[0]*PixelUnit, pt[1]*PixelUnit)
			if k == 0 && j == 0 {
				path.MoveTo(x, y)
			} else {
				path.LineTo(x, y)
			}
		}
	}
	return path.Close()
}
