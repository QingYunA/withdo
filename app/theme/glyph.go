package theme

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

const (
	ringWidth float32 = 1.5
	kappa     float32 = 0.5522847498 // cubic Bézier circle constant
)

// StatusGlyph draws a status as a vector glyph, GlyphSize points square. The shape carries the meaning, the color
// repeats it. complete runs from 0 to 1 while a task is being marked done: the ring fills and the
// check draws itself.
func StatusGlyph(c *ui.Context, t *Tokens, s data.Status, complete float32) ui.Element {
	return StatusGlyphSized(c, t, s, GlyphSize, complete)
}

// StatusGlyphSized is StatusGlyph at another size: the blocked mark in a card's meta line is 14 pt.
func StatusGlyphSized(c *ui.Context, t *Tokens, s data.Status, size, complete float32) ui.Element {
	return ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		drawStatus(p, r, t, s, complete)
	})
}

func drawStatus(p *ui.Painter, r ui.Rect, t *Tokens, s data.Status, complete float32) {
	if complete > 0 {
		drawCompleting(p, r, t, s, complete)
		return
	}
	col := t.Status(s)
	u := r.W / 16 // the shapes are drawn on a 16 unit grid and scaled to the glyph
	ring := ui.Rect{X: r.X + u, Y: r.Y + u, W: r.W - 2*u, H: r.H - 2*u}
	switch s {
	case data.Backlog:
		p.StrokeDashed(ring, col, ring.W/2, ringWidth*u)
	case data.Todo:
		p.Stroke(ring, col, ring.W/2, ringWidth*u)
	case data.Doing:
		p.Stroke(ring, col, ring.W/2, ringWidth*u)
		fillRightHalf(p, r, 4.2*u, col)
	case data.Blocked:
		p.Fill(ring, col, ring.W/2)
		drawBang(p, r, ui.Hex("#FFFFFF"))
	case data.Review:
		p.Stroke(ring, col, ring.W/2, ringWidth*u)
		p.Fill(dotRect(r, 2.6*u), col, 2.6*u)
	case data.Done:
		p.Fill(ring, col, ring.W/2)
		drawCheck(p, r, ui.Hex("#FFFFFF"), 1)
	case data.Cancelled:
		p.Stroke(ring, col, ring.W/2, ringWidth*u)
		slash := (&ui.Path{}).MoveTo(r.X+5*u, r.Y+11*u).LineTo(r.X+11*u, r.Y+5*u)
		p.StrokePath(slash, ringWidth*u, col)
	}
}

// drawCompleting plays the 220 ms fill-and-check: the green disc grows out of the ring, then the
// check draws itself.
func drawCompleting(p *ui.Painter, r ui.Rect, t *Tokens, from data.Status, k float32) {
	done := t.Status(data.Done)
	u := r.W / 16
	ring := ui.Rect{X: r.X + u, Y: r.Y + u, W: r.W - 2*u, H: r.H - 2*u}
	p.Stroke(ring, t.Status(from).Mix(done, k), ring.W/2, ringWidth*u)
	fill := min(k*1.6, 1)
	rad := ring.W / 2 * fill
	p.Fill(dotRect(r, rad), done, rad)
	if k > 0.35 {
		drawCheck(p, r, ui.Hex("#FFFFFF"), min((k-0.35)/0.65, 1))
	}
}

func dotRect(r ui.Rect, radius float32) ui.Rect {
	return ui.Rect{X: r.X + r.W/2 - radius, Y: r.Y + r.H/2 - radius, W: 2 * radius, H: 2 * radius}
}

// fillRightHalf fills the right half of a disc centered in r.
func fillRightHalf(p *ui.Painter, r ui.Rect, radius float32, col ui.Color) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	k := kappa * radius
	path := (&ui.Path{}).MoveTo(cx, cy-radius).
		CubeTo(cx+k, cy-radius, cx+radius, cy-k, cx+radius, cy).
		CubeTo(cx+radius, cy+k, cx+k, cy+radius, cx, cy+radius).
		Close()
	p.FillPath(path, col)
}

func drawBang(p *ui.Painter, r ui.Rect, col ui.Color) {
	u := r.W / 16
	cx := r.X + r.W/2
	stem := (&ui.Path{}).MoveTo(cx, r.Y+4.4*u).LineTo(cx, r.Y+8.6*u)
	p.StrokePath(stem, 1.7*u, col)
	p.Fill(dotRect(ui.Rect{X: cx - 4*u, Y: r.Y + 11.3*u - 4*u, W: 8 * u, H: 8 * u}, 0.95*u), col, 0.95*u)
}

func drawCheck(p *ui.Painter, r ui.Rect, col ui.Color, progress float32) {
	u := r.W / 16
	check := (&ui.Path{}).MoveTo(r.X+4.8*u, r.Y+8.3*u).LineTo(r.X+7.1*u, r.Y+10.6*u).LineTo(r.X+11.4*u, r.Y+5.7*u)
	p.StrokePath(check.Trim(0, progress), 1.7*u, col)
}

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
	return ui.Row(c.Key("badge-"+string(p))).Shrink(0).Padding(0, 5).Radius(5).Border(1, ring).AlignItems(ui.Center).
		Children(func() { Text(c, Badge, ink, string(p)) })
}

// ClockGlyph draws a small clock, the mark of a reminder.
func ClockGlyph(c *ui.Context, col ui.Color) ui.Element {
	return ui.Box(c).Size(12, 12).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		ring := ui.Rect{X: r.X + 0.75, Y: r.Y + 0.75, W: r.W - 1.5, H: r.H - 1.5}
		p.Stroke(ring, col, ring.W/2, 1.25)
		hands := (&ui.Path{}).MoveTo(r.X+6, r.Y+3.2).LineTo(r.X+6, r.Y+6).LineTo(r.X+8, r.Y+7.2)
		p.StrokePath(hands, 1.15, col)
	})
}

// Dot draws a filled dot of the given diameter.
func Dot(c *ui.Context, col ui.Color, size float32) ui.Element {
	return ui.Box(c).Size(size, size).Shrink(0).Radius(size / 2).Background(col)
}
