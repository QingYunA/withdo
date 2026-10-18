package theme

import (
	"github.com/egoist/mygo/ui"
)

// Draw paints the scene in r: the sky over the whole column, with the 264 x 80 grid centred in it,
// then every op of the frame at SceneCell points a cell. A column wider than the grid gets the
// backdrop's first and last columns repeated outward; a narrower one crops both sides alike. The
// pictures are bitmaps drawn in squares of whole cells, so the scene stays pixels at any size.
func (f SceneFrame) Draw(p *ui.Painter, r ui.Rect) {
	p.Clip(r, 0, func() {
		ox := r.X + float32(SceneOffset(r.W))*SceneCell
		sky := fileBitmap(f.Sky.file())
		full := ui.Rect{X: ox, Y: r.Y, W: SceneW * SceneCell, H: SceneH * SceneCell}
		p.ImageRegion(sky, ui.Rect{}, full, ui.ImageOptions{})
		if left := full.X - r.X; left > 0 { // a wider column: the first column of the backdrop, repeated
			p.ImageRegion(sky, ui.Rect{W: 1, H: SceneH}, ui.Rect{X: r.X, Y: r.Y, W: left, H: full.H}, ui.ImageOptions{})
		}
		if right := r.X + r.W - (full.X + full.W); right > 0 {
			p.ImageRegion(sky, ui.Rect{X: SceneW - 1, W: 1, H: SceneH}, ui.Rect{X: full.X + full.W, Y: r.Y, W: right, H: full.H}, ui.ImageOptions{})
		}
		for _, op := range f.Ops {
			op.draw(p, ox, r.Y)
		}
	})
}

// draw paints one op, whose cells are counted from the scene's top left corner at (ox, oy).
func (op SceneOp) draw(p *ui.Painter, ox, oy float32) {
	x, y := ox+float32(op.X)*SceneCell, oy+float32(op.Y)*SceneCell
	if op.Kind == OpRect {
		p.Fill(ui.Rect{X: x, Y: y, W: float32(op.W) * SceneCell, H: float32(op.H) * SceneCell}, op.Color, 0)
		return
	}
	var bmp *ui.Bitmap
	switch op.Kind {
	case OpActor:
		bmp = actorBitmap(op.Actor.Folder, op.Name, op.Actor.Tint)
	case OpGood:
		bmp = fileBitmap("goods/" + op.Name)
	case OpGoodSpin:
		bmp = spinBitmap(op.Name, op.Spin)
	case OpProp:
		bmp = fileBitmap(op.Name)
	}
	w, h := bmp.Size()
	p.ImageRegion(bmp, ui.Rect{}, ui.Rect{X: x, Y: y, W: float32(w) * SceneCell, H: float32(h) * SceneCell}, ui.ImageOptions{FlipX: op.Flip})
}

// digitGlyphs are the 3 x 5 digits and the plus of the pixel counters (+1, +12).
var digitGlyphs = map[rune][5]string{
	'+': {"...", ".x.", "xxx", ".x.", "..."},
	'0': {"xxx", "x.x", "x.x", "x.x", "xxx"},
	'1': {".x.", "xx.", ".x.", ".x.", "xxx"},
	'2': {"xxx", "..x", "xxx", "x..", "xxx"},
	'3': {"xxx", "..x", "xxx", "..x", "xxx"},
	'4': {"x.x", "x.x", "xxx", "..x", "..x"},
	'5': {"xxx", "x..", "xxx", "..x", "xxx"},
	'6': {"xxx", "x..", "xxx", "x.x", "xxx"},
	'7': {"xxx", "..x", ".x.", ".x.", ".x."},
	'8': {"xxx", "x.x", "xxx", "x.x", "xxx"},
	'9': {"xxx", "x.x", "xxx", "..x", "xxx"},
}

var digitOutline = ui.Hex("#3e2731")

// digits draws a number in pixel digits with its top left cell at (x, y): a one-cell dark outline,
// then the digits in fill. Characters other than the digits and the plus are left out.
func digits(s string, x, y int, fill ui.Color) []SceneOp {
	var cells [][2]int
	for i, ch := range s {
		for gy, row := range digitGlyphs[ch] {
			for gx, c := range row {
				if c == 'x' {
					cells = append(cells, [2]int{x + i*4 + gx, y + gy})
				}
			}
		}
	}
	ops := make([]SceneOp, 0, len(cells)*5)
	for _, c := range cells {
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			ops = append(ops, rect(c[0]+d[0], c[1]+d[1], 1, 1, digitOutline))
		}
	}
	for _, c := range cells {
		ops = append(ops, rect(c[0], c[1], 1, 1, fill))
	}
	return ops
}
