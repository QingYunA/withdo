package theme

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

const (
	ringWidth float32 = 1.5
	kappa     float32 = 0.5522847498 // cubic Bézier circle constant
)

// StatusGlyph draws a status as a 16 pt vector glyph. The shape carries the meaning, the color
// repeats it. complete runs from 0 to 1 while a task is being marked done: the ring fills and the
// check draws itself.
func StatusGlyph(c *ui.Context, t *Tokens, s data.Status, complete float32) ui.Element {
	return ui.Box(c).Size(GlyphSize, GlyphSize).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		drawStatus(p, r, t, s, complete)
	})
}

func drawStatus(p *ui.Painter, r ui.Rect, t *Tokens, s data.Status, complete float32) {
	if complete > 0 {
		drawCompleting(p, r, t, s, complete)
		return
	}
	col := t.Status(s)
	ring := ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
	switch s {
	case data.Backlog:
		p.StrokeDashed(ring, col, ring.W/2, ringWidth)
	case data.Todo:
		p.Stroke(ring, col, ring.W/2, ringWidth)
	case data.Doing:
		p.Stroke(ring, col, ring.W/2, ringWidth)
		fillRightHalf(p, r, 4.2, col)
	case data.Blocked:
		p.Fill(ring, col, ring.W/2)
		drawBang(p, r, ui.Hex("#FFFFFF"))
	case data.Review:
		p.Stroke(ring, col, ring.W/2, ringWidth)
		p.Fill(dotRect(r, 2.6), col, 2.6)
	case data.Done:
		p.Fill(ring, col, ring.W/2)
		drawCheck(p, r, ui.Hex("#FFFFFF"), 1)
	case data.Cancelled:
		p.Stroke(ring, col, ring.W/2, ringWidth)
		slash := (&ui.Path{}).MoveTo(r.X+5, r.Y+11).LineTo(r.X+11, r.Y+5)
		p.StrokePath(slash, ringWidth, col)
	}
}

// drawCompleting plays the 220 ms fill-and-check: the green disc grows out of the ring, then the
// check draws itself.
func drawCompleting(p *ui.Painter, r ui.Rect, t *Tokens, from data.Status, k float32) {
	done := t.Status(data.Done)
	ring := ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
	p.Stroke(ring, t.Status(from).Mix(done, k), ring.W/2, ringWidth)
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
	cx := r.X + r.W/2
	stem := (&ui.Path{}).MoveTo(cx, r.Y+4.4).LineTo(cx, r.Y+8.6)
	p.StrokePath(stem, 1.7, col)
	p.Fill(dotRect(ui.Rect{X: cx - 4, Y: r.Y + 11.3 - 4, W: 8, H: 8}, 0.95), col, 0.95)
}

func drawCheck(p *ui.Painter, r ui.Rect, col ui.Color, progress float32) {
	check := (&ui.Path{}).MoveTo(r.X+4.8, r.Y+8.3).LineTo(r.X+7.1, r.Y+10.6).LineTo(r.X+11.4, r.Y+5.7)
	p.StrokePath(check.Trim(0, progress), 1.7, col)
}

// PriorityBars draws three ascending bars. P0 is red, P1 three bars, P2 two, P3 one; bars beyond
// the level stay faint so the set always reads as "n of three".
func PriorityBars(c *ui.Context, t *Tokens, pr data.Priority) ui.Element {
	return ui.Box(c).Size(12, 12).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		level, col := priorityLevel(pr), t.TextSecondary
		if pr == data.P0 {
			col = t.Blocked()
		}
		const w, gap float32 = 2.5, 1.75
		heights := [3]float32{4.5, 7.5, 11}
		for i, h := range heights {
			bar := ui.Rect{X: r.X + float32(i)*(w+gap) + 0.5, Y: r.Y + r.H - h, W: w, H: h}
			if i < level {
				p.Fill(bar, col, 1)
			} else {
				p.Fill(bar, t.TextTertiary.Alpha(0.45), 1)
			}
		}
	})
}

func priorityLevel(pr data.Priority) int {
	switch pr {
	case data.P0, data.P1:
		return 3
	case data.P2:
		return 2
	}
	return 1
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
